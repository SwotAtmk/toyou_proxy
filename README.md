# Toyou Proxy

一个高性能、可扩展的反向代理中间件插件系统，专为现代微服务架构设计。

## 项目概述

Toyou Proxy 是一个基于 Go 语言开发的反向代理服务器，采用"约定优于配置"的设计理念，提供了强大的中间件插件系统。该系统支持插件的自动发现、动态加载和热更新，使开发者能够轻松扩展代理功能而无需修改核心代码。

### 核心特性

- **高性能反向代理**：基于 Go 的高并发特性，支持高吞吐量请求处理
- **灵活的中间件系统**：支持自定义中间件，可在请求处理流程中插入自定义逻辑
- **插件化架构**：支持插件的自动发现、动态加载和热更新
- **多级路由匹配**：支持域名级和路径级路由匹配，满足复杂路由需求
- **多文件配置管理**：支持配置文件分离和合并，便于大型项目配置管理
- **动态路由**：支持基于外部API的动态路由决策，实现智能流量分发

### 应用场景

- **微服务API网关**：作为微服务架构的统一入口，提供路由、认证、限流等功能
- **多租户SaaS平台**：支持多租户隔离和自定义路由规则
- **开发环境代理**：简化开发环境的API调用和调试
- **CDN边缘节点**：作为CDN的边缘节点，提供智能缓存和路由
- **负载均衡器**：结合外部服务发现，实现智能负载均衡

## 功能特性

### 1. 域名/路由匹配

- **精确匹配**：支持精确的域名和路径匹配
- **通配符匹配**：支持 `*` 通配符匹配多个域名或路径
- **正则表达式**：支持使用正则表达式进行复杂模式匹配
- **优先级规则**：按照配置顺序和匹配优先级处理请求

### 2. 中间件系统

- **内置中间件**：
  - `auth`：基于JWT的认证中间件
  - `rate_limit`：请求限流中间件
  - `cors`：跨域资源共享中间件
  - `logging`：请求日志记录中间件
  - `replace`：响应内容替换中间件
  - `dynamic_route`：动态路由中间件
  - `websocket`：WebSocket代理中间件

- **自定义中间件**：支持开发自定义中间件，只需实现 `Middleware` 接口

### 3. 插件系统

- **自动发现**：自动扫描并发现插件目录中的中间件插件
- **动态加载**：支持运行时加载和卸载插件
- **热更新**：支持插件的热更新，无需重启服务
- **版本管理**：支持插件版本控制和依赖管理

### 4. 动态路由

- **外部API集成**：通过调用外部API获取路由决策
- **缓存机制**：内置缓存机制，减少API调用次数
- **故障转移**：API调用失败时自动回退到默认路由
- **实时更新**：支持路由规则的实时更新

### 5. 多文件配置

- **配置分离**：支持将配置拆分为多个文件
- **配置合并**：自动合并多个配置文件
- **环境隔离**：支持不同环境的配置分离
- **配置验证**：提供配置验证和错误提示

### 服务发现

- **静态服务**：支持静态配置后端服务
- **动态服务**：支持从服务发现系统动态获取服务列表
- **健康检查**：支持后端服务健康检查
- **负载均衡**：支持多种负载均衡策略

## 负载均衡功能

Toyou Proxy 内置了强大的负载均衡功能，可以将流量分发到多个后端服务器，提高系统的可用性和性能。

### 负载均衡策略

1. **轮询（Round Robin）**
   - 按顺序依次将请求分发到每个后端服务器
   - 适用于后端服务器性能相近的场景
   - 配置示例：
     ```yaml
     services:
       my-service:
         load_balancer:
           strategy: "round_robin"
           backends:
             - url: "http://backend1:8080"
             - url: "http://backend2:8080"
             - url: "http://backend3:8080"
     ```

2. **加权轮询（Weighted Round Robin）**
   - 根据权重比例分发请求，权重越高接收的请求越多
   - 适用于后端服务器性能不同的场景
   - 配置示例：
     ```yaml
     services:
       my-service:
         load_balancer:
           strategy: "weighted_round_robin"
           backends:
             - url: "http://backend1:8080"
               weight: 3  # 高性能服务器，权重为3
             - url: "http://backend2:8080"
               weight: 2  # 中等性能服务器，权重为2
             - url: "http://backend3:8080"
               weight: 1  # 低性能服务器，权重为1
     ```

3. **IP哈希（IP Hash）**
   - 根据客户端IP地址的哈希值选择后端服务器
   - 确保同一客户端的请求始终发送到同一服务器
   - 适用于需要会话保持的场景
   - 配置示例：
     ```yaml
     services:
       my-service:
         load_balancer:
           strategy: "ip_hash"
           backends:
             - url: "http://backend1:8080"
             - url: "http://backend2:8080"
             - url: "http://backend3:8080"
     ```

4. **最少连接（Least Connections）**
   - 将请求分发到当前连接数最少的后端服务器
   - 适用于请求处理时间差异较大的场景
   - 配置示例：
     ```yaml
     services:
       my-service:
         load_balancer:
           strategy: "least_connections"
           backends:
             - url: "http://backend1:8080"
             - url: "http://backend2:8080"
             - url: "http://backend3:8080"
     ```

### 健康检查

负载均衡器支持自动健康检查，可以定期检查后端服务器的健康状态，自动排除不健康的服务器：

```yaml
services:
  my-service:
    load_balancer:
      strategy: "round_robin"
      backends:
        - url: "http://backend1:8080"
          active: true
        - url: "http://backend2:8080"
          active: true
        - url: "http://backend3:8080"
          active: false  # 可以手动禁用某个后端
      health_check:
        enabled: true
        interval: 10s    # 检查间隔
        timeout: 5s      # 超时时间
        path: "/health"  # 健康检查路径
```

### 完整配置示例

```yaml
services:
  web-service:
    url: "http://localhost:8080"  # 默认后端（可选）
    load_balancer:
      strategy: "round_robin"  # 负载均衡策略
      backends:
        - url: "http://backend1:8080"
          weight: 2            # 权重（仅对加权轮询有效）
          active: true         # 是否启用
        - url: "http://backend2:8080"
          weight: 1
          active: true
        - url: "http://backend3:8080"
          weight: 1
          active: true
      health_check:
        enabled: true
        interval: 10s
        timeout: 5s
        path: "/health"

host_rules:
  - pattern: "example.com"
    port: 80
    target: "web-service"  # 指向上面定义的服务
    route_rules:
      - pattern: "/api/*"
        target: "web-service"
```

### 7. WebSocket代理

