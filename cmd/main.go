package main

import (
	"fmt"
	"log"
	"strings"

	"toyou-proxy/server"
)

func main() {
	// 简化启动：直接使用默认配置文件
	configPath := "config.yaml"

	// 创建并启动服务器
	srv, err := server.NewServer(configPath)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// 获取服务器状态以显示支持的域名和端口
	status := srv.GetStatus()

	// 构建支持的域名列表
	supportedDomains := make([]string, 0)
	if srv.GetConfig() != nil && srv.GetConfig().HostRules != nil {
		for _, rule := range srv.GetConfig().HostRules {
			supportedDomains = append(supportedDomains, rule.Pattern)
		}
	}

	log.Printf("Starting Toyou Proxy Server...")
	log.Printf("Configuration file: %s", configPath)
	log.Printf("Supported domains: %s", strings.Join(supportedDomains, ", "))

	if ports, ok := status["ports"].([]int); ok {
		portsStr := fmt.Sprintf("%v", ports)
		log.Printf("Listening on ports: %s", portsStr)
	}

	// 启动服务器
	if err := srv.Start(); err != nil {
		log.Fatalf("Server stopped with error: %v", err)
	}

	log.Println("Server stopped gracefully")
}
