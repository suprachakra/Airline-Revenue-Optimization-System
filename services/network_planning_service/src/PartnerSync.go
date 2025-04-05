package network

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Codeshare represents partner flight data.
type Codeshare struct {
	PartnerCode  string `json:"partner_code"`
	FlightNumber string `json:"flight_number"`
	Route        string `json:"route"`
	Status       string `json:"status"`
}

// PartnerSync synchronizes codeshare data from partner APIs.
type PartnerSync struct {
	logger    *zap.Logger
	lastValid *CodeshareData
}

// CodeshareData encapsulates codeshare synchronization data.
type CodeshareData struct {
	Data      []Codeshare
	Timestamp time.Time
}

// Sync fetches codeshare data and applies fallback if necessary.
func (p *PartnerSync) Sync(ctx context.Context, apiURL string, partnerID string) ([]Codeshare, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := http.Get(apiURL)
	if err != nil {
		p.logger.Error("Partner sync failed, using last valid data",
			zap.String("partner", partnerID),
			zap.Time("last_valid", p.lastValid.Timestamp),
		)
		return p.applyFallbackSchedule(partnerID)
	}
	defer resp.Body.Close()

	var codeshares []Codeshare
	if err := json.NewDecoder(resp.Body).Decode(&codeshares); err != nil {
		p.logger.Warn("Error decoding codeshare data, applying fallback", zap.Error(err))
		return p.applyFallbackSchedule(partnerID)
	}

	// Validate data: if discrepancy > 5%, use fallback.
	if !validateCodeshareData(codeshares) {
		p.logger.Warn("Codeshare data validation failed; using cached configuration")
		return p.applyFallbackSchedule(partnerID)
	}

	// Update last valid data.
	p.lastValid = &CodeshareData{Data: codeshares, Timestamp: time.Now()}
	return codeshares, nil
}

func validateCodeshareData(data []Codeshare) bool {
	// (Placeholder: Validate that discrepancies are within 5%)
	return true
}

func (p *PartnerSync) applyFallbackSchedule(partnerID string) ([]Codeshare, error) {
	if p.lastValid == nil || time.Since(p.lastValid.Timestamp) > 15*time.Minute {
		return nil, errors.New("fallback data is stale or unavailable")
	}
	return p.lastValid.Data, nil
}
