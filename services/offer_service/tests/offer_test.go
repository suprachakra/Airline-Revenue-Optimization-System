package offer_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"iaros/offer_service"
	"github.com/stretchr/testify/assert"
)

func TestOfferAssembly(t *testing.T) {
	req, _ := http.NewRequest("GET", "/offer?route=JFK-LHR", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(offer.OfferController)
	handler.ServeHTTP(rr, req)

	var response map[string]interface{}
	err := json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err, "Response decoding should not error")
	assert.NotNil(t, response["offer"], "Offer should be present in response")
}
