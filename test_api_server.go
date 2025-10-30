package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	// 简单的测试API服务器
	http.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		// 模拟一些处理时间
		time.Sleep(100 * time.Millisecond)

		// 设置响应头
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "max-age=300")

		// 返回响应
		fmt.Fprintf(w, `{
			"message": "Test API response",
			"timestamp": "%s",
			"method": "%s",
			"path": "%s",
			"headers": {
				"X-Custom-Header": "%s"
			}
		}`, time.Now().Format(time.RFC3339), r.Method, r.URL.Path, r.Header.Get("X-Custom-Header"))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Default service response at %s", time.Now().Format(time.RFC3339))
	})

	fmt.Println("Test API server starting on port 3000...")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
