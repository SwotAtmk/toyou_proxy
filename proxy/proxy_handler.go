package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"toyou-proxy/config"
	"toyou-proxy/matcher"
	"toyou-proxy/middleware"
)

// ProxyHandler 代理处理器
type ProxyHandler struct {
	hostMatcher     *matcher.HostMatcher
	services        map[string]config.Service
	middlewareChain middleware.MiddlewareChain
	factory         middleware.MiddlewareFactory
	autoPluginMgr   *middleware.AutoPluginManager // 自动插件管理器
	cfg             *config.Config
}

// NewProxyHandler 创建新的代理处理器
func NewProxyHandler(cfg *config.Config) (*ProxyHandler, error) {
	// 初始化中间件服务注册表
	if err := middleware.InitMiddlewareServiceRegistry(cfg); err != nil {
		log.Printf("Failed to initialize middleware service registry: %v", err)
	}

	// 创建中间件工厂
	factory := middleware.NewMiddlewareFactory()

	// 确保缓存目录存在
	cacheDir := "cache/plugins"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Printf("Failed to create cache directory: %v", err)
	}

	// 创建自动插件管理器
	pluginSourceDir := "middleware/plugins"
	autoPluginMgr := middleware.NewAutoPluginManager(pluginSourceDir, cacheDir)

	// 自动发现并注册所有插件
	if err := registerAllPlugins(factory, autoPluginMgr); err != nil {
		log.Printf("Failed to register some plugins: %v", err)
	}

	// 创建域名匹配器
	hostMatcher := matcher.NewHostMatcher()
	for _, rule := range cfg.HostRules {
		hostMatcher.AddRule(rule.Pattern, rule.Target)
		log.Printf("Added host rule: %s -> %s (port: %d)", rule.Pattern, rule.Target, rule.Port)
	}

	// 创建中间件链
	middlewareChain := middleware.NewMiddlewareChain()

	for _, mwConfig := range cfg.Middlewares {
		if !mwConfig.Enabled {
			continue
		}

		mw, err := factory.CreateMiddleware(mwConfig.Name, mwConfig.Config)
		if err != nil {
			log.Printf("Failed to create middleware %s: %v", mwConfig.Name, err)
			continue
		}

		middlewareChain.Add(mw)
		log.Printf("Middleware %s loaded", mwConfig.Name)
	}

	return &ProxyHandler{
		hostMatcher:     hostMatcher,
		services:        cfg.Services,
		middlewareChain: middlewareChain,
		factory:         factory,
		autoPluginMgr:   autoPluginMgr,
		cfg:             cfg,
	}, nil
}

// ServeHTTP 处理HTTP请求
func (ph *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// 创建中间件上下文
	ctx := &middleware.Context{
		Request:  r,
		Response: w,
		Values:   make(map[string]interface{}),
	}

	// 确定目标服务和匹配的路由规则
	targetService, hostRule, routeRule, err := ph.determineTarget(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		log.Printf("Failed to determine target: %v", err)
		return
	}

	// 设置初始目标服务到上下文
	ctx.TargetURL = targetService.URL
	ctx.ServiceName = ph.getServiceName(targetService.URL)

	// 创建动态中间件链
	dynamicMiddlewareChain := ph.createDynamicMiddlewareChain(hostRule, routeRule)

	// 执行中间件链
	if !dynamicMiddlewareChain.Execute(ctx) {
		if ctx.StatusCode != 0 {
			w.WriteHeader(ctx.StatusCode)
		}
		log.Printf("Request aborted by middleware: %s %s", r.Method, r.URL.Path)
		return
	}

	// 检查中间件是否修改了目标服务
	if dynamicTarget, exists := ctx.Values["dynamic_target_service"]; exists {
		if dynamicTargetServiceName, ok := dynamicTarget.(string); ok {
			if service, serviceExists := ph.services[dynamicTargetServiceName]; serviceExists {
				targetService = &service
				ctx.TargetURL = targetService.URL
				ctx.ServiceName = ph.getServiceName(targetService.URL)
				log.Printf("Dynamic routing: redirected to service '%s'", dynamicTargetServiceName)
			} else {
				log.Printf("Dynamic routing: service '%s' not found, using original target", dynamicTargetServiceName)
			}
		}
	}

	// 创建反向代理，传递中间件上下文以支持replace中间件
	proxy, err := ph.createReverseProxy(targetService, ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		log.Printf("Failed to create reverse proxy: %v", err)
		return
	}

	// 执行代理，使用中间件上下文中的Response（可能已被包装）
	proxy.ServeHTTP(ctx.Response, r)

	// 注意：finalize()方法不再需要在这里调用，因为httputil.ReverseProxy
	// 会在请求处理完成后自动完成所有写入操作。我们的replaceResponseWrapper
	// 的Write方法会在每次数据写入时自动应用替换规则。

	// 记录请求完成日志
	duration := time.Since(startTime)
	log.Printf("Proxied: %s %s -> %s [%s] %v",
		r.Method, r.URL.Path, targetService.URL, r.Host, duration)
}

