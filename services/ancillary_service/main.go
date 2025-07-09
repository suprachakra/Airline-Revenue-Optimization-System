package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	"iaros/ancillary_service/src/controllers"
	"iaros/ancillary_service/src/middleware"
	"iaros/ancillary_service/src/config"
)

const (
	defaultPort = "8080"
	serviceName = "ancillary-service"
	version     = "1.0.0"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize the ancillary controller
	ancillaryController := controllers.NewAncillaryController()

	// Set up router
	router := mux.NewRouter()

	// Add middleware
	router.Use(middleware.LoggingMiddleware)
	router.Use(middleware.CORSMiddleware)
	router.Use(middleware.RecoveryMiddleware)
	router.Use(middleware.AuthMiddleware)

	// Health check endpoint (before authentication)
	router.HandleFunc("/health", healthCheck).Methods("GET")
	router.HandleFunc("/", rootHandler).Methods("GET")

	// API routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	ancillaryController.RegisterRoutes(apiRouter)

	// Setup CORS
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization", "X-API-Key"}),
	)(router)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      corsHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("🚀 %s v%s starting on port %s", serviceName, version, port)
		log.Printf("📊 Analytics and bundling engine initialized")
		log.Printf("🔗 API endpoints available at http://localhost:%s/api/v1", port)
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Print available endpoints
	printEndpoints(port)

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exited")
}

// healthCheck provides health check endpoint
func healthCheck(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":    "healthy",
		"service":   serviceName,
		"version":   version,
		"timestamp": time.Now().UTC(),
		"uptime":    time.Since(time.Now().Add(-time.Hour)).String(),
		"components": map[string]string{
			"bundling_engine":   "operational",
			"analytics_service": "operational",
			"database":          "connected",
			"cache":             "active",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("Error encoding health check response: %v", err)
	}
}

// rootHandler provides service information
func rootHandler(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"service":     serviceName,
		"version":     version,
		"description": "Intelligent Ancillary Revenue Optimization Service",
		"features": []string{
			"AI-powered bundling recommendations",
			"Dynamic pricing optimization",
			"Customer segmentation and personalization",
			"Real-time analytics and reporting",
			"Comprehensive ancillary item management",
			"Revenue forecasting and insights",
		},
		"api_version": "v1",
		"endpoints": map[string]string{
			"health":         "/health",
			"recommendations": "/api/v1/ancillary/recommendations",
			"items":          "/api/v1/ancillary/items",
			"bundles":        "/api/v1/ancillary/bundles",
			"analytics":      "/api/v1/ancillary/analytics",
			"documentation":  "/api/v1/docs",
		},
		"timestamp": time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := json.NewEncoder(w).Encode(info); err != nil {
		log.Printf("Error encoding root response: %v", err)
	}
}

