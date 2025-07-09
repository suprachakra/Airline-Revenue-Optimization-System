package models

import (
	"time"
	"math"
)

// AncillaryCategory defines the category of ancillary services
type AncillaryCategory string

const (
	CategoryBaggage      AncillaryCategory = "baggage"
	CategoryConnectivity AncillaryCategory = "connectivity"
	CategoryComfort      AncillaryCategory = "comfort"
	CategoryDining       AncillaryCategory = "dining"
	CategoryConvenience  AncillaryCategory = "convenience"
	CategoryProtection   AncillaryCategory = "protection"
	CategoryEntertainment AncillaryCategory = "entertainment"
	CategoryGroundService AncillaryCategory = "ground_service"
)

// AncillaryItem represents a single ancillary service offering
type AncillaryItem struct {
	ID                   string              `json:"id" bson:"_id"`
	Name                 string              `json:"name" bson:"name"`
	Description          string              `json:"description" bson:"description"`
	Category             AncillaryCategory   `json:"category" bson:"category"`
	BasePrice            float64             `json:"base_price" bson:"base_price"`
	Currency             string              `json:"currency" bson:"currency"`
	Available            bool                `json:"available" bson:"available"`
	MaxQuantity          int                 `json:"max_quantity" bson:"max_quantity"`
	PopularityScore      float64             `json:"popularity_score" bson:"popularity_score"`
	RevenueImpact        float64             `json:"revenue_impact" bson:"revenue_impact"`
	ConversionRate       float64             `json:"conversion_rate" bson:"conversion_rate"`
	SeasonalMultiplier   float64             `json:"seasonal_multiplier" bson:"seasonal_multiplier"`
	RouteMultipliers     map[string]float64  `json:"route_multipliers" bson:"route_multipliers"`
	CustomerSegmentPrice map[string]float64  `json:"customer_segment_price" bson:"customer_segment_price"`
	BundleCompatibility  []string            `json:"bundle_compatibility" bson:"bundle_compatibility"`
	Restrictions         AncillaryRestrictions `json:"restrictions" bson:"restrictions"`
	CreatedAt            time.Time           `json:"created_at" bson:"created_at"`
	UpdatedAt            time.Time           `json:"updated_at" bson:"updated_at"`
}

// AncillaryRestrictions defines restrictions for ancillary items
type AncillaryRestrictions struct {
	RouteRestrictions    []string    `json:"route_restrictions" bson:"route_restrictions"`
	ClassRestrictions    []string    `json:"class_restrictions" bson:"class_restrictions"`
	MinAdvanceBooking    *time.Duration `json:"min_advance_booking" bson:"min_advance_booking"`
	MaxAdvanceBooking    *time.Duration `json:"max_advance_booking" bson:"max_advance_booking"`
	AgeRestrictions      *AgeRestriction `json:"age_restrictions" bson:"age_restrictions"`
	FlightDurationMin    *int        `json:"flight_duration_min" bson:"flight_duration_min"`
	FlightDurationMax    *int        `json:"flight_duration_max" bson:"flight_duration_max"`
}

// AgeRestriction defines age-based restrictions
type AgeRestriction struct {
	MinAge *int `json:"min_age" bson:"min_age"`
	MaxAge *int `json:"max_age" bson:"max_age"`
}

// AncillaryBundle represents a collection of ancillary items sold together
type AncillaryBundle struct {
	ID                   string              `json:"id" bson:"_id"`
	Name                 string              `json:"name" bson:"name"`
	Description          string              `json:"description" bson:"description"`
	Items                []string            `json:"items" bson:"items"` // AncillaryItem IDs
	OriginalPrice        float64             `json:"original_price" bson:"original_price"`
	BundlePrice          float64             `json:"bundle_price" bson:"bundle_price"`
	DiscountPercentage   float64             `json:"discount_percentage" bson:"discount_percentage"`
	Currency             string              `json:"currency" bson:"currency"`
	Available            bool                `json:"available" bson:"available"`
	PopularityScore      float64             `json:"popularity_score" bson:"popularity_score"`
	Category             string              `json:"category" bson:"category"`
	TargetSegments       []string            `json:"target_segments" bson:"target_segments"`
	RouteApplicability   []string            `json:"route_applicability" bson:"route_applicability"`
	ValidFrom            time.Time           `json:"valid_from" bson:"valid_from"`
	ValidTo              time.Time           `json:"valid_to" bson:"valid_to"`
	MaxItemsPerBundle    int                 `json:"max_items_per_bundle" bson:"max_items_per_bundle"`
	CreatedAt            time.Time           `json:"created_at" bson:"created_at"`
	UpdatedAt            time.Time           `json:"updated_at" bson:"updated_at"`
}