// registerAllPlugins 自动发现并注册所有插件
func registerAllPlugins(factory middleware.MiddlewareFactory, autoPluginMgr *middleware.AutoPluginManager) error {
	// 发现所有插件
	plugins, err := autoPluginMgr.DiscoverPlugins()
	if err != nil {
		return fmt.Errorf("failed to discover plugins: %v", err)
	}

	log.Printf("Discovered %d plugins: %v", len(plugins), plugins)

	// 注册每个插件
	for _, pluginName := range plugins {
		// 获取插件创建函数
		creator, err := autoPluginMgr.GetPluginCreator(pluginName)
		if err != nil {
			log.Printf("Failed to get creator for plugin '%s': %v", pluginName, err)
			continue
		}

		// 注册插件到工厂
		factory.RegisterMiddleware(pluginName, creator)
		log.Printf("Registered plugin '%s'", pluginName)
	}

	return nil
}

// determineTarget 确定目标服务，返回匹配的服务和路由规则信息
func (ph *ProxyHandler) determineTarget(r *http.Request) (*config.Service, *config.HostRule, *config.RouteRule, error) {
	// 1. 先尝试域名匹配（策略：域名匹配优先）
	host := r.Host
	// 移除端口号
	if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}

	// 使用域名匹配器查找匹配的域名
	targetServiceName, matched := ph.hostMatcher.Match(host)
	if !matched {
		return nil, nil, nil, fmt.Errorf("no matching rule found for host: %s, path: %s", r.Host, r.URL.Path)
	}

	// 查找对应的域名配置
	var matchedHostRule *config.HostRule
	for _, hostRule := range ph.cfg.HostRules {
		if hostRule.Target == targetServiceName {
			// 检查端口号是否匹配
			// 重要：域名规则的端口配置应该表示该规则只在特定端口上生效
			// 如果域名规则指定了端口（Port != 0），那么该规则只在该端口上生效
			// 如果域名规则没有指定端口（Port为0），那么该规则在所有端口上都生效

			// 调试日志：显示域名匹配信息
			log.Printf("Host matching: target=%s, hostRule.Port=%d, r.Host=%s",
				targetServiceName, hostRule.Port, r.Host)

			// 如果域名规则指定了端口，我们需要检查当前请求是否来自正确的端口
			// 但由于HTTP请求的Host头通常不包含端口信息，我们无法从Host头获取端口
			// 因此，我们应该放宽端口检查：只有当域名规则明确指定端口时才进行严格检查
			// 但实际上，更好的做法是：域名规则的端口应该表示该规则只在特定端口上生效
			// 如果域名规则指定了端口，但当前服务器端口不匹配，则跳过

			// 注意：这里我们无法直接获取当前服务器端口，因为请求可能来自任何监听端口
			// 所以我们应该简化逻辑：如果域名规则指定了端口，就接受该规则
			// 因为服务器已经在正确的端口上监听

			matchedHostRule = &hostRule
			log.Printf("Host rule matched: %s -> %s (port: %d)", hostRule.Pattern, hostRule.Target, hostRule.Port)
			break
		}
	}

	if matchedHostRule != nil {
		// 2. 在匹配的域名规则中尝试路由匹配
		for _, routeRule := range matchedHostRule.RouteRules {
			// 简单的路径匹配逻辑
			if routeRule.Pattern == "/" && r.URL.Path == "/" {
				// 精确匹配根路径
				if service, exists := ph.services[routeRule.Target]; exists {
					return &service, matchedHostRule, &routeRule, nil
				}
			} else if strings.HasSuffix(routeRule.Pattern, "/*") {
				// 通配符匹配
				prefix := routeRule.Pattern[:len(routeRule.Pattern)-2]
				if strings.HasPrefix(r.URL.Path, prefix) {
					if r.URL.Path == prefix || strings.HasPrefix(r.URL.Path, prefix+"/") {
						if service, exists := ph.services[routeRule.Target]; exists {
							return &service, matchedHostRule, &routeRule, nil
						}
					}
				}
			} else if strings.HasPrefix(routeRule.Pattern, "^") && strings.HasSuffix(routeRule.Pattern, "$") {
				// 正则表达式匹配
				re, err := regexp.Compile(routeRule.Pattern)
				if err == nil && re.MatchString(r.URL.Path) {
					if service, exists := ph.services[routeRule.Target]; exists {
						return &service, matchedHostRule, &routeRule, nil
					}
				}
			}
		}

		// 3. 如果没有匹配的路由规则，使用域名的默认目标
		if service, exists := ph.services[matchedHostRule.Target]; exists {
			return &service, matchedHostRule, nil, nil
		}
	}

	return nil, nil, nil, fmt.Errorf("no matching rule found for host: %s, path: %s", r.Host, r.URL.Path)
}

