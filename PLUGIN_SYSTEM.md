# 反向代理中间件插件系统

## 概述

本系统是一个统一的反向代理中间件插件系统，采用"约定优于配置"原则，提供标准化的插件开发接口，支持插件的自动发现和动态加载。

## 核心组件

### 1. 中间件接口

#### Middleware 接口

```go
type Middleware interface {
    // Name 返回中间件名称
    Name() string
    
    // Handle 处理请求
    // 返回true表示继续执行下一个中间件，false表示中断请求处理
    Handle(ctx *Context) bool
}
```

#### Context 结构体

```go
type Context struct {
    Request  *http.Request
    Response http.ResponseWriter
    Values   map[string]interface{} // 用于中间件间传递数据
}
```

### 2. 中间件链

#### MiddlewareChain 接口

```go
type MiddlewareChain interface {
    // Add 添加中间件到链的末尾
    Add(middleware Middleware)
    
    // Insert 在指定位置插入中间件
    Insert(index int, middleware Middleware) error
    
    // Remove 移除指定名称的中间件
    Remove(name string) error
    
    // Process 处理请求
    Process(ctx *Context)
    
    // Get 获取指定名称的中间件
    Get(name string) (Middleware, bool)
    
    // List 列出所有中间件
    List() []Middleware
}
```

### 3. 插件接口

#### Plugin 接口

```go
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
```

#### PluginManager 接口

```go
type PluginManager interface {
    // 加载插件
    LoadPlugin(pluginPath string) error
    
    // 卸载插件
    UnloadPlugin(pluginName string) error
    
    // 获取插件
    GetPlugin(pluginName string) (Plugin, bool)
    
    // 列出所有插件
    ListPlugins() []Plugin
    
    // 重新加载插件
    ReloadPlugin(pluginName string) error
    
    // 发现插件目录中的所有插件
    DiscoverPlugins() ([]string, error)
    
    // 加载插件目录中的所有插件
    LoadAllPlugins() error
    
    // 获取插件目录
    GetPluginDir() string
}
```

## 插件开发规范

### 1. 插件目录结构

每个插件必须按照以下目录结构组织：

```
plugins/
├── plugin_name/
│   ├── plugin.json      # 插件元数据和配置
│   ├── plugin.go        # 插件实现文件
│   └── plugin.so        # 编译后的插件文件（可选）
```

### 2. 插件元数据 (plugin.json)

```json
{
    "name": "plugin_name",
    "version": "1.0.0",
    "description": "插件描述",
    "type": "plugin_type",
    "config": {
        // 插件特定配置
    },
    "enabled": true
}
```

### 3. 插件实现 (plugin.go)

插件实现文件必须包含以下函数：

```go
// PluginMain 插件入口函数
func PluginMain(config map[string]interface{}) (Middleware, error)
```

### 4. 插件示例

#### CORS 插件示例

```go
package main

import (
    "net/http"
    "strings"
)

// CORSMiddleware CORS中间件实现
type CORSMiddleware struct {
    allowedOrigins []string
    allowedMethods []string
    allowedHeaders []string
}

// NewCORSMiddleware 创建CORS中间件
func NewCORSMiddleware(config map[string]interface{}) (middleware.Middleware, error) {
    // 解析配置
    allowedOrigins, _ := config["allowed_origins"].([]string)
    allowedMethods, _ := config["allowed_methods"].([]string)
    allowedHeaders, _ := config["allowed_headers"].([]string)
    
    // 设置默认值
    if len(allowedOrigins) == 0 {
        allowedOrigins = []string{"*"}
    }
    if len(allowedMethods) == 0 {
        allowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
    }
    if len(allowedHeaders) == 0 {
        allowedHeaders = []string{"*"}
    }
    
    return &CORSMiddleware{
        allowedOrigins: allowedOrigins,
        allowedMethods: allowedMethods,
        allowedHeaders: allowedHeaders,
    }, nil
}

// Name 返回中间件名称
func (c *CORSMiddleware) Name() string {
    return "cors"
}

// Handle 处理请求
func (c *CORSMiddleware) Handle(ctx *middleware.Context) bool {
    // 设置CORS头
    origin := ctx.Request.Header.Get("Origin")
    if origin != "" && c.isOriginAllowed(origin) {
        ctx.Response.Header().Set("Access-Control-Allow-Origin", origin)
    }
    
    ctx.Response.Header().Set("Access-Control-Allow-Methods", strings.Join(c.allowedMethods, ", "))
    ctx.Response.Header().Set("Access-Control-Allow-Headers", strings.Join(c.allowedHeaders, ", "))
    
    // 处理预检请求
    if ctx.Request.Method == "OPTIONS" {
        ctx.Response.WriteHeader(http.StatusOK)
        return false // 中断请求处理
    }
    
    return true // 继续执行下一个中间件
}

// isOriginAllowed 检查源是否被允许
func (c *CORSMiddleware) isOriginAllowed(origin string) bool {
    for _, allowedOrigin := range c.allowedOrigins {
        if allowedOrigin == "*" || allowedOrigin == origin {
            return true
        }
    }
    return false
}

// PluginMain 插件入口函数
func PluginMain(config map[string]interface{}) (middleware.Middleware, error) {
    return NewCORSMiddleware(config)
}
```

