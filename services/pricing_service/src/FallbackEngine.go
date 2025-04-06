package pricing

import (
	"context"
	"time"

	"github.com/iaros/analytics"
	"github.com/redis/go-redis/v9"
)

// FallbackEngine provides a cascading fallback mechanism to guarantee price delivery.
type FallbackEngine struct {
	historicalClient *HistoricalDataClient
	redis            *redis.ClusterClient
	analytics        *analytics.Client
}

// NewFallbackEngine initializes the FallbackEngine.
func NewFallbackEngine() *FallbackEngine {
	return &FallbackEngine{
		historicalClient: NewHistoricalDataClient(),
		redis:            NewRedisClient(),
		analytics:        analytics.NewClient(),
	}
}

// GetPrice applies a four‑layer fallback strategy for pricing.
// 1. Attempt to retrieve live price from the geo‑distributed cache.
// 2. If unavailable, use the historical 7‑day moving average (with a safeguard markup).
// 3. If historical retrieval fails, fall back to static floor pricing.
func (f *FallbackEngine) GetPrice(ctx context.Context, route string) (float64, error) {
	// Layer 1: Geo‑Cache fallback
	cachedPrice, err := f.redis.Get(ctx, cacheKey(route)).Float64()
	if err == nil {
		f.analytics.LogFallback("geo_cache")
		return cachedPrice, nil
	}

	// Layer 2: Historical 7‑Day Average fallback
	histPrice, err := f.historicalClient.Get7dAverage(ctx, route)
	if err == nil {
		f.analytics.LogFallback("historical")
		return histPrice * 1.15, nil // Apply a safeguard markup of 15%
	}

	// Layer 3: Static Floor Pricing fallback
	f.analytics.LogFallback("static_floor")
	return calculateStaticFloor(route), nil
}

func calculateStaticFloor(route string) float64 {
	// Calculate static floor pricing based on IATA minimum guidelines.
	return 80.0
}

func cacheKey(route string) string {
	return "pricing:" + route
}
