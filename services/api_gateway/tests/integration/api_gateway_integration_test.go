package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"iaros/api_gateway/routes"
	"iaros/api_gateway/config"
	// Assume testServer is defined globally in our test suite.
)

func TestCircuitBreakerFallback(t *testing.T) {
	// Create a mock pricing service that forces a timeout
	mockPricingService := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Second) // simulate delay causing a timeout
		}),
	)
	defer mockPricingService.Close()

	// Load test configuration and override pricing service endpoint
	cfg := config.LoadTestConfig()
	cfg.Endpoints.PricingService = mockPricingService.URL

	// Start test server with updated configuration
	testRouter := mux.NewRouter()
	routes.RegisterPricingRoutes(testRouter)
	testServer := httptest.NewServer(testRouter)
	defer testServer.Close()

	// Make concurrent requests to trigger circuit breaker
	var wg sync.WaitGroup
	concurrentRequests := 50
	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(fmt.Sprintf("%s/pricing", testServer.URL))
			if err != nil {
				t.Errorf("Request error: %v", err)
			}
			// Validate fallback response (e.g., status 200 and fallback headers)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Contains(t, resp.Header.Get("X-Cache-Status"), "fallback")
			assert.NotEmpty(t, resp.Header.Get("X-Circuit-Open"))
		}()
	}
	wg.Wait()

	// Verify circuit breaker metrics using Prometheus if integrated
	// (Pseudo-code; actual metric retrieval depends on the Prometheus client setup)
	// metrics := prometheus.DefaultGatherer
	// family, _ := metrics.GetMetricFamily("circuit_breaker_state")
	// assert.Equal(t, 1, len(family.Metric), "Circuit should be open")
}
