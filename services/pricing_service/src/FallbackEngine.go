package pricing

import (
	"context"
	"time"

	"github.com/iaros/analytics"
	"github.com/redis/go-redis/v9"
)

// FallbackEngine implements a multi-layer fallback strategy.
type FallbackEngine struct {
	historicalClient *HistoricalDataClient
	redis            *redis.ClusterClient
	analytics        *analytics.Client
}

// GetPrice returns a fallback price based on a four-layer strategy.
func (f *FallbackEngine) GetPrice(ctx context.Context, route string) (float64, error) {
	// Layer 1: Live Pricing (attempted earlier; this function is called upon failure).
	// Layer 2: Cached Price from Redis.
	cachedPrice, err := f.redis.Get(ctx, cacheKey(route)).Float64()
	if err == nil {
		f.analytics.LogFallback("geo_cache")
		return cachedPrice, nil
	}

	// Layer 3: Historical Moving Average (7-day).
	histPrice, err := f.historicalClient.Get7dAverage(ctx, route)
	if err == nil {
		f.analytics.LogFallback("historical")
		return histPrice * 1.15, nil // Apply safeguard markup.
	}

	// Layer 4: Static Floor Pricing.
	f.analytics.LogFallback("static_floor")
	return calculateStaticFloor(route), nil
}

func calculateStaticFloor(route string) float64 {
	// Implement IATA-compliant static floor pricing.
	return 80.0
}

func cacheKey(route string) string {
	return "pricing:" + route
}
