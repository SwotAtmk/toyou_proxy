# WebSocket代理功能开发文档

## 1. 项目概述

### 1.1 当前项目分析

Toyou Proxy是一个基于Go语言开发的高性能反向代理服务器，具有以下核心特性：

- **高性能反向代理**：基于Go的高并发特性，支持高吞吐量请求处理
- **灵活的中间件系统**：支持自定义中间件，可在请求处理流程中插入自定义逻辑
- **插件化架构**：支持插件的自动发现、动态加载和热更新
- **多级路由匹配**：支持域名级和路径级路由匹配，满足复杂路由需求
- **SSE支持**：已实现Server-Sent Events(SSE)代理功能，支持实时数据流

### 1.2 WebSocket需求分析

WebSocket是一种在单个TCP连接上进行全双工通信的协议，它使得客户端和服务器之间的数据交换变得更加简单，允许服务端主动向客户端推送数据。在浏览器和服务器之间建立WebSocket连接后，两者就可以随时互相发送数据，而不需要浏览器发起请求。

与HTTP相比，WebSocket具有以下优势：

- **全双工通信**：客户端和服务器可以同时发送数据
- **持久连接**：一旦建立连接，它将保持开放状态，直到客户端或服务器明确关闭它
- **低开销**：WebSocket头部信息比HTTP请求小，减少了数据传输开销
- **实时性**：数据可以立即传输，无需等待请求-响应周期

### 1.3 WebSocket代理的技术挑战

实现WebSocket代理需要解决以下技术挑战：

1. **协议升级**：WebSocket连接始于HTTP请求，需要通过Upgrade头部进行协议升级
2. **长连接管理**：WebSocket连接是长连接，需要有效的连接管理和资源控制
3. **双向数据流**：需要处理客户端到服务器和服务器到客户端的双向数据流
4. **中间件兼容**：确保现有中间件系统与WebSocket代理兼容
5. **安全考虑**：需要处理WSS(WebSocket Secure)的安全连接

## 2. 技术方案设计

### 2.1 WebSocket代理架构

我们将采用以下架构实现WebSocket代理功能：

```
客户端 → Toyou Proxy → 后端WebSocket服务
       ↑
   中间件链
```

关键组件：

1. **WebSocket检测中间件**：识别WebSocket升级请求
2. **WebSocket代理处理器**：处理WebSocket连接的建立和数据转发
3. **连接管理器**：管理活跃的WebSocket连接
4. **协议升级处理器**：处理HTTP到WebSocket的协议升级

### 2.2 实现策略

1. **协议升级处理**：
   - 检测WebSocket升级请求(包含`Upgrade: websocket`头部)
   - 验证升级请求的合法性
   - 与后端服务器建立WebSocket连接
   - 完成客户端和后端之间的协议升级握手

2. **数据转发**：
   - 建立客户端到代理、代理到后端的双向数据通道
   - 实现数据的透明转发
   - 处理连接关闭和错误情况

3. **连接管理**：
   - 跟踪所有活跃的WebSocket连接
   - 实现连接超时和资源清理
   - 提供连接统计和监控功能

4. **中间件集成**：
   - 确保WebSocket连接可以经过中间件链处理
   - 为WebSocket特定的中间件提供上下文信息

### 2.3 配置设计

扩展现有配置结构以支持WebSocket代理：

```yaml
# WebSocket服务定义
services:
  websocket_service:
    url: "ws://localhost:8081"
    # 或
    url: "wss://secure-websocket-server.com"
    # 可选配置
    ping_interval: 30  # 心跳间隔(秒)
    read_timeout: 60   # 读取超时(秒)
    write_timeout: 60  # 写入超时(秒)

# WebSocket路由规则
host_rules:
  - pattern: "ws.example.com"
    port: 80
    target: "websocket_service"
    middlewares: ["websocket", "logging"]
    route_rules:
      - pattern: "/ws/*"
        target: "websocket_service"
        middlewares: ["websocket", "auth"]

# WebSocket中间件配置
middlewares:
  - name: "websocket"
    enabled: true
    config:
      ping_interval: 30
      max_connections: 1000
      enable_origin_check: true
      allowed_origins: ["https://example.com"]
```

## 3. 实现细节

### 3.1 WebSocket检测中间件

创建一个新的WebSocket中间件，用于检测和处理WebSocket连接：

