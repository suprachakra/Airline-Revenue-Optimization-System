package ancillary_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"iaros/ancillary_service"
	"iaros/ancillary_service/src/model"
	"github.com/stretchr/testify/assert"
)

func TestAncillaryBundleFallback(t *testing.T) {
	// Simulate stale customer data.
	customer := model.Customer{
		ID:         "cust123",
		Segment:    "Business Elite",
		LastUpdate: time.Now().Add(-2 * time.Hour), // Stale data
	}
	bundle := ancillary.BundlingEngine(customer)
	defaultBundle := model.DefaultBundle()
	assert.Equal(t, defaultBundle, bundle, "Expected default bundle for stale customer data")
}

func TestAncillaryController(t *testing.T) {
	req, _ := http.NewRequest("GET", "/ancillary", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(ancillary.AncillaryController)
	handler.ServeHTTP(rr, req)

	var response map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&response)
	assert.NotNil(t, response["bundle"], "Response should include an ancillary bundle")
}
