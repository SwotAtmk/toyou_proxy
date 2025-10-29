package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"toyou-proxy/server"
)

func main() {
	// 解析命令行参数
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("Configuration file not found: %s", configPath)
	}

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