```go
type WebSocketMiddleware struct {
    // 配置参数
    pingInterval     time.Duration
    readTimeout      time.Duration
    writeTimeout     time.Duration
    maxConnections   int
    enableOriginCheck bool
    allowedOrigins   []string
    
    // 连接管理
    connectionTracker *WebSocketConnectionTracker
    
    // 统计信息
    stats *WebSocketStats
}

// Handle 处理WebSocket逻辑
func (wsm *WebSocketMiddleware) Handle(ctx *middleware.Context) bool {
    req := ctx.Request
    
    // 检测WebSocket升级请求
    if wsm.isWebSocketUpgradeRequest(req) {
        // 在上下文中标记为WebSocket连接
        ctx.Set("isWebSocketConnection", true)
        
        // 验证Origin头(如果启用)
        if wsm.enableOriginCheck && !wsm.validateOrigin(req) {
            http.Error(ctx.Response, "Origin not allowed", http.StatusForbidden)
            return false
        }
        
        // 更新统计信息
        atomic.AddInt64(&wsm.stats.TotalConnections, 1)
        atomic.AddInt64(&wsm.stats.ActiveConnections, 1)
        
        // 设置清理函数
        defer func() {
            atomic.AddInt64(&wsm.stats.ActiveConnections, -1)
        }()
        
        log.Printf("[WebSocket] New connection established: %s %s", req.Method, req.URL.Path)
    }
    
    return true
}

// isWebSocketUpgradeRequest 检测是否为WebSocket升级请求
func (wsm *WebSocketMiddleware) isWebSocketUpgradeRequest(req *http.Request) bool {
    // 检查Upgrade头
    upgrade := strings.ToLower(req.Header.Get("Upgrade"))
    connection := strings.ToLower(req.Header.Get("Connection"))
    
    return upgrade == "websocket" && 
           strings.Contains(connection, "upgrade") &&
           req.Header.Get("Sec-WebSocket-Key") != "" &&
           req.Header.Get("Sec-WebSocket-Version") != ""
}

// validateOrigin 验证Origin头
func (wsm *WebSocketMiddleware) validateOrigin(req *http.Request) bool {
    origin := req.Header.Get("Origin")
    if origin == "" {
        return true // 非浏览器请求可能没有Origin头
    }
    
    for _, allowedOrigin := range wsm.allowedOrigins {
        if origin == allowedOrigin || allowedOrigin == "*" {
            return true
        }
    }
    
    return false
}
```

### 3.2 WebSocket代理处理器

修改代理处理器以支持WebSocket连接：

