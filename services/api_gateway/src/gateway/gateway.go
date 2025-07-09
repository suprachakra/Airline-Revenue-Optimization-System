package gateway

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	
	"iaros/api_gateway/src/auth"
	"iaros/api_gateway/src/circuit"
	"iaros/api_gateway/src/config"
	"iaros/api_gateway/src/middleware"
	"iaros/api_gateway/src/monitor"
	"iaros/api_gateway/src/ratelimit"
	"iaros/api_gateway/src/registry"
)

// Gateway represents the main API Gateway instance
type Gateway struct {
	config           *config.Config
	router           *mux.Router
	server           *http.Server
	serviceRegistry  *registry.ServiceRegistry
	authService      *auth.AuthService
	rateLimiter      *ratelimit.RateLimiter
	circuitBreaker   *circuit.CircuitBreakerManager
	monitor          *monitor.Monitor
	loadBalancer     *LoadBalancer
	requestCounter   prometheus.Counter
	requestDuration  prometheus.Histogram
	mutex            sync.RWMutex
	shuttingDown     bool
}

// ServiceTarget represents a backend service configuration
type ServiceTarget struct {
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Weight      int               `json:"weight"`
	HealthCheck string            `json:"health_check"`
	Timeout     time.Duration     `json:"timeout"`
	Retries     int               `json:"retries"`
	Headers     map[string]string `json:"headers"`
	IsHealthy   bool              `json:"is_healthy"`
	LastCheck   time.Time         `json:"last_check"`
}

// Route represents a gateway route configuration
type Route struct {
	Path         string          `json:"path"`
	Method       string          `json:"method"`
	Service      string          `json:"service"`
	Targets      []ServiceTarget `json:"targets"`
	AuthRequired bool            `json:"auth_required"`
	RateLimit    int             `json:"rate_limit"`
	Timeout      time.Duration   `json:"timeout"`
	Transform    *Transform      `json:"transform,omitempty"`
	Cache        *CacheConfig    `json:"cache,omitempty"`
}

// Transform represents request/response transformation rules
type Transform struct {
	RequestHeaders  map[string]string `json:"request_headers"`
	ResponseHeaders map[string]string `json:"response_headers"`
	RequestBody     string            `json:"request_body"`
	ResponseBody    string            `json:"response_body"`
}

// CacheConfig represents caching configuration for routes
type CacheConfig struct {
	Enabled bool          `json:"enabled"`
	TTL     time.Duration `json:"ttl"`
	Key     string        `json:"key"`
}

// LoadBalancer handles load balancing strategies
type LoadBalancer struct {
	strategy string
	mutex    sync.RWMutex
}

// NewGateway creates a new API Gateway instance
func NewGateway(cfg *config.Config) (*Gateway, error) {
	// Initialize Prometheus metrics
	requestCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gateway_requests_total",
		Help: "Total number of requests processed by the gateway",
	})

	requestDuration := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "gateway_request_duration_seconds",
		Help:    "Request duration in seconds",
		Buckets: prometheus.DefBuckets,
	})

	prometheus.MustRegister(requestCounter, requestDuration)

	// Initialize core components
	serviceRegistry, err := registry.NewServiceRegistry(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create service registry: %w", err)
	}

	authService, err := auth.NewAuthService(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth service: %w", err)
	}

	rateLimiter, err := ratelimit.NewRateLimiter(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create rate limiter: %w", err)
	}

	circuitBreaker, err := circuit.NewCircuitBreakerManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create circuit breaker: %w", err)
	}

	monitor, err := monitor.NewMonitor(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitor: %w", err)
	}

	loadBalancer := &LoadBalancer{
		strategy: cfg.LoadBalancer.Strategy,
	}

	gateway := &Gateway{
		config:          cfg,
		router:          mux.NewRouter(),
		serviceRegistry: serviceRegistry,
		authService:     authService,
		rateLimiter:     rateLimiter,
		circuitBreaker:  circuitBreaker,
		monitor:         monitor,
		loadBalancer:    loadBalancer,
		requestCounter:  requestCounter,
		requestDuration: requestDuration,
	}

	// Setup routes and middleware
	gateway.setupMiddleware()
	gateway.setupRoutes()
	gateway.setupManagementEndpoints()

	// Configure HTTP server
	gateway.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      gateway.router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
			CipherSuites: []uint16{
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_CHACHA20_POLY1305_SHA256,
			},
		},
	}

	return gateway, nil
}

