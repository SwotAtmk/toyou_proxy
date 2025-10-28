# SSE中间件实现方案

## 1. SSE中间件设计

### 1.1 中间件结构
SSE中间件将实现为一个新的插件，遵循项目的插件架构模式：

```go
type SSEMiddleware struct {
    // 配置参数
    flushInterval    time.Duration
    readTimeout      time.Duration
    writeTimeout     time.Duration
    enableConnectionTracking bool
    maxConnections   int
    
    // 连接管理
    tracker          *ConnectionTracker
    
    // 统计信息
    stats            *SSEStats
}

type SSEStats struct {
    ActiveConnections int64
    TotalConnections  int64
    BytesTransferred  int64
    Errors            int64
}
```

### 1.2 中间件接口实现
SSE中间件需要实现`Middleware`接口：

```go
// Name 返回中间件名称
func (sm *SSEMiddleware) Name() string {
    return "sse"
}

// Handle 处理SSE逻辑
func (sm *SSEMiddleware) Handle(ctx *middleware.Context) bool {
    req := ctx.Request
    resp := ctx.Response
    
    // 检测SSE请求
    if sm.isSSERequest(req) {
        // 设置SSE相关响应头
        sm.setupSSEResponseHeaders(resp)
        
        // 在上下文中标记为SSE连接
        ctx.Set("isSSEConnection", true)
        ctx.Set("sseFlushInterval", sm.flushInterval)
        
        // 记录连接
        if sm.enableConnectionTracking {
            connID := sm.generateConnectionID(req)
            ctx.Set("sseConnectionID", connID)
            sm.tracker.TrackConnection(connID, req)
            
            // 设置清理函数
            defer sm.tracker.UntrackConnection(connID)
        }
        
        // 更新统计信息
        atomic.AddInt64(&sm.stats.TotalConnections, 1)
        atomic.AddInt64(&sm.stats.ActiveConnections, 1)
    }
    
    return true
}
```

### 1.3 SSE请求检测
实现SSE请求的检测逻辑：

```go
func (sm *SSEMiddleware) isSSERequest(req *http.Request) bool {
    // 检查Accept头
    accept := req.Header.Get("Accept")
    if accept != "" && strings.Contains(accept, "text/event-stream") {
        return true
    }
    
    // 检查特定路径模式
    path := req.URL.Path
    for _, pattern := range sm.getSSEPathPatterns() {
        if matched, _ := filepath.Match(pattern, path); matched {
            return true
        }
    }
    
    // 检查查询参数
    if req.URL.Query().Get("stream") == "sse" {
        return true
    }
    
    return false
}
```

### 1.4 SSE响应头设置
设置SSE响应所需的HTTP头：

```go
func (sm *SSEMiddleware) setupSSEResponseHeaders(resp http.ResponseWriter) {
    // 设置内容类型
    resp.Header().Set("Content-Type", "text/event-stream")
    
    // 禁用缓存
    resp.Header().Set("Cache-Control", "no-cache")
    resp.Header().Set("X-Accel-Buffering", "no") // Nginx兼容
    
    // 保持连接
    resp.Header().Set("Connection", "keep-alive")
    
    // 设置CORS头（如果需要）
    resp.Header().Set("Access-Control-Allow-Origin", "*")
    resp.Header().Set("Access-Control-Allow-Headers", "Cache-Control")
}
```

## 2. 连接跟踪器实现

### 2.1 连接跟踪器结构
```go
type ConnectionTracker struct {
    connections map[string]*SSEConnection
    mu          sync.RWMutex
    maxConnections int
    cleanupInterval time.Duration
    inactiveThreshold time.Duration
    stopChan     chan struct{}
    wg           sync.WaitGroup
}

type SSEConnection struct {
    ID           string
    Request      *http.Request
    StartTime    time.Time
    LastActivity time.Time
    Active       bool
    BytesSent    int64
}
```

