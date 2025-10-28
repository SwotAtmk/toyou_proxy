package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// APIResponse API响应结构
type APIResponse struct {
	Data struct {
		GotoServices string `json:"goto_services"`
	} `json:"data"`
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func main() {
	// 设置路由处理器
	http.HandleFunc("/api/host", handleHostAPI)

	// 启动服务器
	port := "7080"
	log.Printf("Mock API server starting on port %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// handleHostAPI 处理主机API请求
func handleHostAPI(w http.ResponseWriter, r *http.Request) {
	// 只处理POST请求
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var request struct {
		Host string `json:"host"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 根据主机名返回不同的目标服务
	var gotoService string
	if strings.Contains(request.Host, "hivision") {
		gotoService = "local-8000" // 将hivision请求重定向到本地8000端口
	} else {
		gotoService = "hivision-service" // 其他请求保持原样
	}

	// 构造响应
	response := APIResponse{
		Data: struct {
			GotoServices string `json:"goto_services"`
		}{
			GotoServices: gotoService,
		},
		Code: 200,
		Msg:  "success",
	}

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")

	// 返回响应
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	log.Printf("Host API: %s -> %s", request.Host, gotoService)
}