```go
// 在proxy_handler.go中添加WebSocket处理逻辑

// createWebSocketProxy 创建WebSocket代理
func (ph *ProxyHandler) createWebSocketProxy(service *config.Service, ctx *middleware.Context) (http.Handler, error) {
    targetURL, err := url.Parse(service.URL)
    if err != nil {
        return nil, fmt.Errorf("invalid target URL: %v", err)
    }
    
    // 确保目标URL使用ws或wss协议
    if targetURL.Scheme != "ws" && targetURL.Scheme != "wss" {
        // 如果是http/https，转换为ws/wss
        if targetURL.Scheme == "http" {
            targetURL.Scheme = "ws"
        } else if targetURL.Scheme == "https" {
            targetURL.Scheme = "wss"
        } else {
            return nil, fmt.Errorf("unsupported URL scheme for WebSocket: %s", targetURL.Scheme)
        }
    }
    
    // 创建WebSocket代理
    return &WebSocketProxy{
        TargetURL: targetURL,
        Context:   ctx,
    }, nil
}

// WebSocketProxy WebSocket代理处理器
type WebSocketProxy struct {
    TargetURL *url.URL
    Context   *middleware.Context
}

// ServeHTTP 处理WebSocket请求
func (wp *WebSocketProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 检查是否为WebSocket升级请求
    if !isWebSocketUpgradeRequest(r) {
        http.Error(w, "Not a WebSocket upgrade request", http.StatusBadRequest)
        return
    }
    
    // 连接到后端WebSocket服务器
    backendConn, err := wp.dialBackend()
    if err != nil {
        log.Printf("[WebSocket] Failed to connect to backend: %v", err)
        http.Error(w, "Failed to connect to backend", http.StatusBadGateway)
        return
    }
    defer backendConn.Close()
    
    // 完成客户端升级握手
    clientConn, err := wp.upgradeClient(w, r)
    if err != nil {
        log.Printf("[WebSocket] Failed to upgrade client connection: %v", err)
        return
    }
    defer clientConn.Close()
    
    // 开始双向数据转发
    wp.proxyData(clientConn, backendConn)
}

// dialBackend 连接到后端WebSocket服务器
func (wp *WebSocketProxy) dialBackend() (*websocket.Conn, error) {
    // 使用gorilla/websocket库连接后端
    dialer := websocket.DefaultDialer
    dialer.HandshakeTimeout = 10 * time.Second
    
    // 构建后端连接URL
    backendURL := *wp.TargetURL
    backendURL.Path = wp.Context.Request.URL.Path
    backendURL.RawQuery = wp.Context.Request.URL.RawQuery
    
    // 连接后端
    conn, _, err := dialer.Dial(backendURL.String(), nil)
    if err != nil {
        return nil, fmt.Errorf("dial backend failed: %v", err)
    }
    
    return conn, nil
}

// upgradeClient 升级客户端连接
func (wp *WebSocketProxy) upgradeClient(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
    upgrader := websocket.Upgrader{
        ReadBufferSize:  1024,
        WriteBufferSize: 1024,
        CheckOrigin: func(r *http.Request) bool {
            // Origin检查已在中间件中完成
            return true
        },
    }
    
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return nil, fmt.Errorf("upgrade failed: %v", err)
    }
    
    return conn, nil
}

// proxyData 代理数据双向转发
func (wp *WebSocketProxy) proxyData(clientConn, backendConn *websocket.Conn) {
    // 创建错误通道
    errChan := make(chan error, 2)
    
    // 客户端到后端的数据转发
    go func() {
        for {
            messageType, message, err := clientConn.ReadMessage()
            if err != nil {
                errChan <- fmt.Errorf("client read error: %v", err)
                return
            }
            
            err = backendConn.WriteMessage(messageType, message)
            if err != nil {
                errChan <- fmt.Errorf("backend write error: %v", err)
                return
            }
        }
    }()
    
    // 后端到客户端的数据转发
    go func() {
        for {
            messageType, message, err := backendConn.ReadMessage()
            if err != nil {
                errChan <- fmt.Errorf("backend read error: %v", err)
                return
            }
            
            err = clientConn.WriteMessage(messageType, message)
            if err != nil {
                errChan <- fmt.Errorf("client write error: %v", err)
                return
            }
        }
    }()
    
    // 等待任一方向出错
    err := <-errChan
    log.Printf("[WebSocket] Connection closed: %v", err)
}
```

### 3.3 连接管理器

实现WebSocket连接管理器：

