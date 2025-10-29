# WebSocket代理测试计划

## 1. 测试概述

本文档详细描述了Toyou Proxy WebSocket代理功能的测试计划，包括功能测试、性能测试、安全测试和兼容性测试。测试将覆盖WebSocket代理的所有核心功能和边界情况。

## 2. 测试环境

### 2.1 测试基础设施

```
┌─────────────────────────────────────────────────────────────┐
│                    测试环境拓扑                              │
│                                                             │
│  ┌─────────────┐    ┌─────────────────────────────────┐   │
│  │   测试客户端  │◄──►│        Toyou Proxy              │   │
│  │  WebSocket   │    │    (WebSocket代理)              │   │
│  └─────────────┘    └─────────────────────────────────┘   │
│                             │                               │
│                             ▼                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              后端WebSocket服务                          │ │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────┐   │ │
│  │  │  Echo服务器  │ │  聊天服务器  │ │  数据流服务器   │   │ │
│  │  └─────────────┘ └─────────────┘ └─────────────────┘   │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                监控与日志系统                            │ │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────┐   │ │
│  │  │  指标收集    │ │   日志收集   │ │   测试报告      │   │ │
│  │  └─────────────┘ └─────────────┘ └─────────────────┘   │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 测试工具

- **WebSocket客户端库**:
  - JavaScript: `ws`, `socket.io-client`
  - Go: `gorilla/websocket`
  - Python: `websockets`
  - Java: `javax.websocket`

- **性能测试工具**:
  - 自定义WebSocket压力测试工具
  - JMeter WebSocket插件
  - Artillery.js

- **网络分析工具**:
  - Wireshark
  - Chrome DevTools
  - tcpdump

- **监控工具**:
  - Prometheus + Grafana
  - 自定义指标收集器

## 3. 功能测试

### 3.1 基本WebSocket代理测试

#### 3.1.1 协议升级测试

| 测试用例ID | 测试描述 | 预期结果 | 测试步骤 |
|------------|----------|----------|----------|
| WS-001 | 基本WebSocket协议升级 | 成功建立WebSocket连接 | 1. 客户端发送WebSocket升级请求<br>2. 验证代理返回101状态码<br>3. 验证响应头包含正确的Upgrade和Connection字段 |
| WS-002 | 无效的WebSocket升级请求 | 返回400错误 | 1. 发送缺少Sec-WebSocket-Key头的请求<br>2. 验证代理返回400错误 |
| WS-003 | 不支持的WebSocket版本 | 返回426错误 | 1. 发送包含不支持的WebSocket版本的请求<br>2. 验证代理返回426错误和Sec-WebSocket-Version头 |

#### 3.1.2 消息传输测试

| 测试用例ID | 测试描述 | 预期结果 | 测试步骤 |
|------------|----------|----------|----------|
| WS-004 | 文本消息传输 | 消息正确转发 | 1. 建立WebSocket连接<br>2. 发送文本消息<br>3. 验证后端服务器收到相同消息<br>4. 验证客户端收到后端响应 |
| WS-005 | 二进制消息传输 | 二进制数据正确转发 | 1. 建立WebSocket连接<br>2. 发送二进制消息<br>3. 验证后端服务器收到相同二进制数据<br>4. 验证客户端收到后端响应 |
| WS-006 | 大消息传输 | 大消息正确转发 | 1. 建立WebSocket连接<br>2. 发送大于配置限制的消息<br>3. 验证消息被正确处理或拒绝 |

#### 3.1.3 连接管理测试

| 测试用例ID | 测试描述 | 预期结果 | 测试步骤 |
|------------|----------|----------|----------|
| WS-007 | 正常连接关闭 | 连接正常关闭 | 1. 建立WebSocket连接<br>2. 客户端发送Close帧<br>3. 验证代理和后端正确关闭连接 |
| WS-008 | 异常连接关闭 | 连接正确清理 | 1. 建立WebSocket连接<br>2. 强制关闭客户端连接<br>3. 验证代理正确清理连接资源 |
| WS-009 | 连接超时处理 | 超时连接被关闭 | 1. 建立WebSocket连接<br>2. 不发送任何消息等待超时<br>3. 验证代理关闭空闲连接 |

### 3.2 安全测试

#### 3.2.1 Origin验证测试

| 测试用例ID | 测试描述 | 预期结果 | 测试步骤 |
|------------|----------|----------|----------|
| WS-010 | 有效Origin请求 | 连接成功 | 1. 发送包含允许Origin头的请求<br>2. 验证连接成功建立 |
| WS-011 | 无效Origin请求 | 连接被拒绝 | 1. 发送包含不允许Origin头的请求<br>2. 验证代理返回403错误 |
| WS-012 | 缺少Origin头 | 根据配置处理 | 1. 发送不包含Origin头的请求<br>2. 验证代理根据配置允许或拒绝连接 |

#### 3.2.2 认证测试

| 测试用例ID | 测试描述 | 预期结果 | 测试步骤 |
|------------|----------|----------|----------|
| WS-013 | 有效认证令牌 | 连接成功 | 1. 发送包含有效认证令牌的请求<br>2. 验证连接成功建立 |
| WS-014 | 无效认证令牌 | 连接被拒绝 | 1. 发送包含无效认证令牌的请求<br>2. 验证代理返回401错误 |
| WS-015 | 缺少认证令牌 | 连接被拒绝 | 1. 发送不包含认证令牌的请求<br>2. 验证代理返回401错误 |

#### 3.2.3 连接限制测试

| 测试用例ID | 测试描述 | 预期结果 | 测试步骤 |
|------------|----------|----------|----------|
| WS-016 | 全局连接限制 | 超过限制的连接被拒绝 | 1. 建立达到最大连接数的连接<br>2. 尝试建立新连接<br>3. 验证新连接被拒绝 |
| WS-017 | 单IP连接限制 | 超过限制的连接被拒绝 | 1. 从同一IP建立达到最大连接数的连接<br>2. 尝试建立新连接<br>3. 验证新连接被拒绝 |

### 3.3 性能测试

#### 3.3.1 并发连接测试

| 测试用例ID | 测试描述 | 预期结果 | 测试步骤 |
|------------|----------|----------|----------|
| WS-018 | 中等并发连接 | 所有连接正常建立 | 1. 并发建立1000个WebSocket连接<br>2. 验证所有连接成功建立<br>3. 验证消息正常传输 |
| WS-019 | 高并发连接 | 系统稳定运行 | 1. 并发建立10000个WebSocket连接<br>2. 监控系统资源使用<br>3. 验证系统稳定运行 |

#### 3.3.2 吞吐量测试

| 测试用例ID | 测试描述 | 预期结果 | 测试步骤 |
|------------|----------|----------|----------|
| WS-020 | 小消息吞吐量 | 达到预期吞吐量 | 1. 建立多个WebSocket连接<br>2. 以高频率发送小消息<br>3. 测量消息吞吐量 |
| WS-021 | 大消息吞吐量 | 达到预期吞吐量 | 1. 建立多个WebSocket连接<br>2. 发送大消息<br>3. 测量消息吞吐量 |

#### 3.3.3 延迟测试

| 测试用例ID | 测试描述 | 预期结果 | 测试步骤 |
|------------|----------|----------|----------|
| WS-022 | 端到端延迟 | 延迟在可接受范围内 | 1. 建立WebSocket连接<br>2. 测量消息往返时间<br>3. 验证延迟在可接受范围内 |
| WS-023 | 高负载下延迟 | 延迟保持在可接受范围内 | 1. 建立大量WebSocket连接<br>2. 在高负载下测量延迟<br>3. 验证延迟保持在可接受范围内 |

### 3.4 兼容性测试

#### 3.4.1 协议版本兼容性

| 测试用例ID | 测试描述 | 预期结果 | 测试步骤 |
|------------|----------|----------|----------|
| WS-024 | WebSocket RFC 6455兼容 | 完全兼容 | 1. 使用标准WebSocket客户端测试<br>2. 验证所有标准功能正常工作 |
| WS-025 | 旧版WebSocket协议 | 正确处理或拒绝 | 1. 使用旧版WebSocket协议测试<br>2. 验证代理正确处理或拒绝连接 |

#### 3.4.2 浏览器兼容性

| 测试用例ID | 测试描述 | 预期结果 | 测试步骤 |
|------------|----------|----------|----------|
| WS-026 | Chrome浏览器 | 正常工作 | 1. 使用Chrome浏览器连接WebSocket<br>2. 验证所有功能正常工作 |
| WS-027 | Firefox浏览器 | 正常工作 | 1. 使用Firefox浏览器连接WebSocket<br>2. 验证所有功能正常工作 |
| WS-028 | Safari浏览器 | 正常工作 | 1. 使用Safari浏览器连接WebSocket<br>2. 验证所有功能正常工作 |

## 4. 自动化测试

### 4.1 单元测试

```go
// 示例：WebSocket中间件单元测试
func TestWebSocketMiddleware(t *testing.T) {
    tests := []struct {
        name           string
        headers        map[string]string
        expectedStatus int
        expectedWS     bool
    }{
        {
            name: "Valid WebSocket upgrade",
            headers: map[string]string{
                "Upgrade":             "websocket",
                "Connection":          "Upgrade",
                "Sec-WebSocket-Key":   "dGhlIHNhbXBsZSBub25jZQ==",
                "Sec-WebSocket-Version": "13",
            },
            expectedStatus: 101,
            expectedWS:     true,
        },
        {
            name: "Invalid WebSocket upgrade",
            headers: map[string]string{
                "Upgrade": "websocket",
            },
            expectedStatus: 400,
            expectedWS:     false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 测试实现
        })
    }
}
```

### 4.2 集成测试

```go
// 示例：WebSocket代理集成测试
func TestWebSocketProxyIntegration(t *testing.T) {
    // 启动测试代理服务器
    proxy := setupTestProxy(t)
    defer proxy.Close()
    
    // 启动测试后端服务器
    backend := setupTestBackend(t)
    defer backend.Close()
    
    // 测试WebSocket连接
    conn, _, err := websocket.DefaultDialer.Dial(
        "ws://"+proxy.Addr()+"/ws", 
        nil,
    )
    require.NoError(t, err)
    defer conn.Close()
    
    // 测试消息传输
    testMessage := "Hello, WebSocket!"
    err = conn.WriteMessage(websocket.TextMessage, []byte(testMessage))
    require.NoError(t, err)
    
    _, message, err := conn.ReadMessage()
    require.NoError(t, err)
    assert.Equal(t, testMessage, string(message))
}
```

### 4.3 性能测试

```go
// 示例：WebSocket性能测试
func BenchmarkWebSocketProxy(b *testing.B) {
    proxy := setupTestProxy(b)
    defer proxy.Close()
    
    backend := setupTestBackend(b)
    defer backend.Close()
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            conn, _, err := websocket.DefaultDialer.Dial(
                "ws://"+proxy.Addr()+"/ws", 
                nil,
            )
            if err != nil {
                b.Fatal(err)
            }
            
            // 发送和接收消息
            err = conn.WriteMessage(websocket.TextMessage, []byte("benchmark"))
            if err != nil {
                b.Fatal(err)
            }
            
            _, _, err = conn.ReadMessage()
            if err != nil {
                b.Fatal(err)
            }
            
            conn.Close()
        }
    })
}
```

## 5. 测试执行计划

### 5.1 测试阶段

| 阶段 | 测试类型 | 预计持续时间 | 负责人 |
|------|----------|--------------|--------|
| 第一阶段 | 单元测试 | 2天 | 开发团队 |
| 第二阶段 | 功能测试 | 3天 | 测试团队 |
| 第三阶段 | 安全测试 | 2天 | 安全团队 |
| 第四阶段 | 性能测试 | 3天 | 性能团队 |
| 第五阶段 | 兼容性测试 | 2天 | 测试团队 |
| 第六阶段 | 端到端测试 | 2天 | 测试团队 |

### 5.2 测试环境准备

1. **硬件环境**:
   - 代理服务器: 4核CPU, 8GB内存
   - 后端服务器: 2核CPU, 4GB内存
   - 客户端机器: 2核CPU, 4GB内存

2. **软件环境**:
   - 操作系统: Linux (Ubuntu 20.04)
   - Go版本: 1.19+
   - Docker: 20.10+
   - 测试工具: 最新版本

3. **网络环境**:
   - 带宽: 1Gbps
   - 延迟: <1ms (局域网)
   - 丢包率: <0.1%

## 6. 测试报告

### 6.1 测试结果记录

每个测试用例的结果应记录以下信息:
- 测试用例ID
- 测试执行时间
- 测试结果 (通过/失败)
- 失败原因 (如果失败)
- 相关日志和截图
- 性能指标 (如果适用)

### 6.2 缺陷管理

所有发现的缺陷应记录以下信息:
- 缺陷ID
- 缺陷标题
- 缺陷描述
- 严重程度
- 优先级
- 复现步骤
- 预期结果
- 实际结果
- 相关测试用例
- 状态 (新建/处理中/已修复/已验证/已关闭)

### 6.3 测试总结报告

测试完成后应生成总结报告，包含:
- 测试概述
- 测试执行情况
- 测试结果统计
- 发现的缺陷列表
- 性能测试结果
- 风险评估
- 建议和结论

## 7. 回归测试

### 7.1 回归测试策略

1. **完全回归测试**:
   - 在重大版本更新前执行
   - 包含所有测试用例
   - 预计持续时间: 2周

2. **部分回归测试**:
   - 在小版本更新前执行
   - 包含核心功能和关键路径测试
   - 预计持续时间: 3天

3. **快速回归测试**:
   - 在紧急修复后执行
   - 包含与修复相关的测试用例
   - 预计持续时间: 4小时

### 7.2 自动化回归测试

建立持续集成流水线，包含以下自动化测试:
1. 单元测试 (每次提交)
2. 集成测试 (每天)
3. 性能基准测试 (每周)
4. 安全扫描 (每月)

## 8. 测试工具和脚本

### 8.1 WebSocket测试客户端

```javascript
// Node.js WebSocket测试客户端示例
const WebSocket = require('ws');

