package middleware

import (
	"net/http"
	"sync"
	"time"
)

// RateLimitMiddleware 限流中间件
type RateLimitMiddleware struct {
	requestsPerMinute int
	burstSize        int
	clients          map[string]*rateLimiter
	mu               sync.RWMutex
}

// rateLimiter 单个客户端的限流器
type rateLimiter struct {
	lastRequest time.Time
	requests    int
	burst       int
}

// NewRateLimitMiddleware 创建限流中间件
func NewRateLimitMiddleware(requestsPerMinute, burstSize int) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		requestsPerMinute: requestsPerMinute,
		burstSize:        burstSize,
		clients:          make(map[string]*rateLimiter),
	}
}

// Name 返回中间件名称
func (rlm *RateLimitMiddleware) Name() string {
	return "rate_limit"
}

// Handle 处理限流逻辑
func (rlm *RateLimitMiddleware) Handle(ctx *Context) bool {
	clientIP := getClientIP(ctx.Request)
	
	rlm.mu.Lock()
	defer rlm.mu.Unlock()

	now := time.Now()
	limiter, exists := rlm.clients[clientIP]
	
	if !exists {
		limiter = &rateLimiter{
			lastRequest: now,
			requests:    0,
			burst:      rlm.burstSize,
		}
		rlm.clients[clientIP] = limiter
	}

	// 检查是否需要重置计数器
	if now.Sub(limiter.lastRequest) > time.Minute {
		limiter.requests = 0
		limiter.burst = rlm.burstSize
		limiter.lastRequest = now
	}

	// 检查是否超过限制
	if limiter.requests >= rlm.requestsPerMinute {
		if limiter.burst <= 0 {
			ctx.StatusCode = http.StatusTooManyRequests
			http.Error(ctx.Response, "Rate limit exceeded", http.StatusTooManyRequests)
			return false
		}
		limiter.burst--
	} else {
		limiter.requests++
	}

	limiter.lastRequest = now
	return true
}

// getClientIP 获取客户端IP
func getClientIP(r *http.Request) string {
	// 检查X-Forwarded-For头部
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	// 检查X-Real-IP头部
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	// 使用远程地址
	return r.RemoteAddr
}