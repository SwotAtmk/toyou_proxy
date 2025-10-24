package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
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
	middlewareChain *middleware.MiddlewareChain
	cfg             *config.Config
}

// NewProxyHandler 创建新的代理处理器
func NewProxyHandler(cfg *config.Config) (*ProxyHandler, error) {
	// 创建域名匹配器
	hostMatcher := matcher.NewHostMatcher()
	for _, rule := range cfg.HostRules {
		hostMatcher.AddRule(rule.Pattern, rule.Target)
	}

	// 创建中间件链
	middlewareChain := middleware.NewMiddlewareChain()
	factory := middleware.NewMiddlewareFactory()

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
	}

	// 执行中间件链
	if !ph.middlewareChain.Execute(ctx) {
		if ctx.StatusCode != 0 {
			w.WriteHeader(ctx.StatusCode)
		}
		log.Printf("Request aborted by middleware: %s %s", r.Method, r.URL.Path)
		return
	}

	// 确定目标服务
	targetService, err := ph.determineTarget(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		log.Printf("Failed to determine target: %v", err)
		return
	}

	// 创建反向代理
	proxy, err := ph.createReverseProxy(targetService)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		log.Printf("Failed to create reverse proxy: %v", err)
		return
	}

	// 设置目标URL到上下文
	ctx.TargetURL = targetService.URL
	ctx.ServiceName = ph.getServiceName(targetService.URL)

	// 执行代理
	proxy.ServeHTTP(w, r)

	// 记录请求日志
	duration := time.Since(startTime)
	log.Printf("Proxied: %s %s -> %s [%s] %v",
		r.Method, r.URL.Path, targetService.URL, r.Host, duration)
}

// determineTarget 确定目标服务
func (ph *ProxyHandler) determineTarget(r *http.Request) (*config.Service, error) {
	// 1. 先尝试域名匹配（nginx策略：域名匹配优先）
	host := r.Host
	// 移除端口号
	if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}

	// 使用域名匹配器查找匹配的域名
	targetServiceName, matched := ph.hostMatcher.Match(host)
	if !matched {
		return nil, fmt.Errorf("no matching rule found for host: %s, path: %s", r.Host, r.URL.Path)
	}

	// 查找对应的域名配置
	var matchedHostRule *config.HostRule
	for _, hostRule := range ph.cfg.HostRules {
		if hostRule.Target == targetServiceName {
			// 检查端口号是否匹配
			requestPort := 80 // 默认端口
			if colonIndex := strings.Index(r.Host, ":"); colonIndex != -1 {
				if port, err := strconv.Atoi(r.Host[colonIndex+1:]); err == nil {
					requestPort = port
				}
			}

			// 如果请求的端口与配置的端口不匹配，跳过这个规则
			if hostRule.Port != 0 && hostRule.Port != requestPort {
				continue
			}

			matchedHostRule = &hostRule
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
					return &service, nil
				}
			} else if strings.HasSuffix(routeRule.Pattern, "/*") {
				// 通配符匹配
				prefix := routeRule.Pattern[:len(routeRule.Pattern)-2]
				if strings.HasPrefix(r.URL.Path, prefix) {
					if r.URL.Path == prefix || strings.HasPrefix(r.URL.Path, prefix+"/") {
						if service, exists := ph.services[routeRule.Target]; exists {
							return &service, nil
						}
					}
				}
			} else if strings.HasPrefix(routeRule.Pattern, "^") && strings.HasSuffix(routeRule.Pattern, "$") {
				// 正则表达式匹配
				re, err := regexp.Compile(routeRule.Pattern)
				if err == nil && re.MatchString(r.URL.Path) {
					if service, exists := ph.services[routeRule.Target]; exists {
						return &service, nil
					}
				}
			}
		}

		// 3. 如果没有匹配的路由规则，使用域名的默认目标
		if service, exists := ph.services[matchedHostRule.Target]; exists {
			return &service, nil
		}
	}

	return nil, fmt.Errorf("no matching rule found for host: %s, path: %s", r.Host, r.URL.Path)
}

// createReverseProxy 创建反向代理
func (ph *ProxyHandler) createReverseProxy(service *config.Service) (*httputil.ReverseProxy, error) {
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
		// 添加代理相关的响应头
		resp.Header.Set("X-Proxy-Server", "toyou-proxy")
		resp.Header.Set("X-Proxy-Target", service.URL)
		return nil
	}

	// 自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		http.Error(w, "Service unavailable", http.StatusBadGateway)
	}

	return proxy, nil
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
