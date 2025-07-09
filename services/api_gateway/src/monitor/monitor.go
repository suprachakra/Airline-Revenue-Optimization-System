package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"iaros/api_gateway/src/config"
)

// Monitor handles monitoring and metrics collection
type Monitor struct {
	config           *config.MonitoringConfig
	logger           *zap.Logger
	metrics          *Metrics
	healthChecks     map[string]HealthCheck
	mutex            sync.RWMutex
	stopChan         chan struct{}
	totalRequests    int64
	totalErrors      int64
	serviceMetrics   map[string]*ServiceMetrics
}

// Metrics holds all Prometheus metrics
type Metrics struct {
	RequestsTotal     *prometheus.CounterVec
	RequestDuration   *prometheus.HistogramVec
	ErrorsTotal       *prometheus.CounterVec
	ActiveConnections prometheus.Gauge
	ServiceHealth     *prometheus.GaugeVec
	RateLimitHits     *prometheus.CounterVec
	CircuitBreakerState *prometheus.GaugeVec
}

// ServiceMetrics tracks metrics for individual services
type ServiceMetrics struct {
	Name             string
	RequestCount     int64
	ErrorCount       int64
	TotalDuration    time.Duration
	LastRequestTime  time.Time
	AverageLatency   time.Duration
	ErrorRate        float64
	mutex            sync.RWMutex
}

// HealthCheck represents a health check function
type HealthCheck func() error

// NewMonitor creates a new monitoring instance
func NewMonitor(cfg *config.Config) (*Monitor, error) {
	logger, _ := zap.NewProduction()

	// Initialize Prometheus metrics
	metrics := &Metrics{
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_requests_total",
				Help: "Total number of requests processed by the API Gateway",
			},
			[]string{"method", "path", "status"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "api_gateway_request_duration_seconds",
				Help:    "Request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "service"},
		),
		ErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_errors_total",
				Help: "Total number of errors",
			},
			[]string{"service", "error_type"},
		),
		ActiveConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "api_gateway_active_connections",
				Help: "Number of active connections",
			},
		),
		ServiceHealth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "api_gateway_service_health",
				Help: "Health status of backend services (1 = healthy, 0 = unhealthy)",
			},
			[]string{"service"},
		),
		RateLimitHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_rate_limit_hits_total",
				Help: "Total number of rate limit hits",
			},
			[]string{"type", "key"},
		),
		CircuitBreakerState: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "api_gateway_circuit_breaker_state",
				Help: "Circuit breaker state (0 = closed, 1 = open, 2 = half-open)",
			},
			[]string{"service"},
		),
	}

	// Register metrics with Prometheus
	prometheus.MustRegister(
		metrics.RequestsTotal,
		metrics.RequestDuration,
		metrics.ErrorsTotal,
		metrics.ActiveConnections,
		metrics.ServiceHealth,
		metrics.RateLimitHits,
		metrics.CircuitBreakerState,
	)

	monitor := &Monitor{
		config:         &cfg.Monitoring,
		logger:         logger,
		metrics:        metrics,
		healthChecks:   make(map[string]HealthCheck),
		stopChan:       make(chan struct{}),
		serviceMetrics: make(map[string]*ServiceMetrics),
	}

	// Initialize service metrics
	monitor.initializeServiceMetrics()

	return monitor, nil
}

// initializeServiceMetrics initializes metrics for all services
func (m *Monitor) initializeServiceMetrics() {
	services := []string{
		"pricing-service",
		"forecasting-service",
		"offer-service",
		"order-service",
		"distribution-service",
		"ancillary-service",
		"user-service",
		"network-service",
		"procurement-service",
		"promotion-service",
	}

	for _, service := range services {
		m.serviceMetrics[service] = &ServiceMetrics{
			Name:            service,
			RequestCount:    0,
			ErrorCount:      0,
			TotalDuration:   0,
			LastRequestTime: time.Now(),
			AverageLatency:  0,
			ErrorRate:       0.0,
		}
	}
}

// RecordRequest records a request metric
func (m *Monitor) RecordRequest(service string, duration time.Duration) {
	// Update Prometheus metrics
	m.metrics.RequestsTotal.WithLabelValues("", "", "200").Inc()
	m.metrics.RequestDuration.WithLabelValues("", "", service).Observe(duration.Seconds())

	// Update service metrics
	m.updateServiceMetrics(service, duration, false)

	// Update total requests
	m.mutex.Lock()
	m.totalRequests++
	m.mutex.Unlock()
}

// RecordError records an error metric
func (m *Monitor) RecordError(service string, err error) {
	// Update Prometheus metrics
	m.metrics.ErrorsTotal.WithLabelValues(service, "request_error").Inc()

	// Update service metrics
	m.updateServiceMetrics(service, 0, true)

	// Update total errors
	m.mutex.Lock()
	m.totalErrors++
	m.mutex.Unlock()

	// Log error
	m.logger.Error("Service error recorded",
		zap.String("service", service),
		zap.Error(err),
	)
}

