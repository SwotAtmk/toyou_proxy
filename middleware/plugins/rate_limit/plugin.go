package main

import (
	"net/http"
	"sync"
	"time"
	"toyou-proxy/middleware"
)

// RateLimitMiddleware 限流中间件
type RateLimitMiddleware struct {
	requestsPerMinute int
	burstSize         int
	clients           map[string]*rateLimiter
	mu                sync.RWMutex
}

// rateLimiter 单个客户端的限流器
type rateLimiter struct {
	count     int
	lastReset time.Time
}

// NewRateLimitMiddleware 创建限流中间件
func NewRateLimitMiddleware(config map[string]interface{}) (middleware.Middleware, error) {
	requestsPerMinute := 100
	if rpm, ok := config["requests_per_minute"].(float64); ok {
		requestsPerMinute = int(rpm)
	}

	burstSize := 20
	if bs, ok := config["burst_size"].(float64); ok {
		burstSize = int(bs)
	}

	return &RateLimitMiddleware{
		requestsPerMinute: requestsPerMinute,
		burstSize:         burstSize,
		clients:           make(map[string]*rateLimiter),
	}, nil
}

// PluginMain 插件入口函数
func PluginMain(config map[string]interface{}) (middleware.Middleware, error) {
	return NewRateLimitMiddleware(config)
}

// Name 返回中间件名称
func (rlm *RateLimitMiddleware) Name() string {
	return "rate_limit"
}

// Handle 处理限流逻辑
func (rlm *RateLimitMiddleware) Handle(context *middleware.Context) bool {

	// 获取客户端IP
	clientIP := getClientIP(context.Request)

	rlm.mu.Lock()
	defer rlm.mu.Unlock()

	// 获取或创建限流器
	limiter, exists := rlm.clients[clientIP]
	if !exists {
		limiter = &rateLimiter{
			count:     0,
			lastReset: time.Now(),
		}
		rlm.clients[clientIP] = limiter
	}

	// 检查是否需要重置计数器
	if time.Since(limiter.lastReset) > time.Minute {
		limiter.count = 0
		limiter.lastReset = time.Now()
	}

	// 检查是否超过限制
	if limiter.count >= rlm.requestsPerMinute+rlm.burstSize {
		context.StatusCode = http.StatusTooManyRequests
		http.Error(context.Response, "Rate limit exceeded", http.StatusTooManyRequests)
		return false
	}

	// 增加计数器
	limiter.count++

	return true
}

// getClientIP 获取客户端IP
func getClientIP(r *http.Request) string {
	// 检查X-Forwarded-For头
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	// 检查X-Real-IP头
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	// 返回远程地址
	return r.RemoteAddr
}