// setupMiddleware configures the middleware stack
func (g *Gateway) setupMiddleware() {
	// Panic recovery middleware
	g.router.Use(middleware.PanicRecovery())

	// Security headers middleware
	g.router.Use(middleware.SecurityHeaders())

	// CORS middleware
	g.router.Use(handlers.CORS(
		handlers.AllowedOrigins(g.config.CORS.AllowedOrigins),
		handlers.AllowedMethods(g.config.CORS.AllowedMethods),
		handlers.AllowedHeaders(g.config.CORS.AllowedHeaders),
		handlers.MaxAge(int(g.config.CORS.MaxAge.Seconds())),
	))

	// Request ID middleware
	g.router.Use(middleware.RequestID())

	// Logging middleware
	g.router.Use(middleware.RequestLogging())

	// Metrics middleware
	g.router.Use(g.metricsMiddleware)

	// Rate limiting middleware
	g.router.Use(g.rateLimitMiddleware)

	// Circuit breaker middleware
	g.router.Use(g.circuitBreakerMiddleware)
}

// setupRoutes configures the main routing
func (g *Gateway) setupRoutes() {
	// API versioning
	v1 := g.router.PathPrefix("/api/v1").Subrouter()

	// Service routes with authentication
	g.setupServiceRoutes(v1)

	// Health and status endpoints (no auth required)
	g.router.HandleFunc("/health", g.healthHandler).Methods("GET")
	g.router.HandleFunc("/ready", g.readinessHandler).Methods("GET")
	g.router.HandleFunc("/status", g.statusHandler).Methods("GET")

	// Metrics endpoint
	g.router.Handle("/metrics", promhttp.Handler()).Methods("GET")
}