## 使用示例

### 1. 基本使用

```go
package main

import (
    "net/http"
    "your_project/middleware"
)

func main() {
    // 创建插件管理器
    pluginManager := middleware.NewPluginManager("./plugins")
    
    // 加载所有插件
    if err := pluginManager.LoadAllPlugins(); err != nil {
        panic(err)
    }
    
    // 创建中间件工厂
    factory := middleware.NewDefaultMiddlewareFactory(pluginManager)
    
    // 从配置创建中间件链
    config := map[string]interface{}{
        "middlewares": []map[string]interface{}{
            {
                "name": "cors",
                "config": map[string]interface{}{
                    "allowed_origins": []string{"https://example.com"},
                    "allowed_methods": []string{"GET", "POST"},
                    "allowed_headers": []string{"Content-Type", "Authorization"},
                },
            },
            {
                "name": "logging",
                "config": map[string]interface{}{
                    "level": "info",
                },
            },
            {
                "name": "rate_limit",
                "config": map[string]interface{}{
                    "requests_per_minute": 60,
                    "burst_size": 10,
                },
            },
        },
    }
    
    chain, err := factory.CreateChainFromConfig(config)
    if err != nil {
        panic(err)
    }
    
    // 创建HTTP处理器
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 处理请求
        w.Write([]byte("Hello, World!"))
    })
    
    // 包装处理器
    wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := &middleware.Context{
            Request:  r,
            Response: w,
            Values:   make(map[string]interface{}),
        }
        
        // 执行中间件链
        chain.Process(ctx)
        
        // 如果请求未被中断，继续处理
        if ctx.Response == w {
            handler.ServeHTTP(w, r)
        }
    })
    
    // 启动HTTP服务器
    http.ListenAndServe(":8080", wrappedHandler)
}
```

### 3. 自定义配置验证

```go
package main

import (
    "your_project/middleware"
)

func main() {
    // 创建自定义配置模式
    schema := middleware.NewConfigSchema()
    
    // 添加规则
    schema.AddRule("api_key", middleware.ConfigRule{
        Required: true,
        Type:     "string",
        Pattern:  `^[a-zA-Z0-9]{32}$`,
    })
    
    schema.AddRule("timeout", middleware.ConfigRule{
        Required: false,
        Type:     "int",
        Default:  30.0,
        Min:      1.0,
        Max:      300.0,
    })
    
    // 验证配置
    config := map[string]interface{}{
        "api_key": "1234567890abcdef1234567890abcdef",
        "timeout": 60.0,
    }
    
    if err := schema.Validate(config); err != nil {
        panic(err)
    }
    
    // 使用配置...
}
```

## 配置验证

系统提供了强大的配置验证机制，支持以下验证规则：

1. **必填字段验证**：检查必填字段是否存在
2. **类型验证**：验证字段值是否符合预期类型
3. **枚举值验证**：验证字段值是否在允许的枚举列表中
4. **正则表达式验证**：验证字符串字段是否符合指定模式
5. **范围验证**：验证数字、字符串长度、数组长度是否在指定范围内
6. **自定义验证**：支持自定义验证函数

## 内置插件

系统提供了以下内置插件：

1. **CORS插件**：处理跨域资源共享
2. **日志插件**：记录请求日志
3. **限流插件**：实现请求频率限制

## 最佳实践

1. **插件设计**：
   - 保持插件简单、单一职责
   - 提供清晰的配置选项
   - 实现优雅的错误处理

2. **性能优化**：
   - 避免在中间件中执行耗时操作
   - 使用连接池管理资源
   - 实现适当的缓存策略

3. **安全性**：
   - 验证所有输入参数
   - 实现适当的访问控制
   - 记录安全相关事件

4. **可维护性**：
   - 编写清晰的文档
   - 提供充分的日志
   - 实现健康检查接口

## 故障排除

### 常见问题

1. **插件加载失败**：
   - 检查插件目录结构是否正确
   - 验证plugin.json格式是否有效
   - 确认plugin.go中是否实现了PluginMain函数

2. **配置验证失败**：
   - 检查配置是否符合插件模式
   - 验证必填字段是否提供
   - 确认字段值类型是否正确

## 扩展开发

### 添加新的内置插件

1. 在middleware/plugins目录下创建新插件目录
2. 实现plugin.go和plugin.json文件
3. 在config_validator.go中添加配置模式
4. 更新GetPluginSchema函数

### 自定义插件管理器

实现PluginManager接口：

```go
type CustomPluginManager struct {
    // 自定义字段
}

func (cpm *CustomPluginManager) LoadPlugin(pluginPath string) error {
    // 自定义实现
}

// 实现其他接口方法...
```

### 自定义中间件链

实现MiddlewareChain接口：

```go
type CustomMiddlewareChain struct {
    // 自定义字段
}

func (cmc *CustomMiddlewareChain) Add(middleware middleware.Middleware) {
    // 自定义实现
}

// 实现其他接口方法...
```

## 总结

本插件系统提供了灵活、可扩展的中间件插件架构，支持动态加载，适用于各种反向代理场景。通过标准化的接口和配置验证机制，开发者可以轻松地开发和集成自定义插件，实现功能的快速扩展。