- **协议转换**：支持HTTP到WebSocket协议的自动转换
- **双向通信**：支持客户端和服务器之间的双向实时通信
- **连接保持**：支持长连接保持和心跳机制
- **路径匹配**：支持基于路径的WebSocket路由
- **自定义头部**：支持自定义WebSocket握手头部

## 快速开始

### 1. 环境要求

- Go 1.19 或更高版本
- 操作系统：Linux、macOS 或 Windows

### 2. 安装

```bash
# 克隆仓库
git clone https://github.com/your-org/toyou-proxy.git
cd toyou-proxy

# 安装依赖
go mod tidy

# 编译
go build -o toyou-proxy cmd/main.go

# 或者使用提供的构建脚本
chmod +x build.sh
./build.sh
```

### 3. 基本配置

创建 `config.yaml` 文件：

```yaml
# 配置文件目录（可选）
config_dir: "conf.d"

# 高级配置
advanced:
  port: 8080  # 代理服务器监听端口
  timeout:
    read_timeout: 30
    write_timeout: 30
    dial_timeout: 10
  security:
    deny_hidden_files: true

# 域名匹配规则
host_rules:
  - pattern: "api.example.com"
    port: 80
    target: "api-service"
    middlewares: ["auth", "rate_limit"]
  - pattern: "*.example.com"
    port: 80
    target: "web-service"
    middlewares: ["cors"]

# 服务定义
services:
  api-service:
    url: "http://localhost:8081"
  web-service:
    url: "http://localhost:8082"

# 中间件配置
middlewares:
  - name: "auth"
    enabled: true
    config:
      jwt_secret: "your-secret-key"
  - name: "rate_limit"
    enabled: true
    config:
      requests_per_minute: 100
  - name: "cors"
    enabled: true
    config:
      allowed_origins: ["*"]
      allowed_methods: ["GET", "POST", "PUT", "DELETE"]
```

### 4. 运行

```bash
# 直接运行
./toyou-proxy

# 或者使用提供的启动脚本
chmod +x start.sh
./start.sh
```

### 5. 测试

```bash
# 测试API服务
curl -H "Host: api.example.com" http://localhost:8080/api/users

# 测试Web服务
curl -H "Host: www.example.com" http://localhost:8080/
```

### 6. 使用Docker

```dockerfile
FROM golang:1.19-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod tidy && go build -o toyou-proxy cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/toyou-proxy .
COPY --from=builder /app/config.yaml .

EXPOSE 8080
CMD ["./toyou-proxy"]
```

```bash
# 构建镜像
docker build -t toyou-proxy .

# 运行容器
docker run -p 8080:8080 -v $(pwd)/config.yaml:/root/config.yaml toyou-proxy
```

## 配置说明

### 配置文件结构

Toyou Proxy 支持单文件和多文件配置方式。配置文件使用 YAML 格式，包含以下主要部分：

- `config_dir`: 配置文件目录（可选）
- `host_rules`: 域名匹配规则
- `route_rules`: 路由匹配规则（主要用于兼容旧配置）
- `services`: 服务定义
- `middlewares`: 中间件配置
- `middleware_services`: 中间件服务注册（支持自定义名称注册）
- `advanced`: 高级配置

### 域名/路由匹配规则

#### 域名匹配规则 (host_rules)

```yaml
host_rules:
  - pattern: "api.example.com"  # 精确匹配
    port: 80                    # 监听端口（可选，默认80）
    target: "api-service"       # 目标服务名称
    middlewares: ["auth", "rate_limit"]  # 应用的中间件列表
    route_rules:                # 嵌套路由规则（可选）
      - pattern: "/api/v1/*"
        target: "api-v1-service"
        middlewares: ["logging"]
  
  - pattern: "*.example.com"    # 通配符匹配
    port: 80
    target: "web-service"
    middlewares: ["cors"]
  
  - pattern: "~^api\\d+\\.example\\.com$"  # 正则表达式匹配
    port: 80
    target: "api-service"
    middlewares: ["auth"]
```

#### 路由匹配规则 (route_rules)

```yaml
route_rules:
  - pattern: "/api/v1/*"        # 路径前缀匹配
    target: "api-v1-service"     # 目标服务名称
    middlewares: ["logging"]     # 应用的中间件列表
  
  - pattern: "/api/v2/*"
    target: "api-v2-service"
    middlewares: ["auth", "logging"]
  
  - pattern: "~^/api/v\\d+/users/\\d+$"  # 正则表达式匹配
    target: "user-service"
    middlewares: ["auth"]
```

#### 匹配优先级

1. 精确匹配 > 通配符匹配 > 正则表达式匹配
2. 域名匹配优先于路由匹配
3. 配置文件中先定义的规则优先于后定义的规则

### 服务定义

```yaml
services:
  api-service:
    url: "http://localhost:8081"           # 后端服务URL
    proxy_host: "backend.api.local"        # 代理时使用的Host头（可选）
  
  web-service:
    url: "http://localhost:8082"
  
  cluster-service:
    url: "http://cluster.example.com"
    proxy_host: "internal.cluster.local"
```

### 中间件配置

#### 基本中间件配置

```yaml
middlewares:
  - name: "auth"                    # 中间件名称
    enabled: true                   # 是否启用
    config:                         # 中间件特定配置
      jwt_secret: "your-secret-key"
      token_header: "X-Auth-Token"
  
  - name: "rate_limit"
    enabled: true
    config:
      requests_per_minute: 100
      burst: 200
      key_extractor: "ip"           # ip, header, custom
  
  - name: "cors"
    enabled: true
    config:
      allowed_origins: ["https://example.com", "https://app.example.com"]
      allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
      allowed_headers: ["Content-Type", "Authorization"]
      exposed_headers: ["X-Total-Count"]
      allow_credentials: true
      max_age: 86400
  
  - name: "websocket"
    enabled: true
    config:
      path_patterns: ["/ws", "/api/ws"]  # WebSocket路径模式
```

#### 中间件服务注册

```yaml
middleware_services:
  - name: "strict_auth"             # 自定义中间件服务名称
    type: "auth"                    # 中间件类型
    enabled: true
    is_global: false                 # 是否全局加载
    description: "严格认证中间件"    # 描述（可选）
    config:
      jwt_secret: "strict-secret"
      require_auth: true
  
  - name: "high_rate_limit"
    type: "rate_limit"
    enabled: true
    is_global: true                 # 全局中间件，应用于所有路由
    description: "高频限流中间件"
    config:
      requests_per_minute: 1000
      burst: 2000
```