// setupServiceRoutes configures routes for backend services
func (g *Gateway) setupServiceRoutes(router *mux.Router) {
	routes := []Route{
		// Pricing Service Routes
		{
			Path:         "/pricing/{action:.*}",
			Method:       "GET,POST,PUT",
			Service:      "pricing-service",
			AuthRequired: true,
			RateLimit:    100,
			Timeout:      5 * time.Second,
			Targets: []ServiceTarget{
				{
					Name:        "pricing-service-1",
					URL:         g.config.Services.Pricing.Primary,
					Weight:      70,
					HealthCheck: "/health",
					Timeout:     5 * time.Second,
					Retries:     3,
				},
				{
					Name:        "pricing-service-2",
					URL:         g.config.Services.Pricing.Secondary,
					Weight:      30,
					HealthCheck: "/health",
					Timeout:     5 * time.Second,
					Retries:     3,
				},
			},
			Cache: &CacheConfig{
				Enabled: true,
				TTL:     15 * time.Minute,
				Key:     "pricing:{path}:{query}",
			},
		},
		// Forecasting Service Routes
		{
			Path:         "/forecasting/{action:.*}",
			Method:       "GET,POST",
			Service:      "forecasting-service",
			AuthRequired: true,
			RateLimit:    50,
			Timeout:      10 * time.Second,
			Targets: []ServiceTarget{
				{
					Name:        "forecasting-service-1",
					URL:         g.config.Services.Forecasting.Primary,
					Weight:      100,
					HealthCheck: "/health",
					Timeout:     10 * time.Second,
					Retries:     2,
				},
			},
			Cache: &CacheConfig{
				Enabled: true,
				TTL:     60 * time.Minute,
				Key:     "forecasting:{path}:{query}",
			},
		},
		// Offer Service Routes
		{
			Path:         "/offers/{action:.*}",
			Method:       "GET,POST,PUT,DELETE",
			Service:      "offer-service",
			AuthRequired: true,
			RateLimit:    200,
			Timeout:      3 * time.Second,
			Targets: []ServiceTarget{
				{
					Name:        "offer-service-1",
					URL:         g.config.Services.Offer.Primary,
					Weight:      100,
					HealthCheck: "/health",
					Timeout:     3 * time.Second,
					Retries:     3,
				},
			},
		},
		// Order Management Service Routes
		{
			Path:         "/orders/{action:.*}",
			Method:       "GET,POST,PUT,DELETE",
			Service:      "order-service",
			AuthRequired: true,
			RateLimit:    150,
			Timeout:      7 * time.Second,
			Targets: []ServiceTarget{
				{
					Name:        "order-service-1",
					URL:         g.config.Services.Order.Primary,
					Weight:      100,
					HealthCheck: "/health",
					Timeout:     7 * time.Second,
					Retries:     3,
				},
			},
		},
		// Distribution Service Routes
		{
			Path:         "/distribution/{action:.*}",
			Method:       "GET,POST,PUT",
			Service:      "distribution-service",
			AuthRequired: true,
			RateLimit:    300,
			Timeout:      5 * time.Second,
			Targets: []ServiceTarget{
				{
					Name:        "distribution-service-1",
					URL:         g.config.Services.Distribution.Primary,
					Weight:      100,
					HealthCheck: "/health",
					Timeout:     5 * time.Second,
					Retries:     2,
				},
			},
		},
		// Ancillary Service Routes
		{
			Path:         "/ancillary/{action:.*}",
			Method:       "GET,POST,PUT,DELETE",
			Service:      "ancillary-service",
			AuthRequired: true,
			RateLimit:    250,
			Timeout:      4 * time.Second,
			Targets: []ServiceTarget{
				{
					Name:        "ancillary-service-1",
					URL:         g.config.Services.Ancillary.Primary,
					Weight:      100,
					HealthCheck: "/health",
					Timeout:     4 * time.Second,
					Retries:     3,
				},
			},
		},
		// User Management Service Routes
		{
			Path:         "/users/{action:.*}",
			Method:       "GET,POST,PUT,DELETE",
			Service:      "user-service",
			AuthRequired: true,
			RateLimit:    100,
			Timeout:      3 * time.Second,
			Targets: []ServiceTarget{
				{
					Name:        "user-service-1",
					URL:         g.config.Services.UserManagement.Primary,
					Weight:      100,
					HealthCheck: "/health",
					Timeout:     3 * time.Second,
					Retries:     3,
				},
			},
		},
		// Network Planning Service Routes
		{
			Path:         "/network/{action:.*}",
			Method:       "GET,POST,PUT",
			Service:      "network-service",
			AuthRequired: true,
			RateLimit:    75,
			Timeout:      8 * time.Second,
			Targets: []ServiceTarget{
				{
					Name:        "network-service-1",
					URL:         g.config.Services.NetworkPlanning.Primary,
					Weight:      100,
					HealthCheck: "/health",
					Timeout:     8 * time.Second,
					Retries:     2,
				},
			},
		},
		// Procurement Service Routes
		{
			Path:         "/procurement/{action:.*}",
			Method:       "GET,POST,PUT,DELETE",
			Service:      "procurement-service",
			AuthRequired: true,
			RateLimit:    50,
			Timeout:      6 * time.Second,
			Targets: []ServiceTarget{
				{
					Name:        "procurement-service-1",
					URL:         g.config.Services.Procurement.Primary,
					Weight:      100,
					HealthCheck: "/health",
					Timeout:     6 * time.Second,
					Retries:     3,
				},
			},
		},
		// Promotion Service Routes
		{
			Path:         "/promotions/{action:.*}",
			Method:       "GET,POST,PUT,DELETE",
			Service:      "promotion-service",
			AuthRequired: true,
			RateLimit:    100,
			Timeout:      4 * time.Second,
			Targets: []ServiceTarget{
				{
					Name:        "promotion-service-1",
					URL:         g.config.Services.Promotion.Primary,
					Weight:      100,
					HealthCheck: "/health",
					Timeout:     4 * time.Second,
					Retries:     3,
				},
			},
		},
	}

	// Register routes
	for _, route := range routes {
		g.registerRoute(router, route)
	}
}