```go
// WebSocketConnectionTracker WebSocket连接跟踪器
type WebSocketConnectionTracker struct {
    connections map[string]*WebSocketConnection
    mu          sync.RWMutex
    maxConnections int
    cleanupInterval time.Duration
    inactiveThreshold time.Duration
    stopChan     chan struct{}
    wg           sync.WaitGroup
}

// WebSocketConnection WebSocket连接信息
type WebSocketConnection struct {
    ID           string
    ClientAddr   string
    TargetURL    string
    StartTime    time.Time
    LastActivity time.Time
    Active       bool
    BytesSent    int64
    BytesReceived int64
}

// NewWebSocketConnectionTracker 创建新的连接跟踪器
func NewWebSocketConnectionTracker(maxConnections int) *WebSocketConnectionTracker {
    tracker := &WebSocketConnectionTracker{
        connections: make(map[string]*WebSocketConnection),
        maxConnections: maxConnections,
        cleanupInterval: 5 * time.Minute,
        inactiveThreshold: 1 * time.Hour,
        stopChan: make(chan struct{}),
    }
    
    // 启动清理协程
    tracker.wg.Add(1)
    go tracker.cleanupRoutine()
    
    return tracker
}

// TrackConnection 跟踪新的WebSocket连接
func (ct *WebSocketConnectionTracker) TrackConnection(id, clientAddr, targetURL string) error {
    ct.mu.Lock()
    defer ct.mu.Unlock()
    
    // 检查连接数限制
    if len(ct.connections) >= ct.maxConnections {
        return fmt.Errorf("maximum WebSocket connections (%d) reached", ct.maxConnections)
    }
    
    // 检查是否已存在
    if _, exists := ct.connections[id]; exists {
        return fmt.Errorf("WebSocket connection %s already tracked", id)
    }
    
    // 创建新连接记录
    ct.connections[id] = &WebSocketConnection{
        ID:           id,
        ClientAddr:   clientAddr,
        TargetURL:    targetURL,
        StartTime:    time.Now(),
        LastActivity: time.Now(),
        Active:       true,
    }
    
    return nil
}

// UntrackConnection 取消跟踪WebSocket连接
func (ct *WebSocketConnectionTracker) UntrackConnection(id string) {
    ct.mu.Lock()
    defer ct.mu.Unlock()
    
    if conn, exists := ct.connections[id]; exists {
        conn.Active = false
        delete(ct.connections, id)
    }
}

// UpdateActivity 更新连接活动时间和数据统计
func (ct *WebSocketConnectionTracker) UpdateActivity(id string, bytesSent, bytesReceived int64) {
    ct.mu.Lock()
    defer ct.mu.Unlock()
    
    if conn, exists := ct.connections[id]; exists {
        conn.LastActivity = time.Now()
        conn.BytesSent += bytesSent
        conn.BytesReceived += bytesReceived
    }
}

// GetActiveConnections 获取活跃连接数
func (ct *WebSocketConnectionTracker) GetActiveConnections() int {
    ct.mu.RLock()
    defer ct.mu.RUnlock()
    
    return len(ct.connections)
}

// GetStats 获取连接统计信息
func (ct *WebSocketConnectionTracker) GetStats() map[string]interface{} {
    ct.mu.RLock()
    defer ct.mu.RUnlock()
    
    activeCount := len(ct.connections)
    var totalSent, totalReceived int64
    
    for _, conn := range ct.connections {
        totalSent += conn.BytesSent
        totalReceived += conn.BytesReceived
    }
    
    return map[string]interface{}{
        "active_connections": activeCount,
        "total_bytes_sent":   totalSent,
        "total_bytes_received": totalReceived,
    }
}

// cleanupRoutine 清理非活跃连接
func (ct *WebSocketConnectionTracker) cleanupRoutine() {
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
}

// cleanupInactiveConnections 清理非活跃连接
func (ct *WebSocketConnectionTracker) cleanupInactiveConnections() {
    ct.mu.Lock()
    defer ct.mu.Unlock()
    
    now := time.Now()
    for id, conn := range ct.connections {
        if now.Sub(conn.LastActivity) > ct.inactiveThreshold {
            delete(ct.connections, id)
            log.Printf("[WebSocket] Cleaned up inactive connection: %s", id)
        }
    }
}

// Stop 停止连接跟踪器
func (ct *WebSocketConnectionTracker) Stop() {
    close(ct.stopChan)
    ct.wg.Wait()
}
```

### 3.4 代理处理器修改

修改`proxy_handler.go`以支持WebSocket连接：

