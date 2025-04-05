package network_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"iaros/network_planning_service"
	"github.com/stretchr/testify/assert"
)

func TestScheduleImporterRetry(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// Simulate empty response on first call.
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("[]"))
		} else {
			// Return valid schedule data on second call.
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"route":"JFK-LHR","departure":"2025-06-01T12:00:00Z","arrival":"2025-06-01T20:00:00Z","aircraft":"A380","status":"on time"}]`))
		}
	}))
	defer ts.Close()

	schedules, err := network_planning_service.ScheduleImporter(ts.URL)
	assert.NoError(t, err, "ScheduleImporter should succeed after retry")
	assert.NotEmpty(t, schedules, "Schedules should be returned after retry")
}

func TestPartnerSyncFallback(t *testing.T) {
	// Simulate partner sync failure using a test server.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	syncer := network_planning_service.NewPartnerSync() // Assume proper initialization.
	codeshares, err := syncer.Sync(context.Background(), ts.URL, "delta_airlines")
	assert.Error(t, err, "PartnerSync should trigger fallback on error")
	assert.NotNil(t, codeshares, "Fallback codeshares should be returned")
}