### 2.2 连接跟踪方法
```go
// TrackConnection 跟踪新的SSE连接
func (ct *ConnectionTracker) TrackConnection(id string, req *http.Request) error {
    ct.mu.Lock()
    defer ct.mu.Unlock()
    
    // 检查连接数限制
    if len(ct.connections) >= ct.maxConnections {
        return fmt.Errorf("maximum SSE connections (%d) reached", ct.maxConnections)
    }
    
    // 检查是否已存在
    if _, exists := ct.connections[id]; exists {
        return fmt.Errorf("SSE connection %s already tracked", id)
    }
    
    // 创建新连接记录
    ct.connections[id] = &SSEConnection{
        ID:           id,
        Request:      req,
        StartTime:    time.Now(),
        LastActivity: time.Now(),
        Active:       true,
    }
    
    return nil
}

// UntrackConnection 取消跟踪SSE连接
func (ct *ConnectionTracker) UntrackConnection(id string) {
    ct.mu.Lock()
    defer ct.mu.Unlock()
    
    if conn, exists := ct.connections[id]; exists {
        conn.Active = false
        delete(ct.connections, id)
    }
}

// UpdateActivity 更新连接活动时间
func (ct *ConnectionTracker) UpdateActivity(id string, bytesSent int64) {
    ct.mu.Lock()
    defer ct.mu.Unlock()
    
    if conn, exists := ct.connections[id]; exists {
        conn.LastActivity = time.Now()
        conn.BytesSent += bytesSent
    }
}

// GetActiveConnections 获取活跃连接数
func (ct *ConnectionTracker) GetActiveConnections() int {
    ct.mu.RLock()
    defer ct.mu.RUnlock()
    
    return len(ct.connections)
}

// StartCleanupRoutine 启动清理例程
func (ct *ConnectionTracker) StartCleanupRoutine() {
    ct.wg.Add(1)
    go func() {
        defer ct.wg.Done()
        
        ticker := time.NewTicker(ct.cleanupInterval)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                ct.cleanupInactiveConnections()
            case <-ct.stopChan:
                return
            }
        }
    }()
}

// Stop 停止连接跟踪器
func (ct *ConnectionTracker) Stop() {
    close(ct.stopChan)
    ct.wg.Wait()
}

// cleanupInactiveConnections 清理不活跃的连接
func (ct *ConnectionTracker) cleanupInactiveConnections() {
    ct.mu.Lock()
    defer ct.mu.Unlock()
    
    now := time.Now()
    for id, conn := range ct.connections {
        if now.Sub(conn.LastActivity) > ct.inactiveThreshold {
            log.Printf("Cleaning up inactive SSE connection: %s", id)
            delete(ct.connections, id)
        }
    }
}
```

## 3. 代理层增强

### 3.1 修改createReverseProxy方法
在`proxy/proxy_handler.go`中的`createReverseProxy`方法中添加SSE支持：

```go
func (ph *ProxyHandler) createReverseProxy(service *config.Service, ctx *middleware.Context) (*httputil.ReverseProxy, error) {
    targetURL, err := url.Parse(service.URL)
    if err != nil {
        return nil, fmt.Errorf("invalid target URL: %s", service.URL)
    }

    proxy := httputil.NewSingleHostReverseProxy(targetURL)
    
    // 检查是否是SSE连接
    isSSE := false
    var flushInterval time.Duration
    
    if ctx != nil {
        if sseConn, exists := ctx.Get("isSSEConnection"); exists {
            isSSE = sseConn.(bool)
        }
        
        if sseInterval, exists := ctx.Get("sseFlushInterval"); exists {
            if interval, ok := sseInterval.(time.Duration); ok {
                flushInterval = interval
            }
        }
    }
    
    // 为SSE连接设置刷新间隔
    if isSSE && flushInterval > 0 {
        proxy.FlushInterval = flushInterval
        log.Printf("SSE connection detected, setting flush interval to %v", flushInterval)
    }

    // 自定义修改请求
    proxy.Director = func(req *http.Request) {
        req.URL.Scheme = targetURL.Scheme
        req.URL.Host = targetURL.Host
        
        // 设置Host头
        hostHeader := targetURL.Host
        if service.ProxyHost != "" {
            hostHeader = service.ProxyHost
        }
        req.Host = hostHeader
        
        // 设置其他必要的头
        req.Header.Set("X-Forwarded-Proto", "http")
        req.Header.Set("X-Forwarded-Host", req.Host)
        req.Header.Set("X-Forwarded-For", req.RemoteAddr)
        
        // 为SSE连接设置特殊头
        if isSSE {
            req.Header.Set("X-SSE-Proxy", "toyou-proxy")
        }
    }

    // 自定义修改响应
    proxy.ModifyResponse = func(resp *http.Response) error {
        // 添加代理相关响应头
        resp.Header.Set("X-Proxy-By", "toyou-proxy")
        resp.Header.Set("X-Target-Service", ph.getServiceName(service.URL))
        
        // 为SSE响应设置特殊头
        if isSSE {
            resp.Header.Set("X-SSE-Proxy", "toyou-proxy")
            // 确保不缓存SSE响应
            resp.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
            resp.Header.Set("Pragma", "no-cache")
            resp.Header.Set("Expires", "0")
            
            // 禁用缓冲（适用于某些代理服务器）
            resp.Header.Set("X-Accel-Buffering", "no")
        }

        // 从上下文中获取替换规则
        if ctx != nil {
            if rules, exists := ctx.Get("replaceRules"); exists {
                if replaceRules, ok := rules.([]middleware.ReplaceRule); ok && len(replaceRules) > 0 {
                    // 读取响应体
                    body, err := io.ReadAll(resp.Body)
                    if err != nil {
                        return err
                    }
                    resp.Body.Close()

                    // 应用替换规则
                    modifiedBody := applyReplaceRules(body, replaceRules)

                    // 重新设置响应体
                    resp.Body = io.NopCloser(bytes.NewReader(modifiedBody))
                    resp.ContentLength = int64(len(modifiedBody))
                    resp.Header.Set("Content-Length", strconv.Itoa(len(modifiedBody)))
                }
            }
        }

        return nil
    }

    // 自定义错误处理
    proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
        log.Printf("Proxy error: %v", err)
        
        // 为SSE连接提供特殊错误处理
        if isSSE {
            w.Header().Set("Content-Type", "text/event-stream")
            w.WriteHeader(http.StatusBadGateway)
            fmt.Fprintf(w, "event: error\ndata: Service unavailable\n\n")
            return
        }
        
        http.Error(w, "Service unavailable", http.StatusBadGateway)
    }

    return proxy, nil
}
```

