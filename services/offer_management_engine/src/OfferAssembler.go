package offer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
	"iaros/offer_service/src/loyalty"
	"iaros/offer_service/src/pricing"
	"iaros/offer_service/src/forecasting"
	"iaros/offer_service/src/ancillary"
)

// OfferRequest represents a comprehensive offer request
type OfferRequest struct {
	Route            string                 `json:"route"`
	CustomerID       string                 `json:"customer_id"`
	CustomerSegment  string                 `json:"customer_segment"`
	Class            string                 `json:"class"`
	Passengers       int                    `json:"passengers"`
	DepartureDate    string                 `json:"departure_date"`
	ReturnDate       string                 `json:"return_date,omitempty"`
	BookingChannel   string                 `json:"booking_channel"`
	RequestID        string                 `json:"request_id"`
	Preferences      map[string]interface{} `json:"preferences"`
	ContextData      map[string]interface{} `json:"context_data"`
	Timestamp        time.Time              `json:"timestamp"`
}

// OfferResponse represents a comprehensive offer response
type OfferResponse struct {
	OfferID              string                 `json:"offer_id"`
	RequestID            string                 `json:"request_id"`
	Route                string                 `json:"route"`
	CustomerID           string                 `json:"customer_id"`
	TotalPrice           float64                `json:"total_price"`
	BasePrice            float64                `json:"base_price"`
	Discounts            float64                `json:"discounts"`
	Taxes                float64                `json:"taxes"`
	Fees                 float64                `json:"fees"`
	Currency             string                 `json:"currency"`
	Components           OfferComponents        `json:"components"`
	Personalization      PersonalizationData    `json:"personalization"`
	Recommendations      []Recommendation       `json:"recommendations"`
	ValidUntil           time.Time              `json:"valid_until"`
	Timestamp            time.Time              `json:"timestamp"`
	Source               string                 `json:"source"`
	ConfidenceScore      float64                `json:"confidence_score"`
	OptimizationScore    float64                `json:"optimization_score"`
	Metadata             map[string]interface{} `json:"metadata"`
}

// OfferComponents represents the breakdown of offer components
type OfferComponents struct {
	Flight       FlightComponent       `json:"flight"`
	Ancillary    AncillaryComponent    `json:"ancillary"`
	Loyalty      LoyaltyComponent      `json:"loyalty"`
	Forecast     ForecastComponent     `json:"forecast"`
}

