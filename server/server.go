package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"toyou-proxy/config"
	"toyou-proxy/proxy"
)

// Server 代理服务器
type Server struct {
	config    *config.Config
	servers   []*http.Server
	portMap   map[int]*proxy.ProxyHandler // 端口到处理器的映射
	stopChan  chan struct{}
	waitGroup sync.WaitGroup
}

// NewServer 创建新的代理服务器
func NewServer(configPath string) (*Server, error) {
	// 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}

	// 扫描host_rules获取所有需要监听的端口
	portHandlers := make(map[int]*proxy.ProxyHandler)

	for _, hostRule := range cfg.HostRules {
		port := hostRule.Port
		if port == 0 {
			port = 80 // 默认端口
		}

		// 如果该端口还没有处理器，创建一个
		if _, exists := portHandlers[port]; !exists {
			handler, err := proxy.NewProxyHandler(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create proxy handler for port %d: %v", port, err)
			}
			portHandlers[port] = handler
		}
	}

	// 如果没有配置任何host_rules，使用默认端口
	if len(portHandlers) == 0 {
		port := 80
		handler, err := proxy.NewProxyHandler(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create proxy handler for default port %d: %v", port, err)
		}
		portHandlers[port] = handler
	}

	return &Server{
		config:   cfg,
		portMap:  portHandlers,
		stopChan: make(chan struct{}),
	}, nil
}

// Start 启动服务器
func (s *Server) Start() error {
	// 记录配置信息
	log.Printf("Starting Toyou Proxy Server...")
	log.Printf("Configuration file: config.yaml")

	// 获取所有监听的端口
	ports := make([]int, 0, len(s.portMap))
	for port := range s.portMap {
		ports = append(ports, port)
	}

	log.Printf("Listening on ports: %v", ports)
	log.Printf("Loaded %d host rules", len(s.config.HostRules))
	log.Printf("Loaded %d route rules", len(s.config.RouteRules))
	log.Printf("Loaded %d services", len(s.config.Services))
	log.Printf("Loaded %d middlewares", len(s.config.Middlewares))

	// 为每个端口创建HTTP服务器
	s.servers = make([]*http.Server, 0, len(s.portMap))

	for port, handler := range s.portMap {
		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: handler,
		}
		s.servers = append(s.servers, server)

		// 启动服务器
		s.waitGroup.Add(1)
		go func(port int, server *http.Server) {
			defer s.waitGroup.Done()

			log.Printf("Starting proxy server on port %d", port)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("Server on port %d failed: %v", port, err)
			}
		}(port, server)
	}

	// 设置信号处理
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号或停止信号
	select {
	case sig := <-signalChan:
		log.Printf("Received signal: %v", sig)
		return s.Stop()
	case <-s.stopChan:
		log.Printf("Received stop signal")
		return s.Stop()
	}
}

// Stop 停止服务器
func (s *Server) Stop() error {
	log.Println("Shutting down servers...")

	// 关闭所有服务器
	for _, server := range s.servers {
		if err := server.Close(); err != nil {
			log.Printf("Error closing server on port: %v", err)
		}
	}

	// 等待所有服务器关闭
	s.waitGroup.Wait()
	log.Println("All servers stopped")

	return nil
}

// GetConfig 获取服务器配置
func (s *Server) GetConfig() *config.Config {
	return s.config
}

// GetStatus 获取服务器状态
func (s *Server) GetStatus() map[string]interface{} {
	// 获取所有监听的端口
	ports := make([]int, 0, len(s.portMap))
	for port := range s.portMap {
		ports = append(ports, port)
	}

	return map[string]interface{}{
		"ports":       ports,
		"host_rules":  len(s.config.HostRules),
		"route_rules": len(s.config.RouteRules),
		"services":    len(s.config.Services),
		"middlewares": len(s.config.Middlewares),
		"running":     true,
	}
}