### 3.2 响应写入器包装
为了更好地支持SSE，我们可以创建一个响应写入器包装器：

```go
type SSEResponseWriter struct {
    http.ResponseWriter
    connectionID string
    tracker      *ConnectionTracker
    flushed      bool
}

func NewSSEResponseWriter(w http.ResponseWriter, connectionID string, tracker *ConnectionTracker) *SSEResponseWriter {
    return &SSEResponseWriter{
        ResponseWriter: w,
        connectionID:   connectionID,
        tracker:        tracker,
        flushed:        false,
    }
}

func (w *SSEResponseWriter) Write(data []byte) (int, error) {
    n, err := w.ResponseWriter.Write(data)
    
    // 更新连接活动
    if w.tracker != nil && w.connectionID != "" {
        w.tracker.UpdateActivity(w.connectionID, int64(n))
    }
    
    // 立即刷新数据
    if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
        flusher.Flush()
        w.flushed = true
    }
    
    return n, err
}

func (w *SSEResponseWriter) Flush() {
    if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
        flusher.Flush()
        w.flushed = true
    }
}
```

## 4. 配置系统扩展

### 4.1 配置结构扩展
扩展配置结构以支持SSE配置：

```go
// 在config/config.go中添加SSE配置结构
type SSEConfig struct {
    Enabled              bool          `yaml:"enabled"`
    FlushInterval        time.Duration `yaml:"flush_interval"`
    ReadTimeout          time.Duration `yaml:"read_timeout"`
    WriteTimeout         time.Duration `yaml:"write_timeout"`
    ConnectionTracking   bool          `yaml:"connection_tracking"`
    MaxConnections       int           `yaml:"max_connections"`
    CleanupInterval      time.Duration `yaml:"cleanup_interval"`
    InactiveThreshold    time.Duration `yaml:"inactive_threshold"`
    PathPatterns         []string      `yaml:"path_patterns"`
}

// 在Service结构中添加SSE配置
type Service struct {
    URL       string    `yaml:"url"`
    ProxyHost string    `yaml:"proxy_host,omitempty"`
    SSE       *SSEConfig `yaml:"sse,omitempty"`
}

// 在AdvancedConfig中添加全局SSE配置
type AdvancedConfig struct {
    Timeout  TimeoutConfig  `yaml:"timeout"`
    Port     int            `yaml:"port"`
    Security SecurityConfig `yaml:"security"`
    SSE      SSEConfig      `yaml:"sse"`
}
```

### 4.2 配置加载与验证
添加SSE配置的加载和验证逻辑：

