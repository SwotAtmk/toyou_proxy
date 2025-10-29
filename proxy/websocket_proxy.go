package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketProxy WebSocket代理处理器
type WebSocketProxy struct {
	// WebSocket升级器
	upgrader websocket.Upgrader

	// 连接管理
	connections map[string]*WebSocketConnection
	connMutex   sync.RWMutex

	// 配置参数
	handshakeTimeout time.Duration
	enablePing       bool
	pingInterval     time.Duration
}

// WebSocketConnection WebSocket连接信息
type WebSocketConnection struct {
	ID           string
	ClientConn   net.Conn
	ServerConn   net.Conn
	StartTime    time.Time
	BytesRead    int64
	BytesWritten int64
}

// NewWebSocketProxy 创建WebSocket代理
func NewWebSocketProxy() *WebSocketProxy {
	return &WebSocketProxy{
		upgrader: websocket.Upgrader{
			HandshakeTimeout: 10 * time.Second,
			ReadBufferSize:   1024,
			WriteBufferSize:  1024,
			CheckOrigin: func(r *http.Request) bool {
				// 允许所有来源，实际生产环境中应该更严格
				return true
			},
		},
		connections:      make(map[string]*WebSocketConnection),
		handshakeTimeout: 10 * time.Second,
		enablePing:       true,
		pingInterval:     30 * time.Second,
	}
}

// ProxyWebSocket 代理WebSocket请求
func (wp *WebSocketProxy) ProxyWebSocket(w http.ResponseWriter, r *http.Request, targetURL string) error {
	// 解析目标URL
	target, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid target URL: %v", err)
	}

	// 将http://或https://或ws://或wss://转换为正确的协议
	scheme := "ws"
	if target.Scheme == "https" || target.Scheme == "wss" {
		scheme = "wss"
	}

	// 构建WebSocket目标URL，使用原始请求的路径
	wsTarget := &url.URL{
		Scheme:   scheme,
		Host:     target.Host,
		Path:     r.URL.Path,     // 使用原始请求的路径
		RawQuery: r.URL.RawQuery, // 使用原始请求的查询参数
	}

	// 劫持客户端连接
	clientConn, _, err := HijackConnection(w)
	if err != nil {
		return fmt.Errorf("failed to hijack client connection: %v", err)
	}
	defer clientConn.Close()

	// 连接到目标WebSocket服务器
	serverConn, err := ConnectToTargetServer(wsTarget, wp.handshakeTimeout)
	if err != nil {
		return fmt.Errorf("failed to connect to target server: %v", err)
	}
	defer serverConn.Close()

	// 创建升级请求
	upgradeReq, err := CreateWebSocketUpgradeRequest(r, wsTarget)
	if err != nil {
		return fmt.Errorf("failed to create upgrade request: %v", err)
	}

	// 发送升级请求到目标服务器
	resp, err := SendUpgradeRequest(serverConn, upgradeReq)
	if err != nil {
		return fmt.Errorf("failed to send upgrade request: %v", err)
	}
	defer resp.Body.Close()

	// 将升级响应直接写入客户端连接
	err = resp.Write(clientConn)
	if err != nil {
		return fmt.Errorf("failed to send upgrade response to client: %v", err)
	}

	// 创建连接信息
	connID := generateConnectionID(r)
	conn := &WebSocketConnection{
		ID:         connID,
		ClientConn: clientConn,
		ServerConn: serverConn,
		StartTime:  time.Now(),
	}

	// 添加到连接管理器
	wp.connMutex.Lock()
	wp.connections[connID] = conn
	wp.connMutex.Unlock()

	// 确保清理连接
	defer func() {
		wp.connMutex.Lock()
		delete(wp.connections, connID)
		wp.connMutex.Unlock()
	}()

	// 启动双向数据转发
	wp.bidirectionalCopy(clientConn, serverConn)

	return nil
}

// bidirectionalCopy 双向复制数据，使用自定义的复制逻辑
func (wp *WebSocketProxy) bidirectionalCopy(clientConn, serverConn net.Conn) {
	// 设置错误通道
	errChan := make(chan error, 2)

	// 客户端到服务器的复制
	go func() {
		buf := make([]byte, 32*1024) // 32KB buffer
		for {
			n, err := clientConn.Read(buf)
			if err != nil {
				errChan <- err
				return
			}

			// 写入到服务器连接
			_, err = serverConn.Write(buf[:n])
			if err != nil {
				errChan <- err
				return
			}
		}
	}()

	// 服务器到客户端的复制
	go func() {
		buf := make([]byte, 32*1024) // 32KB buffer
		for {
			n, err := serverConn.Read(buf)
			if err != nil {
				errChan <- err
				return
			}

			// 写入到客户端连接
			_, err = clientConn.Write(buf[:n])
			if err != nil {
				errChan <- err
				return
			}
		}
	}()

	// 等待任一方向的复制完成
	err := <-errChan

	// 关闭连接
	clientConn.Close()
	serverConn.Close()

	// 记录错误（如果有）
	if err != nil {
		fmt.Printf("WebSocket proxy error: %v\n", err)
	}
}

// generateConnectionID 生成连接ID
func generateConnectionID(r *http.Request) string {
	return fmt.Sprintf("%s-%s-%d", r.RemoteAddr, r.Header.Get("Sec-WebSocket-Key"), time.Now().UnixNano())
}

// GetConnection 获取连接信息
func (wp *WebSocketProxy) GetConnection(id string) (*WebSocketConnection, bool) {
	wp.connMutex.RLock()
	defer wp.connMutex.RUnlock()

	conn, exists := wp.connections[id]
	return conn, exists
}

// GetAllConnections 获取所有连接信息
func (wp *WebSocketProxy) GetAllConnections() []*WebSocketConnection {
	wp.connMutex.RLock()
	defer wp.connMutex.RUnlock()

	connections := make([]*WebSocketConnection, 0, len(wp.connections))
	for _, conn := range wp.connections {
		connections = append(connections, conn)
	}

	return connections
}

// CloseConnection 关闭指定连接
func (wp *WebSocketProxy) CloseConnection(id string) error {
	wp.connMutex.Lock()
	defer wp.connMutex.Unlock()

	conn, exists := wp.connections[id]
	if !exists {
		return fmt.Errorf("connection not found")
	}

	// 关闭客户端连接
	if conn.ClientConn != nil {
		err := conn.ClientConn.Close()
		if err != nil {
			return fmt.Errorf("failed to close client connection: %v", err)
		}
	}

	// 关闭服务器连接
	if conn.ServerConn != nil {
		err := conn.ServerConn.Close()
		if err != nil {
			return fmt.Errorf("failed to close server connection: %v", err)
		}
	}

	// 从连接管理器中移除
	delete(wp.connections, id)

	return nil
}

// CloseAllConnections 关闭所有连接
func (wp *WebSocketProxy) CloseAllConnections() {
	wp.connMutex.Lock()
	defer wp.connMutex.Unlock()

	for id, conn := range wp.connections {
		if conn.ClientConn != nil {
			conn.ClientConn.Close()
		}
		if conn.ServerConn != nil {
			conn.ServerConn.Close()
		}
		delete(wp.connections, id)
	}
}

// IsWebSocketUpgrade 检查是否为WebSocket升级请求
func IsWebSocketUpgrade(r *http.Request) bool {
	return websocket.IsWebSocketUpgrade(r)
}
