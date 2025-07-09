package offer

import (
	"log"
	"iaros/offer_service/src/loyalty"
	"iaros/offer_service/src/model"
)

// PersonalizationEngine applies loyalty-based adjustments to offers.
func PersonalizationEngine(offerValue float64, customer model.Customer) float64 {
	adjustment, err := loyalty.GetAdjustment(customer)
	if err != nil {
		log.Printf("Loyalty adjustment error: %v", err)
		return offerValue
	}
	return offerValue * adjustment
}
