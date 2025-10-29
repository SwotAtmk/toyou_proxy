package proxy

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"toyou-proxy/config"
)

// HandleWebSocketUpgrade 处理WebSocket协议升级
func (ph *ProxyHandler) HandleWebSocketUpgrade(w http.ResponseWriter, r *http.Request, service *config.Service) error {
	// 检查是否是WebSocket升级请求
	if !isWebSocketUpgrade(r) {
		return fmt.Errorf("not a WebSocket upgrade request")
	}

	// 解析目标URL
	targetURL, err := url.Parse(service.URL)
	if err != nil {
		return fmt.Errorf("invalid target URL: %s", service.URL)
	}

	// 创建WebSocket代理
	wsProxy := NewWebSocketProxy()

	// 代理WebSocket连接
	return wsProxy.ProxyWebSocket(w, r, targetURL.String())
}

// isWebSocketUpgrade 检查是否是WebSocket升级请求
func isWebSocketUpgrade(r *http.Request) bool {
	// 检查Connection头
	connection := strings.ToLower(r.Header.Get("Connection"))
	if !strings.Contains(connection, "upgrade") {
		return false
	}

	// 检查Upgrade头
	upgrade := strings.ToLower(r.Header.Get("Upgrade"))
	if upgrade != "websocket" {
		return false
	}

	// 检查WebSocket版本
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		return false
	}

	// 检查WebSocket密钥
	if r.Header.Get("Sec-WebSocket-Key") == "" {
		return false
	}

	return true
}

// HijackConnection 劫持HTTP连接以进行协议升级
func HijackConnection(w http.ResponseWriter) (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}

	conn, buf, err := hijacker.Hijack()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack connection: %v", err)
	}

	return conn, buf, nil
}

// CreateWebSocketUpgradeRequest 创建WebSocket升级请求
func CreateWebSocketUpgradeRequest(r *http.Request, targetURL *url.URL) (*http.Request, error) {
	// 创建新的请求
	req, err := http.NewRequest("GET", targetURL.String(), nil)
	if err != nil {
		return nil, err
	}

	// 复制必要的头
	headersToCopy := []string{
		"Upgrade",
		"Connection",
		"Sec-WebSocket-Key",
		"Sec-WebSocket-Version",
		"Sec-WebSocket-Protocol",
		"Sec-WebSocket-Extensions",
		"Origin",
		"User-Agent",
		"Cookie",
		"Authorization",
	}

	for _, header := range headersToCopy {
		if value := r.Header.Get(header); value != "" {
			req.Header.Set(header, value)
		}
	}

	// 设置X-Forwarded头
	req.Header.Set("X-Forwarded-Proto", "http")
	req.Header.Set("X-Forwarded-Host", r.Host)
	req.Header.Set("X-Forwarded-For", r.RemoteAddr)

	return req, nil
}

// ConnectToTargetServer 连接到目标服务器
func ConnectToTargetServer(targetURL *url.URL, timeout time.Duration) (net.Conn, error) {
	// 确定地址
	addr := targetURL.Host
	if targetURL.Port() == "" {
		if targetURL.Scheme == "https" || targetURL.Scheme == "wss" {
			addr = targetURL.Host + ":443"
		} else {
			addr = targetURL.Host + ":80"
		}
	}

	// 创建连接
	var conn net.Conn
	var err error

	if targetURL.Scheme == "https" || targetURL.Scheme == "wss" {
		// TLS连接
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: timeout}, "tcp", addr, &tls.Config{
			InsecureSkipVerify: true, // 在生产环境中应该验证证书
		})
	} else {
		// 普通连接
		conn, err = net.DialTimeout("tcp", addr, timeout)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to target server: %v", err)
	}

	return conn, nil
}

// SendUpgradeRequest 发送升级请求
func SendUpgradeRequest(conn net.Conn, req *http.Request) (*http.Response, error) {
	// 发送请求
	err := req.Write(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to send upgrade request: %v", err)
	}

	// 读取响应
	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		return nil, fmt.Errorf("failed to read upgrade response: %v", err)
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusSwitchingProtocols {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp, nil
}

// SendUpgradeResponse 发送升级响应
func SendUpgradeResponse(w http.ResponseWriter, resp *http.Response) error {
	// 复制响应头
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// 设置状态码
	w.WriteHeader(resp.StatusCode)

	// 刷新响应
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

// BidirectionalCopy 双向复制数据
func BidirectionalCopy(clientConn, serverConn net.Conn) {
	// 设置错误通道
	errChan := make(chan error, 2)

	// 客户端到服务器的复制
	go func() {
		_, err := io.Copy(serverConn, clientConn)
		errChan <- err
	}()

	// 服务器到客户端的复制
	go func() {
		_, err := io.Copy(clientConn, serverConn)
		errChan <- err
	}()

	// 等待任一方向的复制完成
	err := <-errChan

	// 关闭连接
	clientConn.Close()
	serverConn.Close()

	// 记录错误（如果有）
	if err != nil && err != io.EOF {
		fmt.Printf("WebSocket proxy error: %v\n", err)
	}
}
