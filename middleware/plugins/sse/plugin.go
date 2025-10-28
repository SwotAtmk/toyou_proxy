package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"toyou-proxy/middleware"
)

// SSEMiddleware 自动检测并处理SSE请求的中间件
type SSEMiddleware struct {
	// 连接统计
	activeConnections int64
	totalConnections  int64
	bytesTransferred  int64
	errors            int64
}

// NewSSEMiddleware 创建SSE中间件
func NewSSEMiddleware(config map[string]interface{}) (middleware.Middleware, error) {
	// 这个中间件不需要配置参数
	return &SSEMiddleware{}, nil
}

// Name 返回中间件名称
func (sm *SSEMiddleware) Name() string {
	return "sse"
}

// Handle 处理SSE逻辑
func (sm *SSEMiddleware) Handle(ctx *middleware.Context) bool {
	req := ctx.Request
	resp := ctx.Response

	// 检测SSE请求
	if sm.isSSERequest(req) {
		// 设置SSE相关响应头
		sm.setupSSEResponseHeaders(resp)

		// 在上下文中标记为SSE连接
		ctx.Set("isSSEConnection", true)

		// 包装响应写入器以支持SSE
		sseWriter := &SSEWriter{
			ResponseWriter: resp,
			flushInterval:  100 * time.Millisecond,
			bytesWritten:   0,
			middleware:     sm,
		}

		// 将包装后的写入器设置到上下文中
		ctx.Response = sseWriter

		// 更新统计信息
		atomic.AddInt64(&sm.totalConnections, 1)
		atomic.AddInt64(&sm.activeConnections, 1)

		// 设置清理函数
		defer func() {
			atomic.AddInt64(&sm.activeConnections, -1)
		}()

		// 记录SSE连接
		fmt.Printf("[SSE] New connection established: %s %s\n", req.Method, req.URL.Path)
	}

	return true
}

// isSSERequest 检测是否为SSE请求
func (sm *SSEMiddleware) isSSERequest(req *http.Request) bool {
	// 检查Accept头
	accept := req.Header.Get("Accept")
	if accept != "" && strings.Contains(accept, "text/event-stream") {
		return true
	}

	// 检查特定路径模式
	path := req.URL.Path
	ssePatterns := []string{
		"/events/*",
		"/stream/*",
		"/sse/*",
		"/api/events/*",
		"/api/stream/*",
		"/api/sse/*",
	}

	for _, pattern := range ssePatterns {
		if matched := sm.matchPath(pattern, path); matched {
			return true
		}
	}

	// 检查查询参数
	if req.URL.Query().Get("stream") == "sse" || req.URL.Query().Get("format") == "sse" {
		return true
	}

	return false
}

// matchPath 匹配路径模式
func (sm *SSEMiddleware) matchPath(pattern, path string) bool {
	// 简单的通配符匹配
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, prefix)
	}
	return pattern == path
}

// setupSSEResponseHeaders 设置SSE响应头
func (sm *SSEMiddleware) setupSSEResponseHeaders(resp http.ResponseWriter) {
	// 设置内容类型
	resp.Header().Set("Content-Type", "text/event-stream")

	// 禁用缓存
	resp.Header().Set("Cache-Control", "no-cache")
	resp.Header().Set("X-Accel-Buffering", "no") // Nginx兼容

	// 保持连接
	resp.Header().Set("Connection", "keep-alive")

	// 设置CORS头（如果需要）
	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Set("Access-Control-Allow-Headers", "Cache-Control")
}

// GetStats 获取SSE统计信息
func (sm *SSEMiddleware) GetStats() map[string]int64 {
	return map[string]int64{
		"active_connections": atomic.LoadInt64(&sm.activeConnections),
		"total_connections":  atomic.LoadInt64(&sm.totalConnections),
		"bytes_transferred":  atomic.LoadInt64(&sm.bytesTransferred),
		"errors":             atomic.LoadInt64(&sm.errors),
	}
}

// SSEWriter 包装ResponseWriter以支持SSE
type SSEWriter struct {
	http.ResponseWriter
	flushInterval time.Duration
	bytesWritten  int64
	middleware    *SSEMiddleware
	mu            sync.Mutex
}

// Write 重写Write方法以支持SSE
func (w *SSEWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	n, err := w.ResponseWriter.Write(data)
	if err != nil {
		atomic.AddInt64(&w.middleware.errors, 1)
		return n, err
	}

	w.bytesWritten += int64(n)
	atomic.AddInt64(&w.middleware.bytesTransferred, int64(n))

	// 立即刷新数据
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}

	return n, nil
}

// WriteString 写入字符串
func (w *SSEWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// WriteEvent 写入SSE事件
func (w *SSEWriter) WriteEvent(event, data string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var buf bytes.Buffer

	// 写入事件名称（如果有）
	if event != "" {
		buf.WriteString(fmt.Sprintf("event: %s\n", event))
	}

	// 写入数据
	for _, line := range strings.Split(data, "\n") {
		buf.WriteString(fmt.Sprintf("data: %s\n", line))
	}

	// 写入事件分隔符
	buf.WriteString("\n")

	// 写入响应
	n, err := w.ResponseWriter.Write(buf.Bytes())
	if err != nil {
		atomic.AddInt64(&w.middleware.errors, 1)
		return err
	}

	w.bytesWritten += int64(n)
	atomic.AddInt64(&w.middleware.bytesTransferred, int64(n))

	// 立即刷新数据
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

// Flush 刷新数据
func (w *SSEWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Hijack 劫持连接（用于WebSocket等）
func (w *SSEWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("hijacking not supported")
}

// PluginMain 插件入口函数
func PluginMain(config map[string]interface{}) (middleware.Middleware, error) {
	return NewSSEMiddleware(config)
}