// registerRoute registers a single route with all middleware
func (g *Gateway) registerRoute(router *mux.Router, route Route) {
	handler := g.createProxyHandler(route)

	// Apply authentication middleware if required
	if route.AuthRequired {
		handler = g.authService.AuthRequired(handler)
	}

	// Apply route-specific rate limiting
	if route.RateLimit > 0 {
		handler = g.rateLimiter.RouteLimit(route.RateLimit)(handler)
	}

	// Apply caching if configured
	if route.Cache != nil && route.Cache.Enabled {
		handler = middleware.Cache(route.Cache.TTL, route.Cache.Key)(handler)
	}

	// Apply request transformation if configured
	if route.Transform != nil {
		handler = middleware.Transform(route.Transform)(handler)
	}

	// Register the route
	router.HandleFunc(route.Path, handler).Methods(splitMethods(route.Method)...)
}

// createProxyHandler creates a reverse proxy handler for a route
func (g *Gateway) createProxyHandler(route Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Select target service using load balancing
		target, err := g.selectTarget(route.Targets)
		if err != nil {
			g.handleServiceError(w, r, "Service Unavailable", http.StatusServiceUnavailable)
			return
		}

		// Parse target URL
		targetURL, err := url.Parse(target.URL)
		if err != nil {
			g.handleServiceError(w, r, "Invalid Service URL", http.StatusInternalServerError)
			return
		}

		// Create reverse proxy
		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		
		// Customize the director function
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			
			// Add service-specific headers
			for key, value := range target.Headers {
				req.Header.Set(key, value)
			}
			
			// Add gateway headers
			req.Header.Set("X-Gateway-Request-ID", middleware.GetRequestID(r.Context()))
			req.Header.Set("X-Gateway-Start-Time", start.Format(time.RFC3339Nano))
			req.Header.Set("X-Forwarded-For", r.RemoteAddr)
			req.Header.Set("X-Forwarded-Proto", r.URL.Scheme)
		}

		// Customize error handler
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			g.monitor.RecordError(route.Service, err)
			g.handleServiceError(w, r, "Service Error", http.StatusBadGateway)
		}

		// Set timeout
		ctx, cancel := context.WithTimeout(r.Context(), route.Timeout)
		defer cancel()

		// Execute proxy request
		proxy.ServeHTTP(w, r.WithContext(ctx))

		// Record metrics
		duration := time.Since(start)
		g.monitor.RecordRequest(route.Service, duration)
	}
}

// selectTarget selects a target service using load balancing strategy
func (g *Gateway) selectTarget(targets []ServiceTarget) (*ServiceTarget, error) {
	g.loadBalancer.mutex.RLock()
	defer g.loadBalancer.mutex.RUnlock()

	// Filter healthy targets
	healthyTargets := make([]ServiceTarget, 0)
	for _, target := range targets {
		if target.IsHealthy {
			healthyTargets = append(healthyTargets, target)
		}
	}

	if len(healthyTargets) == 0 {
		return nil, fmt.Errorf("no healthy targets available")
	}

	// Apply load balancing strategy
	switch g.loadBalancer.strategy {
	case "round_robin":
		return g.roundRobinSelection(healthyTargets), nil
	case "weighted":
		return g.weightedSelection(healthyTargets), nil
	case "least_connections":
		return g.leastConnectionsSelection(healthyTargets), nil
	default:
		return &healthyTargets[0], nil
	}
}

// Load balancing implementations
func (g *Gateway) roundRobinSelection(targets []ServiceTarget) *ServiceTarget {
	// Simple round-robin implementation
	// In a real implementation, this would use a counter
	return &targets[0]
}

func (g *Gateway) weightedSelection(targets []ServiceTarget) *ServiceTarget {
	totalWeight := 0
	for _, target := range targets {
		totalWeight += target.Weight
	}
	
	if totalWeight == 0 {
		return &targets[0]
	}

	// Select based on weight (simplified implementation)
	for _, target := range targets {
		if target.Weight > 0 {
			return &target
		}
	}
	
	return &targets[0]
}

