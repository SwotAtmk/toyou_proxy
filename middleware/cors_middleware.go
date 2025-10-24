package middleware

import (
	"net/http"
	"strings"
)

// CORSMiddleware CORS中间件
type CORSMiddleware struct {
	allowedOrigins []string
	allowedMethods []string
}

// NewCORSMiddleware 创建CORS中间件
func NewCORSMiddleware(allowedOrigins, allowedMethods []string) *CORSMiddleware {
	return &CORSMiddleware{
		allowedOrigins: allowedOrigins,
		allowedMethods: allowedMethods,
	}
}

// Name 返回中间件名称
func (cm *CORSMiddleware) Name() string {
	return "cors"
}

// Handle 处理CORS逻辑
func (cm *CORSMiddleware) Handle(ctx *Context) bool {
	origin := ctx.Request.Header.Get("Origin")
	
	// 处理预检请求
	if ctx.Request.Method == "OPTIONS" {
		cm.handlePreflight(ctx, origin)
		return false // 预检请求不需要继续处理
	}

	// 设置CORS头部
	if cm.isOriginAllowed(origin) {
		ctx.Response.Header().Set("Access-Control-Allow-Origin", origin)
		ctx.Response.Header().Set("Access-Control-Allow-Credentials", "true")
		ctx.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		ctx.Response.Header().Set("Access-Control-Allow-Methods", strings.Join(cm.allowedMethods, ", "))
	}

	return true
}

// handlePreflight 处理预检请求
func (cm *CORSMiddleware) handlePreflight(ctx *Context, origin string) {
	if cm.isOriginAllowed(origin) {
		ctx.Response.Header().Set("Access-Control-Allow-Origin", origin)
		ctx.Response.Header().Set("Access-Control-Allow-Credentials", "true")
		ctx.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		ctx.Response.Header().Set("Access-Control-Allow-Methods", strings.Join(cm.allowedMethods, ", "))
		ctx.Response.Header().Set("Access-Control-Max-Age", "86400") // 24小时
	}
	
	ctx.StatusCode = http.StatusNoContent
	ctx.Response.WriteHeader(http.StatusNoContent)
}

// isOriginAllowed 检查源是否允许
func (cm *CORSMiddleware) isOriginAllowed(origin string) bool {
	if len(cm.allowedOrigins) == 0 {
		return false
	}

	for _, allowed := range cm.allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}

	return false
}