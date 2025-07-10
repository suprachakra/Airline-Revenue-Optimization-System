package performance

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/iaros/common/logging"
	"github.com/iaros/common/metrics"
)

// Load Balancer Configuration Types
type BalancingStrategy struct {
	Name           string                 `json:"name"`
	Algorithm      string                 `json:"algorithm"`
	Weight         int                    `json:"weight"`
	Configuration  map[string]interface{} `json:"configuration"`
	HealthCheck    *HealthCheckConfig     `json:"health_check"`
	CircuitBreaker *CircuitBreakerConfig  `json:"circuit_breaker"`
}

type HealthCheckConfig struct {
	Enabled            bool          `json:"enabled"`
	CheckInterval      time.Duration `json:"check_interval"`
	Timeout            time.Duration `json:"timeout"`
	HealthyThreshold   int           `json:"healthy_threshold"`
	UnhealthyThreshold int           `json:"unhealthy_threshold"`
	Path               string        `json:"path"`
	ExpectedStatusCode int           `json:"expected_status_code"`
}

type CircuitBreakerConfig struct {
	Enabled           bool          `json:"enabled"`
	FailureThreshold  int           `json:"failure_threshold"`
	SuccessThreshold  int           `json:"success_threshold"`
	Timeout           time.Duration `json:"timeout"`
	MaxRequests       int           `json:"max_requests"`
}

// Server represents a backend server
type Server struct {
	ID               string            `json:"id"`
	Address          string            `json:"address"`
	Port             int               `json:"port"`
	Weight           int               `json:"weight"`
	IsHealthy        bool              `json:"is_healthy"`
	ActiveConnections int              `json:"active_connections"`
	TotalRequests    int64             `json:"total_requests"`
	FailedRequests   int64             `json:"failed_requests"`
	AverageLatency   time.Duration     `json:"average_latency"`
	LastHealthCheck  time.Time         `json:"last_health_check"`
	Metadata         map[string]string `json:"metadata"`
	mu               sync.RWMutex
}

// RoundRobinBalancer implements round-robin load balancing
type RoundRobinBalancer struct {
	servers []*Server
	current int
	mu      sync.RWMutex
	logger  logging.Logger
}

func NewRoundRobinBalancer() *RoundRobinBalancer {
	return &RoundRobinBalancer{
		servers: make([]*Server, 0),
		current: 0,
		logger:  logging.GetLogger("round_robin_balancer"),
	}
}

func (rrb *RoundRobinBalancer) AddServer(server *Server) {
	rrb.mu.Lock()
	defer rrb.mu.Unlock()
	rrb.servers = append(rrb.servers, server)
	rrb.logger.Info("Server added to round-robin balancer", "server", server.Address)
}

func (rrb *RoundRobinBalancer) RemoveServer(serverID string) {
	rrb.mu.Lock()
	defer rrb.mu.Unlock()
	
	for i, server := range rrb.servers {
		if server.ID == serverID {
			rrb.servers = append(rrb.servers[:i], rrb.servers[i+1:]...)
			rrb.logger.Info("Server removed from round-robin balancer", "server", server.Address)
			break
		}
	}
}

func (rrb *RoundRobinBalancer) SelectServer() *Server {
	rrb.mu.Lock()
	defer rrb.mu.Unlock()
	
	if len(rrb.servers) == 0 {
		return nil
	}
	
	// Find next healthy server
	attempts := 0
	for attempts < len(rrb.servers) {
		server := rrb.servers[rrb.current]
		rrb.current = (rrb.current + 1) % len(rrb.servers)
		
		if server.IsHealthy {
			server.mu.Lock()
			server.ActiveConnections++
			server.TotalRequests++
			server.mu.Unlock()
			return server
		}
		attempts++
	}
	
	return nil // No healthy servers available
}

// LeastConnectionsBalancer implements least connections load balancing
type LeastConnectionsBalancer struct {
	servers []*Server
	mu      sync.RWMutex
	logger  logging.Logger
}

