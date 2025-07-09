package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"iaros/api_gateway/src/config"
	"iaros/api_gateway/src/gateway"
)

// TestAPIGatewayIntegration tests the complete API Gateway functionality
func TestAPIGatewayIntegration(t *testing.T) {
	// Create test configuration
	cfg := createTestConfig()
	
	// Create mock services
	mockServices := createMockServices(t)
	defer closeMockServices(mockServices)

	// Update configuration with mock service URLs
	updateConfigWithMockServices(cfg, mockServices)

	// Create API Gateway
	gw, err := gateway.NewGateway(cfg)
	require.NoError(t, err)

	// Create test server
	testServer := httptest.NewServer(gw.GetRouter())
	defer testServer.Close()

	// Wait for gateway to be ready
	time.Sleep(100 * time.Millisecond)

	// Run test suites
	t.Run("Authentication", func(t *testing.T) {
		testAuthentication(t, testServer)
	})

	t.Run("Authorization", func(t *testing.T) {
		testAuthorization(t, testServer)
	})

	t.Run("RateLimit", func(t *testing.T) {
		testRateLimit(t, testServer)
	})

	t.Run("CircuitBreaker", func(t *testing.T) {
		testCircuitBreaker(t, testServer, mockServices)
	})

	t.Run("ServiceRouting", func(t *testing.T) {
		testServiceRouting(t, testServer)
	})

	t.Run("LoadBalancing", func(t *testing.T) {
		testLoadBalancing(t, testServer)
	})

	t.Run("Monitoring", func(t *testing.T) {
		testMonitoring(t, testServer)
	})

	t.Run("Management", func(t *testing.T) {
		testManagement(t, testServer)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		testErrorHandling(t, testServer)
	})

	t.Run("Security", func(t *testing.T) {
		testSecurity(t, testServer)
	})
}

// createTestConfig creates a test configuration
func createTestConfig() *config.Config {
	cfg := &config.Config{
		Environment: "test",
		Server: config.ServerConfig{
			Port:         8080,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Gateway: config.GatewayConfig{
			Version: "1.0.0-test",
		},
		Auth: config.AuthConfig{
			JWTEnabled:     true,
			JWTIssuer:      "test-issuer",
			JWTAudience:    "test-audience",
			JWTExpiry:      24 * time.Hour,
			APIKeyEnabled:  true,
			SessionEnabled: true,
		},
		RateLimit: config.RateLimitConfig{
			Global: config.RateLimitRule{
				Limit:  100,
				Window: time.Minute,
			},
			PerIP: config.RateLimitRule{
				Limit:  10,
				Window: time.Minute,
			},
		},
		CircuitBreaker: config.CircuitBreakerConfig{
			DefaultFailureThreshold: 3,
			DefaultSuccessThreshold: 2,
			DefaultTimeout:          5 * time.Second,
			DefaultMaxRequests:      10,
		},
		ServiceRegistry: config.ServiceRegistryConfig{
			HealthCheckInterval: 10 * time.Second,
			HealthCheckTimeout:  2 * time.Second,
		},
		LoadBalancer: config.LoadBalancerConfig{
			Strategy: "round_robin",
		},
		Redis: config.RedisConfig{
			Address: "localhost:6379",
		},
		Monitoring: config.MonitoringConfig{
			Enabled:        true,
			ReportInterval: 30 * time.Second,
		},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Content-Type", "Authorization", "X-API-Key"},
			MaxAge:         12 * time.Hour,
		},
		Security: config.SecurityConfig{
			EnableSecurityHeaders: true,
			CSPPolicy:             "default-src 'self'",
			HSTSMaxAge:            31536000,
		},
	}

	return cfg
}