function testWebSocketConnection(url, messageCount) {
    return new Promise((resolve, reject) => {
        const ws = new WebSocket(url);
        let receivedCount = 0;
        const startTime = Date.now();
        
        ws.on('open', () => {
            console.log('WebSocket连接已建立');
            
            // 发送测试消息
            for (let i = 0; i < messageCount; i++) {
                ws.send(`测试消息 ${i}`);
            }
        });
        
        ws.on('message', (data) => {
            receivedCount++;
            console.log(`收到消息: ${data}`);
            
            if (receivedCount === messageCount) {
                const endTime = Date.now();
                const duration = endTime - startTime;
                console.log(`接收${messageCount}条消息耗时: ${duration}ms`);
                ws.close();
                resolve({ duration, messageCount });
            }
        });
        
        ws.on('error', (error) => {
            console.error('WebSocket错误:', error);
            reject(error);
        });
    });
}

// 使用示例
testWebSocketConnection('ws://localhost:8080/ws', 100)
    .then(result => {
        console.log('测试完成:', result);
    })
    .catch(error => {
        console.error('测试失败:', error);
    });
```

### 8.2 性能测试脚本

```python
# Python WebSocket性能测试示例
import asyncio
import websockets
import time
import statistics
from concurrent.futures import ThreadPoolExecutor

