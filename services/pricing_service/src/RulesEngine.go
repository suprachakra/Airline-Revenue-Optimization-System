package pricing

import (
	"log"
	"time"
)

// RulesEngine enforces pricing constraints.
func RulesEngine(fare float64, route string) float64 {
	minFare, maxFare := getFareBounds(route)
	if fare < minFare {
		log.Printf("Price %f is below minimum for route %s; adjusting to %f", fare, route, minFare)
		fare = minFare
	} else if fare > maxFare {
		log.Printf("Price %f exceeds maximum for route %s; adjusting to %f", fare, route, maxFare)
		fare = maxFare
	}
	logManualOverride(route, fare, time.Now())
	return fare
}

func getFareBounds(route string) (float64, float64) {
	// Example: Return default min and max fare.
	return 80.0, 150.0
}

func logManualOverride(route string, fare float64, timestamp time.Time) {
	log.Printf("Manual override logged for route %s: fare set to %f at %v", route, fare, timestamp)
}