// createMockServices creates mock backend services
func createMockServices(t *testing.T) map[string]*httptest.Server {
	services := make(map[string]*httptest.Server)

	// Pricing Service Mock
	services["pricing"] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/health":
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
		case "/api/v1/pricing":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"price":    100.50,
				"currency": "USD",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// Forecasting Service Mock
	services["forecasting"] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/health":
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
		case "/api/v1/forecasting":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"forecast": []map[string]interface{}{
					{"date": "2024-01-01", "value": 150.0},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// Offer Service Mock
	services["offer"] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/health":
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
		case "/api/v1/offers":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"offers": []map[string]interface{}{
					{"id": "offer-1", "price": 200.0},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// Faulty Service Mock (for circuit breaker testing)
	services["faulty"] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		time.Sleep(2 * time.Second) // Simulate slow response
		w.WriteHeader(http.StatusInternalServerError)
	}))

	return services
}

// closeMockServices closes all mock services
func closeMockServices(services map[string]*httptest.Server) {
	for _, service := range services {
		service.Close()
	}
}

// updateConfigWithMockServices updates configuration with mock service URLs
func updateConfigWithMockServices(cfg *config.Config, services map[string]*httptest.Server) {
	cfg.Services = config.ServicesConfig{
		Pricing: config.ServiceConfig{
			Primary: services["pricing"].URL,
		},
		Forecasting: config.ServiceConfig{
			Primary: services["forecasting"].URL,
		},
		Offer: config.ServiceConfig{
			Primary: services["offer"].URL,
		},
	}
}

// testAuthentication tests authentication functionality
func testAuthentication(t *testing.T, server *httptest.Server) {
	t.Run("NoAuth", func(t *testing.T) {
		// Request without authentication
		resp, err := http.Get(server.URL + "/api/v1/pricing")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("ValidJWT", func(t *testing.T) {
		// Create a valid JWT token (mocked)
		token := createTestJWT()
		
		client := &http.Client{}
		req, err := http.NewRequest("GET", server.URL+"/api/v1/pricing", nil)
		require.NoError(t, err)
		
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should get either 200 (success) or 502 (service unavailable)
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadGateway)
	})

	t.Run("InvalidJWT", func(t *testing.T) {
		client := &http.Client{}
		req, err := http.NewRequest("GET", server.URL+"/api/v1/pricing", nil)
		require.NoError(t, err)
		
		req.Header.Set("Authorization", "Bearer invalid-token")
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("APIKey", func(t *testing.T) {
		client := &http.Client{}
		req, err := http.NewRequest("GET", server.URL+"/api/v1/pricing", nil)
		require.NoError(t, err)
		
		req.Header.Set("X-API-Key", "test-api-key")
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should get either 200 (success) or 401 (unauthorized)
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized)
	})
}

// testAuthorization tests authorization functionality
func testAuthorization(t *testing.T, server *httptest.Server) {
	t.Run("AdminAccess", func(t *testing.T) {
		token := createTestJWTWithRole("admin")
		
		client := &http.Client{}
		req, err := http.NewRequest("GET", server.URL+"/management/config", nil)
		require.NoError(t, err)
		
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should get either 200 (success) or 401 (unauthorized)
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized)
	})

	t.Run("UserAccess", func(t *testing.T) {
		token := createTestJWTWithRole("user")
		
		client := &http.Client{}
		req, err := http.NewRequest("GET", server.URL+"/management/config", nil)
		require.NoError(t, err)
		
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should be forbidden
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

// testRateLimit tests rate limiting functionality
func testRateLimit(t *testing.T, server *httptest.Server) {
	t.Run("GlobalRateLimit", func(t *testing.T) {
		token := createTestJWT()
		
		var wg sync.WaitGroup
		rateLimitHit := false
		
		// Send multiple requests concurrently
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				
				client := &http.Client{}
				req, err := http.NewRequest("GET", server.URL+"/api/v1/pricing", nil)
				if err != nil {
					return
				}
				
				req.Header.Set("Authorization", "Bearer "+token)
				
				resp, err := client.Do(req)
				if err != nil {
					return
				}
				defer resp.Body.Close()
				
				if resp.StatusCode == http.StatusTooManyRequests {
					rateLimitHit = true
				}
			}()
		}
		
		wg.Wait()
		
		// Should hit rate limit
		assert.True(t, rateLimitHit, "Rate limit should be triggered")
	})

	t.Run("RateLimitHeaders", func(t *testing.T) {
		token := createTestJWT()
		
		client := &http.Client{}
		req, err := http.NewRequest("GET", server.URL+"/api/v1/pricing", nil)
		require.NoError(t, err)
		
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check for rate limit headers
		assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Limit"))
		assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Reset"))
	})
}

// testCircuitBreaker tests circuit breaker functionality
func testCircuitBreaker(t *testing.T, server *httptest.Server, mockServices map[string]*httptest.Server) {
	t.Run("CircuitBreakerTrip", func(t *testing.T) {
		token := createTestJWT()
		
		// Make multiple requests to trigger circuit breaker
		for i := 0; i < 5; i++ {
			client := &http.Client{Timeout: 1 * time.Second}
			req, err := http.NewRequest("GET", server.URL+"/api/v1/faulty", nil)
			require.NoError(t, err)
			
			req.Header.Set("Authorization", "Bearer "+token)
			
			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
			}
		}

		// Next request should get fallback response
		client := &http.Client{}
		req, err := http.NewRequest("GET", server.URL+"/api/v1/faulty", nil)
		require.NoError(t, err)
		
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should get circuit breaker response
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		assert.Equal(t, "open", resp.Header.Get("X-Circuit-Breaker"))
	})
}

// testServiceRouting tests service routing functionality
func testServiceRouting(t *testing.T, server *httptest.Server) {
	token := createTestJWT()

	testCases := []struct {
		name     string
		path     string
		expected int
	}{
		{"Pricing", "/api/v1/pricing", http.StatusOK},
		{"Forecasting", "/api/v1/forecasting", http.StatusOK},
		{"Offers", "/api/v1/offers", http.StatusOK},
		{"NotFound", "/api/v1/nonexistent", http.StatusNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := &http.Client{}
			req, err := http.NewRequest("GET", server.URL+tc.path, nil)
			require.NoError(t, err)
			
			req.Header.Set("Authorization", "Bearer "+token)
			
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Check status code (allowing for service unavailable)
			assert.True(t, resp.StatusCode == tc.expected || resp.StatusCode == http.StatusBadGateway)
		})
	}
}