async def websocket_client(url, message_count, message_size, results):
    try:
        async with websockets.connect(url) as websocket:
            # 准备测试消息
            message = 'A' * message_size
            
            # 测量开始时间
            start_time = time.time()
            
            # 发送和接收消息
            for i in range(message_count):
                await websocket.send(message)
                response = await websocket.recv()
                
                # 验证响应
                if response != message:
                    raise ValueError(f"响应不匹配: 期望'{message}', 实际'{response}'")
            
            # 测量结束时间
            end_time = time.time()
            duration = end_time - start_time
            
            # 记录结果
            results.append({
                'duration': duration,
                'message_count': message_count,
                'message_size': message_size,
                'throughput': message_count / duration
            })
            
            return duration
    except Exception as e:
        print(f"客户端错误: {e}")
        return None

def run_performance_test(url, client_count, message_count, message_size):
    results = []
    
    # 创建线程池
    with ThreadPoolExecutor(max_workers=client_count) as executor:
        # 创建事件循环
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        
        # 提交任务
        futures = []
        for i in range(client_count):
            future = executor.submit(
                asyncio.run, 
                websocket_client(url, message_count, message_size, results)
            )
            futures.append(future)
        
        # 等待所有任务完成
        for future in futures:
            future.result()
    
    # 计算统计信息
    if results:
        durations = [r['duration'] for r in results]
        throughputs = [r['throughput'] for r in results]
        
        avg_duration = statistics.mean(durations)
        avg_throughput = statistics.mean(throughputs)
        total_messages = client_count * message_count
        total_duration = max(durations)
        
        print(f"性能测试结果:")
        print(f"  客户端数量: {client_count}")
        print(f"  每客户端消息数: {message_count}")
        print(f"  消息大小: {message_size}字节")
        print(f"  总消息数: {total_messages}")
        print(f"  平均持续时间: {avg_duration:.2f}秒")
        print(f"  平均吞吐量: {avg_throughput:.2f}消息/秒")
        print(f"  总吞吐量: {total_messages/total_duration:.2f}消息/秒")

# 使用示例
run_performance_test('ws://localhost:8080/ws', 10, 100, 1024)
```

这些测试计划和脚本将帮助确保WebSocket代理功能的正确性、性能和安全性，为生产环境部署提供可靠保障。