### 高级配置

```yaml
advanced:
  port: 8080                        # 代理服务器监听端口
  timeout:
    read_timeout: 30                # 读取超时（秒）
    write_timeout: 30               # 写入超时（秒）
    dial_timeout: 10                # 连接超时（秒）
  security:
    deny_hidden_files: true         # 是否拒绝访问隐藏文件（以.开头的文件）
```

### 多文件配置

Toyou Proxy 支持将配置拆分为多个文件，便于管理大型项目。配置文件可以放在主配置文件中指定的 `config_dir` 目录下。

#### 主配置文件 (config.yaml)

```yaml
config_dir: "conf.d"                # 配置文件目录

advanced:
  port: 8080
  timeout:
    read_timeout: 30
    write_timeout: 30
    dial_timeout: 10
  security:
    deny_hidden_files: true
```

#### 域名配置文件 (conf.d/domains.yaml)

```yaml
host_rules:
  - pattern: "api.example.com"
    port: 80
    target: "api-service"
    middlewares: ["auth", "rate_limit"]
  
  - pattern: "*.example.com"
    port: 80
    target: "web-service"
    middlewares: ["cors"]

services:
  api-service:
    url: "http://localhost:8081"
  web-service:
    url: "http://localhost:8082"
```

#### 中间件配置文件 (conf.d/middlewares.yaml)

```yaml
middlewares:
  - name: "auth"
    enabled: true
    config:
      jwt_secret: "your-secret-key"
  
  - name: "rate_limit"
    enabled: true
    config:
      requests_per_minute: 100
  
  - name: "cors"
    enabled: true
    config:
      allowed_origins: ["*"]
      allowed_methods: ["GET", "POST", "PUT", "DELETE"]
```

#### 配置合并规则

1. 所有配置文件中的 `host_rules`、`route_rules`、`middlewares` 和 `middleware_services` 会被合并
2. `services` 会被合并，如果出现同名服务，后加载的配置会覆盖先加载的配置
3. `advanced` 配置以主配置文件为准，其他配置文件中的 `advanced` 配置会被忽略
4. 配置文件按文件名字母顺序加载

## 匹配优先级

1. **路由匹配优先**：先检查路由规则
2. **域名匹配次之**：如果路由不匹配，检查域名规则
3. **精确匹配优先**：精确匹配优先于通配符匹配

## 使用示例

### 场景1：多租户SaaS应用

```yaml
host_rules:
  - pattern: "*.company1.com"
    target: "company1-service"
  - pattern: "*.company2.com"
    target: "company2-service"

services:
  "company1-service":
    url: "http://company1-backend:8080"
  "company2-service":
    url: "http://company2-backend:8080"
```

### 场景2：微服务API网关

```yaml
route_rules:
  - pattern: "/api/users/*"
    target: "user-service"
  - pattern: "/api/products/*"
    target: "product-service"
  - pattern: "/api/orders/*"
    target: "order-service"

services:
  "user-service":
    url: "http://user-service:3001"
  "product-service":
    url: "http://product-service:3002"
  "order-service":
    url: "http://order-service:3003"
```

### 场景3：开发环境代理

```yaml
host_rules:
  - pattern: "api.localhost"
    target: "api-dev"
  - pattern: "admin.localhost"
    target: "admin-dev"

services:
  "api-dev":
    url: "http://localhost:3000"
  "admin-dev":
    url: "http://localhost:3001"
```

### 场景4：多文件配置管理（推荐）

**主配置文件 config.yaml**
```yaml
server:
  port: 80

config_dir: "conf.d"

# 系统全局配置
middlewares:
  - name: "logging"
    enabled: true
  - name: "cors"
    enabled: true
```

**conf.d/api-services.yaml**
```yaml
host_rules:
  - pattern: "api.company.com"
    port: 8080
    target: "api-gateway"

route_rules:
  - pattern: "/users/*"
    target: "user-service"
  - pattern: "/products/*"
    target: "product-service"

services:
  "api-gateway":
    url: "http://api-gateway:3000"
  "user-service":
    url: "http://user-service:3001"
  "product-service":
    url: "http://product-service:3002"
```

**conf.d/admin-services.yaml**
```yaml
host_rules:
  - pattern: "admin.company.com"
    port: 8081
    target: "admin-service"

services:
  "admin-service":
    url: "http://admin-service:4000"
```

**conf.d/static-services.yaml**
```yaml
host_rules:
  - pattern: "static.company.com"
    port: 8082
    target: "static-service"

services:
  "static-service":
    url: "http://static-service:5000"
```

## 中间件开发指南

Toyou Proxy 提供了强大的中间件系统，允许开发者轻松扩展代理功能。中间件可以在请求处理流程中插入自定义逻辑，如认证、限流、日志记录等。

### 中间件接口

所有中间件都需要实现 `Middleware` 接口：

```go
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
    TargetURL   string                 // 目标URL
    ServiceName string                 // 服务名称
    StatusCode  int                    // 状态码
}
```

### 创建自定义中间件

#### 1. 创建中间件结构体

```go
package main

import (
    "fmt"
    "net/http"
    "toyou-proxy/middleware"
)

// CustomMiddleware 自定义中间件
type CustomMiddleware struct {
    config map[string]interface{}
}

// NewCustomMiddleware 创建自定义中间件
func NewCustomMiddleware(config map[string]interface{}) (middleware.Middleware, error) {
    // 验证配置
    if config == nil {
        config = make(map[string]interface{})
    }
    
    // 设置默认值
    if _, exists := config["header_name"]; !exists {
        config["header_name"] = "X-Custom-Header"
    }
    
    return &CustomMiddleware{
        config: config,
    }, nil
}
```

#### 2. 实现接口方法

```go
// Name 返回中间件名称
func (cm *CustomMiddleware) Name() string {
    return "custom"
}

// Handle 处理请求
func (cm *CustomMiddleware) Handle(ctx *middleware.Context) bool {
    // 获取配置
    headerName, _ := cm.config["header_name"].(string)
    
    // 检查请求头
    headerValue := ctx.Request.Header.Get(headerName)
    if headerValue == "" {
        // 设置错误状态码
        ctx.StatusCode = http.StatusUnauthorized
        // 写入错误响应
        ctx.Response.WriteHeader(http.StatusUnauthorized)
        fmt.Fprintf(ctx.Response, "Missing required header: %s", headerName)
        // 返回false中断请求处理
        return false
    }
    
    // 在上下文中存储数据，供后续中间件使用
    if ctx.Values == nil {
        ctx.Values = make(map[string]interface{})
    }
    ctx.Values["custom_header_value"] = headerValue
    
    // 记录日志
    fmt.Printf("Custom middleware: %s = %s\n", headerName, headerValue)
    
    // 返回true继续执行下一个中间件
    return true
}
```

