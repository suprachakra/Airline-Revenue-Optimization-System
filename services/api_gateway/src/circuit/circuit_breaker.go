package circuit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"iaros/api_gateway/src/config"
)

// CircuitBreakerManager manages circuit breakers for different services
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	config   *config.CircuitBreakerConfig
	logger   *zap.Logger
	mutex    sync.RWMutex
	ready    bool
}

// CircuitBreaker represents a single circuit breaker
type CircuitBreaker struct {
	name              string
	state             State
	failureThreshold  int
	successThreshold  int
	timeout           time.Duration
	maxRequests       int
	failureCount      int
	successCount      int
	lastFailureTime   time.Time
	lastSuccessTime   time.Time
	nextAttempt       time.Time
	fallbackResponse  interface{}
	healthChecker     HealthChecker
	metrics           *CircuitBreakerMetrics
	mutex             sync.RWMutex
}

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerMetrics tracks circuit breaker metrics
type CircuitBreakerMetrics struct {
	TotalRequests      int64     `json:"total_requests"`
	FailedRequests     int64     `json:"failed_requests"`
	SuccessRequests    int64     `json:"success_requests"`
	CircuitOpenCount   int64     `json:"circuit_open_count"`
	CircuitCloseCount  int64     `json:"circuit_close_count"`
	LastStateChange    time.Time `json:"last_state_change"`
	AverageResponseTime float64  `json:"average_response_time"`
	mutex              sync.RWMutex
}

// HealthChecker interface for service health checking
type HealthChecker interface {
	Check() error
}

// HTTPHealthChecker implements health checking via HTTP
type HTTPHealthChecker struct {
	URL     string
	Timeout time.Duration
	client  *http.Client
}

// FallbackResponse represents a fallback response
type FallbackResponse struct {
	Status  int                    `json:"status"`
	Headers map[string]string      `json:"headers"`
	Body    map[string]interface{} `json:"body"`
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(cfg *config.Config) (*CircuitBreakerManager, error) {
	logger, _ := zap.NewProduction()

	manager := &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
		config:   &cfg.CircuitBreaker,
		logger:   logger,
		ready:    true,
	}

	// Initialize circuit breakers for configured services
	manager.initializeCircuitBreakers()

	// Start monitoring goroutine
	go manager.startMonitoring()

	return manager, nil
}

// initializeCircuitBreakers initializes circuit breakers for all services
func (cbm *CircuitBreakerManager) initializeCircuitBreakers() {
	// Pricing Service Circuit Breaker
	cbm.breakers["pricing-service"] = &CircuitBreaker{
		name:              "pricing-service",
		state:             StateClosed,
		failureThreshold:  5,
		successThreshold:  3,
		timeout:           30 * time.Second,
		maxRequests:       100,
		healthChecker:     &HTTPHealthChecker{URL: "http://pricing-service:8080/health", Timeout: 5 * time.Second},
		fallbackResponse:  cbm.createPricingFallback(),
		metrics:           &CircuitBreakerMetrics{},
	}

	// Forecasting Service Circuit Breaker
	cbm.breakers["forecasting-service"] = &CircuitBreaker{
		name:              "forecasting-service",
		state:             StateClosed,
		failureThreshold:  3,
		successThreshold:  2,
		timeout:           60 * time.Second,
		maxRequests:       50,
		healthChecker:     &HTTPHealthChecker{URL: "http://forecasting-service:8080/health", Timeout: 10 * time.Second},
		fallbackResponse:  cbm.createForecastingFallback(),
		metrics:           &CircuitBreakerMetrics{},
	}

	// Offer Service Circuit Breaker
	cbm.breakers["offer-service"] = &CircuitBreaker{
		name:              "offer-service",
		state:             StateClosed,
		failureThreshold:  5,
		successThreshold:  3,
		timeout:           20 * time.Second,
		maxRequests:       200,
		healthChecker:     &HTTPHealthChecker{URL: "http://offer-service:8080/health", Timeout: 5 * time.Second},
		fallbackResponse:  cbm.createOfferFallback(),
		metrics:           &CircuitBreakerMetrics{},
	}

	// Order Service Circuit Breaker
	cbm.breakers["order-service"] = &CircuitBreaker{
		name:              "order-service",
		state:             StateClosed,
		failureThreshold:  3,
		successThreshold:  2,
		timeout:           30 * time.Second,
		maxRequests:       150,
		healthChecker:     &HTTPHealthChecker{URL: "http://order-service:8080/health", Timeout: 5 * time.Second},
		fallbackResponse:  cbm.createOrderFallback(),
		metrics:           &CircuitBreakerMetrics{},
	}

	// Distribution Service Circuit Breaker
	cbm.breakers["distribution-service"] = &CircuitBreaker{
		name:              "distribution-service",
		state:             StateClosed,
		failureThreshold:  4,
		successThreshold:  3,
		timeout:           25 * time.Second,
		maxRequests:       300,
		healthChecker:     &HTTPHealthChecker{URL: "http://distribution-service:8080/health", Timeout: 5 * time.Second},
		fallbackResponse:  cbm.createDistributionFallback(),
		metrics:           &CircuitBreakerMetrics{},
	}

	// Ancillary Service Circuit Breaker
	cbm.breakers["ancillary-service"] = &CircuitBreaker{
		name:              "ancillary-service",
		state:             StateClosed,
		failureThreshold:  5,
		successThreshold:  3,
		timeout:           20 * time.Second,
		maxRequests:       250,
		healthChecker:     &HTTPHealthChecker{URL: "http://ancillary-service:8080/health", Timeout: 5 * time.Second},
		fallbackResponse:  cbm.createAncillaryFallback(),
		metrics:           &CircuitBreakerMetrics{},
	}

	// User Management Service Circuit Breaker
	cbm.breakers["user-service"] = &CircuitBreaker{
		name:              "user-service",
		state:             StateClosed,
		failureThreshold:  3,
		successThreshold:  2,
		timeout:           15 * time.Second,
		maxRequests:       100,
		healthChecker:     &HTTPHealthChecker{URL: "http://user-service:8080/health", Timeout: 3 * time.Second},
		fallbackResponse:  cbm.createUserFallback(),
		metrics:           &CircuitBreakerMetrics{},
	}

	cbm.logger.Info("Initialized circuit breakers", zap.Int("count", len(cbm.breakers)))
}

