package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	"toyou-proxy/middleware"
)

// DynamicRouteMiddleware 动态路由中间件
type DynamicRouteMiddleware struct {
	apiURL             string
	timeout            time.Duration
	cacheExpiry        time.Duration
	lastCacheUpdate    time.Time
	cachedHostMappings map[string]string
	httpClient         *http.Client
}

// APIResponse 外部API响应结构
type APIResponse struct {
	Data struct {
		GotoServices string `json:"goto_services"`
	} `json:"data"`
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// NewDynamicRouteMiddleware 创建动态路由中间件
func NewDynamicRouteMiddleware(config map[string]interface{}) (middleware.Middleware, error) {
	// 获取API URL，默认为 http://127.0.0.1:7080/api/host
	apiURL, ok := config["api_url"].(string)
	if !ok {
		apiURL = "http://127.0.0.1:7080/api/host"
	}

	// 获取超时时间，默认为5秒
	timeoutSeconds := 5.0
	if ts, ok := config["timeout_seconds"].(float64); ok {
		timeoutSeconds = ts
	}

	// 获取缓存过期时间，默认为60秒
	cacheExpirySeconds := 60.0
	if ces, ok := config["cache_expiry_seconds"].(float64); ok {
		cacheExpirySeconds = ces
	}

	return &DynamicRouteMiddleware{
		apiURL:             apiURL,
		timeout:            time.Duration(timeoutSeconds) * time.Second,
		cacheExpiry:        time.Duration(cacheExpirySeconds) * time.Second,
		lastCacheUpdate:    time.Time{},
		cachedHostMappings: make(map[string]string),
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}, nil
}

// PluginMain 插件入口函数
func PluginMain(config map[string]interface{}) (middleware.Middleware, error) {
	return NewDynamicRouteMiddleware(config)
}

// Name 返回中间件名称
func (drm *DynamicRouteMiddleware) Name() string {
	return "dynamic_route"
}

// Handle 处理动态路由逻辑
func (drm *DynamicRouteMiddleware) Handle(context *middleware.Context) bool {
	// 获取请求的Host
	host := context.Request.Host
	if host == "" {
		// 如果Host为空，从URL中提取
		host = context.Request.URL.Host
	}

	// 提取主机名部分（去除端口）
	hostName := strings.Split(host, ":")[0]

	// 检查缓存是否有效
	targetService, found := drm.getCachedTarget(hostName)
	if !found {
		// 缓存未命中或已过期，调用外部API
		newTarget, err := drm.queryExternalAPI(hostName)
		if err != nil {
			// API调用失败，记录日志但继续执行原始路由
			fmt.Printf("Dynamic route middleware: Failed to query external API for host '%s': %v\n", hostName, err)
			return true
		}

		// 更新缓存
		drm.updateCache(hostName, newTarget)
		targetService = newTarget
	}

	// 如果API返回了有效的目标服务，更新上下文
	if targetService != "" {
		// 将目标服务存储在上下文中，供后续中间件使用
		if context.Values == nil {
			context.Values = make(map[string]interface{})
		}
		context.Values["dynamic_target_service"] = targetService

		fmt.Printf("Dynamic route middleware: Rerouting host '%s' to service '%s'\n", hostName, targetService)
	}

	return true
}

// getCachedTarget 从缓存中获取目标服务
func (drm *DynamicRouteMiddleware) getCachedTarget(host string) (string, bool) {
	// 检查缓存是否已过期
	if time.Since(drm.lastCacheUpdate) > drm.cacheExpiry {
		return "", false
	}

	target, exists := drm.cachedHostMappings[host]
	return target, exists
}

// updateCache 更新缓存
func (drm *DynamicRouteMiddleware) updateCache(host, target string) {
	drm.cachedHostMappings[host] = target
	drm.lastCacheUpdate = time.Now()
}

// queryExternalAPI 查询外部API获取目标服务
func (drm *DynamicRouteMiddleware) queryExternalAPI(host string) (string, error) {
	// 准备请求体
	requestBody := map[string]string{"host": host}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %v", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", drm.apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := drm.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	// 解析响应
	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	// 检查响应状态
	if apiResp.Code != 200 {
		return "", fmt.Errorf("API returned error: %s", apiResp.Msg)
	}

	return apiResp.Data.GotoServices, nil
}
