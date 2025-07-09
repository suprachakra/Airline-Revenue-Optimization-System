package offer

import (
	"encoding/json"
	"net/http"
	"time"
	"log"
)

// OfferController exposes endpoints for final offer retrieval.
func OfferController(w http.ResponseWriter, r *http.Request) {
	route := r.URL.Query().Get("route")
	offerValue, err := OfferAssembler(route)
	if err != nil {
		log.Printf("Offer assembly error for route %s: %v", route, err)
		http.Error(w, "Offer assembly failed", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"route":      route,
		"offer":      offerValue,
		"timestamp":  time.Now().UTC(),
		"fallback":   false, // Update flag if fallback was triggered.
	}
	json.NewEncoder(w).Encode(response)
}
