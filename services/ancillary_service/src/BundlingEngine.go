package ancillary

import (
	"log"
	"time"
	"iaros/ancillary_service/src/model"
)

// BundlingEngine implements dynamic bundling logic.
func BundlingEngine(customer model.Customer) []model.Ancillary {
	// Check if customer data is recent; if stale, use default bundle.
	if time.Since(customer.LastUpdate) > time.Hour {
		log.Println("Customer data stale; using default ancillary bundle")
		return model.DefaultBundle()
	}

	// Attempt RL-based dynamic bundling.
	bundle, err := GenerateRLBundles(customer)
	if err != nil {
		log.Printf("RL bundling failed: %v; reverting to default bundle", err)
		return model.DefaultBundle()
	}
	return bundle
}

// GenerateRLBundles simulates personalized bundle generation using RL.
func GenerateRLBundles(customer model.Customer) ([]model.Ancillary, error) {
	// Example logic based on customer segment.
	switch customer.Segment {
	case "Business Elite":
		return []model.Ancillary{model.PriorityBoarding, model.LoungeAccess, model.WiFi}, nil
	case "Family Traveler":
		return []model.Ancillary{model.ExtraBaggage, model.KidsMeal, model.Entertainment}, nil
	default:
		return nil, nil
	}
}
