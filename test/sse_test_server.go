package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func sseHandler(w http.ResponseWriter, r *http.Request) {
	// 设置SSE响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 创建一个定时器，每秒发送一次事件
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	counter := 0
	for {
		select {
		case <-ticker.C:
			counter++
			// 发送SSE事件
			fmt.Fprintf(w, "data: Message %d from server at %s\n\n",
				counter, time.Now().Format(time.RFC3339))

			// 刷新缓冲区
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

			// 发送10条消息后关闭连接
			if counter >= 10 {
				fmt.Fprintf(w, "event: close\ndata: Stream ended\n\n")
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
				return
			}
		case <-r.Context().Done():
			// 客户端断开连接
			log.Println("Client disconnected")
			return
		}
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	html := `<!DOCTYPE html>
<html>
<head>
    <title>SSE Test Server</title>
</head>
<body>
    <h1>SSE测试服务器</h1>
    <p>这是一个简单的SSE测试服务器。</p>
    <ul>
        <li><a href="/events">SSE事件流</a></li>
        <li><a href="/health">健康检查</a></li>
    </ul>
</body>
</html>`
	fmt.Fprint(w, html)
}

func main() {
	// 注册多个SSE端点
	http.HandleFunc("/events", sseHandler)
	http.HandleFunc("/stream", sseHandler)
	http.HandleFunc("/sse", sseHandler)
	http.HandleFunc("/eventsource", sseHandler)
	http.HandleFunc("/api/events", sseHandler)
	http.HandleFunc("/api/stream", sseHandler)
	http.HandleFunc("/api/sse", sseHandler)

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/", rootHandler)

	log.Println("SSE Test Server starting on port 3000...")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
