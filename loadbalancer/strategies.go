package loadbalancer

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"
)

// RoundRobinLoadBalancer 轮询负载均衡器
type RoundRobinLoadBalancer struct {
	*BaseLoadBalancer
	current int
	mu      sync.Mutex
}

// NewRoundRobinLoadBalancer 创建轮询负载均衡器
func NewRoundRobinLoadBalancer(config LoadBalancerConfig) *RoundRobinLoadBalancer {
	return &RoundRobinLoadBalancer{
		BaseLoadBalancer: NewBaseLoadBalancer(config),
		current:          0,
	}
}

// NextBackend 选择下一个后端服务器
func (lb *RoundRobinLoadBalancer) NextBackend(req *http.Request) (*Backend, error) {
	activeBackends := lb.GetActiveBackends()
	if len(activeBackends) == 0 {
		return nil, errors.New("no active backends available")
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	// 轮询选择
	backend := activeBackends[lb.current%len(activeBackends)]
	lb.current++

	return backend, nil
}

// WeightedRoundRobinLoadBalancer 加权轮询负载均衡器
type WeightedRoundRobinLoadBalancer struct {
	*BaseLoadBalancer
	current int
	weight  int
	mu      sync.Mutex
}

// NewWeightedRoundRobinLoadBalancer 创建加权轮询负载均衡器
func NewWeightedRoundRobinLoadBalancer(config LoadBalancerConfig) *WeightedRoundRobinLoadBalancer {
	return &WeightedRoundRobinLoadBalancer{
		BaseLoadBalancer: NewBaseLoadBalancer(config),
		current:          0,
		weight:           0,
	}
}

// NextBackend 选择下一个后端服务器
func (lb *WeightedRoundRobinLoadBalancer) NextBackend(req *http.Request) (*Backend, error) {
	activeBackends := lb.GetActiveBackends()
	if len(activeBackends) == 0 {
		return nil, errors.New("no active backends available")
	}

	// 计算总权重
	totalWeight := 0
	for _, backend := range activeBackends {
		if backend.Weight <= 0 {
			// 默认权重为1
			totalWeight++
		} else {
			totalWeight += backend.Weight
		}
	}

	if totalWeight == 0 {
		return nil, errors.New("invalid weights for backends")
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	// 加权轮询选择
	targetWeight := lb.weight % totalWeight
	lb.weight++

	currentWeight := 0
	for _, backend := range activeBackends {
		weight := backend.Weight
		if weight <= 0 {
			weight = 1
		}

		currentWeight += weight
		if targetWeight < currentWeight {
			return backend, nil
		}
	}

	// 如果没有找到，返回第一个
	return activeBackends[0], nil
}

// IPHashLoadBalancer IP哈希负载均衡器
type IPHashLoadBalancer struct {
	*BaseLoadBalancer
}

// NewIPHashLoadBalancer 创建IP哈希负载均衡器
func NewIPHashLoadBalancer(config LoadBalancerConfig) *IPHashLoadBalancer {
	return &IPHashLoadBalancer{
		BaseLoadBalancer: NewBaseLoadBalancer(config),
	}
}

// NextBackend 选择下一个后端服务器
func (lb *IPHashLoadBalancer) NextBackend(req *http.Request) (*Backend, error) {
	activeBackends := lb.GetActiveBackends()
	if len(activeBackends) == 0 {
		return nil, errors.New("no active backends available")
	}

	// 获取客户端IP
	clientIP := getClientIP(req)

	// 计算哈希值
	hash := sha256.Sum256([]byte(clientIP))
	hashValue := binary.BigEndian.Uint32(hash[:4])

	// 选择后端
	index := int(hashValue % uint32(len(activeBackends)))
	return activeBackends[index], nil
}

// getClientIP 获取客户端IP地址
func getClientIP(req *http.Request) string {
	// 尝试从X-Forwarded-For头获取
	xForwardedFor := req.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		// X-Forwarded-For可能包含多个IP，取第一个
		if idx := len(xForwardedFor); idx > 0 {
			if commaIdx := 0; commaIdx < idx {
				for i, c := range xForwardedFor {
					if c == ',' {
						commaIdx = i
						break
					}
				}
				if commaIdx > 0 {
					return xForwardedFor[:commaIdx]
				}
			}
			return xForwardedFor
		}
	}

	// 尝试从X-Real-IP头获取
	xRealIP := req.Header.Get("X-Real-IP")
	if xRealIP != "" {
		return xRealIP
	}

	// 从RemoteAddr获取
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return ip
}

// LeastConnectionsLoadBalancer 最少连接负载均衡器
type LeastConnectionsLoadBalancer struct {
	*BaseLoadBalancer
}

// NewLeastConnectionsLoadBalancer 创建最少连接负载均衡器
func NewLeastConnectionsLoadBalancer(config LoadBalancerConfig) *LeastConnectionsLoadBalancer {
	return &LeastConnectionsLoadBalancer{
		BaseLoadBalancer: NewBaseLoadBalancer(config),
	}
}

// NextBackend 选择下一个后端服务器
func (lb *LeastConnectionsLoadBalancer) NextBackend(req *http.Request) (*Backend, error) {
	activeBackends := lb.GetActiveBackends()
	if len(activeBackends) == 0 {
		return nil, errors.New("no active backends available")
	}

	// 找到连接数最少的后端
	minConnections := int(^uint(0) >> 1) // 最大int值
	var selectedBackend *Backend

	for _, backend := range activeBackends {
		if backend.Connections < minConnections {
			minConnections = backend.Connections
			selectedBackend = backend
		}
	}

	return selectedBackend, nil
}

// ResponseTimeLoadBalancer 最短响应时间负载均衡器
type ResponseTimeLoadBalancer struct {
	*BaseLoadBalancer
}

// NewResponseTimeLoadBalancer 创建最短响应时间负载均衡器
func NewResponseTimeLoadBalancer(config LoadBalancerConfig) *ResponseTimeLoadBalancer {
	return &ResponseTimeLoadBalancer{
		BaseLoadBalancer: NewBaseLoadBalancer(config),
	}
}

// NextBackend 选择下一个后端服务器
func (lb *ResponseTimeLoadBalancer) NextBackend(req *http.Request) (*Backend, error) {
	activeBackends := lb.GetActiveBackends()
	if len(activeBackends) == 0 {
		return nil, errors.New("no active backends available")
	}

	// 找到响应时间最短的后端
	var selectedBackend *Backend
	minResponseTime := time.Duration(^int64(0)) // 最大time.Duration值

	for _, backend := range activeBackends {
		// 如果响应时间为0，则认为是新的后端，给予默认值
		responseTime := backend.ResponseTime
		if responseTime == 0 {
			responseTime = 100 * time.Millisecond // 默认100ms
		}

		if responseTime < minResponseTime {
			minResponseTime = responseTime
			selectedBackend = backend
		}
	}

	return selectedBackend, nil
}

// RandomLoadBalancer 随机负载均衡器
type RandomLoadBalancer struct {
	*BaseLoadBalancer
	rand *rand.Rand
	mu   sync.Mutex
}

// NewRandomLoadBalancer 创建随机负载均衡器
func NewRandomLoadBalancer(config LoadBalancerConfig) *RandomLoadBalancer {
	return &RandomLoadBalancer{
		BaseLoadBalancer: NewBaseLoadBalancer(config),
		rand:             rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NextBackend 选择下一个后端服务器
func (lb *RandomLoadBalancer) NextBackend(req *http.Request) (*Backend, error) {
	activeBackends := lb.GetActiveBackends()
	if len(activeBackends) == 0 {
		return nil, errors.New("no active backends available")
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	// 随机选择
	index := lb.rand.Intn(len(activeBackends))
	return activeBackends[index], nil
}

// WeightedRandomLoadBalancer 加权随机负载均衡器
type WeightedRandomLoadBalancer struct {
	*BaseLoadBalancer
	rand *rand.Rand
	mu   sync.Mutex
}

// NewWeightedRandomLoadBalancer 创建加权随机负载均衡器
func NewWeightedRandomLoadBalancer(config LoadBalancerConfig) *WeightedRandomLoadBalancer {
	return &WeightedRandomLoadBalancer{
		BaseLoadBalancer: NewBaseLoadBalancer(config),
		rand:             rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NextBackend 选择下一个后端服务器
func (lb *WeightedRandomLoadBalancer) NextBackend(req *http.Request) (*Backend, error) {
	activeBackends := lb.GetActiveBackends()
	if len(activeBackends) == 0 {
		return nil, errors.New("no active backends available")
	}

	// 计算总权重
	totalWeight := 0
	for _, backend := range activeBackends {
		weight := backend.Weight
		if weight <= 0 {
			weight = 1
		}
		totalWeight += weight
	}

	if totalWeight == 0 {
		return nil, errors.New("invalid weights for backends")
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	// 加权随机选择
	targetWeight := lb.rand.Intn(totalWeight)

	currentWeight := 0
	for _, backend := range activeBackends {
		weight := backend.Weight
		if weight <= 0 {
			weight = 1
		}

		currentWeight += weight
		if targetWeight < currentWeight {
			return backend, nil
		}
	}

	// 如果没有找到，返回第一个
	return activeBackends[0], nil
}

// SessionAffinityLoadBalancer 会话保持负载均衡器包装器
type SessionAffinityLoadBalancer struct {
	LoadBalancer
	config LoadBalancerConfig
}

// NewSessionAffinityLoadBalancer 创建会话保持负载均衡器
func NewSessionAffinityLoadBalancer(lb LoadBalancer, config LoadBalancerConfig) *SessionAffinityLoadBalancer {
	return &SessionAffinityLoadBalancer{
		LoadBalancer: lb,
		config:       config,
	}
}

// NextBackend 选择下一个后端服务器
func (lb *SessionAffinityLoadBalancer) NextBackend(req *http.Request) (*Backend, error) {
	// 如果没有启用会话保持，直接使用内部负载均衡器
	if lb.config.SessionAffinity == nil || !lb.config.SessionAffinity.Enabled {
		return lb.LoadBalancer.NextBackend(req)
	}

	// 尝试从Cookie获取会话信息
	cookie, err := req.Cookie(lb.config.SessionAffinity.CookieName)
	if err == nil && cookie.Value != "" {
		// 如果有会话信息，尝试从会话映射中获取后端
		backend := lb.getBackendFromSession(cookie.Value)
		if backend != nil && backend.Active {
			return backend, nil
		}
	}

	// 如果没有会话信息或后端不可用，使用内部负载均衡器选择
	backend, err := lb.LoadBalancer.NextBackend(req)
	if err != nil {
		return nil, err
	}

	// 设置会话Cookie
	// 注意：这里不能直接设置响应，因为这是在请求处理阶段
	// 需要在代理处理器的响应处理阶段设置Cookie

	return backend, nil
}

// getBackendFromSession 从会话ID获取后端
func (lb *SessionAffinityLoadBalancer) getBackendFromSession(sessionID string) *Backend {
	// 这里简化实现，实际应用中可能需要使用Redis等存储会话映射
	// 这里使用简单的哈希映射
	activeBackends := lb.GetActiveBackends()
	if len(activeBackends) == 0 {
		return nil
	}

	hash := sha256.Sum256([]byte(sessionID))
	index := binary.BigEndian.Uint32(hash[:4]) % uint32(len(activeBackends))
	return activeBackends[index]
}

// GetActiveBackends 获取活跃的后端服务器列表
func (lb *SessionAffinityLoadBalancer) GetActiveBackends() []*Backend {
	// 直接调用内部负载均衡器的GetActiveBackends方法
	return lb.LoadBalancer.GetActiveBackends()
}
