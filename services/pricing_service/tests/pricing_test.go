package pricing_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"iaros/pricing_service"
	"github.com/stretchr/testify/assert"
)

func TestDynamicPricingEngineFallback(t *testing.T) {
	route := "IN-EM"
	// Simulate live pricing failure by forcing fallback conditions.
	price := pricing.DynamicPricingEngine(route, 0.9, "IN", true, 120.0)
	// Fallback expected since lastPriceUpdate simulates a delay >5 minutes.
	expectedFallback := 850.0 // Example static floor price.
	assert.InDelta(t, expectedFallback, price, 100.0, "Fallback should return a price close to the static floor")
}

func TestRulesEngineAdjustment(t *testing.T) {
	route := "IN-EM"
	fare := 70.0 // Below minimum threshold.
	adjusted := pricing.RulesEngine(fare, route)
	minFare, _ := pricing.GetFareBounds(route)
	assert.Equal(t, minFare, adjusted, "RulesEngine should enforce minimum fare bounds")
}