// createDynamicMiddlewareChain 根据路由规则创建动态中间件链
func (ph *ProxyHandler) createDynamicMiddlewareChain(hostRule *config.HostRule, routeRule *config.RouteRule) middleware.MiddlewareChain {
	chain := middleware.NewMiddlewareChain()
	factory := ph.factory // 使用已注册的工厂实例

	// 获取所有已启用的中间件配置
	enabledMiddlewares := make(map[string]config.Middleware)
	for _, mwConfig := range ph.cfg.Middlewares {
		if mwConfig.Enabled {
			enabledMiddlewares[mwConfig.Name] = mwConfig
		}
	}

	// 添加路由级中间件（优先级最高）
	if routeRule != nil && len(routeRule.Middlewares) > 0 {
		for _, mwName := range routeRule.Middlewares {
			// 首先检查是否是注册的中间件服务
			mw, err := factory.CreateMiddleware(mwName, nil)
			if err == nil {
				chain.Add(mw)
				log.Printf("Route-level middleware service %s loaded for path: %s", mwName, routeRule.Pattern)
				continue
			}

			// 如果不是注册的中间件服务，检查标准中间件配置
			if mwConfig, exists := enabledMiddlewares[mwName]; exists {
				mw, err := factory.CreateMiddleware(mwConfig.Name, mwConfig.Config)
				if err != nil {
					log.Printf("Failed to create route-level middleware %s: %v", mwConfig.Name, err)
					continue
				}
				chain.Add(mw)
				log.Printf("Route-level middleware %s loaded for path: %s", mwConfig.Name, routeRule.Pattern)
			} else {
				log.Printf("Warning: Route-level middleware %s not found or disabled", mwName)
			}
		}
	}

	// 添加域名级中间件（优先级次之）
	if hostRule != nil && len(hostRule.Middlewares) > 0 {
		for _, mwName := range hostRule.Middlewares {
			// 首先检查是否是注册的中间件服务
			mw, err := factory.CreateMiddleware(mwName, nil)
			if err == nil {
				chain.Add(mw)
				log.Printf("Host-level middleware service %s loaded for host: %s", mwName, hostRule.Pattern)
				continue
			}

			// 如果不是注册的中间件服务，检查标准中间件配置
			if mwConfig, exists := enabledMiddlewares[mwName]; exists {
				mw, err := factory.CreateMiddleware(mwConfig.Name, mwConfig.Config)
				if err != nil {
					log.Printf("Failed to create host-level middleware %s: %v", mwConfig.Name, err)
					continue
				}
				chain.Add(mw)
				log.Printf("Host-level middleware %s loaded for host: %s", mwConfig.Name, hostRule.Pattern)
			} else {
				log.Printf("Warning: Host-level middleware %s not found or disabled", mwName)
			}
		}
	}

	// 添加全局中间件（优先级最低）
	for _, mwConfig := range ph.cfg.Middlewares {
		if mwConfig.Enabled {
			// 检查是否已经在路由级或域名级添加过
			alreadyAdded := false
			if routeRule != nil {
				for _, mwName := range routeRule.Middlewares {
					if mwName == mwConfig.Name {
						alreadyAdded = true
						break
					}
				}
			}
			if !alreadyAdded && hostRule != nil {
				for _, mwName := range hostRule.Middlewares {
					if mwName == mwConfig.Name {
						alreadyAdded = true
						break
					}
				}
			}

			if !alreadyAdded {
				mw, err := factory.CreateMiddleware(mwConfig.Name, mwConfig.Config)
				if err != nil {
					log.Printf("Failed to create global middleware %s: %v", mwConfig.Name, err)
					continue
				}
				chain.Add(mw)
				log.Printf("Global middleware %s loaded", mwConfig.Name)
			}
		}
	}

	// 添加全局中间件服务（优先级最低）
	// 注意：这里只添加明确标记为全局的中间件服务
	if registry := middleware.GetMiddlewareServiceRegistry(); registry != nil {
		for _, service := range registry.List() {
			// 只有明确标记为全局的中间件服务才会被全局加载
			if service.IsGlobal {
				// 检查是否已经在路由级或域名级添加过
				alreadyAdded := false
				if routeRule != nil {
					for _, mwName := range routeRule.Middlewares {
						if mwName == service.Name {
							alreadyAdded = true
							break
						}
					}
				}
				if !alreadyAdded && hostRule != nil {
					for _, mwName := range hostRule.Middlewares {
						if mwName == service.Name {
							alreadyAdded = true
							break
						}
					}
				}

				if !alreadyAdded {
					mw, err := factory.CreateMiddleware(service.Name, nil)
					if err != nil {
						log.Printf("Failed to create global middleware service %s: %v", service.Name, err)
						continue
					}
					chain.Add(mw)
					log.Printf("Global middleware service %s loaded", service.Name)
				}
			}
		}
	}

	return chain
}

