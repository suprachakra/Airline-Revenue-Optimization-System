package pricing_test

import (
	"testing"
	"iaros/pricing_service"
	"github.com/stretchr/testify/assert"
)

func TestDynamicPricingEngineFallback(t *testing.T) {
	route := "IN-EM"
	price := pricing.DynamicPricingEngine(route, 0.9, "IN", true, 120.0)
	// Expected fallback price should be close to the static floor if fallback is triggered.
	expectedFallback := 850.0
	assert.InDelta(t, expectedFallback, price, 100.0, "Fallback should yield a price near the static floor")
}

func TestRulesEngineAdjustment(t *testing.T) {
	route := "IN-EM"
	fare := 70.0 // Below minimum fare threshold.
	adjustedFare := pricing.RulesEngine(fare, route)
	minFare, _ := pricing.GetFareBounds(route)
	assert.Equal(t, minFare, adjustedFare, "RulesEngine must enforce the minimum fare")
}
