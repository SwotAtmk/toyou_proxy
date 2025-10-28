package main

import (
	"net/http"
	"toyou-proxy/middleware"
)

// CORSMiddleware CORS中间件
type CORSMiddleware struct {
	allowedOrigins []string
	allowedMethods []string
	allowedHeaders []string
}

// NewCORSMiddleware 创建CORS中间件
func NewCORSMiddleware(config map[string]interface{}) (middleware.Middleware, error) {
	var allowedOrigins []string
	if origins, ok := config["allowed_origins"].([]interface{}); ok {
		for _, origin := range origins {
			if o, ok := origin.(string); ok {
				allowedOrigins = append(allowedOrigins, o)
			}
		}
	}

	var allowedMethods []string
	if methods, ok := config["allowed_methods"].([]interface{}); ok {
		for _, method := range methods {
			if m, ok := method.(string); ok {
				allowedMethods = append(allowedMethods, m)
			}
		}
	}

	var allowedHeaders []string
	if headers, ok := config["allowed_headers"].([]interface{}); ok {
		for _, header := range headers {
			if h, ok := header.(string); ok {
				allowedHeaders = append(allowedHeaders, h)
			}
		}
	}

	return &CORSMiddleware{
		allowedOrigins: allowedOrigins,
		allowedMethods: allowedMethods,
		allowedHeaders: allowedHeaders,
	}, nil
}

// PluginMain 插件入口函数
func PluginMain(config map[string]interface{}) (middleware.Middleware, error) {
	return NewCORSMiddleware(config)
}

// Name 返回中间件名称
func (cm *CORSMiddleware) Name() string {
	return "cors"
}

// Handle 处理CORS逻辑
func (cm *CORSMiddleware) Handle(context *middleware.Context) bool {

	request := context.Request
	response := context.Response

	// 设置CORS头
	origin := request.Header.Get("Origin")
	if origin != "" {
		// 检查是否允许该origin
		if len(cm.allowedOrigins) > 0 {
			allowed := false
			for _, allowedOrigin := range cm.allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}
			if !allowed {
				return true
			}
		}

		response.Header().Set("Access-Control-Allow-Origin", origin)
		response.Header().Set("Access-Control-Allow-Credentials", "true")

		if len(cm.allowedMethods) > 0 {
			response.Header().Set("Access-Control-Allow-Methods", join(cm.allowedMethods, ", "))
		}

		if len(cm.allowedHeaders) > 0 {
			response.Header().Set("Access-Control-Allow-Headers", join(cm.allowedHeaders, ", "))
		}

		// 处理预检请求
		if request.Method == "OPTIONS" {
			response.WriteHeader(http.StatusOK)
			return false
		}
	}

	return true
}

// join 辅助函数，将字符串切片连接为字符串
func join(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
