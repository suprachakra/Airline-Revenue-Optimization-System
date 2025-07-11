package constants

import (
	"time"
	"github.com/shopspring/decimal"
)

// PricingConstants contains all configurable pricing values to eliminate hardcoded numbers
type PricingConstants struct {
	// Base Fare Configuration
	DefaultBaseFare     decimal.Decimal
	MinimumFare         decimal.Decimal
	MaximumFare         decimal.Decimal
	EmergencyFallbackFare decimal.Decimal
	
	// Route-Specific Base Fares (USD)
	RouteFares          map[string]decimal.Decimal
	
	// Booking Class Multipliers
	ClassMultipliers    map[string]decimal.Decimal
	
	// Dynamic Adjustment Factors
	DemandAdjustments   map[string]decimal.Decimal
	SeasonalFactors     map[string]decimal.Decimal
	CompetitorFactors   map[string]decimal.Decimal
	
	// Operational Parameters
	TaxRate             decimal.Decimal
	FixedFees           decimal.Decimal
	MaxDiscountPercent  decimal.Decimal
	MaxSurchargePercent decimal.Decimal
	
	// Cache and Performance
	CacheValidityPeriod time.Duration
	PriceValidityPeriod time.Duration
	FallbackTimeout     time.Duration
	
	// Business Rules
	MinProfitMargin     decimal.Decimal
	CorporateDiscount   decimal.Decimal
	LoyaltyDiscount     decimal.Decimal
	
	// Advanced Booking Windows
	EarlyBookingDiscount decimal.Decimal
	LastMinuteSurcharge  decimal.Decimal
	EarlyBookingDays     int
	LastMinuteDays       int
}

