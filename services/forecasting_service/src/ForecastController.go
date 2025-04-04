package forecasting

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
	"iaros/forecasting_service/src/cache"
)

// ForecastController provides endpoints for retrieving forecasts.
func ForecastController(w http.ResponseWriter, r *http.Request) {
	route := r.URL.Query().Get("route")
	modelType := r.URL.Query().Get("model") // e.g., "ARIMA" or "LSTM"
	data, err := cache.GetDataForRoute(route)
	if err != nil {
		log.Printf("Error retrieving data for route %s: %v", route, err)
		http.Error(w, "Data unavailable", http.StatusServiceUnavailable)
		return
	}

	forecast, err := ForecastModel(data, modelType)
	if err != nil {
		log.Printf("Forecasting failed for route %s: %v", route, err)
		forecast = cache.GetCachedForecast(route)
	}

	response := map[string]interface{}{
		"route":     route,
		"forecast":  forecast,
		"timestamp": time.Now().UTC(),
	}
	json.NewEncoder(w).Encode(response)
}
