package loadbalancer

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// LoadBalancerStrategy 负载均衡策略类型
type LoadBalancerStrategy string

const (
	// RoundRobin 轮询策略
	RoundRobin LoadBalancerStrategy = "round_robin"
	// WeightedRoundRobin 加权轮询策略
	WeightedRoundRobin LoadBalancerStrategy = "weighted_round_robin"
	// IPHash IP哈希策略
	IPHash LoadBalancerStrategy = "ip_hash"
	// LeastConnections 最少连接策略
	LeastConnections LoadBalancerStrategy = "least_connections"
	// ResponseTime 最短响应时间策略
	ResponseTime LoadBalancerStrategy = "response_time"
	// Random 随机策略
	Random LoadBalancerStrategy = "random"
	// WeightedRandom 加权随机策略
	WeightedRandom LoadBalancerStrategy = "weighted_random"
)

// Backend 后端服务器信息
type Backend struct {
	URL          string            `yaml:"url"`          // 后端服务器URL
	Weight       int               `yaml:"weight"`       // 权重（用于加权策略）
	Active       bool              `yaml:"active"`       // 是否活跃
	Connections  int               `yaml:"-"`            // 当前连接数（内部使用）
	ResponseTime time.Duration     `yaml:"-"`            // 平均响应时间（内部使用）
	HealthCheck  HealthCheckConfig `yaml:"health_check"` // 健康检查配置
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
	Path     string        `yaml:"path"`
}

// LoadBalancerConfig 负载均衡器配置
type LoadBalancerConfig struct {
	Strategy        LoadBalancerStrategy   `yaml:"strategy"`         // 负载均衡策略
	Backends        []Backend              `yaml:"backends"`         // 后端服务器列表
	HealthCheck     HealthCheckConfig      `yaml:"health_check"`     // 全局健康检查配置
	SessionAffinity *SessionAffinityConfig `yaml:"session_affinity"` // 会话保持配置
}

// SessionAffinityConfig 会话保持配置
type SessionAffinityConfig struct {
	Enabled    bool          `yaml:"enabled"`
	Timeout    time.Duration `yaml:"timeout"`
	CookieName string        `yaml:"cookie_name"`
}

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	// NextBackend 选择下一个后端服务器
	NextBackend(req *http.Request) (*Backend, error)

	// UpdateBackendStatus 更新后端服务器状态
	UpdateBackendStatus(url string, active bool)

	// IncrementConnection 增加后端服务器连接数
	IncrementConnection(url string)

	// DecrementConnection 减少后端服务器连接数
	DecrementConnection(url string)

	// UpdateResponseTime 更新后端服务器响应时间
	UpdateResponseTime(url string, responseTime time.Duration)

	// GetBackends 获取所有后端服务器信息
	GetBackends() []Backend

	// GetActiveBackends 获取活跃的后端服务器信息
	GetActiveBackends() []*Backend

	// StartHealthCheck 启动健康检查
	StartHealthCheck()

	// StopHealthCheck 停止健康检查
	StopHealthCheck()
}

// NewLoadBalancer 创建负载均衡器
func NewLoadBalancer(config LoadBalancerConfig) (LoadBalancer, error) {
	switch config.Strategy {
	case RoundRobin:
		return NewRoundRobinLoadBalancer(config), nil
	case WeightedRoundRobin:
		return NewWeightedRoundRobinLoadBalancer(config), nil
	case IPHash:
		return NewIPHashLoadBalancer(config), nil
	case LeastConnections:
		return NewLeastConnectionsLoadBalancer(config), nil
	case ResponseTime:
		return NewResponseTimeLoadBalancer(config), nil
	case Random:
		return NewRandomLoadBalancer(config), nil
	case WeightedRandom:
		return NewWeightedRandomLoadBalancer(config), nil
	default:
		return nil, fmt.Errorf("unsupported load balancer strategy: %s", config.Strategy)
	}
}

// BaseLoadBalancer 基础负载均衡器，包含公共功能
type BaseLoadBalancer struct {
	config      LoadBalancerConfig
	backends    []*Backend
	mu          sync.RWMutex
	healthCheck *HealthChecker
}

// NewBaseLoadBalancer 创建基础负载均衡器
func NewBaseLoadBalancer(config LoadBalancerConfig) *BaseLoadBalancer {
	// 创建后端服务器指针切片
	backends := make([]*Backend, len(config.Backends))
	for i := range config.Backends {
		backends[i] = &config.Backends[i]
	}

	return &BaseLoadBalancer{
		config:   config,
		backends: backends,
	}
}