// IsOpen checks if a circuit breaker is open
func (cbm *CircuitBreakerManager) IsOpen(service string) bool {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	breaker, exists := cbm.breakers[service]
	if !exists {
		return false
	}

	return breaker.IsOpen()
}

// RecordSuccess records a successful request
func (cbm *CircuitBreakerManager) RecordSuccess(service string, duration time.Duration) {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	breaker, exists := cbm.breakers[service]
	if !exists {
		return
	}

	breaker.RecordSuccess(duration)
}

// RecordFailure records a failed request
func (cbm *CircuitBreakerManager) RecordFailure(service string, err error) {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	breaker, exists := cbm.breakers[service]
	if !exists {
		return
	}

	breaker.RecordFailure(err)
}

// GetFallbackResponse returns the fallback response for a service
func (cbm *CircuitBreakerManager) GetFallbackResponse(service string) interface{} {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	breaker, exists := cbm.breakers[service]
	if !exists {
		return cbm.createGenericFallback()
	}

	return breaker.GetFallbackResponse()
}

// Reset resets a circuit breaker
func (cbm *CircuitBreakerManager) Reset(service string) error {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	breaker, exists := cbm.breakers[service]
	if !exists {
		return fmt.Errorf("circuit breaker not found for service: %s", service)
	}

	breaker.Reset()
	cbm.logger.Info("Circuit breaker reset", zap.String("service", service))
	return nil
}

// GetStatus returns the status of all circuit breakers
func (cbm *CircuitBreakerManager) GetStatus() map[string]interface{} {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	status := make(map[string]interface{})
	for name, breaker := range cbm.breakers {
		status[name] = breaker.GetStatus()
	}

	return status
}

// IsReady returns whether the circuit breaker manager is ready
func (cbm *CircuitBreakerManager) IsReady() bool {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()
	return cbm.ready
}

// CircuitBreaker Methods