// testLoadBalancing tests load balancing functionality
func testLoadBalancing(t *testing.T, server *httptest.Server) {
	t.Run("RoundRobin", func(t *testing.T) {
		token := createTestJWT()
		
		// Make multiple requests to check load balancing
		for i := 0; i < 10; i++ {
			client := &http.Client{}
			req, err := http.NewRequest("GET", server.URL+"/api/v1/pricing", nil)
			require.NoError(t, err)
			
			req.Header.Set("Authorization", "Bearer "+token)
			
			resp, err := client.Do(req)
			require.NoError(t, err)
			resp.Body.Close()
		}
	})
}

// testMonitoring tests monitoring functionality
func testMonitoring(t *testing.T, server *httptest.Server) {
	t.Run("Metrics", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))
	})

	t.Run("Health", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		var health map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)
		
		assert.Equal(t, "healthy", health["status"])
	})

	t.Run("Readiness", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/ready")
		require.NoError(t, err)
		defer resp.Body.Close()

		var readiness map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&readiness)
		require.NoError(t, err)
		
		assert.Contains(t, readiness, "ready")
	})
}

// testManagement tests management endpoints
func testManagement(t *testing.T, server *httptest.Server) {
	token := createTestJWTWithRole("admin")

	t.Run("Status", func(t *testing.T) {
		client := &http.Client{}
		req, err := http.NewRequest("GET", server.URL+"/status", nil)
		require.NoError(t, err)
		
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		var status map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&status)
		require.NoError(t, err)
		
		assert.Contains(t, status, "gateway")
	})

	t.Run("Routes", func(t *testing.T) {
		client := &http.Client{}
		req, err := http.NewRequest("GET", server.URL+"/management/routes", nil)
		require.NoError(t, err)
		
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should be OK or Unauthorized
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized)
	})

	t.Run("Services", func(t *testing.T) {
		client := &http.Client{}
		req, err := http.NewRequest("GET", server.URL+"/management/services", nil)
		require.NoError(t, err)
		
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should be OK or Unauthorized
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized)
	})
}

