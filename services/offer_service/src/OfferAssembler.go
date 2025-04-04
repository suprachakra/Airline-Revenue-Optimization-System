package offer

import (
	"log"
	"time"
	"iaros/offer_service/src/loyalty"
	"iaros/offer_service/src/pricing"
	"iaros/offer_service/src/forecasting"
	"iaros/offer_service/src/ancillary"
)

// OfferAssembler aggregates inputs from pricing, forecasting, ancillary, and loyalty modules.
func OfferAssembler(route string) (float64, error) {
	// Retrieve individual components.
	price, err := pricing.GetPrice(route)
	if err != nil {
		log.Printf("Pricing retrieval error: %v", err)
		return 0, err
	}

	forecastVal, err := forecasting.GetForecast(route)
	if err != nil {
		log.Printf("Forecast retrieval error: %v", err)
		forecastVal = forecasting.GetCachedForecast(route)
	}

	ancillaryBundle, err := ancillary.GetAncillaryBundle(route)
	if err != nil {
		log.Printf("Ancillary bundle retrieval error: %v", err)
		ancillaryBundle = ancillary.DefaultBundle()
	}

	loyaltyAdjustment := loyalty.CalculateLoyaltyAdjustment(route)

	// Compose the final offer based on weighted contributions.
	finalOffer := price*0.6 + forecastVal*0.2 + loyaltyAdjustment*0.2
	return finalOffer, nil
}
