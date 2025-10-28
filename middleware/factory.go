package middleware

import (
	"fmt"
	"log"
	"reflect"
	"sync"
)

// DefaultMiddlewareFactory 默认中间件工厂实现
type DefaultMiddlewareFactory struct {
	creators map[string]func(config map[string]interface{}) (Middleware, error)
	mu       sync.RWMutex
}

// NewMiddlewareFactory 创建新的中间件工厂
func NewMiddlewareFactory() MiddlewareFactory {
	return &DefaultMiddlewareFactory{
		creators: make(map[string]func(config map[string]interface{}) (Middleware, error)),
	}
}

// CreateMiddleware 创建中间件实例
func (dmf *DefaultMiddlewareFactory) CreateMiddleware(name string, config map[string]interface{}) (Middleware, error) {
	dmf.mu.RLock()
	creator, exists := dmf.creators[name]
	dmf.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("middleware creator for '%s' not found", name)
	}

	middleware, err := creator(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create middleware '%s': %v", name, err)
	}

	log.Printf("Successfully created middleware '%s'", name)
	return middleware, nil
}

// RegisterMiddleware 注册中间件创建函数
func (dmf *DefaultMiddlewareFactory) RegisterMiddleware(name string, creator func(config map[string]interface{}) (Middleware, error)) {
	dmf.mu.Lock()
	defer dmf.mu.Unlock()

	dmf.creators[name] = creator
	log.Printf("Registered middleware creator for '%s'", name)
}

// GetRegisteredMiddlewares 获取已注册的中间件列表
func (dmf *DefaultMiddlewareFactory) GetRegisteredMiddlewares() []string {
	dmf.mu.RLock()
	defer dmf.mu.RUnlock()

	names := make([]string, 0, len(dmf.creators))
	for name := range dmf.creators {
		names = append(names, name)
	}
	return names
}

// UnregisterMiddleware 注销中间件创建函数
func (dmf *DefaultMiddlewareFactory) UnregisterMiddleware(name string) error {
	dmf.mu.Lock()
	defer dmf.mu.Unlock()

	if _, exists := dmf.creators[name]; !exists {
		return fmt.Errorf("middleware creator for '%s' not found", name)
	}

	delete(dmf.creators, name)
	log.Printf("Unregistered middleware creator for '%s'", name)
	return nil
}

// RegisterMiddlewareByType 通过类型注册中间件
func (dmf *DefaultMiddlewareFactory) RegisterMiddlewareByType(name string, middlewareType reflect.Type) error {
	if middlewareType.Kind() != reflect.Ptr {
		return fmt.Errorf("middleware type must be a pointer, got %v", middlewareType.Kind())
	}

	// 检查是否实现了Middleware接口
	middlewareInterface := reflect.TypeOf((*Middleware)(nil)).Elem()
	if !middlewareType.Implements(middlewareInterface) {
		return fmt.Errorf("type %v does not implement the Middleware interface", middlewareType)
	}

	creator := func(config map[string]interface{}) (Middleware, error) {
		// 创建新实例
		middlewareValue := reflect.New(middlewareType.Elem())
		middleware := middlewareValue.Interface().(Middleware)

		// 如果中间件有Init方法，调用它
		if initMethod := middlewareValue.MethodByName("Init"); initMethod.IsValid() {
			args := []reflect.Value{reflect.ValueOf(config)}
			results := initMethod.Call(args)
			if len(results) > 0 && !results[0].IsNil() {
				if err, ok := results[0].Interface().(error); ok {
					return nil, err
				}
			}
		}

		return middleware, nil
	}

	dmf.RegisterMiddleware(name, creator)
	return nil
}

// CreateMiddlewareChainFromConfig 根据配置创建中间件链
func (dmf *DefaultMiddlewareFactory) CreateMiddlewareChainFromConfig(middlewareConfigs []map[string]interface{}) (MiddlewareChain, error) {
	chain := NewMiddlewareChain()

	for _, config := range middlewareConfigs {
		name, ok := config["name"].(string)
		if !ok {
			return nil, fmt.Errorf("middleware config missing 'name' field")
		}

		middleware, err := dmf.CreateMiddleware(name, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create middleware '%s': %v", name, err)
		}

		chain.Add(middleware)
	}

	return chain, nil
}

// ValidateMiddlewareConfig 验证中间件配置
func (dmf *DefaultMiddlewareFactory) ValidateMiddlewareConfig(config map[string]interface{}) error {
	name, ok := config["name"].(string)
	if !ok {
		return fmt.Errorf("middleware config missing 'name' field")
	}

	dmf.mu.RLock()
	_, exists := dmf.creators[name]
	dmf.mu.RUnlock()

	if !exists {
		return fmt.Errorf("unknown middleware type: %s", name)
	}

	return nil
}