// IsOpen checks if the circuit breaker is open
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	switch cb.state {
	case StateOpen:
		// Check if we should move to half-open
		if time.Now().After(cb.nextAttempt) {
			cb.mutex.RUnlock()
			cb.mutex.Lock()
			cb.state = StateHalfOpen
			cb.successCount = 0
			cb.metrics.LastStateChange = time.Now()
			cb.mutex.Unlock()
			cb.mutex.RLock()
			return false
		}
		return true
	case StateHalfOpen:
		return false
	case StateClosed:
		return false
	default:
		return false
	}
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess(duration time.Duration) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.lastSuccessTime = time.Now()
	cb.successCount++
	cb.failureCount = 0 // Reset failure count on success

	// Update metrics
	cb.metrics.mutex.Lock()
	cb.metrics.TotalRequests++
	cb.metrics.SuccessRequests++
	cb.metrics.AverageResponseTime = (cb.metrics.AverageResponseTime + duration.Seconds()) / 2
	cb.metrics.mutex.Unlock()

	switch cb.state {
	case StateHalfOpen:
		if cb.successCount >= cb.successThreshold {
			cb.state = StateClosed
			cb.successCount = 0
			cb.metrics.CircuitCloseCount++
			cb.metrics.LastStateChange = time.Now()
		}
	case StateClosed:
		// Already closed, nothing to do
	case StateOpen:
		// Should not happen, but handle gracefully
		cb.state = StateClosed
		cb.successCount = 0
		cb.metrics.CircuitCloseCount++
		cb.metrics.LastStateChange = time.Now()
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure(err error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.lastFailureTime = time.Now()
	cb.failureCount++
	cb.successCount = 0 // Reset success count on failure

	// Update metrics
	cb.metrics.mutex.Lock()
	cb.metrics.TotalRequests++
	cb.metrics.FailedRequests++
	cb.metrics.mutex.Unlock()

	switch cb.state {
	case StateClosed:
		if cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
			cb.nextAttempt = time.Now().Add(cb.timeout)
			cb.metrics.CircuitOpenCount++
			cb.metrics.LastStateChange = time.Now()
		}
	case StateHalfOpen:
		cb.state = StateOpen
		cb.nextAttempt = time.Now().Add(cb.timeout)
		cb.metrics.CircuitOpenCount++
		cb.metrics.LastStateChange = time.Now()
	case StateOpen:
		// Already open, extend timeout
		cb.nextAttempt = time.Now().Add(cb.timeout)
	}
}

// GetFallbackResponse returns the fallback response
func (cb *CircuitBreaker) GetFallbackResponse() interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.fallbackResponse
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.nextAttempt = time.Time{}
	cb.metrics.LastStateChange = time.Now()
}

// GetStatus returns the current status of the circuit breaker
func (cb *CircuitBreaker) GetStatus() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return map[string]interface{}{
		"name":                cb.name,
		"state":               cb.state.String(),
		"failure_count":       cb.failureCount,
		"success_count":       cb.successCount,
		"failure_threshold":   cb.failureThreshold,
		"success_threshold":   cb.successThreshold,
		"timeout":             cb.timeout.String(),
		"last_failure_time":   cb.lastFailureTime,
		"last_success_time":   cb.lastSuccessTime,
		"next_attempt":        cb.nextAttempt,
		"metrics":             cb.getMetrics(),
	}
}

// getMetrics returns current metrics
func (cb *CircuitBreaker) getMetrics() map[string]interface{} {
	cb.metrics.mutex.RLock()
	defer cb.metrics.mutex.RUnlock()

	return map[string]interface{}{
		"total_requests":        cb.metrics.TotalRequests,
		"failed_requests":       cb.metrics.FailedRequests,
		"success_requests":      cb.metrics.SuccessRequests,
		"circuit_open_count":    cb.metrics.CircuitOpenCount,
		"circuit_close_count":   cb.metrics.CircuitCloseCount,
		"last_state_change":     cb.metrics.LastStateChange,
		"average_response_time": cb.metrics.AverageResponseTime,
	}
}

// HTTPHealthChecker implementation
func (hc *HTTPHealthChecker) Check() error {
	if hc.client == nil {
		hc.client = &http.Client{
			Timeout: hc.Timeout,
		}
	}

	resp, err := hc.client.Get(hc.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// startMonitoring starts the monitoring goroutine
func (cbm *CircuitBreakerManager) startMonitoring() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		cbm.performHealthChecks()
	}
}

// performHealthChecks performs health checks on all services
func (cbm *CircuitBreakerManager) performHealthChecks() {
	cbm.mutex.RLock()
	breakers := make(map[string]*CircuitBreaker)
	for name, breaker := range cbm.breakers {
		breakers[name] = breaker
	}
	cbm.mutex.RUnlock()

	for name, breaker := range breakers {
		go func(name string, breaker *CircuitBreaker) {
			if err := breaker.healthChecker.Check(); err != nil {
				cbm.logger.Warn("Health check failed",
					zap.String("service", name),
					zap.Error(err),
				)
				breaker.RecordFailure(err)
			} else {
				breaker.RecordSuccess(0) // Health check success
			}
		}(name, breaker)
	}
}