// UpdateBackendStatus 更新后端服务器状态
func (lb *BaseLoadBalancer) UpdateBackendStatus(url string, active bool) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for _, backend := range lb.backends {
		if backend.URL == url {
			backend.Active = active
			break
		}
	}
}

// IncrementConnection 增加后端服务器连接数
func (lb *BaseLoadBalancer) IncrementConnection(url string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for _, backend := range lb.backends {
		if backend.URL == url {
			backend.Connections++
			break
		}
	}
}

// DecrementConnection 减少后端服务器连接数
func (lb *BaseLoadBalancer) DecrementConnection(url string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for _, backend := range lb.backends {
		if backend.URL == url {
			if backend.Connections > 0 {
				backend.Connections--
			}
			break
		}
	}
}

// UpdateResponseTime 更新后端服务器响应时间
func (lb *BaseLoadBalancer) UpdateResponseTime(url string, responseTime time.Duration) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for _, backend := range lb.backends {
		if backend.URL == url {
			// 使用指数移动平均计算响应时间
			if backend.ResponseTime == 0 {
				backend.ResponseTime = responseTime
			} else {
				// 使用0.7的平滑因子
				backend.ResponseTime = time.Duration(float64(backend.ResponseTime)*0.7 + float64(responseTime)*0.3)
			}
			break
		}
	}
}

// GetBackends 获取所有后端服务器信息
func (lb *BaseLoadBalancer) GetBackends() []Backend {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	result := make([]Backend, len(lb.backends))
	for i, backend := range lb.backends {
		result[i] = *backend
	}
	return result
}

// StartHealthCheck 启动健康检查
func (lb *BaseLoadBalancer) StartHealthCheck() {
	if lb.healthCheck == nil {
		lb.healthCheck = NewHealthChecker(lb)
	}
	lb.healthCheck.Start()
}

// StopHealthCheck 停止健康检查
func (lb *BaseLoadBalancer) StopHealthCheck() {
	if lb.healthCheck != nil {
		lb.healthCheck.Stop()
	}
}

// GetActiveBackends 获取活跃的后端服务器
func (lb *BaseLoadBalancer) GetActiveBackends() []*Backend {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	var activeBackends []*Backend
	for _, backend := range lb.backends {
		if backend.Active {
			activeBackends = append(activeBackends, backend)
		}
	}
	return activeBackends
}

// HealthChecker 健康检查器
type HealthChecker struct {
	loadBalancer *BaseLoadBalancer
	stopCh       chan struct{}
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(loadBalancer *BaseLoadBalancer) *HealthChecker {
	return &HealthChecker{
		loadBalancer: loadBalancer,
		stopCh:       make(chan struct{}),
	}
}

// Start 启动健康检查
func (hc *HealthChecker) Start() {
	// 如果没有配置健康检查，则不启动
	if !hc.loadBalancer.config.HealthCheck.Enabled {
		return
	}

	// 初始化所有后端服务器状态为活跃
	for _, backend := range hc.loadBalancer.backends {
		backend.Active = true
	}

	go func() {
		ticker := time.NewTicker(hc.loadBalancer.config.HealthCheck.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				hc.checkAllBackends()
			case <-hc.stopCh:
				return
			}
		}
	}()
}

// Stop 停止健康检查
func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
}

// checkAllBackends 检查所有后端服务器健康状态
func (hc *HealthChecker) checkAllBackends() {
	for _, backend := range hc.loadBalancer.backends {
		go hc.checkBackend(backend)
	}
}

// checkBackend 检查单个后端服务器健康状态
func (hc *HealthChecker) checkBackend(backend *Backend) {
	// 使用后端自己的健康检查配置，如果没有则使用全局配置
	config := backend.HealthCheck
	if !config.Enabled {
		config = hc.loadBalancer.config.HealthCheck
		if !config.Enabled {
			// 如果都没有启用健康检查，则认为始终健康
			backend.Active = true
			return
		}
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: config.Timeout,
	}

	// 创建健康检查请求
	url := backend.URL
	if config.Path != "" {
		url = backend.URL + config.Path
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		backend.Active = false
		return
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		backend.Active = false
		return
	}
	defer resp.Body.Close()

	// 检查响应状态码
	backend.Active = resp.StatusCode >= 200 && resp.StatusCode < 300
}
