package middleware

import (
	"net/http"
	"strings"
)

// AuthMiddleware 认证中间件
type AuthMiddleware struct {
	headerName string
	validKeys  map[string]bool
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(headerName string, validKeys []string) *AuthMiddleware {
	keysMap := make(map[string]bool)
	for _, key := range validKeys {
		keysMap[key] = true
	}

	return &AuthMiddleware{
		headerName: headerName,
		validKeys:  keysMap,
	}
}

// Name 返回中间件名称
func (am *AuthMiddleware) Name() string {
	return "auth"
}

// Handle 处理认证逻辑
func (am *AuthMiddleware) Handle(ctx *Context) bool {
	authHeader := ctx.Request.Header.Get(am.headerName)
	if authHeader == "" {
		ctx.StatusCode = http.StatusUnauthorized
		http.Error(ctx.Response, "Missing authentication header", http.StatusUnauthorized)
		return false
	}

	// 支持 "Bearer token" 格式
	token := authHeader
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimPrefix(authHeader, "Bearer ")
	}

	if !am.validKeys[token] {
		ctx.StatusCode = http.StatusForbidden
		http.Error(ctx.Response, "Invalid authentication token", http.StatusForbidden)
		return false
	}

	return true
}