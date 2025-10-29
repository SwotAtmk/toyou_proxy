# WebSocket代理配置示例

## 1. 基本WebSocket代理配置

```yaml
# config.yaml
hosts:
  - hostname: "example.com"
    routes:
      - path: "/ws"
        target: "ws://backend-server:8080/websocket"
        websocket: true
        # WebSocket特定配置
        websocket_config:
          # Origin验证配置
          origin_check:
            enabled: true
            allowed_origins:
              - "https://example.com"
              - "https://app.example.com"
          
          # 连接限制配置
          connection_limits:
            max_connections: 1000
            max_connections_per_ip: 10
            connection_timeout: "30s"
            idle_timeout: "300s"
          
          # 消息限制配置
          message_limits:
            max_message_size: "1MB"
            max_message_rate: "100/s"
          
          # 心跳配置
          ping_interval: "30s"
          pong_wait: "10s"
          
          # 缓冲区配置
          buffer_size: "32KB"
          
          # 重试配置
          retry:
            enabled: true
            max_attempts: 3
            backoff: "1s"
        
        # 中间件配置
        middlewares:
          - "auth"
          - "logging"
          - "websocket"
        
        # 健康检查配置
        health_check:
          enabled: true
          path: "/ws-health"
          interval: "30s"
          timeout: "5s"
```

## 2. 多路径WebSocket代理配置

```yaml
# config.yaml
hosts:
  - hostname: "api.example.com"
    routes:
      # 聊天WebSocket
      - path: "/chat"
        target: "ws://chat-server:8080"
        websocket: true
        websocket_config:
          origin_check:
            enabled: true
            allowed_origins:
              - "https://example.com"
              - "https://app.example.com"
          connection_limits:
            max_connections: 5000
            max_connections_per_ip: 5
          ping_interval: "20s"
        middlewares:
          - "auth"
          - "rate_limit"
          - "websocket"
      
      # 实时数据WebSocket
      - path: "/realtime"
        target: "ws://data-server:8080/data"
        websocket: true
        websocket_config:
          origin_check:
            enabled: false
          connection_limits:
            max_connections: 10000
            max_connections_per_ip: 20
          message_limits:
            max_message_size: "5MB"
            max_message_rate: "1000/s"
          ping_interval: "10s"
        middlewares:
          - "auth"
          - "websocket"
      
      # 通知WebSocket
      - path: "/notifications"
        target: "ws://notification-server:8080"
        websocket: true
        websocket_config:
          origin_check:
            enabled: true
            allowed_origins:
              - "https://example.com"
              - "https://app.example.com"
              - "https://admin.example.com"
          connection_limits:
            max_connections: 2000
            max_connections_per_ip: 10
          ping_interval: "60s"
        middlewares:
          - "auth"
          - "websocket"
```

## 3. WSS (WebSocket Secure) 代理配置

```yaml
# config.yaml
hosts:
  - hostname: "secure.example.com"
    ssl:
      cert: "/path/to/cert.pem"
      key: "/path/to/key.pem"
      # 启用HTTP/2支持
      http2: true
    
    routes:
      - path: "/secure-ws"
        target: "wss://secure-backend:8443/websocket"
        websocket: true
        websocket_config:
          # WSS特定配置
          secure: true
          # 后端SSL验证
          backend_ssl:
            verify: true
            ca_cert: "/path/to/ca.pem"
            server_name: "secure-backend"
          
          # 安全增强配置
          security:
            # 强制使用TLS 1.2+
            min_tls_version: "1.2"
            # 禁用弱密码套件
            cipher_suites:
              - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
              - "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305"
              - "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
          
          origin_check:
            enabled: true
            allowed_origins:
              - "https://secure.example.com"
              - "https://app.example.com"
          
          connection_limits:
            max_connections: 1000
            max_connections_per_ip: 5
            connection_timeout: "30s"
            idle_timeout: "600s"
          
          ping_interval: "30s"
          pong_wait: "10s"
        
        middlewares:
          - "auth"
          - "logging"
          - "websocket"
```

## 4. 负载均衡WebSocket代理配置

