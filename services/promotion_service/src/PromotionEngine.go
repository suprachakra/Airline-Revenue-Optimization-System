package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/iaros/segmentation"
)

// PromotionEngine calculates discounts based on CRM segments and market data.
type PromotionEngine struct {
	crmClient    CRMClient      // Interface to CRM system for customer segmentation.
	priceChecker PriceChecker   // Interface to retrieve base prices.
}

// CalculateDiscount computes the promotional discount for a given user.
func (p *PromotionEngine) CalculateDiscount(ctx context.Context, userID string, basePrice float64) (float64, error) {
	// Retrieve customer segment from CRM.
	segment, err := p.crmClient.GetSegment(ctx, userID)
	if err != nil {
		log.Printf("CRM retrieval failed for user %s: %v", userID, err)
		return p.fallbackDiscount(userID, basePrice)
	}

	// Get base discount based on customer segment.
	discountRate, err := p.getDiscountRate(segment)
	if err != nil {
		log.Printf("Discount rate retrieval failed: %v", err)
		return p.fallbackDiscount(userID, basePrice)
	}

	discountedPrice := basePrice * (1 - discountRate)
	return discountedPrice, nil
}

// getDiscountRate returns discount rate based on customer segment.
func (p *PromotionEngine) getDiscountRate(segment string) (float64, error) {
	// Example static mapping; in production, this could be a DB call.
	switch segment {
	case "platinum":
		return 0.15, nil
	case "gold":
		return 0.10, nil
	case "silver":
		return 0.05, nil
	default:
		return 0, errors.New("unknown customer segment")
	}
}

// fallbackDiscount applies a default discount if promotion calculation fails.
func (p *PromotionEngine) fallbackDiscount(userID string, basePrice float64) (float64, error) {
	// Fallback: Use cached CRM segment or default to 5% discount.
	defaultRate := 0.05
	log.Printf("Fallback applied for user %s: default discount %v", userID, defaultRate)
	return basePrice * (1 - defaultRate), nil
}
