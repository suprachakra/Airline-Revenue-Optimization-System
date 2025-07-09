package pricing_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"iaros/pricing_service"
	"iaros/analytics"
	"github.com/stretchr/testify/assert"
)

func TestNuclearFallbackScenario(t *testing.T) {
	// Simulate complete system failure: force live pricing, cache, and historical lookups to fail.
	mockClient := &pricing.MockClient{
		ForceError:       true,
		ForceCacheExpiry: true,
		ForceHistoryFail: true,
	}
	engine := pricing.NewPricingEngine(mockClient)
	price, err := engine.CalculatePrice("JFK-LHR")
	assert.NoError(t, err)
	expectedPrice := 850.0
	assert.InDelta(t, expectedPrice, price, 100.0, "Static floor pricing should trigger under complete failure")

	// Verify that fallback logs record the correct sequence.
	fallbackLogs := analytics.GetFallbackLogs()
	expectedFallbacks := []string{"geo_cache", "historical", "static_floor"}
	assert.Equal(t, expectedFallbacks, fallbackLogs, "Fallback logs must reflect the correct sequence")
}

func TestConcurrentFallbackHandling(t *testing.T) {
	// Test multiple concurrent requests to ensure circuit breaker and fallback resilience.
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get("http://localhost:8080/pricing?route=JFK-LHR")
			assert.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)
		}()
	}
	wg.Wait()
	time.Sleep(6 * time.Minute) // Allow circuit breaker reset.
}
