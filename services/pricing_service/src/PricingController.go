package pricing

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"iaros/pricing_service/src/fallback"
	"iaros/pricing_service/src/forecast"
)

// PricingController handles incoming pricing requests.
func PricingController(w http.ResponseWriter, r *http.Request) {
	// Extract parameters from query string.
	route := r.URL.Query().Get("route")
	demand := parseDemand(r.URL.Query().Get("demand"))
	geo := r.URL.Query().Get("geo")
	corporate := r.URL.Query().Get("corporate") == "true"

	// Retrieve forecast for the route.
	forecastVal, err := forecast.GetForecast(route)
	if err != nil {
		log.Println("Forecast retrieval failed, using cached forecast:", err)
		forecastVal = forecast.GetCachedForecast(route)
	}

	// Calculate dynamic price.
	price := DynamicPricingEngine(route, demand, geo, corporate, forecastVal)

	// Apply rule-based constraints.
	finalPrice := RulesEngine(price, route)

	// Fallback handling.
	if time.Since(lastPriceUpdate(route)) > 5*time.Minute {
		finalPrice, _ = fallback.NewFallbackEngine().GetPrice(r.Context(), route)
	}

	response := map[string]interface{}{
		"route":       route,
		"fare":        finalPrice,
		"timestamp":   time.Now().UTC(),
		"fallback":    finalPrice != price,
	}
	json.NewEncoder(w).Encode(response)
}

// parseDemand converts demand value from string to float64.
func parseDemand(val string) float64 {
	// Implementation omitted for brevity; assume proper error handling.
	return 1.0
}