// GetDefaultPricingConstants returns production-ready pricing configuration
func GetDefaultPricingConstants() *PricingConstants {
	return &PricingConstants{
		// Base Configuration
		DefaultBaseFare:     decimal.NewFromFloat(500.0),
		MinimumFare:         decimal.NewFromFloat(50.0),
		MaximumFare:         decimal.NewFromFloat(5000.0),
		EmergencyFallbackFare: decimal.NewFromFloat(300.0),
		
		// Route-Specific Fares
		RouteFares: map[string]decimal.Decimal{
			"NYC-LON": decimal.NewFromFloat(650.0),
			"NYC-PAR": decimal.NewFromFloat(700.0),
			"NYC-FRA": decimal.NewFromFloat(680.0),
			"NYC-AMS": decimal.NewFromFloat(620.0),
			"NYC-MAD": decimal.NewFromFloat(670.0),
			"DXB-BOM": decimal.NewFromFloat(180.0),
			"DXB-DEL": decimal.NewFromFloat(190.0),
			"DXB-BLR": decimal.NewFromFloat(170.0),
			"LHR-FRA": decimal.NewFromFloat(120.0),
			"LHR-CDG": decimal.NewFromFloat(110.0),
			"SIN-KUL": decimal.NewFromFloat(80.0),
			"SIN-BKK": decimal.NewFromFloat(90.0),
			"HKG-TPE": decimal.NewFromFloat(95.0),
			"LAX-SFO": decimal.NewFromFloat(150.0),
			"JFK-MIA": decimal.NewFromFloat(200.0),
			"ORD-DEN": decimal.NewFromFloat(180.0),
			"ATL-DFW": decimal.NewFromFloat(160.0),
			"SEA-LAX": decimal.NewFromFloat(140.0),
			"BOS-DCA": decimal.NewFromFloat(120.0),
			"YYZ-YVR": decimal.NewFromFloat(220.0),
			"SYD-MEL": decimal.NewFromFloat(100.0),
			"NRT-ICN": decimal.NewFromFloat(130.0),
			"CDG-FCO": decimal.NewFromFloat(90.0),
		},
		
		// Class Multipliers
		ClassMultipliers: map[string]decimal.Decimal{
			"economy":         decimal.NewFromFloat(1.0),
			"premium_economy": decimal.NewFromFloat(1.3),
			"business":        decimal.NewFromFloat(3.0),
			"first":           decimal.NewFromFloat(6.0),
		},
		
		// Demand-Based Adjustments
		DemandAdjustments: map[string]decimal.Decimal{
			"HIGH":   decimal.NewFromFloat(0.30), // +30% for high demand
			"MEDIUM": decimal.NewFromFloat(0.05), // +5% for medium demand
			"LOW":    decimal.NewFromFloat(-0.10), // -10% for low demand
		},
		
		// Seasonal Factors
		SeasonalFactors: map[string]decimal.Decimal{
			"PEAK_SUMMER":   decimal.NewFromFloat(1.25),
			"PEAK_WINTER":   decimal.NewFromFloat(1.20),
			"PEAK_HOLIDAY":  decimal.NewFromFloat(1.35),
			"OFF_SEASON":    decimal.NewFromFloat(0.85),
			"SHOULDER":      decimal.NewFromFloat(0.95),
		},
		
		// Competitor Response Factors
		CompetitorFactors: map[string]decimal.Decimal{
			"AGGRESSIVE": decimal.NewFromFloat(0.95), // Price 5% below market
			"NEUTRAL":    decimal.NewFromFloat(1.0),  // Market rate
			"PREMIUM":    decimal.NewFromFloat(1.05), // Price 5% above market
		},
		
		// Operational Parameters
		TaxRate:             decimal.NewFromFloat(0.15), // 15% tax rate
		FixedFees:           decimal.NewFromFloat(25.0), // $25 fixed fees
		MaxDiscountPercent:  decimal.NewFromFloat(0.30), // 30% max discount
		MaxSurchargePercent: decimal.NewFromFloat(0.50), // 50% max surcharge
		
		// Cache and Timing
		CacheValidityPeriod: 15 * time.Minute,
		PriceValidityPeriod: 15 * time.Minute,
		FallbackTimeout:     5 * time.Second,
		
		// Business Rules
		MinProfitMargin:     decimal.NewFromFloat(0.15), // 15% minimum margin
		CorporateDiscount:   decimal.NewFromFloat(0.10), // 10% corporate discount
		LoyaltyDiscount:     decimal.NewFromFloat(0.05), // 5% loyalty discount
		
		// Booking Window Adjustments
		EarlyBookingDiscount: decimal.NewFromFloat(0.15), // 15% discount for early booking
		LastMinuteSurcharge:  decimal.NewFromFloat(0.25), // 25% surcharge for last minute
		EarlyBookingDays:     60,                         // 60+ days = early booking
		LastMinuteDays:       14,                         // <14 days = last minute
	}
}

// GeoFencingConstants contains geographic pricing adjustments
type GeoFencingConstants struct {
	RegionalAdjustments map[string]decimal.Decimal
	CurrencyFactors     map[string]decimal.Decimal
	LocalTaxRates       map[string]decimal.Decimal
}