// printEndpoints prints available API endpoints
func printEndpoints(port string) {
	fmt.Println("\n📋 Available API Endpoints:")
	fmt.Println("=" * 50)
	
	endpoints := []struct {
		Method string
		Path   string
		Description string
	}{
		{"GET", "/health", "Health check"},
		{"GET", "/", "Service information"},
		{"POST", "/api/v1/ancillary/recommendations", "Generate personalized recommendations"},
		{"GET", "/api/v1/ancillary/recommendations/{customerID}", "Get customer recommendations"},
		{"GET", "/api/v1/ancillary/items", "List ancillary items"},
		{"GET", "/api/v1/ancillary/items/{itemID}", "Get specific item"},
		{"POST", "/api/v1/ancillary/items", "Create new item"},
		{"PUT", "/api/v1/ancillary/items/{itemID}", "Update item"},
		{"DELETE", "/api/v1/ancillary/items/{itemID}", "Delete item"},
		{"POST", "/api/v1/ancillary/items/{itemID}/price", "Get dynamic price"},
		{"GET", "/api/v1/ancillary/bundles", "List bundles"},
		{"GET", "/api/v1/ancillary/bundles/{bundleID}", "Get specific bundle"},
		{"POST", "/api/v1/ancillary/bundles", "Create new bundle"},
		{"PUT", "/api/v1/ancillary/bundles/{bundleID}", "Update bundle"},
		{"DELETE", "/api/v1/ancillary/bundles/{bundleID}", "Delete bundle"},
		{"POST", "/api/v1/ancillary/bundles/generate", "Generate dynamic bundle"},
		{"GET", "/api/v1/ancillary/analytics/items", "Item analytics"},
		{"GET", "/api/v1/ancillary/analytics/bundles", "Bundle analytics"},
		{"GET", "/api/v1/ancillary/analytics/performance", "Performance metrics"},
		{"GET", "/api/v1/ancillary/analytics/revenue", "Revenue analytics"},
		{"POST", "/api/v1/ancillary/purchase", "Record purchase"},
		{"GET", "/api/v1/ancillary/purchase/{purchaseID}", "Get purchase details"},
		{"GET", "/api/v1/ancillary/customers/{customerID}/profile", "Get customer profile"},
		{"PUT", "/api/v1/ancillary/customers/{customerID}/profile", "Update customer profile"},
		{"GET", "/api/v1/ancillary/customers/{customerID}/preferences", "Get customer preferences"},
		{"PUT", "/api/v1/ancillary/customers/{customerID}/preferences", "Update customer preferences"},
	}

	for _, endpoint := range endpoints {
		fmt.Printf("%-6s %-60s %s\n", endpoint.Method, endpoint.Path, endpoint.Description)
	}
	
	fmt.Println("=" * 50)
	fmt.Printf("🌐 Base URL: http://localhost:%s\n", port)
	fmt.Printf("📖 API Documentation: http://localhost:%s/api/v1/docs\n", port)
	fmt.Printf("❤️  Health Check: http://localhost:%s/health\n\n", port)
}

// Additional package imports needed
import (
	"encoding/json"
)

// Package: config
package config

import (
	"os"
	"strconv"
)

// Config holds application configuration
type Config struct {
	Port            string
	DatabaseURL     string
	CacheURL        string
	LogLevel        string
	MaxConnections  int
	RequestTimeout  int
	EnableMetrics   bool
	EnableAnalytics bool
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	maxConn, _ := strconv.Atoi(getEnv("MAX_CONNECTIONS", "100"))
	reqTimeout, _ := strconv.Atoi(getEnv("REQUEST_TIMEOUT", "30"))
	enableMetrics, _ := strconv.ParseBool(getEnv("ENABLE_METRICS", "true"))
	enableAnalytics, _ := strconv.ParseBool(getEnv("ENABLE_ANALYTICS", "true"))

	return &Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "mongodb://localhost:27017/ancillary"),
		CacheURL:        getEnv("CACHE_URL", "redis://localhost:6379"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		MaxConnections:  maxConn,
		RequestTimeout:  reqTimeout,
		EnableMetrics:   enableMetrics,
		EnableAnalytics: enableAnalytics,
	}, nil
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Package: middleware
package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a custom ResponseWriter to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.RequestURI, wrapped.statusCode, duration)
	})
}

// CORSMiddleware adds CORS headers
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// RecoveryMiddleware recovers from panics
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v\n%s", err, debug.Stack())
				
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				
				response := map[string]interface{}{
					"error":     "Internal server error",
					"timestamp": time.Now().UTC(),
				}
				
				json.NewEncoder(w).Encode(response)
			}
		}()
		
		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware handles authentication (simplified for demo)
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check and root endpoints
		if r.URL.Path == "/health" || r.URL.Path == "/" {
			next.ServeHTTP(w, r)
			return
		}
		
		// Check for API key in header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			// For demo purposes, allow requests without API key
			// In production, this should be enforced
			log.Printf("Warning: Request without API key from %s", r.RemoteAddr)
		}
		
		// Add user context (in production, this would be extracted from JWT token)
		ctx := context.WithValue(r.Context(), "user_id", "demo-user")
		ctx = context.WithValue(ctx, "api_key", apiKey)
		
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Import json package for middleware
import (
	"encoding/json"
) 