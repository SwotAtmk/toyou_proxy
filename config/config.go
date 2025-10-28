package config

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 表示整个代理服务的配置
type Config struct {
	// 配置文件目录
	ConfigDir string `yaml:"config_dir"`
	// 域名匹配规则
	HostRules []HostRule `yaml:"host_rules"`
	// 路由匹配规则
	RouteRules []RouteRule `yaml:"route_rules"`
	// 服务定义
	Services map[string]Service `yaml:"services"`
	// 中间件配置
	Middlewares []Middleware `yaml:"middlewares"`
	// 中间件服务注册（支持自定义名称注册）
	MiddlewareServices []MiddlewareService `yaml:"middleware_services"`
	// 高级配置
	Advanced AdvancedConfig `yaml:"advanced"`
}

// HostRule 域名匹配规则
type HostRule struct {
	Pattern     string      `yaml:"pattern"`
	Port        int         `yaml:"port"`
	Target      string      `yaml:"target"`
	Middlewares []string    `yaml:"middlewares,omitempty"` // 域名级中间件装配
	RouteRules  []RouteRule `yaml:"route_rules,omitempty"`
}

// RouteRule 路由匹配规则
type RouteRule struct {
	Pattern     string   `yaml:"pattern"`
	Target      string   `yaml:"target"`
	Middlewares []string `yaml:"middlewares,omitempty"` // 路由级中间件装配
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

// MiddlewareService 中间件服务定义，支持自定义名称注册
// 这些中间件服务可以灵活挂载到各个路由规则进行使用
type MiddlewareService struct {
	Name        string                 `yaml:"name"`        // 中间件服务名称（自定义标识符）
	Type        string                 `yaml:"type"`        // 中间件类型（auth、rate_limit、cors、logging等）
	Enabled     bool                   `yaml:"enabled"`     // 是否启用
	IsGlobal    bool                   `yaml:"is_global"`   // 是否全局加载（默认false）
	Config      map[string]interface{} `yaml:"config"`      // 中间件配置
	Description string                 `yaml:"description"` // 中间件描述（可选）
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
	// 先加载单个配置文件
	config, err := loadSingleConfig(filename)
	if err != nil {
		return nil, err
	}

	// 如果配置了config_dir，则加载多文件配置
	if config.ConfigDir != "" {
		return loadMultiFileConfig(filename, config.ConfigDir)
	}

	return config, nil
}

// loadSingleConfig 加载单个配置文件（不处理多文件配置）
func loadSingleConfig(filename string) (*Config, error) {
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

// loadMultiFileConfig 加载多文件配置
func loadMultiFileConfig(mainConfigFile, configDir string) (*Config, error) {
	// 获取主配置文件所在目录
	mainDir := filepath.Dir(mainConfigFile)
	fullConfigDir := filepath.Join(mainDir, configDir)

	// 检查配置目录是否存在
	if _, err := os.Stat(fullConfigDir); os.IsNotExist(err) {
		log.Printf("配置目录不存在: %s，仅使用主配置文件", fullConfigDir)
		return loadSingleConfig(mainConfigFile)
	}

	// 加载主配置
	mainConfig, err := loadSingleConfig(mainConfigFile)
	if err != nil {
		return nil, err
	}

	// 扫描配置目录下的所有.yaml文件
	files, err := ioutil.ReadDir(fullConfigDir)
	if err != nil {
		return nil, err
	}

	// 合并所有配置
	mergedConfig := mainConfig
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".yaml") {
			configFile := filepath.Join(fullConfigDir, file.Name())
			log.Printf("加载配置文件: %s", configFile)

			partialConfig, err := loadSingleConfig(configFile)
			if err != nil {
				log.Printf("加载配置文件失败 %s: %v", configFile, err)
				continue
			}

			// 合并配置
			mergedConfig = mergeConfigs(mergedConfig, partialConfig)
		}
	}

	return mergedConfig, nil
}

// mergeConfigs 合并两个配置
func mergeConfigs(base, additional *Config) *Config {
	merged := &Config{
		ConfigDir:          base.ConfigDir,
		HostRules:          append([]HostRule{}, base.HostRules...),
		RouteRules:         append([]RouteRule{}, base.RouteRules...),
		Middlewares:        append([]Middleware{}, base.Middlewares...),
		MiddlewareServices: append([]MiddlewareService{}, base.MiddlewareServices...),
		Advanced:           base.Advanced,
	}

	// 合并Services
	if merged.Services == nil {
		merged.Services = make(map[string]Service)
	}
	for k, v := range base.Services {
		merged.Services[k] = v
	}
	for k, v := range additional.Services {
		merged.Services[k] = v
	}

	// 合并HostRules（包含嵌套的路由规则）
	merged.HostRules = append(merged.HostRules, additional.HostRules...)

	// 注意：RouteRules字段现在主要用于兼容性，实际的路由规则应该定义在HostRules的RouteRules字段中
	// 合并RouteRules（主要用于兼容旧的配置格式）
	merged.RouteRules = append(merged.RouteRules, additional.RouteRules...)

	// 合并Middlewares
	merged.Middlewares = append(merged.Middlewares, additional.Middlewares...)

	// 合并MiddlewareServices
	merged.MiddlewareServices = append(merged.MiddlewareServices, additional.MiddlewareServices...)

	return merged
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
