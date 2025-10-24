package config

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v3"
)

// Config 表示整个代理服务的配置
type Config struct {
	// 域名匹配规则
	HostRules []HostRule `yaml:"host_rules"`
	// 路由匹配规则
	RouteRules []RouteRule `yaml:"route_rules"`
	// 服务定义
	Services map[string]Service `yaml:"services"`
	// 中间件配置
	Middlewares []Middleware `yaml:"middlewares"`
	// 高级配置
	Advanced AdvancedConfig `yaml:"advanced"`
}

// HostRule 域名匹配规则
type HostRule struct {
	Pattern    string      `yaml:"pattern"`
	Port       int         `yaml:"port"`
	Target     string      `yaml:"target"`
	RouteRules []RouteRule `yaml:"route_rules,omitempty"`
}

// RouteRule 路由匹配规则
type RouteRule struct {
	Pattern string `yaml:"pattern"`
	Target  string `yaml:"target"`
}

// Service 服务定义
type Service struct {
	URL       string `yaml:"url"`
	ProxyHost string `yaml:"proxy_host,omitempty"` // 反向代理时使用的Host头，可选
}

// Middleware 中间件配置
type Middleware struct {
	Name    string                 `yaml:"name"`
	Enabled bool                   `yaml:"enabled"`
	Config  map[string]interface{} `yaml:"config"`
}

// AdvancedConfig 高级配置
type AdvancedConfig struct {
	Timeout  TimeoutConfig  `yaml:"timeout"`
	Port     int            `yaml:"port"`
	Security SecurityConfig `yaml:"security"`
}

// TimeoutConfig 超时配置
type TimeoutConfig struct {
	ReadTimeout  int `yaml:"read_timeout"`
	WriteTimeout int `yaml:"write_timeout"`
	DialTimeout  int `yaml:"dial_timeout"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	DenyHiddenFiles bool `yaml:"deny_hidden_files"`
}

// LoadConfig 从文件加载配置
func LoadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	// 检查必填字段
	if len(c.HostRules) == 0 && len(c.RouteRules) == 0 {
		log.Println("警告: 没有配置任何域名或路由规则")
	}

	// 验证服务定义
	for _, rule := range c.HostRules {
		if _, exists := c.Services[rule.Target]; !exists {
			log.Printf("警告: 域名规则目标服务 '%s' 未定义", rule.Target)
		}
	}

	for _, rule := range c.RouteRules {
		if _, exists := c.Services[rule.Target]; !exists {
			log.Printf("警告: 路由规则目标服务 '%s' 未定义", rule.Target)
		}
	}

	return nil
}