// Fallback Response Creators

func (cbm *CircuitBreakerManager) createPricingFallback() interface{} {
	return FallbackResponse{
		Status: 200,
		Headers: map[string]string{
			"Content-Type":      "application/json",
			"X-Fallback-Active": "true",
			"X-Service":         "pricing-service",
		},
		Body: map[string]interface{}{
			"price": 0.0,
			"currency": "USD",
			"fallback": true,
			"message": "Pricing service temporarily unavailable. Using cached pricing data.",
			"timestamp": time.Now().UTC(),
		},
	}
}

func (cbm *CircuitBreakerManager) createForecastingFallback() interface{} {
	return FallbackResponse{
		Status: 200,
		Headers: map[string]string{
			"Content-Type":      "application/json",
			"X-Fallback-Active": "true",
			"X-Service":         "forecasting-service",
		},
		Body: map[string]interface{}{
			"forecast": []map[string]interface{}{
				{
					"date":   time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
					"value":  100.0,
					"confidence": 0.5,
				},
			},
			"fallback": true,
			"message": "Forecasting service temporarily unavailable. Using historical data.",
			"timestamp": time.Now().UTC(),
		},
	}
}

func (cbm *CircuitBreakerManager) createOfferFallback() interface{} {
	return FallbackResponse{
		Status: 200,
		Headers: map[string]string{
			"Content-Type":      "application/json",
			"X-Fallback-Active": "true",
			"X-Service":         "offer-service",
		},
		Body: map[string]interface{}{
			"offers": []map[string]interface{}{
				{
					"id":    "fallback-offer-001",
					"title": "Standard Offer",
					"price": 0.0,
					"available": false,
				},
			},
			"fallback": true,
			"message": "Offer service temporarily unavailable. Limited offers available.",
			"timestamp": time.Now().UTC(),
		},
	}
}

func (cbm *CircuitBreakerManager) createOrderFallback() interface{} {
	return FallbackResponse{
		Status: 503,
		Headers: map[string]string{
			"Content-Type":      "application/json",
			"X-Fallback-Active": "true",
			"X-Service":         "order-service",
		},
		Body: map[string]interface{}{
			"error": "Service temporarily unavailable",
			"message": "Order service is currently unavailable. Please try again later.",
			"fallback": true,
			"timestamp": time.Now().UTC(),
		},
	}
}

func (cbm *CircuitBreakerManager) createDistributionFallback() interface{} {
	return FallbackResponse{
		Status: 200,
		Headers: map[string]string{
			"Content-Type":      "application/json",
			"X-Fallback-Active": "true",
			"X-Service":         "distribution-service",
		},
		Body: map[string]interface{}{
			"channels": []string{"direct"},
			"fallback": true,
			"message": "Distribution service temporarily unavailable. Using direct channel only.",
			"timestamp": time.Now().UTC(),
		},
	}
}

func (cbm *CircuitBreakerManager) createAncillaryFallback() interface{} {
	return FallbackResponse{
		Status: 200,
		Headers: map[string]string{
			"Content-Type":      "application/json",
			"X-Fallback-Active": "true",
			"X-Service":         "ancillary-service",
		},
		Body: map[string]interface{}{
			"ancillary_services": []map[string]interface{}{
				{
					"id":    "fallback-baggage",
					"name":  "Baggage",
					"price": 35.0,
					"available": false,
				},
			},
			"fallback": true,
			"message": "Ancillary service temporarily unavailable. Limited services available.",
			"timestamp": time.Now().UTC(),
		},
	}
}

func (cbm *CircuitBreakerManager) createUserFallback() interface{} {
	return FallbackResponse{
		Status: 503,
		Headers: map[string]string{
			"Content-Type":      "application/json",
			"X-Fallback-Active": "true",
			"X-Service":         "user-service",
		},
		Body: map[string]interface{}{
			"error": "Service temporarily unavailable",
			"message": "User service is currently unavailable. Please try again later.",
			"fallback": true,
			"timestamp": time.Now().UTC(),
		},
	}
}

func (cbm *CircuitBreakerManager) createGenericFallback() interface{} {
	return FallbackResponse{
		Status: 503,
		Headers: map[string]string{
			"Content-Type":      "application/json",
			"X-Fallback-Active": "true",
		},
		Body: map[string]interface{}{
			"error": "Service temporarily unavailable",
			"message": "The requested service is currently unavailable. Please try again later.",
			"fallback": true,
			"timestamp": time.Now().UTC(),
		},
	}
} 