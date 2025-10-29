package main

import (
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	// 默认连接地址
	defaultURL := "ws://localhost:8080/ws" // 假设代理服务器在8080端口
	if len(os.Args) > 1 {
		defaultURL = os.Args[1]
	}

	// 解析WebSocket URL
	u, err := url.Parse(defaultURL)
	if err != nil {
		log.Fatal("Failed to parse URL:", err)
	}

	// 连接WebSocket服务器
	log.Printf("Connecting to WebSocket server at %s", u.String())

	c, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("Failed to connect to WebSocket server: %v\nResponse: %+v", err, resp)
	}
	defer c.Close()

	log.Printf("Connected to WebSocket server")

	// 设置读取超时
	c.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.SetPongHandler(func(string) error {
		c.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 设置定时发送ping
	go func() {
		ticker := time.NewTicker(54 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := c.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("Failed to send ping: %v", err)
					return
				}
			}
		}
	}()

	// 读取消息的goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			messageType, message, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				} else {
					log.Printf("WebSocket connection closed: %v", err)
				}
				return
			}

			log.Printf("Received message (%d): %s", messageType, string(message))
		}
	}()

	// 发送测试消息
	testMessages := []string{
		"Hello, WebSocket Proxy!",
		"This is a test message",
		"WebSocket proxy is working",
		"Final test message",
	}

	for i, msg := range testMessages {
		log.Printf("Sending message %d: %s", i+1, msg)
		err := c.WriteMessage(websocket.TextMessage, []byte(msg))
		if err != nil {
			log.Printf("Failed to send message: %v", err)
			return
		}
		time.Sleep(2 * time.Second)
	}

	// 等待中断信号
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	select {
	case <-done:
		return
	case <-interrupt:
		log.Println("Interrupt received, closing connection")

		// 发送关闭消息
		err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Printf("Failed to send close message: %v", err)
			return
		}

		// 等待服务器关闭连接
		select {
		case <-done:
		case <-time.After(time.Second):
		}
		return
	}
}
