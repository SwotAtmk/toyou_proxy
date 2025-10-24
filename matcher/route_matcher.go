package matcher

import (
	"regexp"
	"strings"
)

// RouteMatcher 路由匹配器
type RouteMatcher struct {
	rules map[string]string // pattern -> target
}

// NewRouteMatcher 创建新的路由匹配器
func NewRouteMatcher() *RouteMatcher {
	return &RouteMatcher{
		rules: make(map[string]string),
	}
}

// AddRule 添加路由匹配规则
func (rm *RouteMatcher) AddRule(pattern, target string) {
	rm.rules[pattern] = target
}

// Match 匹配路由路径，返回目标服务
func (rm *RouteMatcher) Match(path string) (string, bool) {
	// 先尝试精确匹配
	if target, exists := rm.rules[path]; exists {
		return target, true
	}

	// 尝试通配符匹配
	for pattern, target := range rm.rules {
		if strings.HasSuffix(pattern, "/*") {
			prefix := pattern[:len(pattern)-2] // 去掉 "/*"
			if strings.HasPrefix(path, prefix) {
				// 检查是否匹配路径前缀
				if path == prefix || strings.HasPrefix(path, prefix+"/") {
					return target, true
				}
			}
		}
	}

	// 尝试正则表达式匹配
	for pattern, target := range rm.rules {
		if strings.HasPrefix(pattern, "^") && strings.HasSuffix(pattern, "$") {
			// 如果模式以^开头且以$结尾，尝试作为正则表达式匹配
			re, err := regexp.Compile(pattern)
			if err == nil {
				if re.MatchString(path) {
					return target, true
				}
			}
		}
	}

	return "", false
}

// GetAllRules 获取所有规则
func (rm *RouteMatcher) GetAllRules() map[string]string {
	return rm.rules
}