func NewLeastConnectionsBalancer() *LeastConnectionsBalancer {
	return &LeastConnectionsBalancer{
		servers: make([]*Server, 0),
		logger:  logging.GetLogger("least_connections_balancer"),
	}
}

func (lcb *LeastConnectionsBalancer) AddServer(server *Server) {
	lcb.mu.Lock()
	defer lcb.mu.Unlock()
	lcb.servers = append(lcb.servers, server)
	lcb.logger.Info("Server added to least-connections balancer", "server", server.Address)
}

func (lcb *LeastConnectionsBalancer) RemoveServer(serverID string) {
	lcb.mu.Lock()
	defer lcb.mu.Unlock()
	
	for i, server := range lcb.servers {
		if server.ID == serverID {
			lcb.servers = append(lcb.servers[:i], lcb.servers[i+1:]...)
			lcb.logger.Info("Server removed from least-connections balancer", "server", server.Address)
			break
		}
	}
}

func (lcb *LeastConnectionsBalancer) SelectServer() *Server {
	lcb.mu.RLock()
	defer lcb.mu.RUnlock()
	
	if len(lcb.servers) == 0 {
		return nil
	}
	
	var selectedServer *Server
	minConnections := int(^uint(0) >> 1) // Max int
	
	for _, server := range lcb.servers {
		if server.IsHealthy && server.ActiveConnections < minConnections {
			minConnections = server.ActiveConnections
			selectedServer = server
		}
	}
	
	if selectedServer != nil {
		selectedServer.mu.Lock()
		selectedServer.ActiveConnections++
		selectedServer.TotalRequests++
		selectedServer.mu.Unlock()
	}
	
	return selectedServer
}

// WeightedRoundRobinBalancer implements weighted round-robin load balancing
type WeightedRoundRobinBalancer struct {
	servers       []*Server
	currentWeights []int
	mu            sync.RWMutex
	logger        logging.Logger
}

func NewWeightedRoundRobinBalancer() *WeightedRoundRobinBalancer {
	return &WeightedRoundRobinBalancer{
		servers:        make([]*Server, 0),
		currentWeights: make([]int, 0),
		logger:         logging.GetLogger("weighted_round_robin_balancer"),
	}
}

func (wrrb *WeightedRoundRobinBalancer) AddServer(server *Server) {
	wrrb.mu.Lock()
	defer wrrb.mu.Unlock()
	wrrb.servers = append(wrrb.servers, server)
	wrrb.currentWeights = append(wrrb.currentWeights, 0)
	wrrb.logger.Info("Server added to weighted round-robin balancer", 
		"server", server.Address, "weight", server.Weight)
}

func (wrrb *WeightedRoundRobinBalancer) RemoveServer(serverID string) {
	wrrb.mu.Lock()
	defer wrrb.mu.Unlock()
	
	for i, server := range wrrb.servers {
		if server.ID == serverID {
			wrrb.servers = append(wrrb.servers[:i], wrrb.servers[i+1:]...)
			wrrb.currentWeights = append(wrrb.currentWeights[:i], wrrb.currentWeights[i+1:]...)
			wrrb.logger.Info("Server removed from weighted round-robin balancer", "server", server.Address)
			break
		}
	}
}

func (wrrb *WeightedRoundRobinBalancer) SelectServer() *Server {
	wrrb.mu.Lock()
	defer wrrb.mu.Unlock()
	
	if len(wrrb.servers) == 0 {
		return nil
	}
	
	// Calculate total weight
	totalWeight := 0
	for _, server := range wrrb.servers {
		if server.IsHealthy {
			totalWeight += server.Weight
		}
	}
	
	if totalWeight == 0 {
		return nil
	}
	
	// Weighted round-robin algorithm
	var selectedServer *Server
	maxCurrentWeight := -1
	
	for i, server := range wrrb.servers {
		if !server.IsHealthy {
			continue
		}
		
		wrrb.currentWeights[i] += server.Weight
		
		if wrrb.currentWeights[i] > maxCurrentWeight {
			maxCurrentWeight = wrrb.currentWeights[i]
			selectedServer = server
		}
	}
	
	if selectedServer != nil {
		// Find the index and reduce current weight
		for i, server := range wrrb.servers {
			if server == selectedServer {
				wrrb.currentWeights[i] -= totalWeight
				break
			}
		}
		
		selectedServer.mu.Lock()
		selectedServer.ActiveConnections++
		selectedServer.TotalRequests++
		selectedServer.mu.Unlock()
	}
	
	return selectedServer
}