```yaml
# config.yaml
hosts:
  - hostname: "ws.example.com"
    routes:
      - path: "/cluster-ws"
        target: "ws://ws-cluster"
        websocket: true
        websocket_config:
          # 负载均衡配置
          load_balancer:
            strategy: "round_robin"  # round_robin, least_connections, ip_hash
            health_check:
              enabled: true
              interval: "10s"
              timeout: "3s"
              path: "/health"
            # 后端服务器列表
            backends:
              - "ws://server1:8080"
              - "ws://server2:8080"
              - "ws://server3:8080"
              - "ws://server4:8080"
          
          # 会话保持配置
          session_affinity:
            enabled: true
            timeout: "3600s"  # 1小时
            cookie_name: "WS_SESSION"
          
          origin_check:
            enabled: true
            allowed_origins:
              - "https://example.com"
              - "https://app.example.com"
          
          connection_limits:
            max_connections: 10000
            max_connections_per_ip: 10
            connection_timeout: "30s"
            idle_timeout: "300s"
          
          ping_interval: "30s"
          pong_wait: "10s"
        
        middlewares:
          - "auth"
          - "logging"
          - "websocket"
```

## 5. WebSocket中间件配置

```yaml
# config.yaml
middlewares:
  # WebSocket中间件
  websocket:
    plugin: "websocket"
    config:
      # 全局WebSocket配置
      global:
        # 默认连接限制
        default_connection_limits:
          max_connections: 1000
          max_connections_per_ip: 10
          connection_timeout: "30s"
          idle_timeout: "300s"
        
        # 默认消息限制
        default_message_limits:
          max_message_size: "1MB"
          max_message_rate: "100/s"
        
        # 默认心跳配置
        default_ping_config:
          ping_interval: "30s"
          pong_wait: "10s"
        
        # 连接跟踪配置
        connection_tracking:
          enabled: true
          track_ip: true
          track_user_agent: true
          track_origin: true
          track_referer: true
        
        # 监控配置
        metrics:
          enabled: true
          export_interval: "10s"
          include_connection_details: true
          include_message_stats: true
```

## 6. 高级WebSocket代理配置

```yaml
# config.yaml
hosts:
  - hostname: "advanced.example.com"
    routes:
      - path: "/advanced-ws"
        target: "ws://backend-server:8080/websocket"
        websocket: true
        websocket_config:
          # 高级连接配置
          advanced_connection:
            # 连接队列配置
            accept_queue_size: 1024
            # 连接池配置
            connection_pool:
              enabled: true
              max_idle_connections: 100
              max_idle_connections_per_host: 10
              idle_connection_timeout: "90s"
            
            # 读写超时配置
            read_timeout: "60s"
            write_timeout: "60s"
            
            # TCP配置
            tcp:
              keep_alive: true
              no_delay: true
              buffer_size: "64KB"
          
          # 消息压缩配置
          compression:
            enabled: true
            algorithm: "deflate"  # deflate, gzip
            level: 6  # 压缩级别 (1-9)
            threshold: "1KB"  # 压缩阈值
          
          # 二进制消息处理
          binary_message:
            max_size: "10MB"
            buffer_pool:
              enabled: true
              buffer_size: "32KB"
              max_buffers: 1000
          
          # 错误处理配置
          error_handling:
            # 错误日志级别
            log_level: "warn"  # debug, info, warn, error
            # 错误响应配置
            error_response:
              include_details: false
              generic_message: "WebSocket connection error"
            # 重试配置
            retry:
              enabled: true
              max_attempts: 3
              backoff_factor: 2.0
              max_backoff: "30s"
          
          # 安全配置
          security:
            # IP白名单
            ip_whitelist:
              - "192.168.1.0/24"
              - "10.0.0.0/8"
            
            # IP黑名单
            ip_blacklist:
              - "192.168.100.100"
            
            # 速率限制
            rate_limit:
              enabled: true
              requests_per_second: 10
              burst: 20
          
          # 监控和日志
          monitoring:
            # 详细日志
            detailed_logging: false
            # 连接事件日志
            log_connection_events: true
            # 消息统计
            message_stats:
              enabled: true
              sample_rate: 0.1  # 10%采样率
            # 性能指标
            performance_metrics:
              enabled: true
              collection_interval: "10s"
        
        middlewares:
          - "auth"
          - "rate_limit"
          - "logging"
          - "websocket"
```