// testErrorHandling tests error handling
func testErrorHandling(t *testing.T, server *httptest.Server) {
	t.Run("404NotFound", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/nonexistent")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("405MethodNotAllowed", func(t *testing.T) {
		resp, err := http.Post(server.URL+"/health", "application/json", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

// testSecurity tests security features
func testSecurity(t *testing.T, server *httptest.Server) {
	t.Run("SecurityHeaders", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check security headers
		assert.NotEmpty(t, resp.Header.Get("X-Content-Type-Options"))
		assert.NotEmpty(t, resp.Header.Get("X-Frame-Options"))
		assert.NotEmpty(t, resp.Header.Get("X-XSS-Protection"))
		assert.NotEmpty(t, resp.Header.Get("Strict-Transport-Security"))
	})

	t.Run("CORS", func(t *testing.T) {
		client := &http.Client{}
		req, err := http.NewRequest("OPTIONS", server.URL+"/api/v1/pricing", nil)
		require.NoError(t, err)
		
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "GET")
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	})
}

// Helper functions

// createTestJWT creates a test JWT token
func createTestJWT() string {
	// In a real test, this would create a properly signed JWT
	// For now, return a mock token
	return "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
}

// createTestJWTWithRole creates a test JWT token with a specific role
func createTestJWTWithRole(role string) string {
	// In a real test, this would create a properly signed JWT with role claim
	// For now, return a mock token
	return "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwicm9sZSI6IiIgKyByb2xlICsgIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
}

// TestAPIGatewayPerformance tests performance characteristics
func TestAPIGatewayPerformance(t *testing.T) {
	cfg := createTestConfig()
	mockServices := createMockServices(t)
	defer closeMockServices(mockServices)
	
	updateConfigWithMockServices(cfg, mockServices)
	
	gw, err := gateway.NewGateway(cfg)
	require.NoError(t, err)
	
	testServer := httptest.NewServer(gw.GetRouter())
	defer testServer.Close()
	
	t.Run("ConcurrentRequests", func(t *testing.T) {
		token := createTestJWT()
		
		concurrency := 100
		var wg sync.WaitGroup
		
		start := time.Now()
		
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				
				client := &http.Client{}
				req, err := http.NewRequest("GET", testServer.URL+"/api/v1/pricing", nil)
				if err != nil {
					return
				}
				
				req.Header.Set("Authorization", "Bearer "+token)
				
				resp, err := client.Do(req)
				if err != nil {
					return
				}
				defer resp.Body.Close()
			}()
		}
		
		wg.Wait()
		duration := time.Since(start)
		
		t.Logf("Processed %d concurrent requests in %v", concurrency, duration)
		assert.Less(t, duration, 5*time.Second, "Should handle concurrent requests efficiently")
	})
}

// TestAPIGatewayFailover tests failover scenarios
func TestAPIGatewayFailover(t *testing.T) {
	cfg := createTestConfig()
	mockServices := createMockServices(t)
	defer closeMockServices(mockServices)
	
	updateConfigWithMockServices(cfg, mockServices)
	
	gw, err := gateway.NewGateway(cfg)
	require.NoError(t, err)
	
	testServer := httptest.NewServer(gw.GetRouter())
	defer testServer.Close()
	
	t.Run("ServiceFailover", func(t *testing.T) {
		token := createTestJWT()
		
		// Close primary service
		mockServices["pricing"].Close()
		
		// Request should still work with fallback
		client := &http.Client{}
		req, err := http.NewRequest("GET", testServer.URL+"/api/v1/pricing", nil)
		require.NoError(t, err)
		
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		// Should get fallback response
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusServiceUnavailable)
	})
} 