#### 3. 实现插件入口函数

```go
// PluginMain 插件入口函数
func PluginMain(config map[string]interface{}) (middleware.Middleware, error) {
    return NewCustomMiddleware(config)
}
```

### 中间件插件目录结构

每个中间件插件都应该有自己的目录，通常位于 `middleware/plugins/` 下：

```
middleware/plugins/
├── custom/
│   ├── plugin.go      # 插件实现
│   ├── plugin.json    # 插件元数据
│   └── README.md      # 插件文档
```

### 插件元数据配置

每个插件目录必须包含 `plugin.json` 文件，定义插件元数据：

```json
{
    "name": "custom",
    "version": "1.0.0",
    "description": "自定义中间件插件",
    "entry_point": "PluginMain",
    "config": {
        "header_name": {
            "type": "string",
            "default": "X-Custom-Header",
            "description": "要检查的请求头名称"
        }
    },
    "enabled": true
}
```

### 中间件配置示例

在配置文件中使用自定义中间件：

```yaml
middlewares:
  - name: "custom"
    enabled: true
    config:
      header_name: "X-API-Key"
```

或者使用中间件服务注册：

```yaml
middleware_services:
  - name: "strict_custom"
    type: "custom"
    enabled: true
    is_global: false
    description: "严格的自定义中间件"
    config:
      header_name: "X-Strict-API-Key"
```

### 中间件最佳实践

1. **错误处理**：妥善处理错误，设置适当的状态码
2. **性能考虑**：避免在中间件中执行耗时操作
3. **上下文使用**：使用 `Context.Values` 在中间件间传递数据
4. **配置验证**：在创建中间件时验证配置参数
5. **日志记录**：添加适当的日志记录，便于调试
6. **资源管理**：正确管理资源，避免内存泄漏

### 高级中间件开发

#### 修改请求和响应

```go
// 修改请求示例
func (cm *CustomMiddleware) Handle(ctx *middleware.Context) bool {
    // 添加请求头
    ctx.Request.Header.Set("X-Proxy-Timestamp", time.Now().Format(time.RFC3339))
    
    // 包装响应写入器，以便修改响应
    ctx.Response = &responseWrapper{
        ResponseWriter: ctx.Response,
        middleware:     cm,
        context:        ctx,
    }
    
    return true
}

// responseWrapper 包装http.ResponseWriter以修改响应
type responseWrapper struct {
    http.ResponseWriter
    middleware *CustomMiddleware
    context    *middleware.Context
}

func (rw *responseWrapper) Write(data []byte) (int, error) {
    // 修改响应数据
    modifiedData := strings.ReplaceAll(string(data), "old", "new")
    return rw.ResponseWriter.Write([]byte(modifiedData))
}
```

#### 异步处理

```go
func (cm *CustomMiddleware) Handle(ctx *middleware.Context) bool {
    // 异步处理，不阻塞请求
    go func() {
        // 执行异步操作
        cm.asyncOperation(ctx)
    }()
    
    return true
}

func (cm *CustomMiddleware) asyncOperation(ctx *middleware.Context) {
    // 执行耗时操作，如日志记录、统计等
    // 注意：不要在这里修改请求或响应，因为主请求可能已经完成
}
```

#### 条件执行

```go
func (cm *CustomMiddleware) Handle(ctx *middleware.Context) bool {
    // 根据条件决定是否执行中间件逻辑
    if cm.shouldSkip(ctx) {
        return true
    }
    
    // 执行中间件逻辑
    return cm.processRequest(ctx)
}

func (cm *CustomMiddleware) shouldSkip(ctx *middleware.Context) bool {
    // 跳过健康检查端点
    if ctx.Request.URL.Path == "/health" {
        return true
    }
    
    // 跳过特定IP
    clientIP := ctx.Request.RemoteAddr
    if cm.isWhitelistedIP(clientIP) {
        return true
    }
    
    return false
}
```

## 动态路由中间件

动态路由中间件是Toyou Proxy的核心功能之一，它允许根据外部API的响应动态调整请求的目标服务，实现灵活的路由策略。

### 功能概述

动态路由中间件通过以下方式工作：

1. **请求拦截**：拦截传入的HTTP请求，提取主机名
2. **缓存查询**：首先检查本地缓存中是否有该主机的路由信息
3. **API查询**：如果缓存未命中或已过期，调用外部API获取路由信息
4. **缓存更新**：将API返回的路由信息存储在本地缓存中
5. **路由调整**：根据API响应动态调整请求的目标服务

### 配置参数

动态路由中间件支持以下配置参数：

| 参数名 | 类型 | 默认值 | 描述 |
|--------|------|--------|------|
| `api_url` | string | `http://127.0.0.1:7080/api/host` | 外部API的URL地址 |
| `timeout_seconds` | float | `5` | API请求超时时间（秒） |
| `cache_expiry_seconds` | float | `60` | 缓存过期时间（秒） |

### 配置示例

在配置文件中使用动态路由中间件：

```yaml
middlewares:
  - name: "dynamic_route"
    enabled: true
    config:
      api_url: "http://route-service.example.com/api/host"
      timeout_seconds: 3
      cache_expiry_seconds: 120
```

### 外部API接口规范

动态路由中间件期望外部API遵循以下接口规范：

#### 请求格式

- **方法**：POST
- **Content-Type**：application/json
- **请求体**：
```json
{
  "host": "example.com"
}
```

#### 响应格式

