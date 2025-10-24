package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// 创建测试后端服务
func createTestServer(port int, name string) {
	handler := http.NewServeMux()
	
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := fmt.Sprintf(`{
			"service": "%s",
			"path": "%s",
			"host": "%s",
			"timestamp": "%s"
		}`, name, r.URL.Path, r.Host, time.Now().Format(time.RFC3339))
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	})
	
	handler.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
	}
	
	log.Printf("Test server %s started on port %d", name, port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Test server %s failed: %v", name, err)
	}
}

func main() {
	// 启动多个测试服务
	go createTestServer(8081, "a-service")
	go createTestServer(8082, "b-service") 
	go createTestServer(8083, "c-service")
	go createTestServer(8080, "health-check")
	
	log.Println("All test servers started")
	log.Println("Test URLs:")
	log.Println("  A Service: http://localhost:8081")
	log.Println("  B Service: http://localhost:8082")
	log.Println("  C Service: http://localhost:8083")
	log.Println("  Health:    http://localhost:8080/health")
	
	// 保持主程序运行
	select {}
}