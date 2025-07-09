package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"iaros/api_gateway/src/config"
)

// ServiceRegistry manages service discovery and health monitoring
type ServiceRegistry struct {
	services     map[string]*Service
	cache        *redis.Client
	logger       *zap.Logger
	config       *config.ServiceRegistryConfig
	httpClient   *http.Client
	mutex        sync.RWMutex
	ready        bool
	stopChan     chan struct{}
}

// Service represents a registered service
type Service struct {
	Name         string            `json:"name"`
	ID           string            `json:"id"`
	Address      string            `json:"address"`
	Port         int               `json:"port"`
	HealthCheck  string            `json:"health_check"`
	Tags         []string          `json:"tags"`
	Metadata     map[string]string `json:"metadata"`
	LastSeen     time.Time         `json:"last_seen"`
	IsHealthy    bool              `json:"is_healthy"`
	HealthScore  float64           `json:"health_score"`
	ResponseTime time.Duration     `json:"response_time"`
	ErrorRate    float64           `json:"error_rate"`
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Service      string        `json:"service"`
	IsHealthy    bool          `json:"is_healthy"`
	ResponseTime time.Duration `json:"response_time"`
	Error        string        `json:"error,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(cfg *config.Config) (*ServiceRegistry, error) {
	logger, _ := zap.NewProduction()

	// Initialize Redis client
	cache := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.ServiceDB,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := cache.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// HTTP client for health checks
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	registry := &ServiceRegistry{
		services:   make(map[string]*Service),
		cache:      cache,
		logger:     logger,
		config:     &cfg.ServiceRegistry,
		httpClient: httpClient,
		ready:      false,
		stopChan:   make(chan struct{}),
	}

	// Initialize with configured services
	registry.initializeServices(cfg)

	// Load services from cache
	if err := registry.loadFromCache(); err != nil {
		logger.Warn("Failed to load services from cache", zap.Error(err))
	}

	registry.ready = true
	return registry, nil
}

// initializeServices initializes services from configuration
func (sr *ServiceRegistry) initializeServices(cfg *config.Config) {
	// Pricing Service
	sr.services["pricing-service"] = &Service{
		Name:         "pricing-service",
		ID:           "pricing-service-001",
		Address:      cfg.Services.Pricing.Host,
		Port:         cfg.Services.Pricing.Port,
		HealthCheck:  fmt.Sprintf("http://%s:%d/health", cfg.Services.Pricing.Host, cfg.Services.Pricing.Port),
		Tags:         []string{"pricing", "core"},
		Metadata:     map[string]string{"version": "1.0.0", "environment": cfg.Environment},
		LastSeen:     time.Now(),
		IsHealthy:    true,
		HealthScore:  1.0,
		ResponseTime: 0,
		ErrorRate:    0.0,
	}

	// Forecasting Service
	sr.services["forecasting-service"] = &Service{
		Name:         "forecasting-service",
		ID:           "forecasting-service-001",
		Address:      cfg.Services.Forecasting.Host,
		Port:         cfg.Services.Forecasting.Port,
		HealthCheck:  fmt.Sprintf("http://%s:%d/health", cfg.Services.Forecasting.Host, cfg.Services.Forecasting.Port),
		Tags:         []string{"forecasting", "ml", "core"},
		Metadata:     map[string]string{"version": "1.0.0", "environment": cfg.Environment},
		LastSeen:     time.Now(),
		IsHealthy:    true,
		HealthScore:  1.0,
		ResponseTime: 0,
		ErrorRate:    0.0,
	}

	// Offer Service
	sr.services["offer-service"] = &Service{
		Name:         "offer-service",
		ID:           "offer-service-001",
		Address:      cfg.Services.Offer.Host,
		Port:         cfg.Services.Offer.Port,
		HealthCheck:  fmt.Sprintf("http://%s:%d/health", cfg.Services.Offer.Host, cfg.Services.Offer.Port),
		Tags:         []string{"offer", "core"},
		Metadata:     map[string]string{"version": "1.0.0", "environment": cfg.Environment},
		LastSeen:     time.Now(),
		IsHealthy:    true,
		HealthScore:  1.0,
		ResponseTime: 0,
		ErrorRate:    0.0,
	}

	// Order Service
	sr.services["order-service"] = &Service{
		Name:         "order-service",
		ID:           "order-service-001",
		Address:      cfg.Services.Order.Host,
		Port:         cfg.Services.Order.Port,
		HealthCheck:  fmt.Sprintf("http://%s:%d/health", cfg.Services.Order.Host, cfg.Services.Order.Port),
		Tags:         []string{"order", "core"},
		Metadata:     map[string]string{"version": "1.0.0", "environment": cfg.Environment},
		LastSeen:     time.Now(),
		IsHealthy:    true,
		HealthScore:  1.0,
		ResponseTime: 0,
		ErrorRate:    0.0,
	}

	// Distribution Service
	sr.services["distribution-service"] = &Service{
		Name:         "distribution-service",
		ID:           "distribution-service-001",
		Address:      cfg.Services.Distribution.Host,
		Port:         cfg.Services.Distribution.Port,
		HealthCheck:  fmt.Sprintf("http://%s:%d/health", cfg.Services.Distribution.Host, cfg.Services.Distribution.Port),
		Tags:         []string{"distribution", "core"},
		Metadata:     map[string]string{"version": "1.0.0", "environment": cfg.Environment},
		LastSeen:     time.Now(),
		IsHealthy:    true,
		HealthScore:  1.0,
		ResponseTime: 0,
		ErrorRate:    0.0,
	}

	// Ancillary Service
	sr.services["ancillary-service"] = &Service{
		Name:         "ancillary-service",
		ID:           "ancillary-service-001",
		Address:      cfg.Services.Ancillary.Host,
		Port:         cfg.Services.Ancillary.Port,
		HealthCheck:  fmt.Sprintf("http://%s:%d/health", cfg.Services.Ancillary.Host, cfg.Services.Ancillary.Port),
		Tags:         []string{"ancillary", "core"},
		Metadata:     map[string]string{"version": "1.0.0", "environment": cfg.Environment},
		LastSeen:     time.Now(),
		IsHealthy:    true,
		HealthScore:  1.0,
		ResponseTime: 0,
		ErrorRate:    0.0,
	}

	// User Management Service
	sr.services["user-service"] = &Service{
		Name:         "user-service",
		ID:           "user-service-001",
		Address:      cfg.Services.UserManagement.Host,
		Port:         cfg.Services.UserManagement.Port,
		HealthCheck:  fmt.Sprintf("http://%s:%d/health", cfg.Services.UserManagement.Host, cfg.Services.UserManagement.Port),
		Tags:         []string{"user", "auth", "core"},
		Metadata:     map[string]string{"version": "1.0.0", "environment": cfg.Environment},
		LastSeen:     time.Now(),
		IsHealthy:    true,
		HealthScore:  1.0,
		ResponseTime: 0,
		ErrorRate:    0.0,
	}

	// Network Planning Service
	sr.services["network-service"] = &Service{
		Name:         "network-service",
		ID:           "network-service-001",
		Address:      cfg.Services.NetworkPlanning.Host,
		Port:         cfg.Services.NetworkPlanning.Port,
		HealthCheck:  fmt.Sprintf("http://%s:%d/health", cfg.Services.NetworkPlanning.Host, cfg.Services.NetworkPlanning.Port),
		Tags:         []string{"network", "planning"},
		Metadata:     map[string]string{"version": "1.0.0", "environment": cfg.Environment},
		LastSeen:     time.Now(),
		IsHealthy:    true,
		HealthScore:  1.0,
		ResponseTime: 0,
		ErrorRate:    0.0,
	}

	// Procurement Service
	sr.services["procurement-service"] = &Service{
		Name:         "procurement-service",
		ID:           "procurement-service-001",
		Address:      cfg.Services.Procurement.Host,
		Port:         cfg.Services.Procurement.Port,
		HealthCheck:  fmt.Sprintf("http://%s:%d/health", cfg.Services.Procurement.Host, cfg.Services.Procurement.Port),
		Tags:         []string{"procurement", "finance"},
		Metadata:     map[string]string{"version": "1.0.0", "environment": cfg.Environment},
		LastSeen:     time.Now(),
		IsHealthy:    true,
		HealthScore:  1.0,
		ResponseTime: 0,
		ErrorRate:    0.0,
	}

	// Promotion Service
	sr.services["promotion-service"] = &Service{
		Name:         "promotion-service",
		ID:           "promotion-service-001",
		Address:      cfg.Services.Promotion.Host,
		Port:         cfg.Services.Promotion.Port,
		HealthCheck:  fmt.Sprintf("http://%s:%d/health", cfg.Services.Promotion.Host, cfg.Services.Promotion.Port),
		Tags:         []string{"promotion", "marketing"},
		Metadata:     map[string]string{"version": "1.0.0", "environment": cfg.Environment},
		LastSeen:     time.Now(),
		IsHealthy:    true,
		HealthScore:  1.0,
		ResponseTime: 0,
		ErrorRate:    0.0,
	}

	sr.logger.Info("Initialized services", zap.Int("count", len(sr.services)))
}

// GetService returns a service by name
func (sr *ServiceRegistry) GetService(name string) (*Service, error) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	service, exists := sr.services[name]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", name)
	}

	return service, nil
}

// GetAllServices returns all registered services
func (sr *ServiceRegistry) GetAllServices() map[string]*Service {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	services := make(map[string]*Service)
	for name, service := range sr.services {
		services[name] = service
	}

	return services
}

// GetHealthyServices returns only healthy services
func (sr *ServiceRegistry) GetHealthyServices() map[string]*Service {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	healthy := make(map[string]*Service)
	for name, service := range sr.services {
		if service.IsHealthy {
			healthy[name] = service
		}
	}

	return healthy
}

// GetServicesByTag returns services with a specific tag
func (sr *ServiceRegistry) GetServicesByTag(tag string) map[string]*Service {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	tagged := make(map[string]*Service)
	for name, service := range sr.services {
		for _, serviceTag := range service.Tags {
			if serviceTag == tag {
				tagged[name] = service
				break
			}
		}
	}

	return tagged
}

// GetHealthStatus returns health status of all services
func (sr *ServiceRegistry) GetHealthStatus() map[string]bool {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	status := make(map[string]bool)
	for name, service := range sr.services {
		status[name] = service.IsHealthy
	}

	return status
}

// StartHealthChecking starts the health checking goroutine
func (sr *ServiceRegistry) StartHealthChecking() {
	go sr.healthCheckLoop()
}

// healthCheckLoop runs the health check loop
func (sr *ServiceRegistry) healthCheckLoop() {
	ticker := time.NewTicker(sr.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sr.performHealthChecks()
		case <-sr.stopChan:
			return
		}
	}
}

// performHealthChecks performs health checks on all services
func (sr *ServiceRegistry) performHealthChecks() {
	sr.mutex.RLock()
	services := make(map[string]*Service)
	for name, service := range sr.services {
		services[name] = service
	}
	sr.mutex.RUnlock()

	var wg sync.WaitGroup
	results := make(chan HealthCheckResult, len(services))

	for _, service := range services {
		wg.Add(1)
		go func(svc *Service) {
			defer wg.Done()
			result := sr.performHealthCheck(svc)
			results <- result
		}(service)
	}

	// Wait for all health checks to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results
	for result := range results {
		sr.updateServiceHealth(result)
	}

	// Save to cache
	if err := sr.saveToCache(); err != nil {
		sr.logger.Error("Failed to save services to cache", zap.Error(err))
	}
}

// performHealthCheck performs a health check on a single service
func (sr *ServiceRegistry) performHealthCheck(service *Service) HealthCheckResult {
	start := time.Now()
	
	resp, err := sr.httpClient.Get(service.HealthCheck)
	responseTime := time.Since(start)

	result := HealthCheckResult{
		Service:      service.Name,
		ResponseTime: responseTime,
		Timestamp:    time.Now(),
	}

	if err != nil {
		result.IsHealthy = false
		result.Error = err.Error()
		return result
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.IsHealthy = true
	} else {
		result.IsHealthy = false
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return result
}

// updateServiceHealth updates service health based on health check result
func (sr *ServiceRegistry) updateServiceHealth(result HealthCheckResult) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	service, exists := sr.services[result.Service]
	if !exists {
		return
	}

	service.IsHealthy = result.IsHealthy
	service.ResponseTime = result.ResponseTime
	service.LastSeen = result.Timestamp

	// Update health score based on recent performance
	if result.IsHealthy {
		service.HealthScore = (service.HealthScore + 1.0) / 2.0
		if service.HealthScore > 1.0 {
			service.HealthScore = 1.0
		}
	} else {
		service.HealthScore = service.HealthScore * 0.8
		if service.HealthScore < 0.0 {
			service.HealthScore = 0.0
		}
	}

	// Calculate error rate (simplified)
	if result.IsHealthy {
		service.ErrorRate = service.ErrorRate * 0.9
	} else {
		service.ErrorRate = (service.ErrorRate + 1.0) / 2.0
		if service.ErrorRate > 1.0 {
			service.ErrorRate = 1.0
		}
	}

	sr.logger.Debug("Updated service health",
		zap.String("service", service.Name),
		zap.Bool("healthy", service.IsHealthy),
		zap.Float64("health_score", service.HealthScore),
		zap.Duration("response_time", service.ResponseTime),
	)
}

// HealthCheckAll performs immediate health checks on all services
func (sr *ServiceRegistry) HealthCheckAll() map[string]HealthCheckResult {
	sr.mutex.RLock()
	services := make(map[string]*Service)
	for name, service := range sr.services {
		services[name] = service
	}
	sr.mutex.RUnlock()

	results := make(map[string]HealthCheckResult)
	var wg sync.WaitGroup
	var mutex sync.Mutex

	for name, service := range services {
		wg.Add(1)
		go func(name string, svc *Service) {
			defer wg.Done()
			result := sr.performHealthCheck(svc)
			
			mutex.Lock()
			results[name] = result
			mutex.Unlock()
		}(name, service)
	}

	wg.Wait()
	return results
}

// Refresh refreshes the service registry
func (sr *ServiceRegistry) Refresh() error {
	sr.logger.Info("Refreshing service registry")
	
	// Reload from cache
	if err := sr.loadFromCache(); err != nil {
		sr.logger.Warn("Failed to reload from cache", zap.Error(err))
	}

	// Perform immediate health checks
	sr.performHealthChecks()

	return nil
}

// saveToCache saves services to Redis cache
func (sr *ServiceRegistry) saveToCache() error {
	ctx := context.Background()
	
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	for name, service := range sr.services {
		serviceJSON, err := json.Marshal(service)
		if err != nil {
			return fmt.Errorf("failed to marshal service %s: %w", name, err)
		}

		key := fmt.Sprintf("service:%s", name)
		if err := sr.cache.Set(ctx, key, serviceJSON, time.Hour).Err(); err != nil {
			return fmt.Errorf("failed to save service %s to cache: %w", name, err)
		}
	}

	return nil
}

// loadFromCache loads services from Redis cache
func (sr *ServiceRegistry) loadFromCache() error {
	ctx := context.Background()
	
	keys, err := sr.cache.Keys(ctx, "service:*").Result()
	if err != nil {
		return fmt.Errorf("failed to get service keys: %w", err)
	}

	for _, key := range keys {
		serviceJSON, err := sr.cache.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var service Service
		if err := json.Unmarshal([]byte(serviceJSON), &service); err != nil {
			continue
		}

		sr.mutex.Lock()
		sr.services[service.Name] = &service
		sr.mutex.Unlock()
	}

	return nil
}

// IsReady returns whether the service registry is ready
func (sr *ServiceRegistry) IsReady() bool {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()
	return sr.ready
}

// Stop stops the service registry
func (sr *ServiceRegistry) Stop() {
	close(sr.stopChan)
	sr.logger.Info("Service registry stopped")
}

// GetServiceURL returns the full URL for a service
func (sr *ServiceRegistry) GetServiceURL(serviceName string) (string, error) {
	service, err := sr.GetService(serviceName)
	if err != nil {
		return "", err
	}

	if !service.IsHealthy {
		return "", fmt.Errorf("service %s is not healthy", serviceName)
	}

	return fmt.Sprintf("http://%s:%d", service.Address, service.Port), nil
}

// GetBestService returns the best service instance based on health score
func (sr *ServiceRegistry) GetBestService(serviceName string) (*Service, error) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	// For simplicity, return the single service instance
	// In a real implementation, this would select from multiple instances
	service, exists := sr.services[serviceName]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", serviceName)
	}

	if !service.IsHealthy {
		return nil, fmt.Errorf("service %s is not healthy", serviceName)
	}

	return service, nil
}

// GetServiceMetrics returns metrics for a service
func (sr *ServiceRegistry) GetServiceMetrics(serviceName string) (map[string]interface{}, error) {
	service, err := sr.GetService(serviceName)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"name":          service.Name,
		"is_healthy":    service.IsHealthy,
		"health_score":  service.HealthScore,
		"response_time": service.ResponseTime.Milliseconds(),
		"error_rate":    service.ErrorRate,
		"last_seen":     service.LastSeen,
	}, nil
} 