// Customer represents customer information for personalization
type Customer struct {
	ID                   string              `json:"id" bson:"_id"`
	Segment              string              `json:"segment" bson:"segment"`
	Tier                 string              `json:"tier" bson:"tier"`
	PreviousPurchases    []string            `json:"previous_purchases" bson:"previous_purchases"`
	PreferredCategories  []AncillaryCategory `json:"preferred_categories" bson:"preferred_categories"`
	SpendingProfile      SpendingProfile     `json:"spending_profile" bson:"spending_profile"`
	TravelFrequency      string              `json:"travel_frequency" bson:"travel_frequency"`
	Route                string              `json:"route" bson:"route"`
	BookingClass         string              `json:"booking_class" bson:"booking_class"`
	TripType             string              `json:"trip_type" bson:"trip_type"`
	CompanionCount       int                 `json:"companion_count" bson:"companion_count"`
	Age                  *int                `json:"age" bson:"age"`
	LastUpdate           time.Time           `json:"last_update" bson:"last_update"`
}

// SpendingProfile represents customer spending patterns
type SpendingProfile struct {
	AverageAncillarySpend float64 `json:"average_ancillary_spend" bson:"average_ancillary_spend"`
	MaxAncillarySpend     float64 `json:"max_ancillary_spend" bson:"max_ancillary_spend"`
	Pricesensitivity     string  `json:"price_sensitivity" bson:"price_sensitivity"` // low, medium, high
	PreferredPriceRange   PriceRange `json:"preferred_price_range" bson:"preferred_price_range"`
}

// PriceRange defines a price range
type PriceRange struct {
	Min float64 `json:"min" bson:"min"`
	Max float64 `json:"max" bson:"max"`
}

// BundleRecommendation represents AI-generated bundle recommendations
type BundleRecommendation struct {
	CustomerID          string                 `json:"customer_id"`
	RecommendedBundles  []RecommendedBundle    `json:"recommended_bundles"`
	IndividualItems     []RecommendedItem      `json:"individual_items"`
	ConfidenceScore     float64                `json:"confidence_score"`
	RecommendationLogic string                 `json:"recommendation_logic"`
	GeneratedAt         time.Time              `json:"generated_at"`
}

// RecommendedBundle represents a recommended bundle with scoring
type RecommendedBundle struct {
	Bundle          AncillaryBundle `json:"bundle"`
	RelevanceScore  float64         `json:"relevance_score"`
	PriceScore      float64         `json:"price_score"`
	PopularityScore float64         `json:"popularity_score"`
	OverallScore    float64         `json:"overall_score"`
	Reasoning       string          `json:"reasoning"`
}

// RecommendedItem represents a recommended individual item with scoring
type RecommendedItem struct {
	Item            AncillaryItem   `json:"item"`
	RelevanceScore  float64         `json:"relevance_score"`
	PriceScore      float64         `json:"price_score"`
	PopularityScore float64         `json:"popularity_score"`
	OverallScore    float64         `json:"overall_score"`
	Reasoning       string          `json:"reasoning"`
}

// AncillaryAnalytics represents analytics data for ancillary services
type AncillaryAnalytics struct {
	ItemID              string    `json:"item_id"`
	TotalSales          int       `json:"total_sales"`
	TotalRevenue        float64   `json:"total_revenue"`
	ConversionRate      float64   `json:"conversion_rate"`
	AveragePrice        float64   `json:"average_price"`
	PopularityTrend     string    `json:"popularity_trend"` // increasing, decreasing, stable
	PerformanceScore    float64   `json:"performance_score"`
	CustomerSatisfaction float64  `json:"customer_satisfaction"`
	ReturnRate          float64   `json:"return_rate"`
	LastUpdated         time.Time `json:"last_updated"`
}

