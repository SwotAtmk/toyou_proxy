package middleware

import (
	"toyou-proxy/config"
)

// 全局中间件服务注册表实例
var globalMiddlewareServiceRegistry MiddlewareServiceRegistry

// InitMiddlewareServiceRegistry 初始化中间件服务注册表
func InitMiddlewareServiceRegistry(cfg *config.Config) error {
	// 创建默认的中间件服务注册表实现
	globalMiddlewareServiceRegistry = NewDefaultMiddlewareServiceRegistry()

	// 初始化注册表
	return globalMiddlewareServiceRegistry.Init(cfg)
}

// GetMiddlewareServiceRegistry 获取全局中间件服务注册表
func GetMiddlewareServiceRegistry() MiddlewareServiceRegistry {
	return globalMiddlewareServiceRegistry
}

// DefaultMiddlewareServiceRegistry 默认中间件服务注册表实现
type DefaultMiddlewareServiceRegistry struct {
	services map[string]config.MiddlewareService
}

// NewDefaultMiddlewareServiceRegistry 创建默认中间件服务注册表
func NewDefaultMiddlewareServiceRegistry() MiddlewareServiceRegistry {
	return &DefaultMiddlewareServiceRegistry{
		services: make(map[string]config.MiddlewareService),
	}
}

// Register 注册中间件服务
func (d *DefaultMiddlewareServiceRegistry) Register(name string, middlewareService config.MiddlewareService) error {
	d.services[name] = middlewareService
	return nil
}

// Get 获取中间件服务
func (d *DefaultMiddlewareServiceRegistry) Get(name string) (config.MiddlewareService, bool) {
	service, exists := d.services[name]
	return service, exists
}

// List 列出所有中间件服务
func (d *DefaultMiddlewareServiceRegistry) List() []config.MiddlewareService {
	services := make([]config.MiddlewareService, 0, len(d.services))
	for _, service := range d.services {
		services = append(services, service)
	}
	return services
}

// Init 初始化注册表
func (d *DefaultMiddlewareServiceRegistry) Init(cfg *config.Config) error {
	// 从配置中加载中间件服务
	if cfg != nil && len(cfg.MiddlewareServices) > 0 {
		for _, service := range cfg.MiddlewareServices {
			if service.Enabled {
				d.Register(service.Name, service)
			}
		}
	}
	return nil
}