func (g *Gateway) leastConnectionsSelection(targets []ServiceTarget) *ServiceTarget {
	// In a real implementation, this would track active connections
	return &targets[0]
}

// setupManagementEndpoints sets up management and monitoring endpoints
func (g *Gateway) setupManagementEndpoints() {
	mgmt := g.router.PathPrefix("/management").Subrouter()

	// Admin endpoints (require admin authentication)
	mgmt.HandleFunc("/routes", g.routesHandler).Methods("GET")
	mgmt.HandleFunc("/services", g.servicesHandler).Methods("GET")
	mgmt.HandleFunc("/config", g.configHandler).Methods("GET")
	mgmt.HandleFunc("/metrics/detailed", g.detailedMetricsHandler).Methods("GET")
	
	// Circuit breaker management
	mgmt.HandleFunc("/circuit-breakers", g.circuitBreakersHandler).Methods("GET")
	mgmt.HandleFunc("/circuit-breakers/{service}/reset", g.resetCircuitBreakerHandler).Methods("POST")
	
	// Rate limiting management
	mgmt.HandleFunc("/rate-limits", g.rateLimitsHandler).Methods("GET")
	mgmt.HandleFunc("/rate-limits/reset", g.resetRateLimitsHandler).Methods("POST")
	
	// Service registry management
	mgmt.HandleFunc("/registry/refresh", g.refreshRegistryHandler).Methods("POST")
	mgmt.HandleFunc("/registry/health-check", g.healthCheckHandler).Methods("POST")
}

// Middleware implementations

func (g *Gateway) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		g.requestCounter.Inc()
		
		next.ServeHTTP(w, r)
		
		duration := time.Since(start)
		g.requestDuration.Observe(duration.Seconds())
	})
}