// RecordServiceHealth records service health status
func (m *Monitor) RecordServiceHealth(service string, healthy bool) {
	var value float64
	if healthy {
		value = 1.0
	} else {
		value = 0.0
	}

	m.metrics.ServiceHealth.WithLabelValues(service).Set(value)
}

// RecordRateLimitHit records a rate limit hit
func (m *Monitor) RecordRateLimitHit(limitType, key string) {
	m.metrics.RateLimitHits.WithLabelValues(limitType, key).Inc()
}

// RecordCircuitBreakerState records circuit breaker state
func (m *Monitor) RecordCircuitBreakerState(service string, state int) {
	m.metrics.CircuitBreakerState.WithLabelValues(service).Set(float64(state))
}

// updateServiceMetrics updates metrics for a specific service
func (m *Monitor) updateServiceMetrics(service string, duration time.Duration, isError bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	serviceMetrics, exists := m.serviceMetrics[service]
	if !exists {
		serviceMetrics = &ServiceMetrics{
			Name:            service,
			RequestCount:    0,
			ErrorCount:      0,
			TotalDuration:   0,
			LastRequestTime: time.Now(),
			AverageLatency:  0,
			ErrorRate:       0.0,
		}
		m.serviceMetrics[service] = serviceMetrics
	}

	serviceMetrics.mutex.Lock()
	defer serviceMetrics.mutex.Unlock()

	// Update request count
	serviceMetrics.RequestCount++
	serviceMetrics.LastRequestTime = time.Now()

	// Update error count
	if isError {
		serviceMetrics.ErrorCount++
	}

	// Update duration metrics
	if duration > 0 {
		serviceMetrics.TotalDuration += duration
		serviceMetrics.AverageLatency = serviceMetrics.TotalDuration / time.Duration(serviceMetrics.RequestCount)
	}

	// Update error rate
	if serviceMetrics.RequestCount > 0 {
		serviceMetrics.ErrorRate = float64(serviceMetrics.ErrorCount) / float64(serviceMetrics.RequestCount)
	}
}

// GetTotalRequests returns total number of requests
func (m *Monitor) GetTotalRequests() int64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.totalRequests
}

// GetTotalErrors returns total number of errors
func (m *Monitor) GetTotalErrors() int64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.totalErrors
}

// GetServiceMetrics returns metrics for a specific service
func (m *Monitor) GetServiceMetrics(service string) *ServiceMetrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if metrics, exists := m.serviceMetrics[service]; exists {
		// Return a copy to avoid race conditions
		metrics.mutex.RLock()
		defer metrics.mutex.RUnlock()
		
		return &ServiceMetrics{
			Name:            metrics.Name,
			RequestCount:    metrics.RequestCount,
			ErrorCount:      metrics.ErrorCount,
			TotalDuration:   metrics.TotalDuration,
			LastRequestTime: metrics.LastRequestTime,
			AverageLatency:  metrics.AverageLatency,
			ErrorRate:       metrics.ErrorRate,
		}
	}

	return nil
}

// GetAllServiceMetrics returns metrics for all services
func (m *Monitor) GetAllServiceMetrics() map[string]*ServiceMetrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]*ServiceMetrics)
	for service, metrics := range m.serviceMetrics {
		metrics.mutex.RLock()
		result[service] = &ServiceMetrics{
			Name:            metrics.Name,
			RequestCount:    metrics.RequestCount,
			ErrorCount:      metrics.ErrorCount,
			TotalDuration:   metrics.TotalDuration,
			LastRequestTime: metrics.LastRequestTime,
			AverageLatency:  metrics.AverageLatency,
			ErrorRate:       metrics.ErrorRate,
		}
		metrics.mutex.RUnlock()
	}

	return result
}

// GetDetailedMetrics returns detailed metrics for management endpoints
func (m *Monitor) GetDetailedMetrics() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Collect service metrics
	serviceMetrics := make(map[string]interface{})
	for service, metrics := range m.serviceMetrics {
		metrics.mutex.RLock()
		serviceMetrics[service] = map[string]interface{}{
			"request_count":     metrics.RequestCount,
			"error_count":       metrics.ErrorCount,
			"average_latency":   metrics.AverageLatency.Milliseconds(),
			"error_rate":        metrics.ErrorRate,
			"last_request_time": metrics.LastRequestTime,
		}
		metrics.mutex.RUnlock()
	}

	return map[string]interface{}{
		"total_requests":  m.totalRequests,
		"total_errors":    m.totalErrors,
		"error_rate":      m.calculateOverallErrorRate(),
		"services":        serviceMetrics,
		"uptime":          time.Since(time.Now()).Seconds(), // Placeholder
		"timestamp":       time.Now().UTC(),
	}
}

