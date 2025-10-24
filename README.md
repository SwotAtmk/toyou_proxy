# Toyou Proxy - Go语言反向代理服务

一个功能强大的Go语言反向代理服务，支持灵活的域名匹配、路由匹配和中间件系统。

## 功能特性

- ✅ **域名匹配规则**：支持通配符域名匹配（如 `*.example.com`）
- ✅ **路由匹配规则**：支持路径前缀匹配（如 `/api/*`）
- ✅ **中间件系统**：支持认证、限流、CORS等中间件
- ✅ **端口监听**：默认监听80端口，可配置
- ✅ **服务发现**：支持动态服务配置和健康检查
- ✅ **高性能**：基于Go标准库，轻量高效

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 配置代理规则

编辑 `config.yaml` 文件，配置你的代理规则：

```yaml
server:
  port: 80

host_rules:
  - pattern: "*.api.example.com"
    target: "api-service"
  - pattern: "*.admin.example.com"
    target: "admin-service"

route_rules:
  - pattern: "/api/v1/*"
    target: "api-v1-service"
  - pattern: "/api/v2/*"
    target: "api-v2-service"

services:
  "api-service":
    url: "http://localhost:3000"
  "admin-service":
    url: "http://localhost:3001"
  "api-v1-service":
    url: "http://localhost:4000"
  "api-v2-service":
    url: "http://localhost:4001"
```

### 3. 启动代理服务

```bash
# 使用默认配置
./toyou-proxy

# 指定配置文件
./toyou-proxy -config custom-config.yaml

# 查看帮助
./toyou-proxy -help
```

### 4. 构建可执行文件

```bash
go build -o toyou-proxy cmd/main.go
```

## 配置说明

### 域名匹配规则

域名匹配规则支持通配符模式：

- `*.example.com` - 匹配所有子域名
- `api.example.com` - 精确匹配

示例：
```yaml
host_rules:
  - pattern: "*.aaa.com"
    target: "aaa-backend"
  - pattern: "api.bbb.com"
    target: "api-service"
```

### 路由匹配规则

路由匹配规则支持路径前缀匹配：

- `/api/*` - 匹配所有以 `/api/` 开头的路径
- `/admin` - 精确匹配

示例：
```yaml
route_rules:
  - pattern: "/api/users/*"
    target: "user-service"
  - pattern: "/api/products/*"
    target: "product-service"
```

### 中间件配置

支持多种中间件，可按需启用：

#### 认证中间件
```yaml
- name: "auth"
  enabled: true
  config:
    header_name: "X-API-Key"
    valid_keys: ["your-secret-key"]
```

#### 限流中间件
```yaml
- name: "rate_limit"
  enabled: true
  config:
    requests_per_minute: 100
    burst_size: 20
```

#### CORS中间件
```yaml
- name: "cors"
  enabled: true
  config:
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE"]
```

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

## 中间件开发

### 自定义中间件

实现 `middleware.Middleware` 接口：

```go
type CustomMiddleware struct {
    // 配置字段
}

func (cm *CustomMiddleware) Name() string {
    return "custom"
}

func (cm *CustomMiddleware) Handle(ctx *middleware.Context) bool {
    // 中间件逻辑
    return true // 返回false中断请求
}
```

### 中间件上下文

中间件可以访问和修改请求上下文：

```go
type Context struct {
    Request     *http.Request
    Response    http.ResponseWriter
    TargetURL   string           // 目标URL
    ServiceName string           // 服务名称
    Aborted     bool             // 是否中断请求
    StatusCode  int              // 状态码
}
```

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