// IPHashBalancer implements IP hash-based load balancing
type IPHashBalancer struct {
	servers []*Server
	mu      sync.RWMutex
	logger  logging.Logger
}

func NewIPHashBalancer() *IPHashBalancer {
	return &IPHashBalancer{
		servers: make([]*Server, 0),
		logger:  logging.GetLogger("ip_hash_balancer"),
	}
}

func (ihb *IPHashBalancer) AddServer(server *Server) {
	ihb.mu.Lock()
	defer ihb.mu.Unlock()
	ihb.servers = append(ihb.servers, server)
	ihb.logger.Info("Server added to IP hash balancer", "server", server.Address)
}

func (ihb *IPHashBalancer) RemoveServer(serverID string) {
	ihb.mu.Lock()
	defer ihb.mu.Unlock()
	
	for i, server := range ihb.servers {
		if server.ID == serverID {
			ihb.servers = append(ihb.servers[:i], ihb.servers[i+1:]...)
			ihb.logger.Info("Server removed from IP hash balancer", "server", server.Address)
			break
		}
	}
}

func (ihb *IPHashBalancer) SelectServer(clientIP string) *Server {
	ihb.mu.RLock()
	defer ihb.mu.RUnlock()
	
	if len(ihb.servers) == 0 {
		return nil
	}
	
	// Get healthy servers
	healthyServers := make([]*Server, 0)
	for _, server := range ihb.servers {
		if server.IsHealthy {
			healthyServers = append(healthyServers, server)
		}
	}
	
	if len(healthyServers) == 0 {
		return nil
	}
	
	// Hash client IP
	hasher := fnv.New32a()
	hasher.Write([]byte(clientIP))
	hash := hasher.Sum32()
	
	// Select server based on hash
	serverIndex := int(hash) % len(healthyServers)
	selectedServer := healthyServers[serverIndex]
	
	selectedServer.mu.Lock()
	selectedServer.ActiveConnections++
	selectedServer.TotalRequests++
	selectedServer.mu.Unlock()
	
	return selectedServer
}

// GeographicRoutingBalancer implements geographic-based load balancing
type GeographicRoutingBalancer struct {
	regions map[string][]*Server
	mu      sync.RWMutex
	logger  logging.Logger
}

func NewGeographicRoutingBalancer() *GeographicRoutingBalancer {
	return &GeographicRoutingBalancer{
		regions: make(map[string][]*Server),
		logger:  logging.GetLogger("geographic_routing_balancer"),
	}
}

func (grb *GeographicRoutingBalancer) AddServer(server *Server, region string) {
	grb.mu.Lock()
	defer grb.mu.Unlock()
	
	if grb.regions[region] == nil {
		grb.regions[region] = make([]*Server, 0)
	}
	
	grb.regions[region] = append(grb.regions[region], server)
	grb.logger.Info("Server added to geographic balancer", 
		"server", server.Address, "region", region)
}

func (grb *GeographicRoutingBalancer) SelectServer(clientRegion string) *Server {
	grb.mu.RLock()
	defer grb.mu.RUnlock()
	
	// Try to find server in client's region first
	if servers, exists := grb.regions[clientRegion]; exists {
		for _, server := range servers {
			if server.IsHealthy {
				server.mu.Lock()
				server.ActiveConnections++
				server.TotalRequests++
				server.mu.Unlock()
				return server
			}
		}
	}
	
	// Fallback to any healthy server in any region
	for _, servers := range grb.regions {
		for _, server := range servers {
			if server.IsHealthy {
				server.mu.Lock()
				server.ActiveConnections++
				server.TotalRequests++
				server.mu.Unlock()
				return server
			}
		}
	}
	
	return nil
}

