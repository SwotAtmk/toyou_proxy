package main

import (
	"bytes"
	"net/http"
	"regexp"
	"toyou-proxy/middleware"
)

// ReplaceMiddleware 响应内容替换中间件
type ReplaceMiddleware struct {
	rules []ReplaceRule
}

// ReplaceRule 替换规则
type ReplaceRule struct {
	Pattern     string `json:"pattern"`
	Replacement string `json:"replacement"`
	Global      bool   `json:"global"`
}

// NewReplaceMiddleware 创建替换中间件
func NewReplaceMiddleware(config map[string]interface{}) (middleware.Middleware, error) {
	var rules []ReplaceRule
	if rulesData, ok := config["rules"].([]interface{}); ok {
		for _, ruleData := range rulesData {
			if rule, ok := ruleData.(map[string]interface{}); ok {
				replaceRule := ReplaceRule{
					Pattern:     getString(rule, "pattern"),
					Replacement: getString(rule, "replacement"),
					Global:      getBool(rule, "global"),
				}
				rules = append(rules, replaceRule)
			}
		}
	}

	return &ReplaceMiddleware{
		rules: rules,
	}, nil
}

// PluginMain 插件入口函数
func PluginMain(config map[string]interface{}) (middleware.Middleware, error) {
	return NewReplaceMiddleware(config)
}

// Name 返回中间件名称
func (rm *ReplaceMiddleware) Name() string {
	return "replace"
}

// Handle 处理替换逻辑
func (rm *ReplaceMiddleware) Handle(context *middleware.Context) bool {

	// 检查是否有替换规则
	if len(rm.rules) == 0 {
		return true
	}

	// 保存原始响应写入器
	originalWriter := context.Response

	// 创建缓冲区来捕获响应
	var buf bytes.Buffer
	context.Response = &responseWriter{
		ResponseWriter: originalWriter,
		body:           &buf,
	}

	// 继续处理请求
	result := true

	// 处理完成后，应用替换规则
	if buf.Len() > 0 {
		content := buf.String()
		modifiedContent := rm.applyReplaceRules(content)

		// 写入修改后的内容
		originalWriter.Header().Set("Content-Length", string(len(modifiedContent)))
		originalWriter.Write([]byte(modifiedContent))
	}

	return result
}

// applyReplaceRules 应用替换规则
func (rm *ReplaceMiddleware) applyReplaceRules(content string) string {
	result := content
	for _, rule := range rm.rules {
		if rule.Global {
			// 全局替换
			re := regexp.MustCompile(rule.Pattern)
			result = re.ReplaceAllString(result, rule.Replacement)
		} else {
			// 单次替换
			re := regexp.MustCompile(rule.Pattern)
			result = re.ReplaceAllString(result, rule.Replacement)
		}
	}
	return result
}

// responseWriter 自定义响应写入器
type responseWriter struct {
	http.ResponseWriter
	body *bytes.Buffer
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.body.Write(b)
}

// 辅助函数
func getString(data map[string]interface{}, key string) string {
	if value, ok := data[key].(string); ok {
		return value
	}
	return ""
}

func getBool(data map[string]interface{}, key string) bool {
	if value, ok := data[key].(bool); ok {
		return value
	}
	return false
}

// ApplyReplaceRules 应用替换规则的公共函数
func ApplyReplaceRules(content string, rules []ReplaceRule) string {
	result := content
	for _, rule := range rules {
		if rule.Global {
			re := regexp.MustCompile(rule.Pattern)
			result = re.ReplaceAllString(result, rule.Replacement)
		} else {
			re := regexp.MustCompile(rule.Pattern)
			result = re.ReplaceAllString(result, rule.Replacement)
		}
	}
	return result
}
