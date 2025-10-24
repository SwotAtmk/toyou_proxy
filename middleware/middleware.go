package middleware

import (
	"net/http"
)

// Context 中间件上下文
type Context struct {
	Request     *http.Request
	Response    http.ResponseWriter
	TargetURL   string
	ServiceName string
	Aborted     bool
	StatusCode  int
}

// Middleware 中间件接口
type Middleware interface {
	Name() string
	Handle(ctx *Context) bool // 返回false表示中断请求处理
}

// MiddlewareChain 中间件链
type MiddlewareChain struct {
	middlewares []Middleware
}

// NewMiddlewareChain 创建新的中间件链
func NewMiddlewareChain() *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: make([]Middleware, 0),
	}
}

// Add 添加中间件
func (mc *MiddlewareChain) Add(mw Middleware) {
	mc.middlewares = append(mc.middlewares, mw)
}

// Execute 执行中间件链
func (mc *MiddlewareChain) Execute(ctx *Context) bool {
	for _, mw := range mc.middlewares {
		if !mw.Handle(ctx) {
			ctx.Aborted = true
			return false
		}
		if ctx.Aborted {
			return false
		}
	}
	return true
}

// GetMiddlewareNames 获取中间件名称列表
func (mc *MiddlewareChain) GetMiddlewareNames() []string {
	names := make([]string, len(mc.middlewares))
	for i, mw := range mc.middlewares {
		names[i] = mw.Name()
	}
	return names
}