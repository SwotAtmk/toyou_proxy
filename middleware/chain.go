package middleware

import (
	"sync"
	"fmt"
	"log"
)

// DefaultMiddlewareChain 默认中间件链实现
type DefaultMiddlewareChain struct {
	middlewares []Middleware
	mu          sync.RWMutex
}

// NewMiddlewareChain 创建新的中间件链
func NewMiddlewareChain() MiddlewareChain {
	return &DefaultMiddlewareChain{
		middlewares: make([]Middleware, 0),
	}
}

// Add 添加中间件到链中
func (dmc *DefaultMiddlewareChain) Add(middleware Middleware) {
	dmc.mu.Lock()
	defer dmc.mu.Unlock()
	
	dmc.middlewares = append(dmc.middlewares, middleware)
	log.Printf("Added middleware '%s' to chain", middleware.Name())
}

// Execute 执行中间件链
func (dmc *DefaultMiddlewareChain) Execute(ctx *Context) bool {
	dmc.mu.RLock()
	defer dmc.mu.RUnlock()
	
	for _, middleware := range dmc.middlewares {
		log.Printf("Executing middleware '%s'", middleware.Name())
		if !middleware.Handle(ctx) {
			log.Printf("Middleware '%s' interrupted the chain", middleware.Name())
			return false
		}
	}
	
	return true
}

// GetMiddlewareNames 获取中间件名称列表
func (dmc *DefaultMiddlewareChain) GetMiddlewareNames() []string {
	dmc.mu.RLock()
	defer dmc.mu.RUnlock()
	
	names := make([]string, len(dmc.middlewares))
	for i, middleware := range dmc.middlewares {
		names[i] = middleware.Name()
	}
	return names
}

// Remove 从链中移除中间件
func (dmc *DefaultMiddlewareChain) Remove(name string) error {
	dmc.mu.Lock()
	defer dmc.mu.Unlock()
	
	for i, middleware := range dmc.middlewares {
		if middleware.Name() == name {
			dmc.middlewares = append(dmc.middlewares[:i], dmc.middlewares[i+1:]...)
			log.Printf("Removed middleware '%s' from chain", name)
			return nil
		}
	}
	
	return fmt.Errorf("middleware '%s' not found in chain", name)
}

// Clear 清空中间件链
func (dmc *DefaultMiddlewareChain) Clear() {
	dmc.mu.Lock()
	defer dmc.mu.Unlock()
	
	dmc.middlewares = make([]Middleware, 0)
	log.Println("Cleared middleware chain")
}

// GetMiddleware 根据名称获取中间件
func (dmc *DefaultMiddlewareChain) GetMiddleware(name string) (Middleware, bool) {
	dmc.mu.RLock()
	defer dmc.mu.RUnlock()
	
	for _, middleware := range dmc.middlewares {
		if middleware.Name() == name {
			return middleware, true
		}
	}
	
	return nil, false
}

// Size 获取中间件链大小
func (dmc *DefaultMiddlewareChain) Size() int {
	dmc.mu.RLock()
	defer dmc.mu.RUnlock()
	
	return len(dmc.middlewares)
}

// InsertAt 在指定位置插入中间件
func (dmc *DefaultMiddlewareChain) InsertAt(index int, middleware Middleware) error {
	dmc.mu.Lock()
	defer dmc.mu.Unlock()
	
	if index < 0 || index > len(dmc.middlewares) {
		return fmt.Errorf("index %d out of bounds for middleware chain of size %d", index, len(dmc.middlewares))
	}
	
	dmc.middlewares = append(dmc.middlewares[:index], append([]Middleware{middleware}, dmc.middlewares[index:]...)...)
	log.Printf("Inserted middleware '%s' at position %d", middleware.Name(), index)
	return nil
}