```go
// 在ServeHTTP方法中添加WebSocket处理逻辑
func (ph *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    startTime := time.Now()
    
    // 创建中间件上下文
    ctx := &middleware.Context{
        Request:  r,
        Response: w,
        Values:   make(map[string]interface{}),
    }
    
    // 检测WebSocket升级请求
    isWebSocket := ph.detectWebSocketRequest(r)
    if isWebSocket {
        ctx.Set("isWebSocketConnection", true)
        log.Printf("[WebSocket] WebSocket upgrade request detected: %s %s", r.Method, r.URL.Path)
    }
    
    // 自动检测SSE请求(保留现有功能)
    isSSE := ph.detectSSERequest(r)
    if isSSE {
        ctx.Set("isSSEConnection", true)
        log.Printf("[SSE] SSE connection detected for: %s %s", r.Method, r.URL.Path)
    }
    
    // 确定目标服务和匹配的路由规则
    targetService, hostRule, routeRule, err := ph.determineTarget(r)
    if err != nil {
        // 为WebSocket和SSE连接提供特殊错误处理
        if isWebSocket {
            ph.handleWebSocketError(w, err.Error())
        } else if isSSE {
            ph.handleSSEError(w, err.Error())
        } else {
            http.Error(w, err.Error(), http.StatusBadGateway)
        }
        log.Printf("Failed to determine target: %v", err)
        return
    }
    
    // 设置初始目标服务到上下文
    ctx.TargetURL = targetService.URL
    ctx.ServiceName = ph.getServiceName(targetService.URL)
    
    // 创建动态中间件链
    dynamicMiddlewareChain := ph.createDynamicMiddlewareChain(hostRule, routeRule)
    
    // 执行中间件链
    if !dynamicMiddlewareChain.Execute(ctx) {
        if ctx.StatusCode != 0 {
            w.WriteHeader(ctx.StatusCode)
        }
        log.Printf("Request aborted by middleware: %s %s", r.Method, r.URL.Path)
        return
    }
    
    // 检查中间件是否修改了目标服务
    if dynamicTarget, exists := ctx.Values["dynamic_target_service"]; exists {
        if dynamicTargetServiceName, ok := dynamicTarget.(string); ok {
            if service, serviceExists := ph.services[dynamicTargetServiceName]; serviceExists {
                targetService = &service
                ctx.TargetURL = targetService.URL
                ctx.ServiceName = ph.getServiceName(targetService.URL)
                log.Printf("Dynamic routing: redirected to service '%s'", dynamicTargetServiceName)
            } else {
                log.Printf("Dynamic routing: service '%s' not found, using original target", dynamicTargetServiceName)
            }
        }
    }
    
    // 根据请求类型创建相应的处理器
    var handler http.Handler
    if isWebSocket {
        // 创建WebSocket代理
        handler, err = ph.createWebSocketProxy(targetService, ctx)
        if err != nil {
            ph.handleWebSocketError(w, err.Error())
            log.Printf("Failed to create WebSocket proxy: %v", err)
            return
        }
    } else {
        // 创建HTTP反向代理
        handler, err = ph.createReverseProxy(targetService, ctx)
        if err != nil {
            if isSSE {
                ph.handleSSEError(w, err.Error())
            } else {
                http.Error(w, err.Error(), http.StatusBadGateway)
            }
            log.Printf("Failed to create reverse proxy: %v", err)
            return
        }
    }
    
    // 执行代理
    handler.ServeHTTP(ctx.Response, r)
    
    // 记录请求完成日志
    duration := time.Since(startTime)
    protocol := "HTTP"
    if isWebSocket {
        protocol = "WebSocket"
    } else if isSSE {
        protocol = "SSE"
    }
    
    log.Printf("Proxied: %s %s %s -> %s [%s] %v",
        protocol, r.Method, r.URL.Path, targetService.URL, r.Host, duration)
}

// detectWebSocketRequest 检测WebSocket升级请求
func (ph *ProxyHandler) detectWebSocketRequest(req *http.Request) bool {
    // 检查Upgrade头
    upgrade := strings.ToLower(req.Header.Get("Upgrade"))
    connection := strings.ToLower(req.Header.Get("Connection"))
    
    return upgrade == "websocket" && 
           strings.Contains(connection, "upgrade") &&
           req.Header.Get("Sec-WebSocket-Key") != "" &&
           req.Header.Get("Sec-WebSocket-Version") != ""
}

// handleWebSocketError 处理WebSocket错误
func (ph *ProxyHandler) handleWebSocketError(w http.ResponseWriter, message string) {
    // 对于WebSocket错误，我们需要确保响应包含适当的头部
    if header, ok := w.(http.Hijacker); ok {
        // 如果可以劫持连接，说明连接还未升级
        http.Error(w, message, http.StatusBadGateway)
    }
    // 如果已经劫持，则无法发送HTTP响应，只能记录日志
    log.Printf("[WebSocket] Error: %s", message)
}
```

## 4. 测试方案

### 4.1 测试环境搭建

1. **WebSocket测试服务器**：
   - 实现一个简单的WebSocket服务器，提供基本的回声功能
   - 支持多种消息类型(文本、二进制)
   - 支持连接统计和监控

2. **测试客户端**：
   - 实现基于浏览器的WebSocket测试页面
   - 实现命令行WebSocket测试工具
   - 支持连接建立、消息发送接收、连接关闭等操作

3. **代理配置**：
   - 配置WebSocket代理规则
   - 配置中间件(认证、日志等)
   - 配置连接限制和超时

### 4.2 测试用例

1. **基本功能测试**：
   - WebSocket连接建立
   - 文本消息发送接收
   - 二进制消息发送接收
   - 连接关闭处理

2. **协议升级测试**：
   - HTTP到WebSocket协议升级
   - 升级失败处理
   - 升级头部验证

