package loyalty

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"sync"
	"time"
	"log"
)

// LoyaltyServiceConfig holds configuration for the loyalty service
type LoyaltyServiceConfig struct {
	BaseURL        string
	Timeout        time.Duration
	RetryAttempts  int
	CacheExpiry    time.Duration
	RateLimitRPS   int
	EnableFallback bool
	TierResetDays  int
	PointsExpiry   int // days
}

// LoyaltyTier represents different loyalty program tiers
type LoyaltyTier int

const (
	Basic LoyaltyTier = iota
	Silver
	Gold
	Platinum
	Diamond
)

// CustomerSegment represents different customer segments for personalization
type CustomerSegment int

const (
	NewCustomer CustomerSegment = iota
	RegularCustomer
	FrequentTraveler
	BusinessTraveler
	VIPCustomer
	InactiveCustomer
)

// LoyaltyProgram represents a comprehensive loyalty program
type LoyaltyProgram struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Tiers           []TierBenefits        `json:"tiers"`
	Active          bool                  `json:"active"`
	PointsEarningRate map[string]float64   `json:"points_earning_rate"`
	RedemptionRates map[string]float64     `json:"redemption_rates"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// TierBenefits represents enhanced benefits for each loyalty tier
type TierBenefits struct {
	Tier               LoyaltyTier `json:"tier"`
	Name               string      `json:"name"`
	MinPoints          int         `json:"min_points"`
	MinFlights         int         `json:"min_flights"`
	DiscountRate       float64     `json:"discount_rate"`
	BonusMultiplier    float64     `json:"bonus_multiplier"`
	Benefits           []string    `json:"benefits"`
	PriorityLevel      int         `json:"priority_level"`
	ExpiryProtection   bool        `json:"expiry_protection"`
	CompanionBenefits  bool        `json:"companion_benefits"`
	UpgradeEligibility bool        `json:"upgrade_eligibility"`
}

// CustomerLoyalty represents comprehensive customer loyalty information
type CustomerLoyalty struct {
	CustomerID         string          `json:"customer_id"`
	ProgramID          string          `json:"program_id"`
	MemberNumber       string          `json:"member_number"`
	CurrentTier        LoyaltyTier     `json:"current_tier"`
	NextTier           LoyaltyTier     `json:"next_tier"`
	Points             int             `json:"points"`
	LifetimePoints     int             `json:"lifetime_points"`
	FlightCount        int             `json:"flight_count"`
	LifetimeFlights    int             `json:"lifetime_flights"`
	Segment            CustomerSegment `json:"segment"`
	TierProgress       float64         `json:"tier_progress"`
	PointsToNextTier   int             `json:"points_to_next_tier"`
	FlightsToNextTier  int             `json:"flights_to_next_tier"`
	JoinDate           time.Time       `json:"join_date"`
	LastActivity       time.Time       `json:"last_activity"`
	TierAchievedDate   time.Time       `json:"tier_achieved_date"`
	Status             string          `json:"status"`
	PreferredRoutes    []string        `json:"preferred_routes"`
	AvgSpendPerTrip    float64         `json:"avg_spend_per_trip"`
	Metadata           map[string]interface{} `json:"metadata"`
}

// LoyaltyAdjustment represents comprehensive loyalty-based pricing adjustment
type LoyaltyAdjustment struct {
	CustomerID         string                 `json:"customer_id"`
	Route              string                 `json:"route"`
	OriginalPrice      float64                `json:"original_price"`
	AdjustedPrice      float64                `json:"adjusted_price"`
	DiscountAmount     float64                `json:"discount_amount"`
	DiscountRate       float64                `json:"discount_rate"`
	Tier               LoyaltyTier           `json:"tier"`
	Segment            CustomerSegment       `json:"segment"`
	Points             int                   `json:"points"`
	RouteMultiplier    float64               `json:"route_multiplier"`
	SegmentMultiplier  float64               `json:"segment_multiplier"`
	SeasonalBonus      float64               `json:"seasonal_bonus"`
	PersonalizationScore float64             `json:"personalization_score"`
	Timestamp          time.Time             `json:"timestamp"`
	Metadata           map[string]interface{} `json:"metadata"`
}

// LoyaltyCache stores cached loyalty data with thread-safe operations
type LoyaltyCache struct {
	mu          sync.RWMutex
	customers   map[string]CustomerLoyalty
	adjustments map[string]LoyaltyAdjustment
	expiry      map[string]time.Time
	hitCount    int64
	missCount   int64
}

// LoyaltyService handles all loyalty operations
type LoyaltyService struct {
	config      LoyaltyServiceConfig
	cache       *LoyaltyCache
	httpClient  *http.Client
	rateLimiter chan struct{}
	metrics     *LoyaltyMetrics
}

// LoyaltyMetrics tracks loyalty service metrics
type LoyaltyMetrics struct {
	mu                sync.RWMutex
	TotalRequests     int64
	CacheHits         int64
	CacheMisses       int64
	ServiceCalls      int64
	FallbackCalls     int64
	ErrorCount        int64
	AvgResponseTime   time.Duration
	TierDistribution  map[LoyaltyTier]int64
	SegmentDistribution map[CustomerSegment]int64
	LastUpdated       time.Time
}

// Global loyalty service instance
var loyaltyService *LoyaltyService
var once sync.Once

// Enhanced loyalty program with comprehensive tier system
var enhancedProgram = LoyaltyProgram{
	ID:          "airline-rewards-plus",
	Name:        "Airline Rewards Plus Program",
	Description: "Enhanced loyalty program with personalized benefits and tier progression",
	Tiers: []TierBenefits{
		{
			Tier:               Basic,
			Name:               "Basic Member",
			MinPoints:          0,
			MinFlights:         0,
			DiscountRate:       0.0,
			BonusMultiplier:    1.0,
			Benefits:           []string{"Standard check-in", "Newsletter", "Basic customer support"},
			PriorityLevel:      0,
			ExpiryProtection:   false,
			CompanionBenefits:  false,
			UpgradeEligibility: false,
		},
		{
			Tier:               Silver,
			Name:               "Silver Elite",
			MinPoints:          5000,
			MinFlights:         4,
			DiscountRate:       0.05,
			BonusMultiplier:    1.25,
			Benefits:           []string{"Priority check-in", "Extra baggage allowance", "Lounge access discounts", "Seat selection priority"},
			PriorityLevel:      1,
			ExpiryProtection:   false,
			CompanionBenefits:  false,
			UpgradeEligibility: true,
		},
		{
			Tier:               Gold,
			Name:               "Gold Elite",
			MinPoints:          15000,
			MinFlights:         12,
			DiscountRate:       0.08,
			BonusMultiplier:    1.5,
			Benefits:           []string{"Priority boarding", "Free seat selection", "Complimentary upgrades", "Premium customer service", "Lounge access"},
			PriorityLevel:      2,
			ExpiryProtection:   true,
			CompanionBenefits:  true,
			UpgradeEligibility: true,
		},
		{
			Tier:               Platinum,
			Name:               "Platinum Elite",
			MinPoints:          35000,
			MinFlights:         25,
			DiscountRate:       0.12,
			BonusMultiplier:    1.75,
			Benefits:           []string{"Premium lounge access", "Fast track security", "Guaranteed upgrades", "Dedicated phone line", "Companion benefits"},
			PriorityLevel:      3,
			ExpiryProtection:   true,
			CompanionBenefits:  true,
			UpgradeEligibility: true,
		},
		{
			Tier:               Diamond,
			Name:               "Diamond Elite",
			MinPoints:          75000,
			MinFlights:         50,
			DiscountRate:       0.15,
			BonusMultiplier:    2.0,
			Benefits:           []string{"Concierge service", "Free companions", "Lifetime benefits", "First-class lounge access", "Personal travel advisor"},
			PriorityLevel:      4,
			ExpiryProtection:   true,
			CompanionBenefits:  true,
			UpgradeEligibility: true,
		},
	},
	Active: true,
	PointsEarningRate: map[string]float64{
		"base":         1.0,
		"premium":      1.5,
		"business":     2.0,
		"first":        3.0,
		"partner":      0.5,
		"bonus_promo":  2.5,
	},
	RedemptionRates: map[string]float64{
		"flight_discount": 100.0, // 100 points = $1
		"upgrade":         150.0, // 150 points = $1 upgrade value
		"ancillary":       80.0,  // 80 points = $1 ancillary value
		"partner":         120.0, // 120 points = $1 partner value
	},
	CreatedAt: time.Now(),
	UpdatedAt: time.Now(),
	Metadata: map[string]interface{}{
		"program_version": "2.1",
		"max_tier":        "Diamond",
		"tier_reset":      "annual",
	},
}

// Initialize initializes the loyalty service
func Initialize() error {
	once.Do(func() {
		config := LoyaltyServiceConfig{
			BaseURL:        "http://loyalty-service:8080",
			Timeout:        5 * time.Second,
			RetryAttempts:  3,
			CacheExpiry:    2 * time.Hour,
			RateLimitRPS:   100,
			EnableFallback: true,
			TierResetDays:  365,
			PointsExpiry:   730, // 2 years
		}

		loyaltyService = &LoyaltyService{
			config: config,
			cache: &LoyaltyCache{
				customers:   make(map[string]CustomerLoyalty),
				adjustments: make(map[string]LoyaltyAdjustment),
				expiry:      make(map[string]time.Time),
			},
			httpClient: &http.Client{
				Timeout: config.Timeout,
			},
			rateLimiter: make(chan struct{}, config.RateLimitRPS),
			metrics: &LoyaltyMetrics{
				LastUpdated:         time.Now(),
				TierDistribution:    make(map[LoyaltyTier]int64),
				SegmentDistribution: make(map[CustomerSegment]int64),
			},
		}

		// Initialize rate limiter
		go loyaltyService.rateLimiterWorker()
	})

	return nil
}

// CalculateLoyaltyAdjustment calculates comprehensive loyalty-based pricing adjustment
func CalculateLoyaltyAdjustment(route string) float64 {
	// Ensure service is initialized
	if err := Initialize(); err != nil {
		return calculateDefaultAdjustment(route)
	}

	// For demo purposes, simulate a Silver tier customer
	customerID := "demo-customer"
	adjustment, err := loyaltyService.GetCustomerAdjustment(customerID, route, 500.0)
	if err != nil {
		log.Printf("Failed to get customer adjustment: %v", err)
		return calculateDefaultAdjustment(route)
	}

	return adjustment.DiscountAmount
}

// GetCustomerAdjustment calculates personalized loyalty adjustment for a specific customer
func (ls *LoyaltyService) GetCustomerAdjustment(customerID, route string, originalPrice float64) (LoyaltyAdjustment, error) {
	startTime := time.Now()

	// Update metrics
	ls.metrics.mu.Lock()
	ls.metrics.TotalRequests++
	ls.metrics.mu.Unlock()

	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", customerID, route)
	if cachedAdjustment, exists := ls.getCachedAdjustment(cacheKey); exists {
		ls.metrics.mu.Lock()
		ls.metrics.CacheHits++
		ls.metrics.mu.Unlock()

		// Update price in cached adjustment
		cachedAdjustment.OriginalPrice = originalPrice
		cachedAdjustment.AdjustedPrice = originalPrice - cachedAdjustment.DiscountAmount

		return cachedAdjustment, nil
	}

	// Cache miss
	ls.metrics.mu.Lock()
	ls.metrics.CacheMisses++
	ls.metrics.mu.Unlock()

	// Get customer loyalty information
	customer, err := ls.GetEnhancedCustomerLoyalty(customerID)
	if err != nil {
		ls.metrics.mu.Lock()
		ls.metrics.ErrorCount++
		ls.metrics.mu.Unlock()

		// Return default adjustment
		return LoyaltyAdjustment{
			CustomerID:    customerID,
			Route:         route,
			OriginalPrice: originalPrice,
			AdjustedPrice: originalPrice,
			DiscountAmount: 0.0,
			DiscountRate:  0.0,
			Tier:         Basic,
			Segment:      RegularCustomer,
			Timestamp:    time.Now(),
		}, nil
	}

	// Calculate comprehensive adjustment
	adjustment := ls.calculateComprehensiveAdjustment(customer, route, originalPrice)

	// Cache the adjustment
	ls.setCachedAdjustment(cacheKey, adjustment)

	// Update response time metrics
	responseTime := time.Since(startTime)
	ls.metrics.mu.Lock()
	ls.metrics.AvgResponseTime = (ls.metrics.AvgResponseTime + responseTime) / 2
	ls.metrics.LastUpdated = time.Now()
	ls.metrics.mu.Unlock()

	return adjustment, nil
}

// calculateComprehensiveAdjustment applies sophisticated loyalty adjustment logic
func (ls *LoyaltyService) calculateComprehensiveAdjustment(customer CustomerLoyalty, route string, originalPrice float64) LoyaltyAdjustment {
	// Get tier benefits
	tierBenefits := getTierBenefits(customer.CurrentTier)

	// Base discount from tier
	baseDiscount := tierBenefits.DiscountRate

	// Route-specific multiplier
	routeMultiplier := getRouteMultiplier(route)

	// Customer segment multiplier
	segmentMultiplier := getSegmentMultiplier(customer.Segment)

	// Seasonal bonus (if applicable)
	seasonalBonus := calculateSeasonalBonus(customer, route)

	// Loyalty history bonus
	historyBonus := calculateHistoryBonus(customer)

	// Preferred route bonus
	preferredRouteBonus := calculatePreferredRouteBonus(customer, route)

	// Calculate total discount rate
	totalDiscountRate := baseDiscount * routeMultiplier * segmentMultiplier + seasonalBonus + historyBonus + preferredRouteBonus

	// Cap the discount at 25%
	if totalDiscountRate > 0.25 {
		totalDiscountRate = 0.25
	}

	// Calculate discount amount
	discountAmount := originalPrice * totalDiscountRate
	adjustedPrice := originalPrice - discountAmount

	// Calculate personalization score
	personalizationScore := calculatePersonalizationScore(customer, route, totalDiscountRate)

	adjustment := LoyaltyAdjustment{
		CustomerID:           customer.CustomerID,
		Route:                route,
		OriginalPrice:        originalPrice,
		AdjustedPrice:        adjustedPrice,
		DiscountAmount:       discountAmount,
		DiscountRate:         totalDiscountRate,
		Tier:                 customer.CurrentTier,
		Segment:              customer.Segment,
		Points:               customer.Points,
		RouteMultiplier:      routeMultiplier,
		SegmentMultiplier:    segmentMultiplier,
		SeasonalBonus:        seasonalBonus,
		PersonalizationScore: personalizationScore,
		Timestamp:            time.Now(),
		Metadata: map[string]interface{}{
			"base_discount":          baseDiscount,
			"history_bonus":          historyBonus,
			"preferred_route_bonus":  preferredRouteBonus,
			"tier_name":              tierBenefits.Name,
			"lifetime_points":        customer.LifetimePoints,
			"lifetime_flights":       customer.LifetimeFlights,
		},
	}

	return adjustment
}

// GetEnhancedCustomerLoyalty retrieves comprehensive customer loyalty information
func (ls *LoyaltyService) GetEnhancedCustomerLoyalty(customerID string) (CustomerLoyalty, error) {
	if customerID == "" {
		return CustomerLoyalty{}, errors.New("customer ID cannot be empty")
	}

	// Check cache first
	if cachedCustomer, exists := ls.getCachedCustomer(customerID); exists {
		return cachedCustomer, nil
	}

	// Generate comprehensive customer loyalty data
	customer := ls.generateEnhancedCustomer(customerID)

	// Cache the customer
	ls.setCachedCustomer(customerID, customer)

	return customer, nil
}

// generateEnhancedCustomer creates comprehensive loyalty data for a customer
func (ls *LoyaltyService) generateEnhancedCustomer(customerID string) CustomerLoyalty {
	// Simulate different customer profiles with realistic data
	profiles := []CustomerLoyalty{
		// Silver Elite Customer
		{
			CustomerID:         customerID,
			ProgramID:          enhancedProgram.ID,
			MemberNumber:       fmt.Sprintf("ARP%s", customerID[:6]),
			CurrentTier:        Silver,
			NextTier:           Gold,
			Points:             8500,
			LifetimePoints:     28500,
			FlightCount:        6,
			LifetimeFlights:    18,
			Segment:            FrequentTraveler,
			TierProgress:       0.57, // 57% to Gold
			PointsToNextTier:   6500,
			FlightsToNextTier:  6,
			JoinDate:           time.Now().AddDate(-2, -3, 0),
			LastActivity:       time.Now().AddDate(0, 0, -15),
			TierAchievedDate:   time.Now().AddDate(0, -8, 0),
			Status:             "ACTIVE",
			PreferredRoutes:    []string{"NYC-LON", "LON-PAR", "NYC-LAX"},
			AvgSpendPerTrip:    485.50,
			Metadata: map[string]interface{}{
				"profile_type":     "frequent_traveler",
				"travel_purpose":   "business",
				"preferred_class":  "Y",
				"communication":    "email",
			},
		},
		// Gold Elite Customer
		{
			CustomerID:         customerID,
			ProgramID:          enhancedProgram.ID,
			MemberNumber:       fmt.Sprintf("ARP%s", customerID[:6]),
			CurrentTier:        Gold,
			NextTier:           Platinum,
			Points:             22000,
			LifetimePoints:     75000,
			FlightCount:        15,
			LifetimeFlights:    45,
			Segment:            BusinessTraveler,
			TierProgress:       0.35, // 35% to Platinum
			PointsToNextTier:   13000,
			FlightsToNextTier:  10,
			JoinDate:           time.Now().AddDate(-3, -6, 0),
			LastActivity:       time.Now().AddDate(0, 0, -8),
			TierAchievedDate:   time.Now().AddDate(0, -4, 0),
			Status:             "ACTIVE",
			PreferredRoutes:    []string{"NYC-LON", "NYC-FRA", "LON-FRA", "DXB-BOM"},
			AvgSpendPerTrip:    750.25,
			Metadata: map[string]interface{}{
				"profile_type":     "business_traveler",
				"travel_purpose":   "business",
				"preferred_class":  "W",
				"communication":    "sms",
			},
		},
		// Regular Customer
		{
			CustomerID:         customerID,
			ProgramID:          enhancedProgram.ID,
			MemberNumber:       fmt.Sprintf("ARP%s", customerID[:6]),
			CurrentTier:        Basic,
			NextTier:           Silver,
			Points:             1200,
			LifetimePoints:     3200,
			FlightCount:        2,
			LifetimeFlights:    4,
			Segment:            RegularCustomer,
			TierProgress:       0.24, // 24% to Silver
			PointsToNextTier:   3800,
			FlightsToNextTier:  2,
			JoinDate:           time.Now().AddDate(-1, -2, 0),
			LastActivity:       time.Now().AddDate(0, 0, -45),
			TierAchievedDate:   time.Now().AddDate(-1, -2, 0),
			Status:             "ACTIVE",
			PreferredRoutes:    []string{"NYC-LON", "LAX-SFO"},
			AvgSpendPerTrip:    320.75,
			Metadata: map[string]interface{}{
				"profile_type":     "leisure_traveler",
				"travel_purpose":   "leisure",
				"preferred_class":  "Y",
				"communication":    "email",
			},
		},
	}

	// Select profile based on customer ID hash
	profileIndex := int(customerID[len(customerID)-1]) % len(profiles)
	selectedProfile := profiles[profileIndex]

	// Customize based on customer ID
	selectedProfile.CustomerID = customerID

	return selectedProfile
}

// Helper functions for adjustment calculations
func getTierBenefits(tier LoyaltyTier) TierBenefits {
	for _, tierBenefit := range enhancedProgram.Tiers {
		if tierBenefit.Tier == tier {
			return tierBenefit
		}
	}
	return enhancedProgram.Tiers[0] // Default to Basic
}

func getRouteMultiplier(route string) float64 {
	// Route-specific loyalty multipliers
	multipliers := map[string]float64{
		"NYC-LON": 1.2,  // Premium international route
		"NYC-PAR": 1.15, // High-value route
		"NYC-FRA": 1.1,  // Business route
		"LON-PAR": 0.9,  // Short-haul discount
		"LON-FRA": 0.95, // Regional route
		"DXB-BOM": 1.05, // Emerging market route
		"DXB-DEL": 1.1,  // High-demand route
		"NYC-LAX": 1.0,  // Domestic route
		"NYC-SFO": 1.05, // Tech corridor route
	}

	if multiplier, exists := multipliers[route]; exists {
		return multiplier
	}
	return 1.0
}

func getSegmentMultiplier(segment CustomerSegment) float64 {
	multipliers := map[CustomerSegment]float64{
		NewCustomer:       1.1, // Welcome bonus
		RegularCustomer:   1.0, // Standard
		FrequentTraveler:  1.2, // Higher value
		BusinessTraveler:  1.3, // Premium segment
		VIPCustomer:       1.4, // Highest value
		InactiveCustomer:  0.8, // Win-back discount
	}

	if multiplier, exists := multipliers[segment]; exists {
		return multiplier
	}
	return 1.0
}

func calculateSeasonalBonus(customer CustomerLoyalty, route string) float64 {
	// Seasonal bonuses based on route and customer tier
	if customer.CurrentTier >= Gold {
		month := time.Now().Month()
		
		// Holiday season bonus (November-December)
		if month >= 11 || month <= 1 {
			return 0.02 // 2% additional discount
		}
		
		// Summer travel bonus (June-August)
		if month >= 6 && month <= 8 {
			return 0.015 // 1.5% additional discount
		}
	}
	
	return 0.0
}

func calculateHistoryBonus(customer CustomerLoyalty) float64 {
	// Loyalty history bonus based on lifetime value
	if customer.LifetimeFlights >= 50 {
		return 0.02 // 2% for very loyal customers
	}
	if customer.LifetimeFlights >= 25 {
		return 0.015 // 1.5% for loyal customers
	}
	if customer.LifetimeFlights >= 10 {
		return 0.01 // 1% for regular customers
	}
	
	return 0.0
}

func calculatePreferredRouteBonus(customer CustomerLoyalty, route string) float64 {
	// Bonus for preferred routes
	for _, preferredRoute := range customer.PreferredRoutes {
		if preferredRoute == route {
			return 0.01 // 1% bonus for preferred routes
		}
	}
	
	return 0.0
}

func calculatePersonalizationScore(customer CustomerLoyalty, route string, discountRate float64) float64 {
	// Calculate how personalized this offer is
	score := 0.5 // Base score
	
	// Tier-based personalization
	tierScore := float64(customer.CurrentTier) / 4.0 // 0.0 to 1.0
	score += tierScore * 0.3
	
	// Route preference
	for _, preferredRoute := range customer.PreferredRoutes {
		if preferredRoute == route {
			score += 0.2
			break
		}
	}
	
	// Discount value personalization
	if discountRate > 0.1 {
		score += 0.1
	}
	
	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}
	
	return score
}

// Cache management functions
func (ls *LoyaltyService) getCachedCustomer(customerID string) (CustomerLoyalty, bool) {
	ls.cache.mu.RLock()
	defer ls.cache.mu.RUnlock()

	// Check if key exists and hasn't expired
	if expiry, exists := ls.cache.expiry[customerID]; exists {
		if time.Now().Before(expiry) {
			if customer, exists := ls.cache.customers[customerID]; exists {
				ls.cache.hitCount++
				return customer, true
			}
		} else {
			// Clean up expired entry
			delete(ls.cache.customers, customerID)
			delete(ls.cache.expiry, customerID)
		}
	}

	ls.cache.missCount++
	return CustomerLoyalty{}, false
}

func (ls *LoyaltyService) setCachedCustomer(customerID string, customer CustomerLoyalty) {
	ls.cache.mu.Lock()
	defer ls.cache.mu.Unlock()

	ls.cache.customers[customerID] = customer
	ls.cache.expiry[customerID] = time.Now().Add(ls.config.CacheExpiry)
}

func (ls *LoyaltyService) getCachedAdjustment(key string) (LoyaltyAdjustment, bool) {
	ls.cache.mu.RLock()
	defer ls.cache.mu.RUnlock()

	// Check if key exists and hasn't expired
	adjustmentKey := fmt.Sprintf("adj_%s", key)
	if expiry, exists := ls.cache.expiry[adjustmentKey]; exists {
		if time.Now().Before(expiry) {
			if adjustment, exists := ls.cache.adjustments[adjustmentKey]; exists {
				ls.cache.hitCount++
				return adjustment, true
			}
		} else {
			// Clean up expired entry
			delete(ls.cache.adjustments, adjustmentKey)
			delete(ls.cache.expiry, adjustmentKey)
		}
	}

	ls.cache.missCount++
	return LoyaltyAdjustment{}, false
}

func (ls *LoyaltyService) setCachedAdjustment(key string, adjustment LoyaltyAdjustment) {
	ls.cache.mu.Lock()
	defer ls.cache.mu.Unlock()

	adjustmentKey := fmt.Sprintf("adj_%s", key)
	ls.cache.adjustments[adjustmentKey] = adjustment
	ls.cache.expiry[adjustmentKey] = time.Now().Add(ls.config.CacheExpiry)
}

// Rate limiter worker
func (ls *LoyaltyService) rateLimiterWorker() {
	ticker := time.NewTicker(time.Second / time.Duration(ls.config.RateLimitRPS))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			select {
			case <-ls.rateLimiter:
				// Rate limit slot released
			default:
				// No goroutine waiting
			}
		}
	}
}

// Legacy functions for backward compatibility
func calculateDefaultAdjustment(route string) float64 {
	// Default to Silver tier benefits for demonstration
	silverDiscount := 0.05

	// Route-specific adjustments
	routeMultipliers := map[string]float64{
		"NYC-LON": 1.2,
		"NYC-PAR": 1.1,
		"NYC-FRA": 1.0,
		"LON-PAR": 0.8,
		"LON-FRA": 0.9,
		"DXB-BOM": 0.7,
		"DXB-DEL": 0.8,
		"BOM-DEL": 0.6,
	}

	multiplier := 1.0
	if routeMultiplier, exists := routeMultipliers[route]; exists {
		multiplier = routeMultiplier
	}

	return silverDiscount * multiplier * 100.0 // Return as discount amount
}

// Additional service functions
func GetCustomerLoyalty(customerID string) (CustomerLoyalty, error) {
	if err := Initialize(); err != nil {
		return CustomerLoyalty{}, err
	}

	return loyaltyService.GetEnhancedCustomerLoyalty(customerID)
}

func CalculateCustomerAdjustment(customerID, route string, originalPrice float64) (LoyaltyAdjustment, error) {
	if err := Initialize(); err != nil {
		return LoyaltyAdjustment{}, err
	}

	return loyaltyService.GetCustomerAdjustment(customerID, route, originalPrice)
}

func GetTierBenefits(customerID string) (TierBenefits, error) {
	customer, err := GetCustomerLoyalty(customerID)
	if err != nil {
		return TierBenefits{}, err
	}

	return getTierBenefits(customer.CurrentTier), nil
}

func GetProgramInfo() LoyaltyProgram {
	return enhancedProgram
}

func GetCustomerSegment(customerID string) (string, error) {
	customer, err := GetCustomerLoyalty(customerID)
	if err != nil {
		return "Unknown", err
	}

	segmentNames := map[CustomerSegment]string{
		NewCustomer:       "New Customer",
		RegularCustomer:   "Regular Customer",
		FrequentTraveler:  "Frequent Traveler",
		BusinessTraveler:  "Business Traveler",
		VIPCustomer:       "VIP Customer",
		InactiveCustomer:  "Inactive Customer",
	}

	if name, exists := segmentNames[customer.Segment]; exists {
		return name, nil
	}

	return "Unknown", nil
}

func GetPersonalizedOffers(customerID string) ([]string, error) {
	customer, err := GetCustomerLoyalty(customerID)
	if err != nil {
		return []string{}, err
	}

	offers := []string{}

	// Tier-based offers
	tierBenefits := getTierBenefits(customer.CurrentTier)
	if customer.CurrentTier >= Silver {
		offers = append(offers, "Priority check-in available")
		offers = append(offers, "Extra baggage allowance discounts")
	}
	if customer.CurrentTier >= Gold {
		offers = append(offers, "Complimentary seat selection")
		offers = append(offers, "Lounge access included")
	}

	// Segment-based offers
	switch customer.Segment {
	case BusinessTraveler:
		offers = append(offers, "Business travel package deals")
		offers = append(offers, "Corporate rate discounts")
	case FrequentTraveler:
		offers = append(offers, "Multi-trip booking discounts")
		offers = append(offers, "Route-specific loyalty bonuses")
	case NewCustomer:
		offers = append(offers, "Welcome bonus: 500 points")
		offers = append(offers, "First flight insurance included")
	}

	// Tier progression offers
	if customer.TierProgress > 0.7 {
		offers = append(offers, fmt.Sprintf("Only %d points to %s tier!", customer.PointsToNextTier, tierBenefits.Name))
	}

	return offers, nil
}

func GetLoyaltyMetrics() map[string]interface{} {
	if loyaltyService == nil {
		return map[string]interface{}{"error": "service not initialized"}
	}

	loyaltyService.metrics.mu.RLock()
	defer loyaltyService.metrics.mu.RUnlock()

	hitRate := 0.0
	if loyaltyService.metrics.CacheHits+loyaltyService.metrics.CacheMisses > 0 {
		hitRate = float64(loyaltyService.metrics.CacheHits) / float64(loyaltyService.metrics.CacheHits+loyaltyService.metrics.CacheMisses)
	}

	return map[string]interface{}{
		"total_requests":      loyaltyService.metrics.TotalRequests,
		"cache_hits":          loyaltyService.metrics.CacheHits,
		"cache_misses":        loyaltyService.metrics.CacheMisses,
		"cache_hit_rate":      hitRate,
		"service_calls":       loyaltyService.metrics.ServiceCalls,
		"fallback_calls":      loyaltyService.metrics.FallbackCalls,
		"error_count":         loyaltyService.metrics.ErrorCount,
		"avg_response_time":   loyaltyService.metrics.AvgResponseTime.String(),
		"tier_distribution":   loyaltyService.metrics.TierDistribution,
		"segment_distribution": loyaltyService.metrics.SegmentDistribution,
		"last_updated":        loyaltyService.metrics.LastUpdated,
	}
}

func ClearCache() {
	if loyaltyService == nil {
		return
	}

	loyaltyService.cache.mu.Lock()
	defer loyaltyService.cache.mu.Unlock()

	loyaltyService.cache.customers = make(map[string]CustomerLoyalty)
	loyaltyService.cache.adjustments = make(map[string]LoyaltyAdjustment)
	loyaltyService.cache.expiry = make(map[string]time.Time)
	loyaltyService.cache.hitCount = 0
	loyaltyService.cache.missCount = 0
}

func GetCacheStatus() map[string]interface{} {
	if loyaltyService == nil {
		return map[string]interface{}{"error": "service not initialized"}
	}

	loyaltyService.cache.mu.RLock()
	defer loyaltyService.cache.mu.RUnlock()

	return map[string]interface{}{
		"cached_customers":   len(loyaltyService.cache.customers),
		"cached_adjustments": len(loyaltyService.cache.adjustments),
		"hit_count":          loyaltyService.cache.hitCount,
		"miss_count":         loyaltyService.cache.missCount,
		"cache_size":         len(loyaltyService.cache.customers) + len(loyaltyService.cache.adjustments),
	}
} 