package pricing

import (
	"log"
	"time"
)

// RulesEngine enforces regulatory and business constraints on the computed fare.
func RulesEngine(fare float64, route string) float64 {
	minFare, maxFare := getFareBounds(route)
	if fare < minFare {
		log.Printf("Adjusted fare %f below minimum for route %s; enforcing minimum fare %f", fare, route, minFare)
		fare = minFare
	} else if fare > maxFare {
		log.Printf("Adjusted fare %f above maximum for route %s; enforcing maximum fare %f", fare, route, maxFare)
		fare = maxFare
	}
	logManualOverride(route, fare, time.Now())
	return fare
}

func getFareBounds(route string) (float64, float64) {
	// Retrieve fare bounds based on route-specific rules.
	return 80.0, 150.0
}

func logManualOverride(route string, fare float64, timestamp time.Time) {
	// Log any manual override events for audit purposes.
	log.Printf("Manual override logged for route %s: fare set to %f at %v", route, fare, timestamp)
}
