package loadbalancer

import (
	"fmt"
)

// LoadBalancerFactory 负载均衡器工厂接口
type LoadBalancerFactory interface {
	// CreateLoadBalancer 创建负载均衡器
	CreateLoadBalancer(config LoadBalancerConfig) (LoadBalancer, error)

	// GetSupportedStrategies 获取支持的策略列表
	GetSupportedStrategies() []LoadBalancerStrategy
}

// DefaultLoadBalancerFactory 默认负载均衡器工厂
type DefaultLoadBalancerFactory struct{}

// NewDefaultLoadBalancerFactory 创建默认负载均衡器工厂
func NewDefaultLoadBalancerFactory() *DefaultLoadBalancerFactory {
	return &DefaultLoadBalancerFactory{}
}

// CreateLoadBalancer 创建负载均衡器
func (f *DefaultLoadBalancerFactory) CreateLoadBalancer(config LoadBalancerConfig) (LoadBalancer, error) {
	// 验证配置
	if err := f.validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid load balancer config: %w", err)
	}

	// 创建基础负载均衡器
	var lb LoadBalancer

	switch config.Strategy {
	case RoundRobin:
		lb = NewRoundRobinLoadBalancer(config)
	case WeightedRoundRobin:
		lb = NewWeightedRoundRobinLoadBalancer(config)
	case IPHash:
		lb = NewIPHashLoadBalancer(config)
	case LeastConnections:
		lb = NewLeastConnectionsLoadBalancer(config)
	case ResponseTime:
		lb = NewResponseTimeLoadBalancer(config)
	case Random:
		lb = NewRandomLoadBalancer(config)
	case WeightedRandom:
		lb = NewWeightedRandomLoadBalancer(config)
	default:
		return nil, fmt.Errorf("unsupported load balancer strategy: %s", config.Strategy)
	}

	// 如果配置了会话保持，则包装负载均衡器
	if config.SessionAffinity != nil && config.SessionAffinity.Enabled {
		lb = NewSessionAffinityLoadBalancer(lb, config)
	}

	return lb, nil
}

// GetSupportedStrategies 获取支持的策略列表
func (f *DefaultLoadBalancerFactory) GetSupportedStrategies() []LoadBalancerStrategy {
	return []LoadBalancerStrategy{
		RoundRobin,
		WeightedRoundRobin,
		IPHash,
		LeastConnections,
		ResponseTime,
		Random,
		WeightedRandom,
	}
}

// validateConfig 验证负载均衡器配置
func (f *DefaultLoadBalancerFactory) validateConfig(config LoadBalancerConfig) error {
	// 检查策略是否有效
	validStrategies := f.GetSupportedStrategies()
	valid := false
	for _, strategy := range validStrategies {
		if config.Strategy == strategy {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid strategy: %s", config.Strategy)
	}

	// 检查后端列表
	if len(config.Backends) == 0 {
		return fmt.Errorf("at least one backend is required")
	}

	// 检查每个后端
	for i, backend := range config.Backends {
		if backend.URL == "" {
			return fmt.Errorf("backend %d: URL is required", i)
		}

		// 对于加权策略，检查权重
		if config.Strategy == WeightedRoundRobin || config.Strategy == WeightedRandom {
			if backend.Weight <= 0 {
				// 默认权重为1
				config.Backends[i].Weight = 1
			}
		}
	}

	// 检查健康检查配置
	if config.HealthCheck.Enabled {
		if config.HealthCheck.Interval <= 0 {
			return fmt.Errorf("health check interval must be greater than 0")
		}

		if config.HealthCheck.Timeout <= 0 {
			return fmt.Errorf("health check timeout must be greater than 0")
		}
	}

	// 检查会话保持配置
	if config.SessionAffinity != nil && config.SessionAffinity.Enabled {
		if config.SessionAffinity.Timeout <= 0 {
			return fmt.Errorf("session affinity timeout must be greater than 0")
		}

		if config.SessionAffinity.CookieName == "" {
			return fmt.Errorf("session affinity cookie name is required")
		}
	}

	return nil
}

// 全局默认工厂实例
var defaultFactory = NewDefaultLoadBalancerFactory()

// CreateLoadBalancer 使用默认工厂创建负载均衡器
func CreateLoadBalancer(config LoadBalancerConfig) (LoadBalancer, error) {
	return defaultFactory.CreateLoadBalancer(config)
}

// GetSupportedStrategies 使用默认工厂获取支持的策略列表
func GetSupportedStrategies() []LoadBalancerStrategy {
	return defaultFactory.GetSupportedStrategies()
}
