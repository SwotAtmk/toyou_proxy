package loadbalancer

import (
	"time"

	"toyou-proxy/config"
)

// ConvertConfig 将配置结构转换为负载均衡器配置
func ConvertConfig(cfg *config.LoadBalancerConfig) LoadBalancerConfig {
	if cfg == nil {
		return LoadBalancerConfig{}
	}

	// 转换策略
	strategy := LoadBalancerStrategy(cfg.Strategy)

	// 转换后端服务器
	backends := make([]Backend, len(cfg.Backends))
	for i, backend := range cfg.Backends {
		backends[i] = Backend{
			URL:    backend.URL,
			Weight: backend.Weight,
			Active: true, // 默认为活跃状态
		}

		// 转换健康检查配置
		if backend.HealthCheck != nil {
			backends[i].HealthCheck = HealthCheckConfig{
				Enabled:  backend.HealthCheck.Enabled,
				Interval: backend.HealthCheck.Interval,
				Timeout:  backend.HealthCheck.Timeout,
				Path:     backend.HealthCheck.Path,
			}
		}
	}

	// 转换全局健康检查配置
	var healthCheck HealthCheckConfig
	if cfg.HealthCheck != nil {
		healthCheck = HealthCheckConfig{
			Enabled:  cfg.HealthCheck.Enabled,
			Interval: cfg.HealthCheck.Interval,
			Timeout:  cfg.HealthCheck.Timeout,
			Path:     cfg.HealthCheck.Path,
		}
	}

	// 转换会话保持配置
	var sessionAffinity *SessionAffinityConfig
	if cfg.SessionAffinity != nil {
		sessionAffinity = &SessionAffinityConfig{
			Enabled:    cfg.SessionAffinity.Enabled,
			Timeout:    cfg.SessionAffinity.Timeout,
			CookieName: cfg.SessionAffinity.CookieName,
		}
	}

	return LoadBalancerConfig{
		Strategy:        strategy,
		Backends:        backends,
		HealthCheck:     healthCheck,
		SessionAffinity: sessionAffinity,
	}
}

// ConvertServiceConfig 将服务配置转换为负载均衡器配置
func ConvertServiceConfig(service *config.Service) (LoadBalancerConfig, bool) {
	if service.LoadBalancer == nil {
		return LoadBalancerConfig{}, false
	}

	return ConvertConfig(service.LoadBalancer), true
}

// SetDefaultValues 设置默认值
func SetDefaultValues(cfg *LoadBalancerConfig) {
	if cfg.Strategy == "" {
		cfg.Strategy = RoundRobin
	}

	// 设置默认健康检查配置
	if !cfg.HealthCheck.Enabled {
		cfg.HealthCheck = HealthCheckConfig{
			Enabled:  false,
			Interval: 30 * time.Second,
			Timeout:  5 * time.Second,
			Path:     "/health",
		}
	}

	// 设置默认会话保持配置
	if cfg.SessionAffinity == nil {
		cfg.SessionAffinity = &SessionAffinityConfig{
			Enabled:    false,
			Timeout:    30 * time.Minute,
			CookieName: "LB_SESSION",
		}
	}

	// 设置后端服务器默认值
	for i := range cfg.Backends {
		if cfg.Backends[i].Weight <= 0 {
			cfg.Backends[i].Weight = 1
		}

		if !cfg.Backends[i].HealthCheck.Enabled {
			cfg.Backends[i].HealthCheck = HealthCheckConfig{
				Enabled:  false,
				Interval: 30 * time.Second,
				Timeout:  5 * time.Second,
				Path:     "/health",
			}
		}
	}
}
