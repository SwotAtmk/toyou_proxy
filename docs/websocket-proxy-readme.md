# WebSocket代理功能

本文档介绍了Toyou Proxy的WebSocket代理功能，包括其实现原理、配置方法和使用示例。

## 功能概述

Toyou Proxy现在支持WebSocket代理功能，可以透明地代理WebSocket连接，实现客户端与WebSocket服务器之间的双向通信。

## 实现原理

WebSocket代理功能通过以下组件实现：

1. **WebSocket检测中间件** (`middleware/plugins/websocket/plugin.go`)
   - 检测WebSocket升级请求
   - 标记WebSocket连接
   - 跟踪连接状态

2. **WebSocket代理处理器** (`proxy/websocket_proxy.go`)
   - 处理WebSocket协议升级
   - 建立与目标服务器的连接
   - 实现双向数据转发

3. **协议升级处理** (`proxy/websocket_upgrade.go`)
   - 劫持HTTP连接
   - 处理协议升级握手
   - 管理连接生命周期

4. **代理处理器集成** (`proxy/proxy_handler.go`)
   - 集成WebSocket检测
   - 路由WebSocket请求
   - 处理WebSocket错误

## 配置方法

### 1. 启用WebSocket中间件

在配置文件中添加WebSocket中间件：

```yaml
middlewares:
  - name: "websocket"
    enabled: true
    config:
      path_patterns: ["/ws", "/api/ws"]
      connection_tracking: true
      max_connections: 100
```

### 2. 配置路由规则

在域名规则中添加WebSocket路由：

```yaml
host_rules:
  - pattern: "example.com"
    target: "websocket-service"
    port: 80
    route_rules:
      - pattern: "/ws"
        target: "websocket-service"
        middlewares: ["websocket"]
```

### 3. 配置服务

定义WebSocket服务：

```yaml
services:
  - name: "websocket-service"
    url: "http://websocket-server:8081"
    proxy_host: "websocket-server:8081"
    health_check:
      path: "/"
      interval: 30
      timeout: 5
```

## 测试方法

### 1. 运行测试脚本

项目提供了完整的测试脚本，可以一键测试WebSocket代理功能：

```bash
./test/websocket_test.sh
```

测试脚本会：
1. 构建代理服务器和测试服务器
2. 启动WebSocket测试服务器
3. 启动代理服务器
4. 运行测试客户端（直接连接和通过代理连接）
5. 清理进程

### 2. 手动测试

#### 启动测试服务器

```bash
go run test/websocket_test_server.go
```

#### 启动代理服务器

```bash
./bin/toyou-proxy -config test/websocket_test_config.yaml
```

#### 运行测试客户端

直接连接测试服务器：
```bash
go run test/websocket_test_client.go "ws://localhost:8081/ws"
```

通过代理连接测试服务器：
```bash
go run test/websocket_test_client.go "ws://localhost:8080/ws"
```

## WebSocket检测规则

代理服务器通过以下规则检测WebSocket请求：

1. **Connection头**：包含"upgrade"
2. **Upgrade头**：值为"websocket"
3. **Sec-WebSocket-Version头**：值为"13"
4. **Sec-WebSocket-Key头**：非空

## 限制与注意事项

1. **协议支持**：目前支持WebSocket协议（RFC 6455）
2. **子协议**：支持自定义子协议协商
3. **扩展**：支持WebSocket扩展协商
4. **连接跟踪**：可选的连接跟踪功能，用于监控连接状态
5. **最大连接数**：可配置最大连接数限制

## 故障排除

### 1. 连接失败

检查以下项目：
- 目标服务器是否正常运行
- 网络连接是否正常
- 代理配置是否正确

### 2. 协议升级失败

检查以下项目：
- 请求头是否正确
- 代理服务器是否正确识别WebSocket请求
- 中间件是否正确配置

### 3. 数据传输问题

检查以下项目：
- 防火墙设置
- 代理超时配置
- 连接保持设置

## 性能优化

1. **连接池**：考虑实现连接池以减少连接建立开销
2. **缓冲区大小**：根据实际需求调整读写缓冲区大小
3. **压缩**：考虑启用WebSocket压缩扩展
4. **负载均衡**：在多服务器环境中实现负载均衡

## 未来计划

1. **SSL/TLS支持**：增强对wss://协议的支持
2. **认证集成**：与现有认证系统集成
3. **监控指标**：添加详细的性能监控指标
4. **配置热更新**：支持配置热更新
5. **高级路由**：支持基于路径、头部等的复杂路由规则