```go
// 在config/config.go中添加SSE配置验证
func validateSSEConfig(sseConfig SSEConfig) error {
    if sseConfig.Enabled {
        if sseConfig.MaxConnections <= 0 {
            return fmt.Errorf("sse.max_connections must be positive when sse is enabled")
        }
        
        if sseConfig.FlushInterval <= 0 {
            return fmt.Errorf("sse.flush_interval must be positive when sse is enabled")
        }
        
        if sseConfig.CleanupInterval <= 0 {
            return fmt.Errorf("sse.cleanup_interval must be positive when sse is enabled")
        }
        
        if sseConfig.InactiveThreshold <= 0 {
            return fmt.Errorf("sse.inactive_threshold must be positive when sse is enabled")
        }
    }
    
    return nil
}
```

## 5. 插件实现

### 5.1 插件主文件
创建`middleware/plugins/sse/plugin.go`文件：

```go
package main

import (
    "time"
    "toyou-proxy/middleware"
)

// SSEMiddleware SSE中间件
type SSEMiddleware struct {
    // 配置参数
    flushInterval    time.Duration
    readTimeout      time.Duration
    writeTimeout     time.Duration
    enableConnectionTracking bool
    maxConnections   int
    pathPatterns     []string
    
    // 连接管理
    tracker          *ConnectionTracker
    
    // 统计信息
    stats            *SSEStats
}

// NewSSEMiddleware 创建SSE中间件
func NewSSEMiddleware(config map[string]interface{}) (middleware.Middleware, error) {
    // 解析配置
    flushInterval := 100 * time.Millisecond
    if fi, ok := config["flush_interval"].(string); ok {
        if duration, err := time.ParseDuration(fi); err == nil {
            flushInterval = duration
        }
    }
    
    readTimeout := 0 * time.Second
    if rt, ok := config["read_timeout"].(string); ok {
        if duration, err := time.ParseDuration(rt); err == nil {
            readTimeout = duration
        }
    }
    
    writeTimeout := 0 * time.Second
    if wt, ok := config["write_timeout"].(string); ok {
        if duration, err := time.ParseDuration(wt); err == nil {
            writeTimeout = duration
        }
    }
    
    enableConnectionTracking := true
    if ect, ok := config["connection_tracking"].(bool); ok {
        enableConnectionTracking = ect
    }
    
    maxConnections := 1000
    if mc, ok := config["max_connections"].(int); ok {
        maxConnections = mc
    }
    
    var pathPatterns []string
    if pp, ok := config["path_patterns"].([]interface{}); ok {
        for _, pattern := range pp {
            if p, ok := pattern.(string); ok {
                pathPatterns = append(pathPatterns, p)
            }
        }
    }
    
    // 创建中间件
    sm := &SSEMiddleware{
        flushInterval:           flushInterval,
        readTimeout:             readTimeout,
        writeTimeout:            writeTimeout,
        enableConnectionTracking: enableConnectionTracking,
        maxConnections:          maxConnections,
        pathPatterns:            pathPatterns,
        stats: &SSEStats{},
    }
    
    // 创建连接跟踪器
    if enableConnectionTracking {
        sm.tracker = NewConnectionTracker(maxConnections)
        sm.tracker.StartCleanupRoutine()
    }
    
    return sm, nil
}

// PluginMain 插件入口函数
func PluginMain(config map[string]interface{}) (middleware.Middleware, error) {
    return NewSSEMiddleware(config)
}

// 其他方法实现...
```

### 5.2 插件配置文件
创建`middleware/plugins/sse/plugin.json`文件：

```json
{
    "name": "sse",
    "version": "1.0.0",
    "description": "Server-Sent Events (SSE) support middleware",
    "type": "middleware",
    "entry": "plugin.go",
    "config_schema": {
        "flush_interval": {
            "type": "duration",
            "default": "100ms",
            "description": "Interval for flushing SSE data"
        },
        "read_timeout": {
            "type": "duration",
            "default": "0s",
            "description": "Read timeout for SSE connections (0 to disable)"
        },
        "write_timeout": {
            "type": "duration",
            "default": "0s",
            "description": "Write timeout for SSE connections (0 to disable)"
        },
        "connection_tracking": {
            "type": "boolean",
            "default": true,
            "description": "Enable SSE connection tracking"
        },
        "max_connections": {
            "type": "integer",
            "default": 1000,
            "description": "Maximum number of concurrent SSE connections"
        },
        "path_patterns": {
            "type": "array",
            "default": ["/events/*", "/stream/*"],
            "description": "Path patterns that should be treated as SSE endpoints"
        }
    }
}
```

## 6. 测试方案

### 6.1 单元测试
创建`middleware/plugins/sse/plugin_test.go`文件：

