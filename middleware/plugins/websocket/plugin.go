package main

import (
	"net/http"
	"strings"
	"sync/atomic"

	"toyou-proxy/middleware"
)

// WebSocketMiddleware 检测并处理WebSocket请求的中间件
type WebSocketMiddleware struct {
	// 连接统计
	activeConnections int64
	totalConnections  int64
	errors            int64

	// 配置参数
	pathPatterns   []string
	maxConnections int64
}

// NewWebSocketMiddleware 创建WebSocket中间件
func NewWebSocketMiddleware(config map[string]interface{}) (middleware.Middleware, error) {
	// 解析路径模式
	var pathPatterns []string
	if pp, ok := config["path_patterns"].([]interface{}); ok {
		for _, pattern := range pp {
			if p, ok := pattern.(string); ok {
				pathPatterns = append(pathPatterns, p)
			}
		}
	}

	// 设置默认路径模式
	if len(pathPatterns) == 0 {
		pathPatterns = []string{
			"/ws/*",
			"/websocket/*",
			"/socket.io/*",
		}
	}

	// 解析最大连接数
	maxConnections := int64(1000) // 默认值
	if mc, ok := config["max_connections"].(float64); ok {
		maxConnections = int64(mc)
	}

	return &WebSocketMiddleware{
		pathPatterns:   pathPatterns,
		maxConnections: maxConnections,
	}, nil
}

// Name 返回中间件名称
func (wm *WebSocketMiddleware) Name() string {
	return "websocket"
}

// Handle 处理WebSocket检测逻辑
func (wm *WebSocketMiddleware) Handle(ctx *middleware.Context) bool {
	req := ctx.Request

	// 检测WebSocket请求
	if wm.isWebSocketRequest(req) {
		// 在上下文中标记为WebSocket连接
		ctx.Set("isWebSocketConnection", true)

		// 更新统计信息
		atomic.AddInt64(&wm.totalConnections, 1)
		atomic.AddInt64(&wm.activeConnections, 1)

		// 设置清理函数
		defer func() {
			atomic.AddInt64(&wm.activeConnections, -1)
		}()

		// 记录WebSocket连接
		// 注意：这里不直接输出日志，而是使用上下文存储，由日志中间件处理
		ctx.Set("websocket_connection", true)
	}

	return true
}

// isWebSocketRequest 检测是否为WebSocket请求
func (wm *WebSocketMiddleware) isWebSocketRequest(req *http.Request) bool {
	// 检查Upgrade头
	if strings.ToLower(req.Header.Get("Upgrade")) == "websocket" {
		return true
	}

	// 检查Connection头
	connection := req.Header.Get("Connection")
	if connection != "" && strings.Contains(strings.ToLower(connection), "upgrade") {
		// 同时检查Upgrade头
		if strings.ToLower(req.Header.Get("Upgrade")) == "websocket" {
			return true
		}
	}

	// 检查特定路径模式
	path := req.URL.Path
	for _, pattern := range wm.pathPatterns {
		if wm.matchPath(pattern, path) {
			return true
		}
	}

	// 检查查询参数
	if req.URL.Query().Get("websocket") == "true" ||
		req.URL.Query().Get("ws") == "true" {
		return true
	}

	return false
}

// matchPath 匹配路径模式
func (wm *WebSocketMiddleware) matchPath(pattern, path string) bool {
	// 简单的通配符匹配
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, prefix)
	}
	return pattern == path
}

// GetStats 获取WebSocket统计信息
func (wm *WebSocketMiddleware) GetStats() map[string]int64 {
	return map[string]int64{
		"active_connections": atomic.LoadInt64(&wm.activeConnections),
		"total_connections":  atomic.LoadInt64(&wm.totalConnections),
		"errors":             atomic.LoadInt64(&wm.errors),
	}
}

// PluginMain 插件入口函数
func PluginMain(config map[string]interface{}) (middleware.Middleware, error) {
	return NewWebSocketMiddleware(config)
}