// HealthChecker monitors server health
type HealthChecker struct {
	config  *HealthCheckerConfig
	servers map[string]*Server
	mu      sync.RWMutex
	logger  logging.Logger
}

func NewHealthChecker(config *HealthCheckerConfig) *HealthChecker {
	return &HealthChecker{
		config:  config,
		servers: make(map[string]*Server),
		logger:  logging.GetLogger("health_checker"),
	}
}

func (hc *HealthChecker) AddServer(server *Server) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.servers[server.ID] = server
}

func (hc *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(hc.config.CheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hc.performHealthChecks()
		}
	}
}

func (hc *HealthChecker) performHealthChecks() {
	hc.mu.RLock()
	servers := make([]*Server, 0, len(hc.servers))
	for _, server := range hc.servers {
		servers = append(servers, server)
	}
	hc.mu.RUnlock()
	
	for _, server := range servers {
		go hc.checkServer(server)
	}
}

func (hc *HealthChecker) checkServer(server *Server) {
	isHealthy := hc.isServerHealthy(server)
	
	server.mu.Lock()
	previousHealth := server.IsHealthy
	server.IsHealthy = isHealthy
	server.LastHealthCheck = time.Now()
	server.mu.Unlock()
	
	if previousHealth != isHealthy {
		if isHealthy {
			hc.logger.Info("Server became healthy", "server", server.Address)
		} else {
			hc.logger.Warn("Server became unhealthy", "server", server.Address)
		}
	}
}

func (hc *HealthChecker) isServerHealthy(server *Server) bool {
	client := &http.Client{
		Timeout: hc.config.Timeout,
	}
	
	url := fmt.Sprintf("http://%s:%d%s", server.Address, server.Port, hc.config.Path)
	
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == hc.config.ExpectedStatusCode
}

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	config           *CircuitBreakerConfig
	state            CircuitBreakerState
	failureCount     int
	successCount     int
	lastFailureTime  time.Time
	mu               sync.RWMutex
	logger           logging.Logger
}

type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  CircuitBreakerClosed,
		logger: logging.GetLogger("circuit_breaker"),
	}
}

func (cb *CircuitBreaker) Call(fn func() error) error {
	if !cb.allowRequest() {
		return fmt.Errorf("circuit breaker is open")
	}
	
	err := fn()
	cb.recordResult(err == nil)
	
	return err
}

func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		return time.Since(cb.lastFailureTime) >= cb.config.Timeout
	case CircuitBreakerHalfOpen:
		return cb.successCount < cb.config.MaxRequests
	default:
		return false
	}
}

func (cb *CircuitBreaker) recordResult(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	if success {
		cb.onSuccess()
	} else {
		cb.onFailure()
	}
}

func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case CircuitBreakerHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.state = CircuitBreakerClosed
			cb.failureCount = 0
			cb.successCount = 0
			cb.logger.Info("Circuit breaker closed")
		}
	case CircuitBreakerClosed:
		cb.failureCount = 0
	}
}

func (cb *CircuitBreaker) onFailure() {
	switch cb.state {
	case CircuitBreakerClosed:
		cb.failureCount++
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.state = CircuitBreakerOpen
			cb.lastFailureTime = time.Now()
			cb.logger.Warn("Circuit breaker opened")
		}
	case CircuitBreakerHalfOpen:
		cb.state = CircuitBreakerOpen
		cb.lastFailureTime = time.Now()
		cb.successCount = 0
		cb.logger.Warn("Circuit breaker opened from half-open state")
	}
}

// Performance tracking components
type ResponseTimeTracker struct {
	responseTimes []time.Duration
	mu            sync.RWMutex
	logger        logging.Logger
}