- **Content-Type**：application/json
- **响应体**：
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "goto_services": "service-name"
  }
}
```

- **code**：响应状态码，200表示成功
- **msg**：响应消息
- **data.goto_services**：目标服务名称，如果为空字符串则表示不改变原始路由

### 实现原理

动态路由中间件的核心实现逻辑如下：

```go
// Handle 处理动态路由逻辑
func (drm *DynamicRouteMiddleware) Handle(context *middleware.Context) bool {
    // 获取请求的Host
    host := context.Request.Host
    if host == "" {
        host = context.Request.URL.Host
    }

    // 提取主机名部分（去除端口）
    hostName := strings.Split(host, ":")[0]

    // 检查缓存是否有效
    targetService, found := drm.getCachedTarget(hostName)
    if !found {
        // 缓存未命中或已过期，调用外部API
        newTarget, err := drm.queryExternalAPI(hostName)
        if err != nil {
            // API调用失败，记录日志但继续执行原始路由
            fmt.Printf("Dynamic route middleware: Failed to query external API for host '%s': %v\n", hostName, err)
            return true
        }

        // 更新缓存
        drm.updateCache(hostName, newTarget)
        targetService = newTarget
    }

    // 如果API返回了有效的目标服务，更新上下文
    if targetService != "" {
        // 将目标服务存储在上下文中，供后续中间件使用
        if context.Values == nil {
            context.Values = make(map[string]interface{})
        }
        context.Values["dynamic_target_service"] = targetService

        fmt.Printf("Dynamic route middleware: Rerouting host '%s' to service '%s'\n", hostName, targetService)
    }

    return true
}
```

### 缓存机制

动态路由中间件实现了高效的缓存机制：

1. **内存缓存**：使用Go的map结构存储主机名到目标服务的映射
2. **时间过期**：根据配置的`cache_expiry_seconds`参数自动过期缓存
3. **批量更新**：支持批量更新缓存，提高性能

### 错误处理

动态路由中间件实现了健壮的错误处理机制：

1. **API调用失败**：当API调用失败时，中间件会记录错误日志，但不会中断请求处理，而是继续使用原始路由
2. **API响应错误**：当API返回非200状态码时，中间件会解析错误消息并记录日志
3. **网络超时**：当API请求超时时，中间件会捕获超时错误并记录日志

### 性能优化

为了提高性能，动态路由中间件实现了以下优化：

1. **缓存机制**：减少对外部API的调用次数
2. **连接池**：复用HTTP连接，减少连接建立开销
3. **并发控制**：使用Go的并发特性处理多个请求
4. **超时控制**：避免长时间等待API响应

### 使用场景

动态路由中间件适用于以下场景：

1. **A/B测试**：根据用户特征将请求路由到不同的服务版本
2. **蓝绿部署**：将部分流量路由到新版本服务
3. **多租户系统**：根据租户信息将请求路由到对应的服务实例
4. **故障转移**：当主服务不可用时，自动将请求路由到备用服务
5. **负载均衡**：根据服务负载情况动态调整路由策略

### 最佳实践

1. **API设计**：外部API应该快速响应，避免成为性能瓶颈
2. **缓存策略**：根据业务需求设置合适的缓存过期时间
3. **错误处理**：外部API应该提供清晰的错误码和错误消息
4. **监控告警**：监控API调用成功率和响应时间，设置告警阈值
5. **容错设计**：设计合理的降级策略，当API不可用时保证服务可用性

### 扩展开发

如果需要扩展动态路由中间件的功能，可以考虑以下方向：

1. **多条件路由**：不仅基于主机名，还可以基于请求头、路径等条件
2. **权重路由**：支持按权重分配流量到不同的服务
3. **健康检查**：集成服务健康检查，自动路由到健康的服务实例
4. **路由规则**：支持更复杂的路由规则配置
5. **路由统计**：收集路由统计数据，用于分析和优化路由策略

## WebSocket代理中间件

WebSocket代理中间件是Toyou Proxy的重要功能之一，它允许代理服务器处理WebSocket连接，实现客户端和后端WebSocket服务器之间的双向通信。

### 功能概述

WebSocket代理中间件通过以下方式工作：

1. **协议检测**：自动检测WebSocket升级请求
2. **协议转换**：将HTTP连接升级为WebSocket连接
3. **双向代理**：在客户端和服务器之间建立双向数据通道
4. **连接保持**：维护长连接，支持心跳机制
5. **错误处理**：提供完善的错误处理和连接恢复机制

### 配置参数

WebSocket代理中间件支持以下配置参数：

| 参数名 | 类型 | 默认值 | 描述 |
|--------|------|--------|------|
| `path_patterns` | []string | `["/ws"]` | WebSocket路径模式列表 |
| `allowed_origins` | []string | `["*"]` | 允许的来源列表 |
| `ping_interval` | int | `30` | 心跳间隔（秒） |
| `pong_wait` | int | `60` | 等待pong响应的时间（秒） |
| `write_wait` | int | `10` | 写入超时时间（秒） |
| `max_message_size` | int64 | `512` | 最大消息大小（KB） |

### 配置示例

#### 基本配置

```yaml
middlewares:
  - name: "websocket"
    enabled: true
    config:
      path_patterns: ["/ws", "/api/ws"]
```

#### 完整配置

```yaml
middlewares:
  - name: "websocket"
    enabled: true
    config:
      path_patterns: ["/ws", "/api/ws", "/socket.io"]
      allowed_origins: ["https://example.com", "https://app.example.com"]
      ping_interval: 30
      pong_wait: 60
      write_wait: 10
      max_message_size: 1024
```

#### 域名和路由配置

```yaml
host_rules:
  - pattern: "ws.example.com"
    port: 8080
    target: "websocket-service"
    middlewares: ["websocket"]

route_rules:
  - pattern: "/ws/*"
    target: "websocket-service"
    middlewares: ["websocket"]

services:
  websocket-service:
    url: "ws://localhost:8081"  # 注意使用ws://或wss://协议
```

### 使用示例

#### 简单聊天应用

```yaml
# 配置文件
host_rules:
  - pattern: "chat.example.com"
    port: 8080
    target: "chat-service"
    middlewares: ["websocket"]

services:
  chat-service:
    url: "ws://localhost:3001"

middlewares:
  - name: "websocket"
    enabled: true
    config:
      path_patterns: ["/chat"]
      allowed_origins: ["https://chat.example.com"]
```

```javascript
// 客户端代码
const ws = new WebSocket('wss://chat.example.com/chat');

ws.onopen = function(event) {
    console.log('Connected to chat server');
    ws.send(JSON.stringify({
        type: 'join',
        room: 'general',
        username: 'user123'
    }));
};

ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log('Received message:', message);
    // 处理聊天消息
};

ws.onclose = function(event) {
    console.log('Disconnected from chat server');
};

ws.onerror = function(error) {
    console.error('WebSocket error:', error);
};
```

#### 实时数据推送

```yaml
# 配置文件
route_rules:
  - pattern: "/api/realtime/*"
    target: "data-service"
    middlewares: ["websocket"]

services:
  data-service:
    url: "ws://localhost:4001"

middlewares:
  - name: "websocket"
    enabled: true
    config:
      path_patterns: ["/api/realtime"]
      ping_interval: 15  # 更频繁的心跳