## 7. 多环境WebSocket代理配置

### 开发环境配置 (config.dev.yaml)

```yaml
hosts:
  - hostname: "dev.example.com"
    routes:
      - path: "/ws"
        target: "ws://localhost:8080"
        websocket: true
        websocket_config:
          origin_check:
            enabled: false  # 开发环境禁用Origin检查
          connection_limits:
            max_connections: 100
            max_connections_per_ip: 50
          ping_interval: "60s"
          monitoring:
            detailed_logging: true
            log_connection_events: true
        middlewares:
          - "websocket"
```

### 测试环境配置 (config.test.yaml)

```yaml
hosts:
  - hostname: "test.example.com"
    routes:
      - path: "/ws"
        target: "ws://test-backend:8080"
        websocket: true
        websocket_config:
          origin_check:
            enabled: true
            allowed_origins:
              - "https://test.example.com"
          connection_limits:
            max_connections: 500
            max_connections_per_ip: 20
          ping_interval: "30s"
          monitoring:
            detailed_logging: true
            message_stats:
              enabled: true
              sample_rate: 1.0  # 100%采样率
        middlewares:
          - "auth"
          - "logging"
          - "websocket"
```

### 生产环境配置 (config.prod.yaml)

```yaml
hosts:
  - hostname: "prod.example.com"
    ssl:
      cert: "/path/to/prod/cert.pem"
      key: "/path/to/prod/key.pem"
      http2: true
    
    routes:
      - path: "/ws"
        target: "wss://prod-backend:8443"
        websocket: true
        websocket_config:
          secure: true
          backend_ssl:
            verify: true
            ca_cert: "/path/to/prod/ca.pem"
            server_name: "prod-backend"
          
          origin_check:
            enabled: true
            allowed_origins:
              - "https://prod.example.com"
              - "https://app.example.com"
          
          connection_limits:
            max_connections: 10000
            max_connections_per_ip: 10
            connection_timeout: "30s"
            idle_timeout: "600s"
          
          message_limits:
            max_message_size: "1MB"
            max_message_rate: "100/s"
          
          ping_interval: "30s"
          pong_wait: "10s"
          
          compression:
            enabled: true
            algorithm: "deflate"
            level: 6
            threshold: "1KB"
          
          monitoring:
            detailed_logging: false
            message_stats:
              enabled: true
              sample_rate: 0.01  # 1%采样率
            performance_metrics:
              enabled: true
              collection_interval: "30s"
        
        middlewares:
          - "auth"
          - "rate_limit"
          - "logging"
          - "websocket"
```

## 8. WebSocket代理健康检查配置

```yaml
# config.yaml
hosts:
  - hostname: "health.example.com"
    routes:
      - path: "/ws"
        target: "ws://backend-server:8080"
        websocket: true
        websocket_config:
          # 健康检查配置
          health_check:
            enabled: true
            # 健康检查路径
            path: "/ws-health"
            # 检查间隔
            interval: "30s"
            # 超时时间
            timeout: "5s"
            # 成功阈值
            success_threshold: 2
            # 失败阈值
            failure_threshold: 3
            # 健康检查消息
            health_check_message: '{"type":"health_check"}'
            # 预期响应
            expected_response: '{"status":"ok"}'
            # 不健康状态处理
            on_unhealthy:
              # 返回错误状态码
              return_status: 503
              # 返回错误消息
              return_message: "WebSocket service unavailable"
              # 记录日志
              log_error: true
        
        # 路由级健康检查
        health_check:
          enabled: true
          path: "/health"
          interval: "10s"
          timeout: "2s"
```

这些配置示例展示了如何在不同场景下配置WebSocket代理，包括基本配置、多路径配置、安全配置、负载均衡配置和高级功能配置。开发团队可以根据实际需求选择和调整这些配置。