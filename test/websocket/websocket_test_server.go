package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源的连接
	},
}

// handleWebSocket 处理WebSocket连接
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升级HTTP连接为WebSocket连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("WebSocket connection established from %s", conn.RemoteAddr())

	// 发送欢迎消息
	welcomeMsg := fmt.Sprintf("Welcome to WebSocket Test Server! Time: %s", time.Now().Format(time.RFC3339))
	err = conn.WriteMessage(websocket.TextMessage, []byte(welcomeMsg))
	if err != nil {
		log.Printf("Failed to send welcome message: %v", err)
		return
	}

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 定期发送心跳消息
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				heartbeatMsg := fmt.Sprintf("Heartbeat from server at %s", time.Now().Format(time.RFC3339))
				if err := conn.WriteMessage(websocket.TextMessage, []byte(heartbeatMsg)); err != nil {
					log.Printf("Failed to send heartbeat: %v", err)
					return
				}
			}
		}
	}()

	// 读取客户端消息
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			} else {
				log.Printf("WebSocket connection closed: %v", err)
			}
			break
		}

		log.Printf("Received message: %s", string(message))

		// 回显消息
		echoMsg := fmt.Sprintf("Echo: %s (received at %s)", string(message), time.Now().Format(time.RFC3339))
		if err := conn.WriteMessage(messageType, []byte(echoMsg)); err != nil {
			log.Printf("Failed to send echo message: %v", err)
			break
		}
	}

	log.Printf("WebSocket connection closed")
}

// handleHTTP 处理普通HTTP请求
func handleHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "This is a regular HTTP page. Time: %s", time.Now().Format(time.RFC3339))
}

func main() {
	// 设置路由
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/", handleHTTP)

	// 启动服务器
	port := ":8081"
	log.Printf("WebSocket Test Server starting on %s", port)
	log.Printf("WebSocket endpoint: ws://localhost%s/ws", port)
	log.Printf("HTTP endpoint: http://localhost%s/", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}
