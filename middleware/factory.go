package middleware

import (
	"fmt"
	"strconv"
)

// MiddlewareFactory 中间件工厂
type MiddlewareFactory struct{}

// NewMiddlewareFactory 创建中间件工厂
func NewMiddlewareFactory() *MiddlewareFactory {
	return &MiddlewareFactory{}
}

// CreateMiddleware 根据配置创建中间件
func (mf *MiddlewareFactory) CreateMiddleware(name string, config map[string]interface{}) (Middleware, error) {
	switch name {
	case "auth":
		return mf.createAuthMiddleware(config)
	case "rate_limit":
		return mf.createRateLimitMiddleware(config)
	case "cors":
		return mf.createCORSMiddleware(config)
	default:
		return nil, fmt.Errorf("unknown middleware: %s", name)
	}
}

// createAuthMiddleware 创建认证中间件
func (mf *MiddlewareFactory) createAuthMiddleware(config map[string]interface{}) (Middleware, error) {
	headerName, ok := config["header_name"].(string)
	if !ok {
		headerName = "X-API-Key"
	}

	validKeysInterface, ok := config["valid_keys"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("valid_keys must be an array")
	}

	validKeys := make([]string, len(validKeysInterface))
	for i, key := range validKeysInterface {
		validKeys[i] = key.(string)
	}

	return NewAuthMiddleware(headerName, validKeys), nil
}

// createRateLimitMiddleware 创建限流中间件
func (mf *MiddlewareFactory) createRateLimitMiddleware(config map[string]interface{}) (Middleware, error) {
	requestsPerMinute := 100
	if rpm, ok := config["requests_per_minute"]; ok {
		switch v := rpm.(type) {
		case int:
			requestsPerMinute = v
		case float64:
			requestsPerMinute = int(v)
		case string:
			if parsed, err := strconv.Atoi(v); err == nil {
				requestsPerMinute = parsed
			}
		}
	}

	burstSize := 20
	if bs, ok := config["burst_size"]; ok {
		switch v := bs.(type) {
		case int:
			burstSize = v
		case float64:
			burstSize = int(v)
		case string:
			if parsed, err := strconv.Atoi(v); err == nil {
				burstSize = parsed
			}
		}
	}

	return NewRateLimitMiddleware(requestsPerMinute, burstSize), nil
}

// createCORSMiddleware 创建CORS中间件
func (mf *MiddlewareFactory) createCORSMiddleware(config map[string]interface{}) (Middleware, error) {
	allowedOriginsInterface, ok := config["allowed_origins"].([]interface{})
	if !ok {
		allowedOriginsInterface = []interface{}{"*"}
	}

	allowedOrigins := make([]string, len(allowedOriginsInterface))
	for i, origin := range allowedOriginsInterface {
		allowedOrigins[i] = origin.(string)
	}

	allowedMethodsInterface, ok := config["allowed_methods"].([]interface{})
	if !ok {
		allowedMethodsInterface = []interface{}{"GET", "POST", "PUT", "DELETE"}
	}

	allowedMethods := make([]string, len(allowedMethodsInterface))
	for i, method := range allowedMethodsInterface {
		allowedMethods[i] = method.(string)
	}

	return NewCORSMiddleware(allowedOrigins, allowedMethods), nil
}