// BundleAnalytics represents analytics data for bundles
type BundleAnalytics struct {
	BundleID            string    `json:"bundle_id"`
	TotalSales          int       `json:"total_sales"`
	TotalRevenue        float64   `json:"total_revenue"`
	ConversionRate      float64   `json:"conversion_rate"`
	AverageDiscount     float64   `json:"average_discount"`
	PopularityTrend     string    `json:"popularity_trend"`
	PerformanceScore    float64   `json:"performance_score"`
	CustomerSatisfaction float64  `json:"customer_satisfaction"`
	AttachRate          float64   `json:"attach_rate"`
	LastUpdated         time.Time `json:"last_updated"`
}

// Methods for AncillaryItem

// GetDynamicPrice calculates dynamic price based on various factors
func (item *AncillaryItem) GetDynamicPrice(customer Customer, route string) float64 {
	price := item.BasePrice

	// Apply seasonal multiplier
	price *= item.SeasonalMultiplier

	// Apply route-specific multiplier
	if multiplier, exists := item.RouteMultipliers[route]; exists {
		price *= multiplier
	}

	// Apply customer segment pricing
	if segmentPrice, exists := item.CustomerSegmentPrice[customer.Segment]; exists {
		price = segmentPrice
	}

	// Apply customer tier discount
	switch customer.Tier {
	case "Diamond":
		price *= 0.85 // 15% discount
	case "Platinum":
		price *= 0.9  // 10% discount
	case "Gold":
		price *= 0.95 // 5% discount
	}

	// Price sensitivity adjustment
	switch customer.SpendingProfile.PriceEnsitivity {
	case "high":
		price *= 0.9  // 10% discount for price-sensitive customers
	case "low":
		price *= 1.1  // 10% premium for price-insensitive customers
	}

	return math.Round(price*100) / 100
}