```go
package main

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
    "toyou-proxy/middleware"
)

func TestSSEMiddleware(t *testing.T) {
    // 创建SSE中间件
    config := map[string]interface{}{
        "flush_interval": "50ms",
        "connection_tracking": true,
        "max_connections": 100,
        "path_patterns": []string{"/events/*", "/stream/*"},
    }
    
    sm, err := NewSSEMiddleware(config)
    if err != nil {
        t.Fatalf("Failed to create SSE middleware: %v", err)
    }
    
    // 测试SSE请求检测
    tests := []struct {
        name           string
        path           string
        acceptHeader   string
        expectedSSE    bool
    }{
        {"SSE by Accept header", "/events", "text/event-stream", true},
        {"SSE by path pattern", "/events/data", "", true},
        {"Non-SSE request", "/api/data", "application/json", false},
        {"SSE by query param", "/api/data?stream=sse", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest("GET", tt.path, nil)
            if tt.acceptHeader != "" {
                req.Header.Set("Accept", tt.acceptHeader)
            }
            
            w := httptest.NewRecorder()
            ctx := &middleware.Context{
                Request:  req,
                Response: w,
                Values:   make(map[string]interface{}),
            }
            
            result := sm.Handle(ctx)
            
            // 检查中间件是否继续执行
            if !result {
                t.Error("Middleware should continue execution")
            }
            
            // 检查SSE标记
            isSSE, exists := ctx.Get("isSSEConnection")
            if tt.expectedSSE {
                if !exists || !isSSE.(bool) {
                    t.Error("Expected SSE connection to be detected")
                }
                
                // 检查响应头
                contentType := w.Header().Get("Content-Type")
                if contentType != "text/event-stream" {
                    t.Errorf("Expected Content-Type to be text/event-stream, got %s", contentType)
                }
            } else {
                if exists && isSSE.(bool) {
                    t.Error("Expected SSE connection not to be detected")
                }
            }
        })
    }
}

func TestConnectionTracker(t *testing.T) {
    tracker := NewConnectionTracker(2)
    defer tracker.Stop()
    
    // 测试连接跟踪
    req := httptest.NewRequest("GET", "/events", nil)
    err := tracker.TrackConnection("conn1", req)
    if err != nil {
        t.Fatalf("Failed to track connection: %v", err)
    }
    
    // 检查连接数
    if count := tracker.GetActiveConnections(); count != 1 {
        t.Errorf("Expected 1 active connection, got %d", count)
    }
    
    // 测试连接更新
    tracker.UpdateActivity("conn1", 1024)
    
    // 测试连接取消跟踪
    tracker.UntrackConnection("conn1")
    
    // 检查连接数
    if count := tracker.GetActiveConnections(); count != 0 {
        t.Errorf("Expected 0 active connections, got %d", count)
    }
    
    // 测试连接数限制
    tracker.TrackConnection("conn2", req)
    err = tracker.TrackConnection("conn3", req)
    if err == nil {
        t.Error("Expected error when exceeding max connections")
    }
}
```

### 6.2 集成测试
创建集成测试，验证SSE数据传输：

```go
func TestSSEIntegration(t *testing.T) {
    // 创建测试服务器
    backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/event-stream")
        w.Header().Set("Cache-Control", "no-cache")
        w.Header().Set("Connection", "keep-alive")
        
        // 发送SSE数据
        for i := 0; i < 5; i++ {
            fmt.Fprintf(w, "data: Message %d\n\n", i)
            if f, ok := w.(http.Flusher); ok {
                f.Flush()
            }
            time.Sleep(100 * time.Millisecond)
        }
    }))
    defer backend.Close()
    
    // 配置代理
    config := &config.Config{
        Services: map[string]config.Service{
            "sse-service": {
                URL: backend.URL,
                SSE: &config.SSEConfig{
                    Enabled:       true,
                    FlushInterval: 50 * time.Millisecond,
                },
            },
        },
        HostRules: []config.HostRule{
            {
                Pattern: "sse.example.com",
                Target: "sse-service",
                Middlewares: []string{"sse"},
            },
        },
        MiddlewareServices: []config.MiddlewareService{
            {
                Name:    "sse",
                Type:    "sse",
                Enabled: true,
                Config: map[string]interface{}{
                    "flush_interval": "50ms",
                },
            },
        },
    }
    
    // 创建代理处理器
    proxyHandler, err := NewProxyHandler(config)
    if err != nil {
        t.Fatalf("Failed to create proxy handler: %v", err)
    }
    
    // 创建测试请求
    req := httptest.NewRequest("GET", "http://sse.example.com/events", nil)
    req.Header.Set("Accept", "text/event-stream")
    
    w := httptest.NewRecorder()
    
    // 执行代理
    proxyHandler.ServeHTTP(w, req)
    
    // 验证响应
    resp := w.Result()
    if resp.StatusCode != http.StatusOK {
        t.Errorf("Expected status 200, got %d", resp.StatusCode)
    }
    
    contentType := resp.Header.Get("Content-Type")
    if contentType != "text/event-stream" {
        t.Errorf("Expected Content-Type text/event-stream, got %s", contentType)
    }
    
    // 验证响应体
    body := w.Body.String()
    expected := "data: Message 0\n\ndata: Message 1\n\ndata: Message 2\n\ndata: Message 3\n\ndata: Message 4\n\n"
    if body != expected {
        t.Errorf("Expected body %q, got %q", expected, body)
    }
}
```