3. **中间件集成测试**：
   - 认证中间件集成
   - 日志中间件集成
   - 限流中间件集成

4. **性能测试**：
   - 并发连接测试
   - 大消息传输测试
   - 长时间连接稳定性测试

5. **错误处理测试**：
   - 后端连接失败处理
   - 客户端异常断开处理
   - 网络中断恢复测试

### 4.3 测试脚本

创建自动化测试脚本，验证WebSocket代理功能：

```bash
#!/bin/bash

# WebSocket代理测试脚本

# 测试1: 基本WebSocket连接
echo "测试1: 基本WebSocket连接"
# 使用websocat或其他WebSocket客户端工具

# 测试2: 通过代理的WebSocket连接
echo "测试2: 通过代理的WebSocket连接"
# 连接到代理服务器，验证代理转发

# 测试3: 消息传输测试
echo "测试3: 消息传输测试"
# 发送各种类型的消息，验证正确转发

# 测试4: 并发连接测试
echo "测试4: 并发连接测试"
# 创建多个并发连接，验证代理处理能力

# 测试5: 长时间连接测试
echo "测试5: 长时间连接测试"
# 保持连接长时间活跃，验证稳定性

# 测试6: 错误处理测试
echo "测试6: 错误处理测试"
# 模拟各种错误情况，验证错误处理

# 测试7: 中间件集成测试
echo "测试7: 中间件集成测试"
# 验证中间件正确处理WebSocket连接

# 测试8: 安全测试
echo "测试8: 安全测试"
# 测试Origin验证、WSS连接等安全功能

# 测试9: 性能测试
echo "测试9: 性能测试"
# 测量延迟、吞吐量等性能指标

echo "所有测试完成"
```

## 5. 部署与运维

### 5.1 部署配置

1. **配置文件示例**：

```yaml
# config.yaml
config_dir: "conf.d"

# 高级配置
advanced:
  port: 8080
  timeout:
    read_timeout: 60
    write_timeout: 60
    dial_timeout: 10
  security:
    deny_hidden_files: true

# WebSocket服务定义
services:
  websocket_service:
    url: "ws://localhost:8081"
    # 可选配置
    ping_interval: 30
    read_timeout: 60
    write_timeout: 60

# 域名匹配规则
host_rules:
  - pattern: "ws.example.com"
    port: 80
    target: "websocket_service"
    middlewares: ["websocket", "logging"]
    route_rules:
      - pattern: "/ws/*"
        target: "websocket_service"
        middlewares: ["websocket", "auth"]

# 中间件配置
middlewares:
  - name: "websocket"
    enabled: true
    config:
      ping_interval: 30
      max_connections: 1000
      enable_origin_check: true
      allowed_origins: ["https://example.com"]
  - name: "auth"
    enabled: true
    config:
      jwt_secret: "your-secret-key"
  - name: "logging"
    enabled: true
    config:
      log_level: "info"
```

2. **环境变量配置**：

```bash
# WebSocket代理环境变量
export WEBSOCKET_MAX_CONNECTIONS=1000
export WEBSOCKET_PING_INTERVAL=30
export WEBSOCKET_READ_TIMEOUT=60
export WEBSOCKET_WRITE_TIMEOUT=60
export WEBSOCKET_ENABLE_ORIGIN_CHECK=true
export WEBSOCKET_ALLOWED_ORIGINS="https://example.com,https://app.example.com"
```

### 5.2 监控指标

实现以下监控指标：

1. **连接指标**：
   - 当前活跃连接数
   - 总连接数
   - 连接建立速率
   - 连接关闭速率

2. **流量指标**：
   - 接收字节数
   - 发送字节数
   - 消息数量
   - 错误率

3. **性能指标**：
   - 连接延迟
   - 消息延迟
   - 代理吞吐量

### 5.3 日志记录

实现详细的日志记录：

1. **连接日志**：
   - 连接建立/关闭
   - 连接错误
   - 连接超时

2. **消息日志**：
   - 消息类型和大小
   - 消息方向(客户端到服务器/服务器到客户端)
   - 消息处理时间

3. **错误日志**：
   - 协议升级错误
   - 消息处理错误
   - 网络错误

## 6. 安全考虑

### 6.1 安全措施

1. **Origin验证**：
   - 检查请求的Origin头
   - 配置允许的源列表
   - 防止CSRF攻击