// calculateOverallErrorRate calculates the overall error rate
func (m *Monitor) calculateOverallErrorRate() float64 {
	if m.totalRequests == 0 {
		return 0.0
	}
	return float64(m.totalErrors) / float64(m.totalRequests)
}

// StartReporting starts the monitoring reporting goroutine
func (m *Monitor) StartReporting() {
	go m.reportingLoop()
}

// reportingLoop runs the reporting loop
func (m *Monitor) reportingLoop() {
	ticker := time.NewTicker(m.config.ReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.generateReport()
		case <-m.stopChan:
			return
		}
	}
}

// generateReport generates and logs monitoring report
func (m *Monitor) generateReport() {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Generate report
	report := map[string]interface{}{
		"timestamp":      time.Now().UTC(),
		"total_requests": m.totalRequests,
		"total_errors":   m.totalErrors,
		"error_rate":     m.calculateOverallErrorRate(),
		"services":       m.getServiceSummary(),
	}

	// Log report
	m.logger.Info("Monitoring report",
		zap.Any("report", report),
	)

	// TODO: Send report to external monitoring systems
	// - Prometheus Push Gateway
	// - Jaeger
	// - Custom monitoring endpoints
}

// getServiceSummary returns a summary of service metrics
func (m *Monitor) getServiceSummary() map[string]interface{} {
	summary := make(map[string]interface{})
	
	for service, metrics := range m.serviceMetrics {
		metrics.mutex.RLock()
		summary[service] = map[string]interface{}{
			"requests":       metrics.RequestCount,
			"errors":         metrics.ErrorCount,
			"error_rate":     metrics.ErrorRate,
			"avg_latency_ms": metrics.AverageLatency.Milliseconds(),
		}
		metrics.mutex.RUnlock()
	}

	return summary
}

// AddHealthCheck adds a health check function
func (m *Monitor) AddHealthCheck(name string, check HealthCheck) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.healthChecks[name] = check
}

// RunHealthChecks runs all registered health checks
func (m *Monitor) RunHealthChecks() map[string]bool {
	m.mutex.RLock()
	checks := make(map[string]HealthCheck)
	for name, check := range m.healthChecks {
		checks[name] = check
	}
	m.mutex.RUnlock()

	results := make(map[string]bool)
	for name, check := range checks {
		results[name] = check() == nil
	}

	return results
}

// GetHealthStatus returns overall health status
func (m *Monitor) GetHealthStatus() map[string]interface{} {
	healthChecks := m.RunHealthChecks()
	
	healthy := true
	for _, result := range healthChecks {
		if !result {
			healthy = false
			break
		}
	}

	return map[string]interface{}{
		"healthy":       healthy,
		"checks":        healthChecks,
		"total_requests": m.GetTotalRequests(),
		"total_errors":   m.GetTotalErrors(),
		"error_rate":     m.calculateOverallErrorRate(),
		"timestamp":      time.Now().UTC(),
	}
}

// Stop stops the monitoring service
func (m *Monitor) Stop() {
	close(m.stopChan)
	m.logger.Info("Monitoring service stopped")
}

// SetActiveConnections sets the number of active connections
func (m *Monitor) SetActiveConnections(count int) {
	m.metrics.ActiveConnections.Set(float64(count))
}

// IncrementActiveConnections increments active connections
func (m *Monitor) IncrementActiveConnections() {
	m.metrics.ActiveConnections.Inc()
}

// DecrementActiveConnections decrements active connections
func (m *Monitor) DecrementActiveConnections() {
	m.metrics.ActiveConnections.Dec()
}

// RecordHTTPRequest records an HTTP request with method, path, and status
func (m *Monitor) RecordHTTPRequest(method, path, status string, duration time.Duration) {
	m.metrics.RequestsTotal.WithLabelValues(method, path, status).Inc()
	m.metrics.RequestDuration.WithLabelValues(method, path, "gateway").Observe(duration.Seconds())
}

// RecordServiceRequest records a request to a specific service
func (m *Monitor) RecordServiceRequest(service, method, path string, duration time.Duration) {
	m.metrics.RequestDuration.WithLabelValues(method, path, service).Observe(duration.Seconds())
}

// GetPrometheusMetrics returns Prometheus metrics handler
func (m *Monitor) GetPrometheusMetrics() *Metrics {
	return m.metrics
}

// Reset resets all metrics (for testing)
func (m *Monitor) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.totalRequests = 0
	m.totalErrors = 0

	for _, metrics := range m.serviceMetrics {
		metrics.mutex.Lock()
		metrics.RequestCount = 0
		metrics.ErrorCount = 0
		metrics.TotalDuration = 0
		metrics.AverageLatency = 0
		metrics.ErrorRate = 0.0
		metrics.mutex.Unlock()
	}
}

// IsHealthy returns whether the monitoring service is healthy
func (m *Monitor) IsHealthy() bool {
	// Check if critical components are working
	return m.config.Enabled && len(m.serviceMetrics) > 0
} 