## 7. 部署与配置

### 7.1 配置示例
提供完整的配置示例：

```yaml
# config.yaml
config_dir: "conf.d"

# 全局SSE配置
advanced:
  port: 80
  sse:
    enabled: true
    flush_interval: 100ms
    read_timeout: 0s
    write_timeout: 0s
    connection_tracking: true
    max_connections: 1000
    cleanup_interval: 5m
    inactive_threshold: 30m
    path_patterns:
      - "/events/*"
      - "/stream/*"
      - "/sse/*"

# 中间件服务注册
middleware_services:
  - name: "sse_support"
    type: "sse"
    enabled: true
    is_global: true
    description: "Server-Sent Events support middleware"
    config:
      flush_interval: "100ms"
      connection_tracking: true
      max_connections: 1000
      path_patterns:
        - "/events/*"
        - "/stream/*"

# 服务定义
services:
  sse-backend:
    url: "http://localhost:8080"
    sse:
      enabled: true
      flush_interval: "50ms"
      connection_tracking: true
      max_connections: 500

# 域名规则
host_rules:
  - pattern: "sse.example.com"
    port: 80
    target: "sse-backend"
    middlewares: ["sse_support"]
    route_rules:
      - pattern: "/events/*"
        target: "sse-backend"
        middlewares: ["sse_support"]
      - pattern: "/api/*"
        target: "sse-backend"
```

### 7.2 部署步骤
1. **编译插件**：
   ```bash
   cd middleware/plugins/sse
   go build -buildmode=plugin -o sse.so plugin.go
   ```

2. **配置中间件**：
   - 在配置文件中添加SSE中间件配置
   - 指定需要SSE支持的服务和路径

3. **启动代理服务**：
   ```bash
   ./toyou-proxy -config config.yaml
   ```

4. **验证SSE功能**：
   - 使用curl测试SSE端点
   - 检查日志确认SSE连接被正确处理

### 7.3 监控与日志
添加SSE相关的监控指标和日志：

```go
// 在SSE中间件中添加监控指标
func (sm *SSEMiddleware) logConnectionEvent(event string, connID string) {
    log.Printf("[SSE] %s: connection=%s, active=%d, total=%d", 
        event, connID, 
        atomic.LoadInt64(&sm.stats.ActiveConnections),
        atomic.LoadInt64(&sm.stats.TotalConnections))
}

// 定期输出统计信息
func (sm *SSEMiddleware) logStats() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        active := atomic.LoadInt64(&sm.stats.ActiveConnections)
        total := atomic.LoadInt64(&sm.stats.TotalConnections)
        bytes := atomic.LoadInt64(&sm.stats.BytesTransferred)
        errors := atomic.LoadInt64(&sm.stats.Errors)
        
        log.Printf("[SSE] Stats - Active: %d, Total: %d, Bytes: %d, Errors: %d", 
            active, total, bytes, errors)
    }
}
```

## 8. 总结

本实现方案详细描述了如何在当前反向代理项目中添加SSE支持，包括：

1. **SSE中间件设计**：实现了SSE请求检测、响应头设置和连接管理
2. **连接跟踪器**：提供了SSE连接的跟踪、限制和清理机制
3. **代理层增强**：修改了反向代理实现，添加了SSE特定的处理逻辑
4. **配置系统扩展**：扩展了配置结构，支持SSE相关配置选项
5. **插件实现**：按照项目插件架构实现了SSE中间件插件
6. **测试方案**：提供了单元测试和集成测试，确保功能正确性
7. **部署与配置**：提供了完整的配置示例和部署步骤

通过这些改进，项目将能够很好地支持SSE数据传输，满足现代Web应用对实时数据传输的需求。