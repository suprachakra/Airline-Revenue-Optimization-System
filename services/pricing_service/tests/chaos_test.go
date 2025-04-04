package pricing_test

import (
	"fmt"
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
	// Simulate a scenario where all live pricing layers fail.
	mockClient := &pricing.MockClient{
		ForceError:        true,
		ForceCacheExpiry:  true,
		ForceHistoryFail:  true,
	}
	engine := pricing.NewPricingEngine(mockClient)
	price, err := engine.CalculatePrice("JFK-LHR")
	assert.NoError(t, err)
	expectedPrice := 850.0 // Expected fallback price from static floor.
	assert.InDelta(t, expectedPrice, price, 100.0, "Static floor pricing should be used in complete failure")

	// Validate fallback logging.
	entries := analytics.GetFallbackLogs()
	expectedFallbacks := []string{"geo_cache", "historical", "static_floor"}
	assert.Equal(t, expectedFallbacks, entries, "Fallback log order should match expected sequence")
}
