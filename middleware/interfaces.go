package middleware

import (
	"net/http"
	"toyou-proxy/config"
)

// Middleware 中间件接口
type Middleware interface {
	// Name 返回中间件名称
	Name() string

	// Handle 处理请求
	// 返回true表示继续执行下一个中间件，false表示中断请求处理
	Handle(ctx *Context) bool
}

// Context 中间件上下文
type Context struct {
	Request     *http.Request
	Response    http.ResponseWriter
	Values      map[string]interface{} // 用于中间件间传递数据
	TargetURL   string                 // 目标服务URL
	ServiceName string                 // 服务名称
	StatusCode  int                    // 状态码，用于中间件设置响应状态
}

// Get 从上下文中获取值
func (c *Context) Get(key string) (interface{}, bool) {
	if c.Values == nil {
		return nil, false
	}
	value, exists := c.Values[key]
	return value, exists
}

// Set 在上下文中设置值
func (c *Context) Set(key string, value interface{}) {
	if c.Values == nil {
		c.Values = make(map[string]interface{})
	}
	c.Values[key] = value
}

// Plugin 插件接口
type Plugin interface {
	// Name 返回插件名称
	Name() string

	// Version 返回插件版本
	Version() string

	// Description 返回插件描述
	Description() string

	// Init 初始化插件
	Init(config map[string]interface{}) error

	// CreateMiddleware 创建中间件实例
	CreateMiddleware() (Middleware, error)

	// Stop 停止插件
	Stop() error
}

// PluginManager 插件管理器接口
type PluginManager interface {
	// LoadPlugin 加载插件
	LoadPlugin(pluginPath string) error

	// UnloadPlugin 卸载插件
	UnloadPlugin(pluginName string) error

	// GetPlugin 获取插件
	GetPlugin(pluginName string) (Plugin, bool)

	// ListPlugins 列出所有插件
	ListPlugins() []Plugin

	// ReloadPlugin 重新加载插件
	ReloadPlugin(pluginName string) error

	// GetPluginDir 获取插件目录
	GetPluginDir() string
}

// MiddlewareChain 中间件链接口
type MiddlewareChain interface {
	// Add 添加中间件到链中
	Add(middleware Middleware)

	// Execute 执行中间件链
	Execute(ctx *Context) bool

	// GetMiddlewareNames 获取中间件名称列表
	GetMiddlewareNames() []string

	// GetMiddlewares 获取中间件列表
	GetMiddlewares() []Middleware
}

// MiddlewareFactory 中间件工厂接口
type MiddlewareFactory interface {
	// CreateMiddleware 创建中间件实例
	CreateMiddleware(name string, config map[string]interface{}) (Middleware, error)

	// RegisterMiddleware 注册中间件创建函数
	RegisterMiddleware(name string, creator func(config map[string]interface{}) (Middleware, error))

	// GetRegisteredMiddlewares 获取已注册的中间件列表
	GetRegisteredMiddlewares() []string
}

// PluginMetadata 插件元数据
type PluginMetadata struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"config"`
	Enabled     bool                   `json:"enabled"`
}

// MiddlewareServiceRegistry 中间件服务注册表接口
type MiddlewareServiceRegistry interface {
	// Register 注册中间件服务
	Register(name string, middlewareService config.MiddlewareService) error

	// Get 获取中间件服务
	Get(name string) (config.MiddlewareService, bool)

	// List 列出所有中间件服务
	List() []config.MiddlewareService

	// Init 初始化注册表
	Init(cfg *config.Config) error
}