// createReverseProxy 创建反向代理
func (ph *ProxyHandler) createReverseProxy(service *config.Service, ctx *middleware.Context) (*httputil.ReverseProxy, error) {
	targetURL, err := url.Parse(service.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %s", service.URL)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// 自定义修改请求 - 设置正确的Host头（二级代理场景）
	proxy.Director = func(req *http.Request) {
		// 保留原始请求的URL路径和查询参数
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host

		// 关键修复：设置Host头
		// 如果配置了proxy_host，使用配置的proxy_host，否则使用目标URL的Host
		hostHeader := targetURL.Host
		if service.ProxyHost != "" {
			hostHeader = service.ProxyHost
		}
		req.Host = hostHeader

		// 设置其他必要的头
		req.Header.Set("X-Forwarded-Proto", "http")
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Forwarded-For", req.RemoteAddr)
	}

	// 自定义修改响应
	proxy.ModifyResponse = func(resp *http.Response) error {
		// 添加代理相关响应头
		resp.Header.Set("X-Proxy-By", "toyou-proxy")
		resp.Header.Set("X-Target-Service", ph.getServiceName(service.URL))

		// 从上下文中获取替换规则
		if ctx != nil {
			if rules, exists := ctx.Get("replaceRules"); exists {
				if replaceRules, ok := rules.([]middleware.ReplaceRule); ok && len(replaceRules) > 0 {
					// 读取响应体
					body, err := io.ReadAll(resp.Body)
					if err != nil {
						return err
					}
					resp.Body.Close()

					// 应用替换规则
					modifiedBody := applyReplaceRules(body, replaceRules)

					// 重新设置响应体
					resp.Body = io.NopCloser(bytes.NewReader(modifiedBody))
					resp.ContentLength = int64(len(modifiedBody))
					resp.Header.Set("Content-Length", strconv.Itoa(len(modifiedBody)))
				}
			}
		}

		return nil
	}

	// 自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		http.Error(w, "Service unavailable", http.StatusBadGateway)
	}

	return proxy, nil
}

// applyReplaceRules 应用替换规则到响应内容
func applyReplaceRules(content []byte, rules []middleware.ReplaceRule) []byte {
	return middleware.ApplyReplaceRules(content, rules)
}

// getServiceName 从URL中提取服务名称
func (ph *ProxyHandler) getServiceName(urlStr string) string {
	if u, err := url.Parse(urlStr); err == nil {
		return u.Hostname()
	}
	return urlStr
}

// GetMiddlewareInfo 获取中间件信息
func (ph *ProxyHandler) GetMiddlewareInfo() []string {
	return ph.middlewareChain.GetMiddlewareNames()
}

// GetRulesInfo 获取规则信息
func (ph *ProxyHandler) GetRulesInfo() (map[string]string, map[string]string) {
	// 返回域名规则和空的路由规则（路由规则现在属于域名配置的子节点）
	return ph.hostMatcher.GetAllRules(), make(map[string]string)
}