// GetGeoFencingConstants returns geographic pricing configuration
func GetGeoFencingConstants() *GeoFencingConstants {
	return &GeoFencingConstants{
		// Regional Price Adjustments
		RegionalAdjustments: map[string]decimal.Decimal{
			"US":     decimal.NewFromFloat(1.0),   // Base rate
			"EU":     decimal.NewFromFloat(1.1),   // 10% higher
			"ASIA":   decimal.NewFromFloat(0.85),  // 15% lower
			"MENA":   decimal.NewFromFloat(0.9),   // 10% lower
			"LATAM":  decimal.NewFromFloat(0.8),   // 20% lower
			"AFRICA": decimal.NewFromFloat(0.75),  // 25% lower
		},
		
		// Currency Conversion Factors (relative to USD)
		CurrencyFactors: map[string]decimal.Decimal{
			"USD": decimal.NewFromFloat(1.0),
			"EUR": decimal.NewFromFloat(0.92),
			"GBP": decimal.NewFromFloat(0.79),
			"JPY": decimal.NewFromFloat(149.5),
			"AUD": decimal.NewFromFloat(1.52),
			"CAD": decimal.NewFromFloat(1.36),
			"INR": decimal.NewFromFloat(83.2),
			"SGD": decimal.NewFromFloat(1.35),
		},
		
		// Local Tax Rates by Country/Region
		LocalTaxRates: map[string]decimal.Decimal{
			"US":  decimal.NewFromFloat(0.072), // 7.2% average
			"EU":  decimal.NewFromFloat(0.20),  // 20% VAT
			"UK":  decimal.NewFromFloat(0.20),  // 20% VAT
			"CA":  decimal.NewFromFloat(0.13),  // 13% HST
			"AU":  decimal.NewFromFloat(0.10),  // 10% GST
			"IN":  decimal.NewFromFloat(0.18),  // 18% GST
			"SG":  decimal.NewFromFloat(0.07),  // 7% GST
		},
	}
}

// RegulatoryConstants contains compliance-related pricing parameters
type RegulatoryConstants struct {
	DOTPricingCaps      map[string]decimal.Decimal
	IATAComplianceRules map[string]interface{}
	DisclosureRequirements map[string][]string
}

// GetRegulatoryConstants returns regulatory compliance configuration
func GetRegulatoryConstants() *RegulatoryConstants {
	return &RegulatoryConstants{
		// DOT Price Caps (US Domestic Routes)
		DOTPricingCaps: map[string]decimal.Decimal{
			"DOMESTIC_SHORT": decimal.NewFromFloat(500.0),  // <500 miles
			"DOMESTIC_MEDIUM": decimal.NewFromFloat(800.0), // 500-1500 miles
			"DOMESTIC_LONG": decimal.NewFromFloat(1200.0),  // >1500 miles
		},
		
		// IATA NDC Compliance Rules
		IATAComplianceRules: map[string]interface{}{
			"NDC_LEVEL_4": true,
			"FARE_BREAKDOWN_REQUIRED": true,
			"TAX_TRANSPARENCY": true,
			"CURRENCY_DISCLOSURE": true,
		},
		
		// Required Disclosure by Region
		DisclosureRequirements: map[string][]string{
			"US": {"total_price", "tax_breakdown", "fees", "cancellation_terms"},
			"EU": {"total_price", "tax_breakdown", "fees", "passenger_rights", "environmental_impact"},
			"APAC": {"total_price", "tax_breakdown", "fees", "baggage_policy"},
		},
	}
}

// PerformanceConstants contains performance-related thresholds
type PerformanceConstants struct {
	ResponseTimeTargets map[string]time.Duration
	CacheHitRateTargets map[string]float64
	ErrorRateThresholds map[string]float64
}

// GetPerformanceConstants returns performance monitoring configuration
func GetPerformanceConstants() *PerformanceConstants {
	return &PerformanceConstants{
		ResponseTimeTargets: map[string]time.Duration{
			"PRICING_CALCULATION": 200 * time.Millisecond,
			"FALLBACK_ACTIVATION": 50 * time.Millisecond,
			"CACHE_LOOKUP": 10 * time.Millisecond,
			"DATABASE_QUERY": 100 * time.Millisecond,
		},
		
		CacheHitRateTargets: map[string]float64{
			"ROUTE_PRICING": 0.85,
			"COMPETITOR_DATA": 0.90,
			"CUSTOMER_SEGMENTS": 0.95,
		},
		
		ErrorRateThresholds: map[string]float64{
			"PRICING_ERRORS": 0.001,  // 0.1% max error rate
			"FALLBACK_USAGE": 0.01,   // 1% max fallback usage
			"CACHE_MISSES": 0.15,     // 15% max cache miss rate
		},
	}
} 