// IsAvailableForCustomer checks if the item is available for a specific customer
func (item *AncillaryItem) IsAvailableForCustomer(customer Customer) bool {
	if !item.Available {
		return false
	}

	// Check route restrictions
	if len(item.Restrictions.RouteRestrictions) > 0 {
		allowed := false
		for _, route := range item.Restrictions.RouteRestrictions {
			if route == customer.Route {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	// Check class restrictions
	if len(item.Restrictions.ClassRestrictions) > 0 {
		allowed := false
		for _, class := range item.Restrictions.ClassRestrictions {
			if class == customer.BookingClass {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	// Check age restrictions
	if item.Restrictions.AgeRestrictions != nil && customer.Age != nil {
		ageRestriction := item.Restrictions.AgeRestrictions
		if ageRestriction.MinAge != nil && *customer.Age < *ageRestriction.MinAge {
			return false
		}
		if ageRestriction.MaxAge != nil && *customer.Age > *ageRestriction.MaxAge {
			return false
		}
	}

	return true
}

// Methods for AncillaryBundle

// CalculateBundlePrice calculates the total price of all items in the bundle
func (bundle *AncillaryBundle) CalculateBundlePrice(items []AncillaryItem) float64 {
	var total float64
	for _, item := range items {
		total += item.BasePrice
	}
	return total
}

// CalculateDiscount calculates the discount amount for the bundle
func (bundle *AncillaryBundle) CalculateDiscount() float64 {
	return bundle.OriginalPrice - bundle.BundlePrice
}

// IsValidForDates checks if the bundle is valid for the given date
func (bundle *AncillaryBundle) IsValidForDates(date time.Time) bool {
	return date.After(bundle.ValidFrom) && date.Before(bundle.ValidTo)
}

// Default ancillary items catalog
func GetDefaultAncillaryItems() []AncillaryItem {
	return []AncillaryItem{
		{
			ID:                  "baggage-20kg",
			Name:                "Extra Baggage (20kg)",
			Description:         "Additional 20kg checked baggage allowance",
			Category:            CategoryBaggage,
			BasePrice:           35.00,
			Currency:            "USD",
			Available:           true,
			MaxQuantity:         3,
			PopularityScore:     0.82,
			RevenueImpact:       2.3,
			ConversionRate:      0.24,
			SeasonalMultiplier:  1.0,
			RouteMultipliers:    map[string]float64{
				"NYC-LON": 1.2,
				"LAX-TOK": 1.3,
				"SFO-FRA": 1.1,
			},
			CustomerSegmentPrice: map[string]float64{
				"Business Elite": 42.00,
				"Family Traveler": 30.00,
			},
			BundleCompatibility: []string{"comfort", "family", "business"},
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		},
		{
			ID:                  "wifi-premium",
			Name:                "Premium WiFi",
			Description:         "High-speed internet access throughout the flight",
			Category:            CategoryConnectivity,
			BasePrice:           12.99,
			Currency:            "USD",
			Available:           true,
			MaxQuantity:         1,
			PopularityScore:     0.91,
			RevenueImpact:       1.8,
			ConversionRate:      0.45,
			SeasonalMultiplier:  1.0,
			RouteMultipliers:    map[string]float64{
				"NYC-LON": 1.1,
				"LAX-TOK": 1.2,
			},
			CustomerSegmentPrice: map[string]float64{
				"Business Elite": 15.99,
				"Tech Savvy": 10.99,
			},
			BundleCompatibility: []string{"business", "comfort", "tech"},
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		},
		{
			ID:                  "seat-premium",
			Name:                "Premium Seat Selection",
			Description:         "Extra legroom and preferred seating",
			Category:            CategoryComfort,
			BasePrice:           45.00,
			Currency:            "USD",
			Available:           true,
			MaxQuantity:         1,
			PopularityScore:     0.76,
			RevenueImpact:       3.2,
			ConversionRate:      0.31,
			SeasonalMultiplier:  1.0,
			RouteMultipliers:    map[string]float64{
				"NYC-LON": 1.3,
				"LAX-TOK": 1.4,
			},
			CustomerSegmentPrice: map[string]float64{
				"Business Elite": 55.00,
				"Comfort Seeker": 40.00,
			},
			BundleCompatibility: []string{"comfort", "business", "premium"},
			Restrictions: AncillaryRestrictions{
				ClassRestrictions: []string{"Economy", "Premium Economy"},
			},
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
		{
			ID:                  "meal-premium",
			Name:                "Premium Meal",
			Description:         "Chef-curated premium dining experience",
			Category:            CategoryDining,
			BasePrice:           28.50,
			Currency:            "USD",
			Available:           true,
			MaxQuantity:         2,
			PopularityScore:     0.68,
			RevenueImpact:       2.1,
			ConversionRate:      0.29,
			SeasonalMultiplier:  1.0,
			CustomerSegmentPrice: map[string]float64{
				"Business Elite": 35.00,
				"Food Lover": 25.00,
			},
			BundleCompatibility: []string{"dining", "premium", "comfort"},
			Restrictions: AncillaryRestrictions{
				FlightDurationMin: &[]int{180}[0], // 3 hours minimum
			},
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
		{
			ID:                  "lounge-access",
			Name:                "Airport Lounge Access",
			Description:         "Access to premium airport lounges",
			Category:            CategoryGroundService,
			BasePrice:           45.00,
			Currency:            "USD",
			Available:           true,
			MaxQuantity:         1,
			PopularityScore:     0.54,
			RevenueImpact:       2.8,
			ConversionRate:      0.18,
			SeasonalMultiplier:  1.0,
			CustomerSegmentPrice: map[string]float64{
				"Business Elite": 55.00,
				"Frequent Flyer": 40.00,
			},
			BundleCompatibility: []string{"business", "premium", "comfort"},
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
		{
			ID:                  "fast-track",
			Name:                "Fast Track Security",
			Description:         "Priority security screening",
			Category:            CategoryConvenience,
			BasePrice:           18.00,
			Currency:            "USD",
			Available:           true,
			MaxQuantity:         1,
			PopularityScore:     0.71,
			RevenueImpact:       1.5,
			ConversionRate:      0.33,
			SeasonalMultiplier:  1.0,
			CustomerSegmentPrice: map[string]float64{
				"Business Elite": 22.00,
				"Time Conscious": 15.00,
			},
			BundleCompatibility: []string{"convenience", "business", "premium"},
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
		{
			ID:                  "priority-boarding",
			Name:                "Priority Boarding",
			Description:         "Board first and settle in comfortably",
			Category:            CategoryConvenience,
			BasePrice:           15.00,
			Currency:            "USD",
			Available:           true,
			MaxQuantity:         1,
			PopularityScore:     0.79,
			RevenueImpact:       1.2,
			ConversionRate:      0.38,
			SeasonalMultiplier:  1.0,
			CustomerSegmentPrice: map[string]float64{
				"Business Elite": 18.00,
				"Frequent Flyer": 12.00,
			},
			BundleCompatibility: []string{"convenience", "comfort", "business"},
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
		{
			ID:                  "insurance-travel",
			Name:                "Travel Insurance",
			Description:         "Comprehensive travel protection",
			Category:            CategoryProtection,
			BasePrice:           24.99,
			Currency:            "USD",
			Available:           true,
			MaxQuantity:         1,
			PopularityScore:     0.43,
			RevenueImpact:       1.9,
			ConversionRate:      0.15,
			SeasonalMultiplier:  1.0,
			CustomerSegmentPrice: map[string]float64{
				"Risk Averse": 20.99,
				"International Traveler": 27.99,
			},
			BundleCompatibility: []string{"protection", "international"},
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
	}
}

// Default bundle catalog
func GetDefaultBundles() []AncillaryBundle {
	return []AncillaryBundle{
		{
			ID:                "comfort-plus",
			Name:              "Comfort Plus Bundle",
			Description:       "Premium seating, WiFi, and priority boarding",
			Items:             []string{"seat-premium", "wifi-premium", "priority-boarding"},
			OriginalPrice:     72.99,
			BundlePrice:       59.99,
			DiscountPercentage: 18,
			Currency:          "USD",
			Available:         true,
			PopularityScore:   0.85,
			Category:          "comfort",
			TargetSegments:    []string{"Comfort Seeker", "Business Elite"},
			RouteApplicability: []string{"NYC-LON", "LAX-TOK", "SFO-FRA"},
			ValidFrom:         time.Now(),
			ValidTo:           time.Now().AddDate(1, 0, 0),
			MaxItemsPerBundle: 4,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                "business-traveler",
			Name:              "Business Traveler Bundle",
			Description:       "Fast track, lounge access, and premium WiFi",
			Items:             []string{"fast-track", "lounge-access", "wifi-premium"},
			OriginalPrice:     75.99,
			BundlePrice:       64.99,
			DiscountPercentage: 14,
			Currency:          "USD",
			Available:         true,
			PopularityScore:   0.72,
			Category:          "business",
			TargetSegments:    []string{"Business Elite", "Frequent Flyer"},
			RouteApplicability: []string{"NYC-LON", "LAX-TOK", "SFO-FRA", "MIA-MAD"},
			ValidFrom:         time.Now(),
			ValidTo:           time.Now().AddDate(1, 0, 0),
			MaxItemsPerBundle: 4,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                "family-pack",
			Name:              "Family Pack Bundle",
			Description:       "Extra baggage, meals, and entertainment",
			Items:             []string{"baggage-20kg", "meal-premium", "wifi-premium"},
			OriginalPrice:     76.49,
			BundlePrice:       65.99,
			DiscountPercentage: 14,
			Currency:          "USD",
			Available:         true,
			PopularityScore:   0.69,
			Category:          "family",
			TargetSegments:    []string{"Family Traveler"},
			RouteApplicability: []string{"NYC-LON", "LAX-TOK", "SFO-FRA", "MIA-MAD"},
			ValidFrom:         time.Now(),
			ValidTo:           time.Now().AddDate(1, 0, 0),
			MaxItemsPerBundle: 5,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                "premium-experience",
			Name:              "Premium Experience Bundle",
			Description:       "Complete premium experience package",
			Items:             []string{"seat-premium", "wifi-premium", "meal-premium", "lounge-access"},
			OriginalPrice:     131.49,
			BundlePrice:       99.99,
			DiscountPercentage: 24,
			Currency:          "USD",
			Available:         true,
			PopularityScore:   0.58,
			Category:          "premium",
			TargetSegments:    []string{"Business Elite", "Luxury Traveler"},
			RouteApplicability: []string{"NYC-LON", "LAX-TOK", "SFO-FRA"},
			ValidFrom:         time.Now(),
			ValidTo:           time.Now().AddDate(1, 0, 0),
			MaxItemsPerBundle: 6,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
	}
} 