2. **连接限制**：
   - 限制最大连接数
   - 限制单个客户端连接数
   - 防止资源耗尽

3. **认证授权**：
   - 集成现有认证中间件
   - 支持JWT令牌验证
   - 支持基于角色的访问控制

4. **数据加密**：
   - 支持WSS(WebSocket Secure)
   - 自动处理SSL/TLS证书
   - 确保数据传输安全

### 6.2 安全配置

```yaml
# 安全配置示例
middlewares:
  - name: "websocket"
    enabled: true
    config:
      # 启用Origin验证
      enable_origin_check: true
      # 允许的源
      allowed_origins: 
        - "https://example.com"
        - "https://app.example.com"
      # 最大连接数
      max_connections: 1000
      # 单个客户端最大连接数
      max_connections_per_client: 10
      # 心跳间隔
      ping_interval: 30
      # 连接超时
      read_timeout: 60
      write_timeout: 60
      # 启用速率限制
      enable_rate_limit: true
      # 每分钟最大消息数
      max_messages_per_minute: 1000
```

## 7. 性能优化

### 7.1 优化策略

1. **连接池管理**：
   - 复用后端连接
   - 减少连接建立开销
   - 实现连接预热

2. **数据缓冲**：
   - 优化读写缓冲区大小
   - 实现批量消息处理
   - 减少系统调用次数

3. **并发处理**：
   - 使用高效的并发模型
   - 优化Goroutine使用
   - 减少锁竞争

4. **内存管理**：
   - 减少内存分配
   - 实现对象池
   - 优化垃圾回收

### 7.2 性能配置

```yaml
# 性能优化配置
advanced:
  # 缓冲区大小
  buffer_size:
    read_buffer: 4096
    write_buffer: 4096
  # 连接池配置
  connection_pool:
    max_idle_connections: 100
    max_active_connections: 1000
    idle_timeout: 90
  # 并发配置
  concurrency:
    max_goroutines: 1000
    goroutine_stack_size: 8192
```

## 8. 实施计划

### 8.1 开发阶段

1. **第一阶段：基础功能实现**
   - 实现WebSocket检测中间件
   - 实现基本WebSocket代理功能
   - 实现协议升级处理

2. **第二阶段：连接管理**
   - 实现连接跟踪器
   - 实现连接限制和超时
   - 实现连接统计和监控

3. **第三阶段：中间件集成**
   - 确保与现有中间件兼容
   - 实现WebSocket特定中间件
   - 实现安全中间件集成

4. **第四阶段：测试和优化**
   - 实现全面的测试套件
   - 性能优化和调优
   - 文档完善

### 8.2 时间安排

- **第一阶段**：2周
- **第二阶段**：1周
- **第三阶段**：1周
- **第四阶段**：2周

总计：6周

## 9. 风险评估与应对

### 9.1 技术风险

1. **协议兼容性**：
   - 风险：WebSocket协议实现差异
   - 应对：使用成熟的WebSocket库，充分测试各种客户端

2. **性能影响**：
   - 风险：WebSocket代理可能影响现有HTTP代理性能
   - 应对：实现独立的处理路径，优化资源使用

3. **资源消耗**：
   - 风险：长连接可能导致资源耗尽
   - 应对：实现连接限制和超时机制

### 9.2 运维风险

1. **监控盲区**：
   - 风险：WebSocket连接可能难以监控
   - 应对：实现专门的监控指标和日志

2. **故障排查**：
   - 风险：WebSocket连接问题难以排查
   - 应对：实现详细的日志记录和诊断工具

## 10. 总结

本文档详细分析了Toyou Proxy项目实现WebSocket代理功能的需求、设计和实现方案。通过扩展现有的中间件系统和代理处理器，我们可以无缝集成WebSocket代理功能，同时保持与现有功能的兼容性。

关键实现点包括：

1. **协议升级处理**：正确处理HTTP到WebSocket的协议升级
2. **双向数据转发**：实现客户端和后端服务器之间的透明数据转发
3. **连接管理**：有效管理长连接，防止资源耗尽
4. **中间件集成**：确保WebSocket连接可以经过中间件链处理
5. **安全考虑**：实现Origin验证、认证授权等安全措施

通过分阶段实施，我们可以在不影响现有功能的情况下，逐步添加WebSocket代理支持，最终实现一个功能完整、性能优异的反向代理系统。