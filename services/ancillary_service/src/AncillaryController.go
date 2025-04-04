package ancillary

import (
	"encoding/json"
	"net/http"
	"time"
	"iaros/ancillary_service/src/model"
	"log"
)

// AncillaryController handles API requests for ancillary offerings.
func AncillaryController(w http.ResponseWriter, r *http.Request) {
	customer := model.GetCustomerFromRequest(r)
	bundle := BundlingEngine(customer)
	response := map[string]interface{}{
		"customer_id": customer.ID,
		"bundle":      bundle,
		"timestamp":   time.Now().UTC(),
	}
	json.NewEncoder(w).Encode(response)
}