func (g *Gateway) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !g.rateLimiter.Allow(r) {
			g.handleRateLimitExceeded(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (g *Gateway) circuitBreakerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		service := g.getServiceFromPath(r.URL.Path)
		
		if g.circuitBreaker.IsOpen(service) {
			g.handleCircuitBreakerOpen(w, r, service)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// Handler implementations

func (g *Gateway) healthHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   g.config.Gateway.Version,
		"services":  g.getServiceHealthStatus(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (g *Gateway) readinessHandler(w http.ResponseWriter, r *http.Request) {
	ready := g.checkReadiness()
	
	status := map[string]interface{}{
		"ready":     ready,
		"timestamp": time.Now().UTC(),
		"checks":    g.getReadinessChecks(),
	}

	if ready {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (g *Gateway) statusHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"gateway": map[string]interface{}{
			"version":    g.config.Gateway.Version,
			"uptime":     time.Since(g.config.Gateway.StartTime),
			"requests":   g.monitor.GetTotalRequests(),
			"errors":     g.monitor.GetTotalErrors(),
		},
		"services":      g.serviceRegistry.GetAllServices(),
		"rate_limits":   g.rateLimiter.GetStatus(),
		"circuit_breakers": g.circuitBreaker.GetStatus(),
		"timestamp":     time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Management handler implementations

func (g *Gateway) routesHandler(w http.ResponseWriter, r *http.Request) {
	routes := g.getAllRoutes()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(routes)
}

func (g *Gateway) servicesHandler(w http.ResponseWriter, r *http.Request) {
	services := g.serviceRegistry.GetAllServices()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(services)
}

func (g *Gateway) configHandler(w http.ResponseWriter, r *http.Request) {
	// Return sanitized configuration (remove sensitive data)
	config := g.sanitizeConfig()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

func (g *Gateway) detailedMetricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := g.monitor.GetDetailedMetrics()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (g *Gateway) circuitBreakersHandler(w http.ResponseWriter, r *http.Request) {
	status := g.circuitBreaker.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (g *Gateway) resetCircuitBreakerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	service := vars["service"]
	
	if err := g.circuitBreaker.Reset(service); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "reset"})
}

func (g *Gateway) rateLimitsHandler(w http.ResponseWriter, r *http.Request) {
	status := g.rateLimiter.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (g *Gateway) resetRateLimitsHandler(w http.ResponseWriter, r *http.Request) {
	g.rateLimiter.Reset()
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "reset"})
}

func (g *Gateway) refreshRegistryHandler(w http.ResponseWriter, r *http.Request) {
	if err := g.serviceRegistry.Refresh(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "refreshed"})
}

func (g *Gateway) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	results := g.serviceRegistry.HealthCheckAll()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// Error handlers

func (g *Gateway) handleServiceError(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
	response := map[string]interface{}{
		"error":      message,
		"status":     statusCode,
		"timestamp":  time.Now().UTC(),
		"request_id": middleware.GetRequestID(r.Context()),
		"path":       r.URL.Path,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func (g *Gateway) handleRateLimitExceeded(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"error":      "Rate limit exceeded",
		"status":     http.StatusTooManyRequests,
		"timestamp":  time.Now().UTC(),
		"request_id": middleware.GetRequestID(r.Context()),
		"retry_after": g.rateLimiter.GetRetryAfter(r),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", fmt.Sprintf("%d", g.rateLimiter.GetRetryAfter(r)))
	w.WriteHeader(http.StatusTooManyRequests)
	json.NewEncoder(w).Encode(response)
}

func (g *Gateway) handleCircuitBreakerOpen(w http.ResponseWriter, r *http.Request, service string) {
	fallbackResponse := g.circuitBreaker.GetFallbackResponse(service)
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Circuit-Breaker", "open")
	w.Header().Set("X-Service", service)
	w.WriteHeader(http.StatusServiceUnavailable)
	json.NewEncoder(w).Encode(fallbackResponse)
}

// Utility functions

func (g *Gateway) getServiceFromPath(path string) string {
	// Extract service name from path
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/"), "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

func (g *Gateway) getServiceHealthStatus() map[string]bool {
	return g.serviceRegistry.GetHealthStatus()
}

func (g *Gateway) checkReadiness() bool {
	// Check if gateway is ready to serve requests
	return !g.shuttingDown && 
		   g.authService.IsReady() && 
		   g.rateLimiter.IsReady() && 
		   g.serviceRegistry.IsReady()
}

func (g *Gateway) getReadinessChecks() map[string]bool {
	return map[string]bool{
		"auth_service":      g.authService.IsReady(),
		"rate_limiter":      g.rateLimiter.IsReady(),
		"service_registry":  g.serviceRegistry.IsReady(),
		"circuit_breakers":  g.circuitBreaker.IsReady(),
	}
}

func (g *Gateway) getAllRoutes() []Route {
	// Return all configured routes
	// Implementation would return actual route configurations
	return []Route{}
}

func (g *Gateway) sanitizeConfig() map[string]interface{} {
	// Return configuration with sensitive data removed
	return map[string]interface{}{
		"gateway": map[string]interface{}{
			"version": g.config.Gateway.Version,
			"port":    g.config.Server.Port,
		},
		"services": g.config.Services,
		// Exclude auth tokens, secrets, etc.
	}
}

func splitMethods(methods string) []string {
	return strings.Split(methods, ",")
}

// Start starts the API Gateway server
func (g *Gateway) Start() error {
	log.Printf("Starting API Gateway on port %d", g.config.Server.Port)
	
	// Start background services
	go g.serviceRegistry.StartHealthChecking()
	go g.monitor.StartReporting()
	
	// Start server
	if g.config.Server.TLS.Enabled {
		return g.server.ListenAndServeTLS(
			g.config.Server.TLS.CertFile,
			g.config.Server.TLS.KeyFile,
		)
	}
	
	return g.server.ListenAndServe()
}

// Shutdown gracefully shuts down the API Gateway
func (g *Gateway) Shutdown(ctx context.Context) error {
	g.mutex.Lock()
	g.shuttingDown = true
	g.mutex.Unlock()
	
	log.Println("Shutting down API Gateway...")
	
	// Stop background services
	g.serviceRegistry.Stop()
	g.monitor.Stop()
	
	// Shutdown HTTP server
	return g.server.Shutdown(ctx)
} 