func NewResponseTimeTracker() *ResponseTimeTracker {
	return &ResponseTimeTracker{
		responseTimes: make([]time.Duration, 0),
		logger:        logging.GetLogger("response_time_tracker"),
	}
}

func (rtt *ResponseTimeTracker) RecordResponseTime(duration time.Duration) {
	rtt.mu.Lock()
	defer rtt.mu.Unlock()
	
	rtt.responseTimes = append(rtt.responseTimes, duration)
	
	// Keep only last 1000 measurements
	if len(rtt.responseTimes) > 1000 {
		rtt.responseTimes = rtt.responseTimes[1:]
	}
}

func (rtt *ResponseTimeTracker) GetAverageResponseTime() time.Duration {
	rtt.mu.RLock()
	defer rtt.mu.RUnlock()
	
	if len(rtt.responseTimes) == 0 {
		return 0
	}
	
	var total time.Duration
	for _, duration := range rtt.responseTimes {
		total += duration
	}
	
	return total / time.Duration(len(rtt.responseTimes))
}

func (rtt *ResponseTimeTracker) GetPercentile(percentile float64) time.Duration {
	rtt.mu.RLock()
	defer rtt.mu.RUnlock()
	
	if len(rtt.responseTimes) == 0 {
		return 0
	}
	
	// Create a copy and sort
	sorted := make([]time.Duration, len(rtt.responseTimes))
	copy(sorted, rtt.responseTimes)
	
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	
	index := int(float64(len(sorted)) * percentile / 100.0)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	
	return sorted[index]
}

type ThroughputTracker struct {
	requestCounts []int
	timestamps    []time.Time
	mu            sync.RWMutex
	logger        logging.Logger
}

func NewThroughputTracker() *ThroughputTracker {
	return &ThroughputTracker{
		requestCounts: make([]int, 0),
		timestamps:    make([]time.Time, 0),
		logger:        logging.GetLogger("throughput_tracker"),
	}
}

func (tt *ThroughputTracker) RecordRequest() {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	
	now := time.Now()
	
	// Clean old entries (older than 1 minute)
	cutoff := now.Add(-time.Minute)
	for len(tt.timestamps) > 0 && tt.timestamps[0].Before(cutoff) {
		tt.timestamps = tt.timestamps[1:]
		tt.requestCounts = tt.requestCounts[1:]
	}
	
	// Add current request
	tt.timestamps = append(tt.timestamps, now)
	tt.requestCounts = append(tt.requestCounts, 1)
}

func (tt *ThroughputTracker) GetRequestsPerSecond() float64 {
	tt.mu.RLock()
	defer tt.mu.RUnlock()
	
	if len(tt.requestCounts) == 0 {
		return 0
	}
	
	totalRequests := 0
	for _, count := range tt.requestCounts {
		totalRequests += count
	}
	
	return float64(totalRequests) / 60.0 // requests per second over last minute
}

// Server utility functions
func (s *Server) DecrementConnections() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ActiveConnections > 0 {
		s.ActiveConnections--
	}
}

func (s *Server) RecordFailure() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FailedRequests++
}

func (s *Server) RecordLatency(latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Simple moving average
	if s.AverageLatency == 0 {
		s.AverageLatency = latency
	} else {
		s.AverageLatency = (s.AverageLatency + latency) / 2
	}
}

func (s *Server) GetConnectionString() string {
	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

func (s *Server) GetMetrics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return map[string]interface{}{
		"id":                 s.ID,
		"address":           s.Address,
		"port":              s.Port,
		"weight":            s.Weight,
		"is_healthy":        s.IsHealthy,
		"active_connections": s.ActiveConnections,
		"total_requests":    s.TotalRequests,
		"failed_requests":   s.FailedRequests,
		"average_latency":   s.AverageLatency.String(),
		"last_health_check": s.LastHealthCheck.Format(time.RFC3339),
	}
} 