// FlightComponent represents flight pricing details
type FlightComponent struct {
	BaseFare         float64                `json:"base_fare"`
	FinalFare        float64                `json:"final_fare"`
	Adjustments      []PricingAdjustment    `json:"adjustments"`
	Taxes            float64                `json:"taxes"`
	Fees             float64                `json:"fees"`
	Source           string                 `json:"source"`
	ConfidenceScore  float64                `json:"confidence_score"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// AncillaryComponent represents ancillary services
type AncillaryComponent struct {
	Bundle           AncillaryBundleInfo    `json:"bundle"`
	Items            []AncillaryItemInfo    `json:"items"`
	TotalValue       float64                `json:"total_value"`
	Savings          float64                `json:"savings"`
	AIConfidence     float64                `json:"ai_confidence"`
	RecommendedItems []string               `json:"recommended_items"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// LoyaltyComponent represents loyalty benefits and adjustments
type LoyaltyComponent struct {
	CustomerTier         string                 `json:"customer_tier"`
	CustomerSegment      string                 `json:"customer_segment"`
	DiscountAmount       float64                `json:"discount_amount"`
	DiscountRate         float64                `json:"discount_rate"`
	PointsEarned         int                    `json:"points_earned"`
	BonusMultiplier      float64                `json:"bonus_multiplier"`
	PersonalizationScore float64                `json:"personalization_score"`
	Benefits             []string               `json:"benefits"`
	Metadata             map[string]interface{} `json:"metadata"`
}

// ForecastComponent represents demand and pricing forecasts
type ForecastComponent struct {
	DemandForecast       float64                `json:"demand_forecast"`
	PriceForecast        float64                `json:"price_forecast"`
	RecommendedTiming    string                 `json:"recommended_timing"`
	PriceTrend           string                 `json:"price_trend"`
	ConfidenceLevel      float64                `json:"confidence_level"`
	ModelUsed            string                 `json:"model_used"`
	Metadata             map[string]interface{} `json:"metadata"`
}

// PersonalizationData represents personalization metrics
type PersonalizationData struct {
	PersonalizationScore float64                `json:"personalization_score"`
	SegmentMatch         float64                `json:"segment_match"`
	PreferenceMatch      float64                `json:"preference_match"`
	HistoryMatch         float64                `json:"history_match"`
	RecommendationEngine string                 `json:"recommendation_engine"`
	OptimizationFactors  []string               `json:"optimization_factors"`
	Metadata             map[string]interface{} `json:"metadata"`
}

// Recommendation represents offer recommendations
type Recommendation struct {
	Type             string                 `json:"type"`
	Title            string                 `json:"title"`
	Description      string                 `json:"description"`
	Value            float64                `json:"value"`
	Confidence       float64                `json:"confidence"`
	Priority         int                    `json:"priority"`
	ApplicableUntil  time.Time              `json:"applicable_until"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// Supporting types for component details
type PricingAdjustment struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Percentage  float64 `json:"percentage"`
}

type AncillaryBundleInfo struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Description      string  `json:"description"`
	TotalPrice       float64 `json:"total_price"`
	DiscountPrice    float64 `json:"discount_price"`
	PopularityScore  float64 `json:"popularity_score"`
}

type AncillaryItemInfo struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Recommended bool    `json:"recommended"`
}

// OfferAssembler orchestrates all offer components with comprehensive business logic
func OfferAssembler(route string) (float64, error) {
	// Create default offer request
	request := OfferRequest{
		Route:           route,
		CustomerID:      "demo-customer",
		CustomerSegment: "FrequentTraveler",
		Class:           "Y",
		Passengers:      1,
		DepartureDate:   time.Now().AddDate(0, 0, 30).Format("2006-01-02"),
		BookingChannel:  "web",
		RequestID:       fmt.Sprintf("offer-%d", time.Now().UnixNano()),
		Timestamp:       time.Now(),
		ContextData: map[string]interface{}{
			"market_conditions": "normal",
			"booking_advance":   30,
		},
	}

	// Get comprehensive offer
	response, err := GetComprehensiveOffer(request)
	if err != nil {
		log.Printf("Failed to get comprehensive offer: %v", err)
		return 0, err
	}

	return response.TotalPrice, nil
}

// GetComprehensiveOffer creates a comprehensive offer with all components
func GetComprehensiveOffer(request OfferRequest) (OfferResponse, error) {
	startTime := time.Now()

	// Initialize all services
	if err := initializeAllServices(); err != nil {
		return OfferResponse{}, fmt.Errorf("failed to initialize services: %v", err)
	}

	// Create context for concurrent operations
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use channels for concurrent component retrieval
	flightChan := make(chan FlightComponent, 1)
	ancillaryChan := make(chan AncillaryComponent, 1)
	loyaltyChan := make(chan LoyaltyComponent, 1)
	forecastChan := make(chan ForecastComponent, 1)

	var wg sync.WaitGroup
	var componentsErr error

	// Retrieve flight component
	wg.Add(1)
	go func() {
		defer wg.Done()
		component, err := getFlightComponent(ctx, request)
		if err != nil {
			componentsErr = fmt.Errorf("flight component error: %v", err)
			return
		}
		flightChan <- component
	}()

	// Retrieve ancillary component
	wg.Add(1)
	go func() {
		defer wg.Done()
		component, err := getAncillaryComponent(ctx, request)
		if err != nil {
			log.Printf("Ancillary component warning: %v", err)
			// Use default for ancillary
			component = getDefaultAncillaryComponent(request)
		}
		ancillaryChan <- component
	}()

	// Retrieve loyalty component
	wg.Add(1)
	go func() {
		defer wg.Done()
		component, err := getLoyaltyComponent(ctx, request)
		if err != nil {
			log.Printf("Loyalty component warning: %v", err)
			// Use default for loyalty
			component = getDefaultLoyaltyComponent(request)
		}
		loyaltyChan <- component
	}()

	// Retrieve forecast component
	wg.Add(1)
	go func() {
		defer wg.Done()
		component, err := getForecastComponent(ctx, request)
		if err != nil {
			log.Printf("Forecast component warning: %v", err)
			// Use default for forecast
			component = getDefaultForecastComponent(request)
		}
		forecastChan <- component
	}()

	// Wait for all components
	wg.Wait()

	if componentsErr != nil {
		return OfferResponse{}, componentsErr
	}

	// Collect components
	flightComponent := <-flightChan
	ancillaryComponent := <-ancillaryChan
	loyaltyComponent := <-loyaltyChan
	forecastComponent := <-forecastChan

	// Apply cross-component optimizations
	optimizedComponents := optimizeComponents(
		flightComponent,
		ancillaryComponent,
		loyaltyComponent,
		forecastComponent,
		request,
	)

	// Calculate total pricing
	totalPrice := calculateTotalPrice(optimizedComponents)
	basePrice := flightComponent.BaseFare
	discounts := loyaltyComponent.DiscountAmount + optimizedComponents.Ancillary.Savings
	taxes := flightComponent.Taxes
	fees := flightComponent.Fees

	// Generate personalization data
	personalization := generatePersonalizationData(optimizedComponents, request)

	// Generate recommendations
	recommendations := generateRecommendations(optimizedComponents, request)

	// Calculate confidence and optimization scores
	confidenceScore := calculateConfidenceScore(optimizedComponents)
	optimizationScore := calculateOptimizationScore(optimizedComponents, personalization)

	// Create comprehensive offer response
	response := OfferResponse{
		OfferID:           fmt.Sprintf("offer-%s-%d", request.Route, time.Now().Unix()),
		RequestID:         request.RequestID,
		Route:             request.Route,
		CustomerID:        request.CustomerID,
		TotalPrice:        totalPrice,
		BasePrice:         basePrice,
		Discounts:         discounts,
		Taxes:             taxes,
		Fees:              fees,
		Currency:          "USD",
		Components:        optimizedComponents,
		Personalization:   personalization,
		Recommendations:   recommendations,
		ValidUntil:        time.Now().Add(24 * time.Hour),
		Timestamp:         time.Now(),
		Source:            "comprehensive_assembler",
		ConfidenceScore:   confidenceScore,
		OptimizationScore: optimizationScore,
		Metadata: map[string]interface{}{
			"processing_time_ms": time.Since(startTime).Milliseconds(),
			"components_count":   4,
			"optimization_level": "full",
			"ai_enabled":         true,
			"cache_enabled":      true,
		},
	}

	return response, nil
}

// Component retrieval functions
func getFlightComponent(ctx context.Context, request OfferRequest) (FlightComponent, error) {
	// Get detailed pricing information
	priceResponse, err := pricing.GetDetailedPrice(request.Route, request.Passengers, request.Class)
	if err != nil {
		return FlightComponent{}, err
	}

	// Convert pricing adjustments
	adjustments := make([]PricingAdjustment, len(priceResponse.Adjustments))
	for i, adj := range priceResponse.Adjustments {
		adjustments[i] = PricingAdjustment{
			Type:        adj.Type,
			Description: adj.Description,
			Amount:      adj.Amount,
			Percentage:  adj.Percentage,
		}
	}

	component := FlightComponent{
		BaseFare:        priceResponse.BaseFare,
		FinalFare:       priceResponse.FinalFare,
		Adjustments:     adjustments,
		Taxes:           priceResponse.Taxes,
		Fees:            priceResponse.Fees,
		Source:          priceResponse.Source,
		ConfidenceScore: priceResponse.ConfidenceScore,
		Metadata: map[string]interface{}{
			"breakdown":     priceResponse.Breakdown,
			"valid_until":   priceResponse.ValidUntil,
			"request_id":    priceResponse.RequestID,
		},
	}

	return component, nil
}

func getAncillaryComponent(ctx context.Context, request OfferRequest) (AncillaryComponent, error) {
	// Initialize ancillary service
	if err := ancillary.Initialize(); err != nil {
		return AncillaryComponent{}, err
	}

	// Get AI-optimized bundle (using internal service method)
	bundleValue, err := ancillary.GetAncillaryBundle(request.Route)
	if err != nil {
		return AncillaryComponent{}, err
	}

	// Get available items for reference
	availableItems, err := ancillary.GetAvailableItems(request.Route)
	if err != nil {
		return AncillaryComponent{}, err
	}

	// Convert to component format
	items := make([]AncillaryItemInfo, len(availableItems))
	totalValue := 0.0
	for i, item := range availableItems {
		items[i] = AncillaryItemInfo{
			ID:          item.ID,
			Name:        item.Name,
			Price:       item.Price,
			Recommended: item.PopularityScore > 0.7,
		}
		if items[i].Recommended {
			totalValue += item.Price
		}
	}

	savings := totalValue - bundleValue

	component := AncillaryComponent{
		Bundle: AncillaryBundleInfo{
			ID:              "ai-optimized-bundle",
			Name:            "AI-Optimized Travel Bundle",
			Description:     "Personalized bundle of ancillary services",
			TotalPrice:      totalValue,
			DiscountPrice:   bundleValue,
			PopularityScore: 0.85,
		},
		Items:            items,
		TotalValue:       totalValue,
		Savings:          savings,
		AIConfidence:     0.88,
		RecommendedItems: []string{"wifi-premium", "bag-20kg", "seat-extra-legroom"},
		Metadata: map[string]interface{}{
			"bundle_type":       "ai_optimized",
			"personalization":   true,
			"items_count":       len(items),
		},
	}

	return component, nil
}

func getLoyaltyComponent(ctx context.Context, request OfferRequest) (LoyaltyComponent, error) {
	// Get customer loyalty adjustment
	adjustment, err := loyalty.CalculateCustomerAdjustment(request.CustomerID, request.Route, 500.0)
	if err != nil {
		return LoyaltyComponent{}, err
	}

	// Get customer loyalty information
	customer, err := loyalty.GetCustomerLoyalty(request.CustomerID)
	if err != nil {
		return LoyaltyComponent{}, err
	}

	// Get tier benefits
	tierBenefits, err := loyalty.GetTierBenefits(request.CustomerID)
	if err != nil {
		return LoyaltyComponent{}, err
	}

	// Get customer segment
	segment, err := loyalty.GetCustomerSegment(request.CustomerID)
	if err != nil {
		segment = "Regular Customer"
	}

	// Calculate points earned (simplified)
	pointsEarned := int(500.0 * tierBenefits.BonusMultiplier)

	component := LoyaltyComponent{
		CustomerTier:         tierBenefits.Name,
		CustomerSegment:      segment,
		DiscountAmount:       adjustment.DiscountAmount,
		DiscountRate:         adjustment.DiscountRate,
		PointsEarned:         pointsEarned,
		BonusMultiplier:      tierBenefits.BonusMultiplier,
		PersonalizationScore: adjustment.PersonalizationScore,
		Benefits:             tierBenefits.Benefits,
		Metadata: map[string]interface{}{
			"tier_level":        customer.CurrentTier,
			"lifetime_points":   customer.LifetimePoints,
			"lifetime_flights":  customer.LifetimeFlights,
			"tier_progress":     customer.TierProgress,
		},
	}

	return component, nil
}

func getForecastComponent(ctx context.Context, request OfferRequest) (ForecastComponent, error) {
	// Get demand forecast
	demandForecast, err := forecasting.GetForecast(request.Route)
	if err != nil {
		return ForecastComponent{}, err
	}

	// Get advanced forecasts
	advancedForecasts, err := forecasting.GetAdvancedForecast(request.Route, 7)
	if err != nil {
		log.Printf("Warning: Could not get advanced forecasts: %v", err)
	}

	// Analyze price trend
	priceTrend := "stable"
	if demandForecast > 0.8 {
		priceTrend = "increasing"
	} else if demandForecast < 0.4 {
		priceTrend = "decreasing"
	}

	// Recommend timing
	recommendedTiming := "book_now"
	if priceTrend == "increasing" {
		recommendedTiming = "book_immediately"
	} else if priceTrend == "decreasing" {
		recommendedTiming = "wait_for_deals"
	}

	component := ForecastComponent{
		DemandForecast:    demandForecast,
		PriceForecast:     demandForecast * 1.2, // Simplified correlation
		RecommendedTiming: recommendedTiming,
		PriceTrend:        priceTrend,
		ConfidenceLevel:   0.87,
		ModelUsed:         "ARIMA",
		Metadata: map[string]interface{}{
			"advanced_forecasts": advancedForecasts,
			"forecast_horizon":   7,
			"model_accuracy":     0.88,
		},
	}

	return component, nil
}

// Default component functions for fallback scenarios
func getDefaultAncillaryComponent(request OfferRequest) AncillaryComponent {
	return AncillaryComponent{
		Bundle: AncillaryBundleInfo{
			ID:              "default-bundle",
			Name:            "Essential Travel Bundle",
			Description:     "Basic travel essentials",
			TotalPrice:      50.0,
			DiscountPrice:   45.0,
			PopularityScore: 0.7,
		},
		Items: []AncillaryItemInfo{
			{ID: "bag-20kg", Name: "20kg Baggage", Price: 35.0, Recommended: true},
			{ID: "meal-standard", Name: "Standard Meal", Price: 15.0, Recommended: false},
		},
		TotalValue:       50.0,
		Savings:          5.0,
		AIConfidence:     0.6,
		RecommendedItems: []string{"bag-20kg"},
		Metadata:         map[string]interface{}{"fallback": true},
	}
}

func getDefaultLoyaltyComponent(request OfferRequest) LoyaltyComponent {
	return LoyaltyComponent{
		CustomerTier:         "Basic Member",
		CustomerSegment:      "Regular Customer",
		DiscountAmount:       0.0,
		DiscountRate:         0.0,
		PointsEarned:         500,
		BonusMultiplier:      1.0,
		PersonalizationScore: 0.5,
		Benefits:             []string{"Standard check-in", "Newsletter"},
		Metadata:             map[string]interface{}{"fallback": true},
	}
}

func getDefaultForecastComponent(request OfferRequest) ForecastComponent {
	return ForecastComponent{
		DemandForecast:    0.5,
		PriceForecast:     0.5,
		RecommendedTiming: "book_now",
		PriceTrend:        "stable",
		ConfidenceLevel:   0.6,
		ModelUsed:         "fallback",
		Metadata:          map[string]interface{}{"fallback": true},
	}
}

// Component optimization and calculation functions
func optimizeComponents(flight FlightComponent, ancillary AncillaryComponent, loyalty LoyaltyComponent, forecast ForecastComponent, request OfferRequest) OfferComponents {
	// Apply cross-component optimizations
	
	// Optimize ancillary based on loyalty tier
	if loyalty.CustomerTier != "Basic Member" {
		ancillary.Bundle.DiscountPrice *= 0.95 // Additional 5% discount for loyalty members
		ancillary.Savings = ancillary.TotalValue - ancillary.Bundle.DiscountPrice
	}

	// Optimize pricing based on forecast
	if forecast.DemandForecast > 0.8 {
		// High demand - reduce loyalty discount slightly
		loyalty.DiscountAmount *= 0.9
	} else if forecast.DemandForecast < 0.4 {
		// Low demand - increase ancillary discounts
		ancillary.Bundle.DiscountPrice *= 0.9
		ancillary.Savings = ancillary.TotalValue - ancillary.Bundle.DiscountPrice
	}

	return OfferComponents{
		Flight:    flight,
		Ancillary: ancillary,
		Loyalty:   loyalty,
		Forecast:  forecast,
	}
}

func calculateTotalPrice(components OfferComponents) float64 {
	return components.Flight.FinalFare + 
		   components.Flight.Taxes + 
		   components.Flight.Fees + 
		   components.Ancillary.Bundle.DiscountPrice - 
		   components.Loyalty.DiscountAmount
}

func generatePersonalizationData(components OfferComponents, request OfferRequest) PersonalizationData {
	// Calculate personalization metrics
	segmentMatch := 0.8 // Based on customer segment matching
	preferenceMatch := 0.7 // Based on route and class preferences
	historyMatch := components.Loyalty.PersonalizationScore

	personalizationScore := (segmentMatch + preferenceMatch + historyMatch) / 3.0

	return PersonalizationData{
		PersonalizationScore: personalizationScore,
		SegmentMatch:         segmentMatch,
		PreferenceMatch:      preferenceMatch,
		HistoryMatch:         historyMatch,
		RecommendationEngine: "ai_v2.1",
		OptimizationFactors:  []string{"loyalty_tier", "route_preference", "forecast_data", "ancillary_affinity"},
		Metadata: map[string]interface{}{
			"customer_segment": request.CustomerSegment,
			"route_familiarity": true,
			"cross_sell_opportunity": components.Ancillary.AIConfidence > 0.8,
		},
	}
}

func generateRecommendations(components OfferComponents, request OfferRequest) []Recommendation {
	recommendations := []Recommendation{}

	// Pricing recommendation
	if components.Forecast.PriceTrend == "increasing" {
		recommendations = append(recommendations, Recommendation{
			Type:            "pricing",
			Title:           "Book Now - Prices Rising",
			Description:     "Our forecast shows prices increasing. Book now to secure this rate.",
			Value:           0.0,
			Confidence:      components.Forecast.ConfidenceLevel,
			Priority:        1,
			ApplicableUntil: time.Now().Add(24 * time.Hour),
			Metadata:        map[string]interface{}{"urgency": "high"},
		})
	}

	// Ancillary recommendation
	if components.Ancillary.AIConfidence > 0.8 {
		recommendations = append(recommendations, Recommendation{
			Type:            "ancillary",
			Title:           "Personalized Travel Bundle",
			Description:     fmt.Sprintf("Save $%.2f with our AI-recommended travel bundle", components.Ancillary.Savings),
			Value:           components.Ancillary.Savings,
			Confidence:      components.Ancillary.AIConfidence,
			Priority:        2,
			ApplicableUntil: time.Now().Add(24 * time.Hour),
			Metadata:        map[string]interface{}{"bundle_id": components.Ancillary.Bundle.ID},
		})
	}

	// Loyalty recommendation
	if components.Loyalty.CustomerTier == "Basic Member" {
		recommendations = append(recommendations, Recommendation{
			Type:            "loyalty",
			Title:           "Join Silver Elite",
			Description:     "You're close to Silver Elite status! Book 2 more flights to unlock premium benefits.",
			Value:           0.0,
			Confidence:      0.9,
			Priority:        3,
			ApplicableUntil: time.Now().Add(30 * 24 * time.Hour),
			Metadata:        map[string]interface{}{"tier_upgrade": "silver"},
		})
	}

	return recommendations
}

func calculateConfidenceScore(components OfferComponents) float64 {
	scores := []float64{
		components.Flight.ConfidenceScore,
		components.Ancillary.AIConfidence,
		components.Loyalty.PersonalizationScore,
		components.Forecast.ConfidenceLevel,
	}

	sum := 0.0
	for _, score := range scores {
		sum += score
	}

	return sum / float64(len(scores))
}

func calculateOptimizationScore(components OfferComponents, personalization PersonalizationData) float64 {
	// Optimization score based on various factors
	priceOptimization := components.Flight.ConfidenceScore
	ancillaryOptimization := components.Ancillary.AIConfidence
	loyaltyOptimization := components.Loyalty.PersonalizationScore
	personalizationOptimization := personalization.PersonalizationScore

	// Weighted average
	weights := []float64{0.3, 0.25, 0.25, 0.2}
	scores := []float64{priceOptimization, ancillaryOptimization, loyaltyOptimization, personalizationOptimization}

	weightedSum := 0.0
	for i, score := range scores {
		weightedSum += score * weights[i]
	}

	return weightedSum
}

func initializeAllServices() error {
	// Initialize all services concurrently
	var wg sync.WaitGroup
	errors := make(chan error, 4)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := pricing.Initialize(); err != nil {
			errors <- fmt.Errorf("pricing service: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := forecasting.Initialize(); err != nil {
			errors <- fmt.Errorf("forecasting service: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := ancillary.Initialize(); err != nil {
			errors <- fmt.Errorf("ancillary service: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := loyalty.Initialize(); err != nil {
			errors <- fmt.Errorf("loyalty service: %v", err)
		}
	}()

	wg.Wait()
	close(errors)

	// Check for any initialization errors
	for err := range errors {
		if err != nil {
			return err
		}
	}

	return nil
}
