package middleware

import (
	"regexp"
)

// ReplaceRule 替换规则
type ReplaceRule struct {
	Pattern     string `json:"pattern"`
	Replacement string `json:"replacement"`
	Global      bool   `json:"global"`
}

// ApplyReplaceRules 应用替换规则的公共函数
func ApplyReplaceRules(content []byte, rules []ReplaceRule) []byte {
	result := string(content)
	for _, rule := range rules {
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
	return []byte(result)
}