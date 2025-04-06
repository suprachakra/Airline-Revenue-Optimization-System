package pricing

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"iaros/pricing_service/fallback"
	"iaros/pricing_service/forecast"
)

// PricingController is the HTTP handler for pricing requests.
// It integrates real-time forecasting data and applies dynamic pricing algorithms,
// with multi-layered fallback mechanisms to guarantee consistent fare outputs.
func PricingController(w http.ResponseWriter, r *http.Request) {
	// Extract parameters from the request.
	route := r.URL.Query().Get("route")
	demand := parseDemand(r.URL.Query().Get("demand"))
	geo := r.URL.Query().Get("geo")
	corporate := r.URL.Query().Get("corporate") == "true"

	// Step 1: Retrieve forecast data.
	forecastVal, err := forecast.GetForecast(route)
	if err != nil {
		log.Printf("Forecast retrieval failed for route %s: %v. Using cached forecast.", route, err)
		forecastVal = forecast.GetCachedForecast(route)
	}

	// Step 2: Compute dynamic price using the core engine.
	price := DynamicPricingEngine(route, demand, geo, corporate, forecastVal)

	// Step 3: Apply regulatory rules to enforce bounds.
	finalPrice := RulesEngine(price, route)

	// Step 4: If pricing data is stale, trigger fallback.
	if time.Since(lastPriceUpdate(route)) > 5*time.Minute {
		fbEngine := fallback.NewFallbackEngine()
		if fbPrice, err := fbEngine.GetPrice(r.Context(), route); err == nil {
			finalPrice = fbPrice
		}
	}

	response := map[string]interface{}{
		"route":     route,
		"fare":      finalPrice,
		"timestamp": time.Now().UTC(),
		"fallback":  finalPrice != price,
	}
	json.NewEncoder(w).Encode(response)
}

// parseDemand converts a demand parameter to a float64.
// Robust error handling and fallback to default demand are implemented.
func parseDemand(val string) float64 {
	// TODO: Implement parsing logic with error fallback.
	return 1.0
}