```

### 实现原理

WebSocket代理中间件的核心实现逻辑如下：

```go
// Handle 处理WebSocket代理逻辑
func (wsm *WebSocketMiddleware) Handle(context *middleware.Context) bool {
    // 检查是否是WebSocket升级请求
    if !isWebSocketUpgrade(context.Request) {
        return true // 不是WebSocket请求，继续处理
    }

    // 获取目标服务URL
    targetURL := getTargetURL(context)
    
    // 创建WebSocket代理
    proxy := &WebSocketProxy{
        TargetURL: targetURL,
        Config:    wsm.config,
    }
    
    // 执行代理
    return proxy.ProxyWebSocket(context.Response, context.Request)
}
```

### 连接流程

1. **客户端请求**：客户端发送HTTP升级请求到代理服务器
2. **协议检测**：代理服务器检测WebSocket升级请求
3. **目标连接**：代理服务器连接到后端WebSocket服务器
4. **协议升级**：代理服务器与客户端和服务器分别完成WebSocket握手
5. **数据转发**：代理服务器在客户端和服务器之间双向转发数据
6. **连接关闭**：任一端关闭连接时，代理服务器关闭另一端连接

### 错误处理

WebSocket代理中间件实现了健壮的错误处理机制：

1. **连接失败**：当无法连接到目标服务器时，返回适当的HTTP错误
2. **协议错误**：当WebSocket协议出现错误时，记录日志并关闭连接
3. **超时处理**：实现读写超时机制，防止连接挂起
4. **资源清理**：连接关闭时正确清理所有相关资源

### 性能优化

为了提高性能，WebSocket代理中间件实现了以下优化：

1. **连接池**：复用到目标服务器的连接，减少连接建立开销
2. **缓冲区管理**：优化读写缓冲区大小，提高数据传输效率
3. **并发控制**：使用Go的并发特性处理多个WebSocket连接
4. **内存复用**：复用内存缓冲区，减少垃圾回收压力

### 监控和日志

WebSocket代理中间件提供了丰富的监控和日志功能：

1. **连接统计**：记录当前活跃连接数、总连接数等统计信息
2. **错误日志**：记录连接错误、协议错误等异常情况
3. **性能指标**：记录连接建立时间、数据传输量等性能指标
4. **调试日志**：提供详细的调试日志，便于问题排查

### 安全考虑

WebSocket代理中间件实现了以下安全措施：

1. **来源验证**：验证请求来源，防止跨站WebSocket劫持
2. **协议限制**：只允许有效的WebSocket协议升级
3. **消息大小限制**：限制消息大小，防止内存耗尽攻击
4. **速率限制**：可与其他中间件结合，实现连接速率限制

### 最佳实践

1. **路径设计**：使用明确的WebSocket路径，如`/ws`、`/api/ws`
2. **心跳机制**：配置适当的心跳间隔，保持连接活跃
3. **错误处理**：在客户端实现完善的错误处理和重连机制
4. **资源管理**：监控连接数和资源使用情况，防止资源泄漏
5. **安全配置**：限制允许的来源，使用安全的WebSocket协议(wss://)

### 负载均衡使用示例

以下是一个完整的负载均衡使用示例，展示如何快速设置和测试负载均衡功能。

### 1. 准备后端服务

假设我们有三个后端服务，分别运行在9001、9002和9003端口：

```go
// backend_server.go
package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "time"
)

