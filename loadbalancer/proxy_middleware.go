package loadbalancer

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// LoadBalancedProxy 负载均衡代理
type LoadBalancedProxy struct {
	loadBalancer LoadBalancer
	transport    http.RoundTripper
}

// NewLoadBalancedProxy 创建负载均衡代理
func NewLoadBalancedProxy(lb LoadBalancer) *LoadBalancedProxy {
	return &LoadBalancedProxy{
		loadBalancer: lb,
		transport: &http.Transport{
			// 配置传输参数
			ResponseHeaderTimeout: 30 * time.Second,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
		},
	}
}

// ServeHTTP 处理HTTP请求
func (p *LoadBalancedProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// 选择后端服务器
	backend, err := p.loadBalancer.NextBackend(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("No available backend: %v", err), http.StatusServiceUnavailable)
		return
	}

	// 增加连接计数
	p.loadBalancer.IncrementConnection(backend.URL)
	defer p.loadBalancer.DecrementConnection(backend.URL)

	// 记录开始时间
	startTime := time.Now()

	// 创建新的请求
	outReq := new(http.Request)
	*outReq = *req

	// 更新URL
	targetURL, err := url.Parse(backend.URL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid backend URL: %v", err), http.StatusInternalServerError)
		return
	}

	// 设置请求URL
	outReq.URL.Scheme = targetURL.Scheme
	outReq.URL.Host = targetURL.Host

	// 保留原始路径和查询参数
	if targetURL.Path != "" && targetURL.Path != "/" {
		outReq.URL.Path = targetURL.Path
	}

	// 设置Host头
	outReq.Host = targetURL.Host

	// 创建响应写入器包装器，用于捕获响应
	recorder := &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
	}

	// 发送请求
	resp, err := p.transport.RoundTrip(outReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Backend request failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 复制响应头
	for key, values := range resp.Header {
		for _, value := range values {
			recorder.Header().Add(key, value)
		}
	}

	// 设置状态码
	recorder.statusCode = resp.StatusCode

	// 复制响应体
	_, err = io.Copy(recorder.body, resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read response body: %v", err), http.StatusInternalServerError)
		return
	}

	// 更新响应时间
	responseTime := time.Since(startTime)
	p.loadBalancer.UpdateResponseTime(backend.URL, responseTime)

	// 将响应写入原始响应写入器
	recorder.flush()
}

// responseRecorder 响应记录器，用于捕获和修改响应
type responseRecorder struct {
	http.ResponseWriter
	statusCode  int
	body        *bytes.Buffer
	wroteHeader bool
}

// WriteHeader 记录状态码
func (r *responseRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.statusCode = code
		r.wroteHeader = true
		r.ResponseWriter.WriteHeader(code)
	}
}

// Write 记录响应体
func (r *responseRecorder) Write(data []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.body.Write(data)
}

// flush 将记录的响应写入原始响应写入器
func (r *responseRecorder) flush() {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	// 写入响应体
	io.Copy(r.ResponseWriter, r.body)
}

// LoadBalancerMiddleware 负载均衡中间件
type LoadBalancerMiddleware struct {
	proxy *LoadBalancedProxy
}

// NewLoadBalancerMiddleware 创建负载均衡中间件
func NewLoadBalancerMiddleware(lb LoadBalancer) *LoadBalancerMiddleware {
	return &LoadBalancerMiddleware{
		proxy: NewLoadBalancedProxy(lb),
	}
}

// Name 返回中间件名称
func (m *LoadBalancerMiddleware) Name() string {
	return "LoadBalancerMiddleware"
}

// Handle 处理HTTP请求
func (m *LoadBalancerMiddleware) Handle(c *middlewareContext) error {
	m.proxy.ServeHTTP(c.Response, c.Request)
	return nil
}

// middlewareContext 中间件上下文，用于兼容现有的中间件接口
type middlewareContext struct {
	Request  *http.Request
	Response http.ResponseWriter
	Values   map[string]interface{}
}

// 为了兼容现有的中间件系统，我们需要定义一些接口
// 这里我们假设现有的中间件系统有类似的接口定义

// MiddlewareContext 中间件上下文接口
type MiddlewareContext interface {
	// GetRequest 获取HTTP请求
	GetRequest() *http.Request

	// GetResponse 获取HTTP响应写入器
	GetResponse() http.ResponseWriter

	// SetValue 设置上下文值
	SetValue(key string, value interface{})

	// GetValue 获取上下文值
	GetValue(key string) (interface{}, bool)
}

// 实现MiddlewareContext接口
func (c *middlewareContext) GetRequest() *http.Request {
	return c.Request
}

func (c *middlewareContext) GetResponse() http.ResponseWriter {
	return c.Response
}

func (c *middlewareContext) SetValue(key string, value interface{}) {
	if c.Values == nil {
		c.Values = make(map[string]interface{})
	}
	c.Values[key] = value
}

func (c *middlewareContext) GetValue(key string) (interface{}, bool) {
	if c.Values == nil {
		return nil, false
	}
	value, exists := c.Values[key]
	return value, exists
}

