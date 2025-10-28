package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
	"toyou-proxy/middleware"
)

// LoggingMiddleware 日志中间件
type LoggingMiddleware struct {
	level string
}

// NewLoggingMiddleware 创建日志中间件
func NewLoggingMiddleware(config map[string]interface{}) (middleware.Middleware, error) {
	level := "info"
	if l, ok := config["level"].(string); ok {
		level = l
	}

	return &LoggingMiddleware{
		level: level,
	}, nil
}

// PluginMain 插件入口函数
func PluginMain(config map[string]interface{}) (middleware.Middleware, error) {
	return NewLoggingMiddleware(config)
}

// Name 返回中间件名称
func (lm *LoggingMiddleware) Name() string {
	return "logging"
}

// Handle 处理日志逻辑
func (lm *LoggingMiddleware) Handle(context *middleware.Context) bool {

	start := time.Now()

	// 记录请求开始
	if lm.level == "debug" {
		log.Printf("[%s] %s %s - Started", lm.level, context.Request.Method, context.Request.URL.Path)
	}

	// 继续处理请求
	result := true

	// 记录请求结束
	if lm.level == "info" || lm.level == "debug" {
		duration := time.Since(start)
		statusCode := context.StatusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		log.Printf("[%s] %s %s - %d - %v", lm.level, context.Request.Method, context.Request.URL.Path, statusCode, duration)
	}

	return result
}

// 辅助函数，用于格式化日志
func (lm *LoggingMiddleware) formatLog(message string) string {
	return fmt.Sprintf("[%s] %s", lm.level, message)
}
