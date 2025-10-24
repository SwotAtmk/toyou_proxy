package matcher

import (
	"strings"
)

// HostMatcher 域名匹配器
type HostMatcher struct {
	rules map[string]string // pattern -> target
}

// NewHostMatcher 创建新的域名匹配器
func NewHostMatcher() *HostMatcher {
	return &HostMatcher{
		rules: make(map[string]string),
	}
}

// AddRule 添加域名匹配规则
func (hm *HostMatcher) AddRule(pattern, target string) {
	hm.rules[pattern] = target
}

// Match 匹配域名，返回目标服务
func (hm *HostMatcher) Match(host string) (string, bool) {
	// 先尝试精确匹配
	if target, exists := hm.rules[host]; exists {
		return target, true
	}

	// 尝试通配符匹配
	for pattern, target := range hm.rules {
		if strings.HasPrefix(pattern, "*.") {
			domain := pattern[2:] // 去掉 "*."
			if strings.HasSuffix(host, domain) {
				// 检查是否匹配子域名
				if host == domain || strings.HasSuffix(host, "."+domain) {
					return target, true
				}
			}
		}
	}

	return "", false
}

// GetAllRules 获取所有规则
func (hm *HostMatcher) GetAllRules() map[string]string {
	return hm.rules
}