// ConvertToMiddlewareContext 将现有的中间件上下文转换为我们的上下文
// 这个函数需要根据实际的中间件系统实现进行调整
func ConvertToMiddlewareContext(ctx interface{}) *middlewareContext {
	// 这里需要根据实际的中间件系统实现进行调整
	// 假设传入的ctx已经实现了GetRequest和GetResponse方法

	// 使用类型断言检查是否实现了我们期望的方法
	type requestGetter interface {
		GetRequest() *http.Request
	}

	type responseGetter interface {
		GetResponse() http.ResponseWriter
	}

	var req *http.Request
	var resp http.ResponseWriter

	if rg, ok := ctx.(requestGetter); ok {
		req = rg.GetRequest()
	}

	if rg, ok := ctx.(responseGetter); ok {
		resp = rg.GetResponse()
	}

	return &middlewareContext{
		Request:  req,
		Response: resp,
		Values:   make(map[string]interface{}),
	}
}

// CreateLoadBalancedHandler 创建负载均衡处理器
// 这个函数可以用于创建一个标准的http.Handler，用于集成到现有的代理系统中
func CreateLoadBalancedHandler(lb LoadBalancer) http.Handler {
	return NewLoadBalancedProxy(lb)
}

// CreateLoadBalancedReverseProxy 创建负载均衡反向代理
// 这个函数可以用于创建一个反向代理，用于替换现有的反向代理实现
func CreateLoadBalancedReverseProxy(lb LoadBalancer, target *url.URL) *ReverseProxy {
	return &ReverseProxy{
		Director: func(req *http.Request) {
			// 选择后端服务器
			backend, err := lb.NextBackend(req)
			if err != nil {
				// 如果没有可用的后端，使用默认目标
				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host
				return
			}

			// 更新URL
			backendURL, err := url.Parse(backend.URL)
			if err != nil {
				// 如果解析失败，使用默认目标
				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host
				return
			}

			req.URL.Scheme = backendURL.Scheme
			req.URL.Host = backendURL.Host

			// 设置Host头
			req.Host = backendURL.Host
		},
		Transport: &LoadBalancerTransport{
			LoadBalancer: lb,
		},
		ModifyResponse: func(resp *http.Response) error {
			// 这里可以添加响应修改逻辑
			return nil
		},
	}
}

// LoadBalancerTransport 负载均衡传输器
type LoadBalancerTransport struct {
	LoadBalancer LoadBalancer
	Transport    http.RoundTripper
}

// RoundTrip 实现http.RoundTripper接口
func (t *LoadBalancerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 使用默认传输器发送请求
	if t.Transport == nil {
		t.Transport = http.DefaultTransport
	}

	startTime := time.Now()
	resp, err := t.Transport.RoundTrip(req)

	// 更新响应时间
	if err == nil && resp != nil {
		responseTime := time.Since(startTime)

		// 从URL中提取后端URL
		backendURL := req.URL.Scheme + "://" + req.URL.Host
		t.LoadBalancer.UpdateResponseTime(backendURL, responseTime)
	}

	return resp, err
}

// ReverseProxy 反向代理，复制自标准库但添加了负载均衡支持
type ReverseProxy struct {
	// Director 函数必须在发送请求前修改请求
	// 请求URL的路径必须保持不变
	Director func(*http.Request)

	// Transport 指定执行请求的传输机制
	// 如果为nil，则使用http.DefaultTransport
	Transport http.RoundTripper

	// ModifyResponse 是一个可选函数，用于修改从后端收到的响应
	// 如果返回错误，则调用ErrorHandler处理错误
	ModifyResponse func(*http.Response) error

	// ErrorHandler 是一个可选函数，用于处理从后端返回的错误
	// 如果为nil，则使用默认错误处理
	ErrorHandler func(http.ResponseWriter, *http.Request, error)
}

// ServeHTTP 实现http.Handler接口
func (p *ReverseProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	transport := p.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	// 复制请求
	outreq := req.Clone(req.Context())
	if req.ContentLength == 0 {
		outreq.Body = nil
	}

	// 调用Director函数修改请求
	if p.Director != nil {
		p.Director(outreq)
	}

	// 发送请求
	res, err := transport.RoundTrip(outreq)
	if err != nil {
		p.errorHandler(rw, req, err)
		return
	}

	// 修改响应
	if p.ModifyResponse != nil {
		if err := p.ModifyResponse(res); err != nil {
			p.errorHandler(rw, req, err)
			res.Body.Close()
			return
		}
	}

	// 复制响应头
	for k, vv := range res.Header {
		for _, v := range vv {
			rw.Header().Add(k, v)
		}
	}

	// 设置状态码
	rw.WriteHeader(res.StatusCode)

	// 复制响应体
	io.Copy(rw, res.Body)
	res.Body.Close()
}

// errorHandler 处理错误的默认实现
func (p *ReverseProxy) errorHandler(rw http.ResponseWriter, req *http.Request, err error) {
	http.Error(rw, "Error proxying request: "+err.Error(), http.StatusBadGateway)
}
