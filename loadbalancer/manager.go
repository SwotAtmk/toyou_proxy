package loadbalancer

import (
	"fmt"
	"sync"
)

// LoadBalancerManager 负载均衡器管理器接口
type LoadBalancerManager interface {
	// CreateLoadBalancer 创建并注册负载均衡器
	CreateLoadBalancer(name string, config LoadBalancerConfig) error

	// GetLoadBalancer 获取负载均衡器
	GetLoadBalancer(name string) (LoadBalancer, error)

	// UpdateLoadBalancer 更新负载均衡器配置
	UpdateLoadBalancer(name string, config LoadBalancerConfig) error

	// DeleteLoadBalancer 删除负载均衡器
	DeleteLoadBalancer(name string) error

	// ListLoadBalancers 列出所有负载均衡器名称
	ListLoadBalancers() []string

	// StartAll 启动所有负载均衡器的健康检查
	StartAll()

	// StopAll 停止所有负载均衡器的健康检查
	StopAll()
}

// DefaultLoadBalancerManager 默认负载均衡器管理器实现
type DefaultLoadBalancerManager struct {
	loadBalancers map[string]LoadBalancer
	factory       LoadBalancerFactory
	mu            sync.RWMutex
}

// NewDefaultLoadBalancerManager 创建默认负载均衡器管理器
func NewDefaultLoadBalancerManager() *DefaultLoadBalancerManager {
	return &DefaultLoadBalancerManager{
		loadBalancers: make(map[string]LoadBalancer),
		factory:       NewDefaultLoadBalancerFactory(),
	}
}

// NewLoadBalancerManagerWithFactory 使用指定工厂创建负载均衡器管理器
func NewLoadBalancerManagerWithFactory(factory LoadBalancerFactory) *DefaultLoadBalancerManager {
	return &DefaultLoadBalancerManager{
		loadBalancers: make(map[string]LoadBalancer),
		factory:       factory,
	}
}

// CreateLoadBalancer 创建并注册负载均衡器
func (m *DefaultLoadBalancerManager) CreateLoadBalancer(name string, config LoadBalancerConfig) error {
	if name == "" {
		return fmt.Errorf("load balancer name cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已存在
	if _, exists := m.loadBalancers[name]; exists {
		return fmt.Errorf("load balancer with name '%s' already exists", name)
	}

	// 创建负载均衡器
	lb, err := m.factory.CreateLoadBalancer(config)
	if err != nil {
		return fmt.Errorf("failed to create load balancer '%s': %w", name, err)
	}

	// 注册负载均衡器
	m.loadBalancers[name] = lb

	// 启动健康检查
	lb.StartHealthCheck()

	return nil
}

// GetLoadBalancer 获取负载均衡器
func (m *DefaultLoadBalancerManager) GetLoadBalancer(name string) (LoadBalancer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	lb, exists := m.loadBalancers[name]
	if !exists {
		return nil, fmt.Errorf("load balancer with name '%s' not found", name)
	}

	return lb, nil
}

// UpdateLoadBalancer 更新负载均衡器配置
func (m *DefaultLoadBalancerManager) UpdateLoadBalancer(name string, config LoadBalancerConfig) error {
	if name == "" {
		return fmt.Errorf("load balancer name cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否存在
	oldLb, exists := m.loadBalancers[name]
	if !exists {
		return fmt.Errorf("load balancer with name '%s' not found", name)
	}

	// 停止旧负载均衡器的健康检查
	oldLb.StopHealthCheck()

	// 创建新负载均衡器
	newLb, err := m.factory.CreateLoadBalancer(config)
	if err != nil {
		// 如果创建失败，重新启动旧负载均衡器的健康检查
		oldLb.StartHealthCheck()
		return fmt.Errorf("failed to create load balancer '%s': %w", name, err)
	}

	// 替换负载均衡器
	m.loadBalancers[name] = newLb

	return nil
}

// DeleteLoadBalancer 删除负载均衡器
func (m *DefaultLoadBalancerManager) DeleteLoadBalancer(name string) error {
	if name == "" {
		return fmt.Errorf("load balancer name cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否存在
	lb, exists := m.loadBalancers[name]
	if !exists {
		return fmt.Errorf("load balancer with name '%s' not found", name)
	}

	// 停止健康检查
	lb.StopHealthCheck()

	// 删除负载均衡器
	delete(m.loadBalancers, name)

	return nil
}

// ListLoadBalancers 列出所有负载均衡器名称
func (m *DefaultLoadBalancerManager) ListLoadBalancers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.loadBalancers))
	for name := range m.loadBalancers {
		names = append(names, name)
	}

	return names
}

// StartAll 启动所有负载均衡器的健康检查
func (m *DefaultLoadBalancerManager) StartAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, lb := range m.loadBalancers {
		lb.StartHealthCheck()
	}
}

// StopAll 停止所有负载均衡器的健康检查
func (m *DefaultLoadBalancerManager) StopAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, lb := range m.loadBalancers {
		lb.StopHealthCheck()
	}
}

// 全局默认管理器实例
var defaultManager = NewDefaultLoadBalancerManager()

// CreateLoadBalancerWithManager 使用默认管理器创建负载均衡器
func CreateLoadBalancerWithManager(name string, config LoadBalancerConfig) error {
	return defaultManager.CreateLoadBalancer(name, config)
}

// GetLoadBalancer 使用默认管理器获取负载均衡器
func GetLoadBalancer(name string) (LoadBalancer, error) {
	return defaultManager.GetLoadBalancer(name)
}

// UpdateLoadBalancer 使用默认管理器更新负载均衡器
func UpdateLoadBalancer(name string, config LoadBalancerConfig) error {
	return defaultManager.UpdateLoadBalancer(name, config)
}

// DeleteLoadBalancer 使用默认管理器删除负载均衡器
func DeleteLoadBalancer(name string) error {
	return defaultManager.DeleteLoadBalancer(name)
}

// ListLoadBalancers 使用默认管理器列出所有负载均衡器名称
func ListLoadBalancers() []string {
	return defaultManager.ListLoadBalancers()
}

// StartAll 使用默认管理器启动所有负载均衡器的健康检查
func StartAll() {
	defaultManager.StartAll()
}

// StopAll 使用默认管理器停止所有负载均衡器的健康检查
func StopAll() {
	defaultManager.StopAll()
}

// GetDefaultManager 获取默认负载均衡器管理器实例
func GetDefaultManager() LoadBalancerManager {
	return defaultManager
}