func main() {
    port := os.Args[1]
    
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        time.Sleep(100 * time.Millisecond) // 模拟处理时间
        fmt.Fprintf(w, "Response from backend server on port %s at %s", port, time.Now().Format(time.RFC3339))
    })
    
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        fmt.Fprintf(w, "OK")
    })
    
    log.Printf("Backend server starting on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

启动三个后端服务：

```bash
go run backend_server.go 9001 &
go run backend_server.go 9002 &
go run backend_server.go 9003 &
```

### 2. 配置负载均衡

创建配置文件 `lb_config.yaml`：

```yaml
services:
  web-service:
    load_balancer:
      strategy: "round_robin"  # 使用轮询策略
      backends:
        - url: "http://localhost:9001"
          weight: 1
          active: true
        - url: "http://localhost:9002"
          weight: 1
          active: true
        - url: "http://localhost:9003"
          weight: 1
          active: true
      health_check:
        enabled: true
        interval: 10s
        timeout: 5s
        path: "/health"

host_rules:
  - pattern: "localhost"
    port: 8080
    target: "web-service"
```

### 3. 启动代理服务器

```bash
./toyou-proxy -config lb_config.yaml
```

### 4. 测试负载均衡

发送多个请求，观察请求被分发到不同的后端服务器：

```bash
for i in {1..6}; do 
  echo "Request $i:"
  curl -s http://localhost:8080/
  echo
done
```

预期输出：

```
Request 1:
Response from backend server on port 9001 at 2023-07-20T10:30:45+08:00
Request 2:
Response from backend server on port 9002 at 2023-07-20T10:30:45+08:00
Request 3:
Response from backend server on port 9003 at 2023-07-20T10:30:45+08:00
Request 4:
Response from backend server on port 9001 at 2023-07-20T10:30:45+08:00
Request 5:
Response from backend server on port 9002 at 2023-07-20T10:30:45+08:00
Request 6:
Response from backend server on port 9003 at 2023-07-20T10:30:45+08:00
```

### 5. 测试加权轮询

修改配置文件，使用加权轮询策略：

```yaml
services:
  web-service:
    load_balancer:
      strategy: "weighted_round_robin"  # 使用加权轮询策略
      backends:
        - url: "http://localhost:9001"
          weight: 3  # 高权重，接收更多请求
          active: true
        - url: "http://localhost:9002"
          weight: 2  # 中等权重
          active: true
        - url: "http://localhost:9003"
          weight: 1  # 低权重，接收较少请求
          active: true
      health_check:
        enabled: true
        interval: 10s
        timeout: 5s
        path: "/health"

host_rules:
  - pattern: "localhost"
    port: 8080
    target: "web-service"
```

重启代理服务器并再次测试，你会发现9001端口的服务器接收的请求大约是9003的3倍。

### 6. 测试健康检查

停止其中一个后端服务：

```bash
kill %1  # 停止9001端口的服务
```

继续发送请求，你会发现代理服务器会自动检测到9001端口的服务不可用，并将请求分发到其他健康的服务器：

```bash
for i in {1..6}; do 
  echo "Request $i:"
  curl -s http://localhost:8080/
  echo
done
```

预期输出：

```
Request 1:
Response from backend server on port 9002 at 2023-07-20T10:35:15+08:00
Request 2:
Response from backend server on port 9003 at 2023-07-20T10:35:15+08:00
Request 3:
Response from backend server on port 9002 at 2023-07-20T10:35:15+08:00
Request 4:
Response from backend server on port 9003 at 2023-07-20T10:35:15+08:00
Request 5:
Response from backend server on port 9002 at 2023-07-20T10:35:15+08:00
Request 6:
Response from backend server on port 9003 at 2023-07-20T10:35:15+08:00
```

### 7. 监控和日志

代理服务器会记录负载均衡相关的日志，包括：

- 后端服务器的健康状态变化
- 请求分发情况
- 负载均衡策略选择

查看日志：

```bash
tail -f proxy.log
```

## 故障排除

常见问题及解决方案：

1. **连接失败**：检查目标服务器URL和网络连接
2. **协议错误**：确保客户端发送正确的WebSocket升级请求
3. **数据丢失**：检查缓冲区大小和超时设置
4. **性能问题**：监控连接数和数据传输量，优化配置参数

## 插件系统

Toyou Proxy 提供了强大的插件系统，支持中间件的自动发现、动态加载和热更新。插件系统基于Go语言的插件机制，允许开发者在不修改主程序的情况下扩展代理功能。

### 系统架构

插件系统由以下核心组件组成：

1. **AutoPluginManager**：自动插件管理器，负责插件的发现、编译、加载和生命周期管理
2. **PluginMetadata**：插件元数据，定义插件的基本信息和配置结构
3. **Middleware Interface**：中间件接口，定义插件必须实现的标准接口
4. **Configuration System**：配置系统，管理插件配置和系统配置

### 目录结构

插件系统遵循标准的目录结构：

```
middleware/
├── interfaces.go          # 中间件接口定义
├── manager.go            # 中间件管理器
├── auto_plugin_manager.go # 自动插件管理器
├── context.go            # 中间件上下文
└── plugins/              # 插件目录
    ├── auth/             # 认证插件
    │   ├── plugin.go     # 插件实现
    │   └── plugin.json   # 插件元数据
    ├── rate_limit/       # 限流插件
    │   ├── plugin.go
    │   └── plugin.json
    └── ...               # 其他插件
```

### 插件元数据

每个插件目录必须包含`plugin.json`文件，定义插件的元数据：

```json
{
  "name": "auth",
  "version": "1.0.0",
  "description": "认证中间件，支持多种认证方式",
  "entry_point": "PluginMain",
  "config": {
    "type": "basic",
    "users": [
      {"username": "admin", "password": "admin123"}
    ]
  },
  "enabled": true
}
```

插件元数据包含以下字段：

- `name`：插件名称，必须唯一
- `version`：插件版本号
- `description`：插件描述
- `entry_point`：插件入口函数名称，通常为`PluginMain`
- `config`：插件默认配置
- `enabled`：是否启用插件

### 插件接口

所有插件必须实现`Middleware`接口：

```go
// Middleware 中间件接口
type Middleware interface {
    // Name 返回中间件名称
    Name() string
    
    // Handle 处理请求
    // 返回true表示继续执行下一个中间件，false表示中断请求处理
    Handle(ctx *Context) bool
}
```

`Context`结构体提供了中间件之间共享数据的机制：

```go
// Context 中间件上下文
type Context struct {
    Request  *http.Request
    Response http.ResponseWriter
    Values   map[string]interface{} // 用于中间件间传递数据
}
```

### 插件实现

每个插件必须提供一个入口函数，用于创建中间件实例：

```go
// PluginMain 插件入口函数
func PluginMain(config map[string]interface{}) (middleware.Middleware, error)
```

以下是一个简单的认证插件实现示例：

```go
package main

import (
    "fmt"
    "net/http"
    "toyou-proxy/middleware"
)

// AuthMiddleware 认证中间件
type AuthMiddleware struct {
    users map[string]string // 用户名到密码的映射
}

// NewAuthMiddleware 创建认证中间件实例
func NewAuthMiddleware(config map[string]interface{}) (*AuthMiddleware, error) {
    // 从配置中获取用户信息
    usersConfig, ok := config["users"].([]interface{})
    if !ok {
        return nil, fmt.Errorf("invalid users configuration")
    }
    
    users := make(map[string]string)
    for _, userConfig := range usersConfig {
        userMap, ok := userConfig.(map[string]interface{})
        if !ok {
            continue
        }
        
        username, _ := userMap["username"].(string)
        password, _ := userMap["password"].(string)
        
        if username != "" && password != "" {
            users[username] = password
        }
    }
    
    return &AuthMiddleware{users: users}, nil
}

// Name 返回中间件名称
func (am *AuthMiddleware) Name() string {
    return "auth"
}

// Handle 处理请求
func (am *AuthMiddleware) Handle(ctx *middleware.Context) bool {
    // 获取HTTP基本认证信息
    username, password, ok := ctx.Request.BasicAuth()
    if !ok {
        // 未提供认证信息，返回401
        ctx.Response.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
        http.Error(ctx.Response, "Unauthorized", http.StatusUnauthorized)
        return false
    }
    
    // 验证用户名和密码
    storedPassword, exists := am.users[username]
    if !exists || storedPassword != password {
        // 认证失败，返回401
        http.Error(ctx.Response, "Unauthorized", http.StatusUnauthorized)
        return false
    }
    
    // 认证成功，继续执行下一个中间件
    return true
}

// PluginMain 插件入口函数
func PluginMain(config map[string]interface{}) (middleware.Middleware, error) {
    return NewAuthMiddleware(config)
}
```

### 插件生命周期

插件系统管理插件的完整生命周期：

1. **发现**：自动扫描插件目录，发现有效的插件
2. **编译**：将Go源代码编译为插件库(.so文件)
3. **加载**：动态加载插件库到内存
4. **初始化**：调用插件入口函数创建中间件实例
5. **注册**：将中间件实例注册到中间件链
6. **执行**：按优先级顺序执行中间件
7. **卸载**：支持插件的动态卸载和重新加载

### 自动编译机制

插件系统实现了自动编译机制，可以将Go源代码编译为插件库：

```go
// compilePlugin 编译插件
func (apm *AutoPluginManager) compilePlugin(pluginPath string) (string, error) {
    // 创建缓存目录
    cacheDir := filepath.Join(apm.cacheDir, filepath.Base(pluginPath))
    if err := os.MkdirAll(cacheDir, 0755); err != nil {
        return "", fmt.Errorf("failed to create cache directory: %v", err)
    }
    
    // 生成输出文件路径
    outputFile := filepath.Join(cacheDir, "plugin.so")
    
    // 构建编译命令
    cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", outputFile, pluginPath)
    cmd.Dir = pluginPath
    
    // 执行编译
    if output, err := cmd.CombinedOutput(); err != nil {
        return "", fmt.Errorf("failed to compile plugin: %v, output: %s", err, string(output))
    }
    
    return outputFile, nil
}
```

### 插件配置

插件配置分为两个级别：

1. **插件元数据配置**：在`plugin.json`文件中定义，包含插件的默认配置
2. **系统配置文件**：在主配置文件中覆盖插件的默认配置

```yaml
# 在主配置文件中配置插件
middlewares:
  - name: "auth"
    enabled: true
    config:
      type: "basic"
      users:
        - username: "admin"
          password: "admin123"
        - username: "user"
          password: "user123"
  - name: "rate_limit"
    enabled: true
    config:
      requests_per_minute: 60
      burst: 10
```

### 插件管理API

插件系统提供了丰富的管理API：

```go
// AutoPluginManager 提供的主要方法
type AutoPluginManager struct {
    // ...
}

// LoadPlugin 加载插件
func (apm *AutoPluginManager) LoadPlugin(pluginName string) (plugin.Symbol, error)

// GetPluginCreator 获取插件创建函数
func (apm *AutoPluginManager) GetPluginCreator(pluginName string) (func(map[string]interface{}) (middleware.Middleware, error), error)

// GetPluginMetadata 获取插件元数据
func (apm *AutoPluginManager) GetPluginMetadata(pluginName string) (*middleware.PluginMetadata, error)

// DiscoverPlugins 发现所有有效插件
func (apm *AutoPluginManager) DiscoverPlugins() ([]string, error)

// ReloadPlugin 重新加载插件
func (apm *AutoPluginManager) ReloadPlugin(pluginName string) error

// ClearCache 清空插件缓存
func (apm *AutoPluginManager) ClearCache() error
```

### 热更新机制

插件系统支持热更新，可以在不重启服务的情况下更新插件：

1. **文件监控**：监控插件目录变化
2. **自动重载**：检测到插件文件变化时自动重新加载
3. **平滑切换**：确保在插件更新过程中服务不中断
4. **错误恢复**：更新失败时自动回滚到旧版本

### 开发最佳实践

1. **接口设计**：遵循标准的中间件接口，确保兼容性
2. **错误处理**：实现健壮的错误处理机制，避免插件错误影响主程序
3. **性能考虑**：避免在插件中执行耗时操作，考虑异步处理
4. **配置验证**：验证插件配置的有效性，提供友好的错误提示
5. **日志记录**：记录关键操作和错误信息，便于调试和监控
6. **资源管理**：合理管理资源，避免内存泄漏
7. **并发安全**：确保插件在并发环境下的安全性

### 插件示例

以下是一个完整的限流插件示例：

```go
package main

import (
    "fmt"
    "net/http"
    "sync"
    "time"
    "toyou-proxy/middleware"
)

// RateLimitMiddleware 限流中间件
type RateLimitMiddleware struct {
    requestsPerMinute int
    burst             int
    tokens            int
    lastRefill        time.Time
    mutex             sync.Mutex
}

// NewRateLimitMiddleware 创建限流中间件实例
func NewRateLimitMiddleware(config map[string]interface{}) (*RateLimitMiddleware, error) {
    // 从配置中获取参数
    rpm, ok := config["requests_per_minute"].(int)
    if !ok {
        rpm = 60 // 默认值
    }
    
    burst, ok := config["burst"].(int)
    if !ok {
        burst = 10 // 默认值
    }
    
    return &RateLimitMiddleware{
        requestsPerMinute: rpm,
        burst:             burst,
        tokens:            burst,
        lastRefill:        time.Now(),
    }, nil
}

// Name 返回中间件名称
func (rlm *RateLimitMiddleware) Name() string {
    return "rate_limit"
}

// Handle 处理请求
func (rlm *RateLimitMiddleware) Handle(ctx *middleware.Context) bool {
    rlm.mutex.Lock()
    defer rlm.mutex.Unlock()
    
    // 计算需要添加的令牌数
    now := time.Now()
    elapsed := now.Sub(rlm.lastRefill)
    tokensToAdd := int(elapsed.Minutes() * float64(rlm.requestsPerMinute))
    
    // 更新令牌数，不超过burst
    rlm.tokens = rlm.burst
    if tokensToAdd < rlm.burst {
        rlm.tokens += tokensToAdd
    }
    
    // 更新最后填充时间
    rlm.lastRefill = now
    
    // 检查是否有足够的令牌
    if rlm.tokens <= 0 {
        // 令牌不足，拒绝请求
        http.Error(ctx.Response, "Too Many Requests", http.StatusTooManyRequests)
        return false
    }
    
    // 消耗一个令牌
    rlm.tokens--
    
    // 继续执行下一个中间件
    return true
}

// PluginMain 插件入口函数
func PluginMain(config map[string]interface{}) (middleware.Middleware, error) {
    return NewRateLimitMiddleware(config)
}
```

### 扩展方向

插件系统支持多种扩展方向：

1. **插件依赖**：支持插件之间的依赖关系管理
2. **插件市场**：建立插件市场，方便插件的分发和共享
3. **插件沙箱**：实现插件沙箱，提高安全性
4. **插件热部署**：支持插件的热部署和版本管理
5. **插件监控**：提供插件性能监控和告警功能
6. **插件测试**：提供插件测试框架和工具

### 故障排除

常见插件问题及解决方案：

1. **编译失败**：检查Go环境和依赖项
2. **加载失败**：检查插件路径和权限
3. **配置错误**：验证插件配置格式和内容
4. **运行时错误**：查看日志，检查插件实现
5. **性能问题**：分析插件执行时间，优化代码

## 性能优化

- 使用连接池复用HTTP客户端
- 支持响应缓存（通过缓存中间件）
- 异步日志记录，减少I/O阻塞
- 支持GZIP压缩

## 监控和日志

代理服务会自动记录：

- 请求处理时间
- 目标服务响应
- 中间件执行状态
- 错误和异常信息

## 故障排除

### 常见问题

1. **端口占用**：检查80端口是否被其他进程占用
2. **配置错误**：验证YAML格式和路径配置
3. **服务不可达**：检查后端服务是否正常运行
4. **权限问题**：Linux系统可能需要sudo权限绑定80端口

### 调试模式

启动时添加详细日志：

```bash
./toyou-proxy -config config.yaml 2>&1 | tee proxy.log
```

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request！