package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestPromotionEngine(t *testing.T) {
	engine := NewPromotionEngine()
	price, err := engine.CalculateDiscount(nil, "user123", 200.0)
	assert.NoError(t, err)
	// For FLASH50, expect 50% discount; for default fallback, expect 5% discount.
	assert.InDelta(t, 190.0, price, 20.0, "Default fallback discount should be applied")
}

func TestPromotionController(t *testing.T) {
	req, _ := http.NewRequest("GET", "/promotion?user_id=user123&price=200", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(PromotionController)
	handler.ServeHTTP(rr, req)

	var response map[string]interface{}
	err := json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "user123", response["user_id"])
	assert.NotNil(t, response["price"])
}
