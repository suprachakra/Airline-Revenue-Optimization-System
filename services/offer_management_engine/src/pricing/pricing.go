package pricing

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
	"log"
	"math"
	"context"
)

// PricingServiceConfig holds configuration for the pricing service
type PricingServiceConfig struct {
	BaseURL        string
	Timeout        time.Duration
	RetryAttempts  int
	CacheExpiry    time.Duration
	RateLimitRPS   int
	EnableFallback bool
}

// PriceRequest represents a comprehensive pricing request
type PriceRequest struct {
	Route           string            `json:"route"`
	Origin          string            `json:"origin"`
	Destination     string            `json:"destination"`
	DepartureDate   string            `json:"departure_date"`
	ReturnDate      string            `json:"return_date,omitempty"`
	Passengers      int               `json:"passengers"`
	Class           string            `json:"class"`
	CustomerID      string            `json:"customer_id,omitempty"`
	CorporateID     string            `json:"corporate_id,omitempty"`
	LoyaltyTier     string            `json:"loyalty_tier,omitempty"`
	BookingChannel  string            `json:"booking_channel"`
	MarketData      map[string]interface{} `json:"market_data,omitempty"`
	Preferences     map[string]interface{} `json:"preferences,omitempty"`
	RequestID       string            `json:"request_id"`
	Timestamp       time.Time         `json:"timestamp"`
}

// PriceResponse represents a comprehensive pricing response
type PriceResponse struct {
	Route           string            `json:"route"`
	RequestID       string            `json:"request_id"`
	BaseFare        float64           `json:"base_fare"`
	FinalFare       float64           `json:"final_fare"`
	Currency        string            `json:"currency"`
	Taxes           float64           `json:"taxes"`
	Fees            float64           `json:"fees"`
	Total           float64           `json:"total"`
	Adjustments     []PriceAdjustment `json:"adjustments"`
	Breakdown       PriceBreakdown    `json:"breakdown"`
	Metadata        map[string]interface{} `json:"metadata"`
	ValidUntil      time.Time         `json:"valid_until"`
	Timestamp       time.Time         `json:"timestamp"`
	Source          string            `json:"source"` // "service", "cache", "fallback"
	ConfidenceScore float64           `json:"confidence_score"`
}

// PriceAdjustment represents individual pricing adjustments
type PriceAdjustment struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Percentage  float64 `json:"percentage"`
	Applied     bool    `json:"applied"`
}

// PriceBreakdown provides detailed price breakdown
type PriceBreakdown struct {
	BaseFare       float64 `json:"base_fare"`
	DemandSurge    float64 `json:"demand_surge"`
	SeasonalAdj    float64 `json:"seasonal_adjustment"`
	RouteMultiplier float64 `json:"route_multiplier"`
	ClassMultiplier float64 `json:"class_multiplier"`
	LoyaltyDiscount float64 `json:"loyalty_discount"`
	CorporateDiscount float64 `json:"corporate_discount"`
	ChannelAdj     float64 `json:"channel_adjustment"`
	CompetitorAdj  float64 `json:"competitor_adjustment"`
	FuelSurcharge  float64 `json:"fuel_surcharge"`
	SubTotal       float64 `json:"subtotal"`
	Taxes          float64 `json:"taxes"`
	Fees           float64 `json:"fees"`
	Total          float64 `json:"total"`
}

// PriceCache stores cached pricing data with thread-safe operations
type PriceCache struct {
	mu       sync.RWMutex
	prices   map[string]PriceResponse
	expiry   map[string]time.Time
	hitCount int64
	missCount int64
}

// PricingService handles all pricing operations
type PricingService struct {
	config      PricingServiceConfig
	cache       *PriceCache
	httpClient  *http.Client
	rateLimiter chan struct{}
	metrics     *PricingMetrics
}

// PricingMetrics tracks pricing service metrics
type PricingMetrics struct {
	mu            sync.RWMutex
	TotalRequests int64
	CacheHits     int64
	CacheMisses   int64
	ServiceCalls  int64
	FallbackCalls int64
	ErrorCount    int64
	AvgResponseTime time.Duration
	LastUpdated   time.Time
}

// Global pricing service instance
var pricingService *PricingService
var once sync.Once

// Initialize initializes the pricing service
func Initialize() error {
	once.Do(func() {
		config := PricingServiceConfig{
			BaseURL:        "http://pricing-service:8080",
			Timeout:        5 * time.Second,
			RetryAttempts:  3,
			CacheExpiry:    15 * time.Minute,
			RateLimitRPS:   100,
			EnableFallback: true,
		}

		pricingService = &PricingService{
			config: config,
			cache: &PriceCache{
				prices: make(map[string]PriceResponse),
				expiry: make(map[string]time.Time),
			},
			httpClient: &http.Client{
				Timeout: config.Timeout,
			},
			rateLimiter: make(chan struct{}, config.RateLimitRPS),
			metrics: &PricingMetrics{
				LastUpdated: time.Now(),
			},
		}

		// Initialize rate limiter
		go pricingService.rateLimiterWorker()
	})

	return nil
}

// GetPrice retrieves price for a given route with comprehensive business logic
func GetPrice(route string) (float64, error) {
	if route == "" {
		return 0, errors.New("route cannot be empty")
	}

	// Ensure service is initialized
	if err := Initialize(); err != nil {
		return 0, fmt.Errorf("failed to initialize pricing service: %v", err)
	}

	// Create comprehensive pricing request
	request := PriceRequest{
		Route:          route,
		Origin:         strings.Split(route, "-")[0],
		Destination:    strings.Split(route, "-")[1],
		DepartureDate:  time.Now().AddDate(0, 0, 30).Format("2006-01-02"),
		Passengers:     1,
		Class:          "Y",
		BookingChannel: "web",
		RequestID:      fmt.Sprintf("req-%d", time.Now().UnixNano()),
		Timestamp:      time.Now(),
		MarketData: map[string]interface{}{
			"demand_level": 0.7,
			"competitor_avg": 0.0,
			"season_factor": 1.0,
		},
	}

	// Get comprehensive price response
	response, err := pricingService.GetComprehensivePrice(request)
	if err != nil {
		log.Printf("Failed to get comprehensive price: %v", err)
		// Fallback to basic pricing
		return getBasicPrice(route)
	}

	return response.Total, nil
}

// GetComprehensivePrice handles comprehensive pricing with all business logic
func (ps *PricingService) GetComprehensivePrice(request PriceRequest) (PriceResponse, error) {
	startTime := time.Now()
	
	// Update metrics
	ps.metrics.mu.Lock()
	ps.metrics.TotalRequests++
	ps.metrics.mu.Unlock()

	// Check cache first
	cacheKey := ps.generateCacheKey(request)
	if cachedPrice, exists := ps.getCachedPrice(cacheKey); exists {
		ps.metrics.mu.Lock()
		ps.metrics.CacheHits++
		ps.metrics.mu.Unlock()
		
		cachedPrice.Source = "cache"
		return cachedPrice, nil
	}

	// Cache miss
	ps.metrics.mu.Lock()
	ps.metrics.CacheMisses++
	ps.metrics.mu.Unlock()

	// Rate limiting
	select {
	case ps.rateLimiter <- struct{}{}:
		defer func() { <-ps.rateLimiter }()
	case <-time.After(1 * time.Second):
		return PriceResponse{}, errors.New("rate limit exceeded")
	}

	// Try to get price from pricing service
	response, err := ps.callPricingService(request)
	if err != nil {
		ps.metrics.mu.Lock()
		ps.metrics.ErrorCount++
		ps.metrics.mu.Unlock()

		if ps.config.EnableFallback {
			// Fallback to local pricing logic
			response, err = ps.generateFallbackPrice(request)
			if err != nil {
				return PriceResponse{}, err
			}
			response.Source = "fallback"
			
			ps.metrics.mu.Lock()
			ps.metrics.FallbackCalls++
			ps.metrics.mu.Unlock()
		} else {
			return PriceResponse{}, err
		}
	} else {
		response.Source = "service"
		ps.metrics.mu.Lock()
		ps.metrics.ServiceCalls++
		ps.metrics.mu.Unlock()
	}

	// Cache the response
	ps.setCachedPrice(cacheKey, response)

	// Update response time metrics
	responseTime := time.Since(startTime)
	ps.metrics.mu.Lock()
	ps.metrics.AvgResponseTime = (ps.metrics.AvgResponseTime + responseTime) / 2
	ps.metrics.LastUpdated = time.Now()
	ps.metrics.mu.Unlock()

	return response, nil
}

// callPricingService makes HTTP call to the pricing service
func (ps *PricingService) callPricingService(request PriceRequest) (PriceResponse, error) {
	// Marshal request
	requestBody, err := json.Marshal(request)
	if err != nil {
		return PriceResponse{}, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/pricing/comprehensive", ps.config.BaseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return PriceResponse{}, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Request-ID", request.RequestID)

	// Make request with retry logic
	var response PriceResponse
	for attempt := 0; attempt < ps.config.RetryAttempts; attempt++ {
		resp, err := ps.httpClient.Do(req)
		if err != nil {
			log.Printf("Attempt %d failed: %v", attempt+1, err)
			if attempt == ps.config.RetryAttempts-1 {
				return PriceResponse{}, fmt.Errorf("all retry attempts failed: %v", err)
			}
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("HTTP error: %d", resp.StatusCode)
			if attempt == ps.config.RetryAttempts-1 {
				return PriceResponse{}, fmt.Errorf("HTTP error: %d", resp.StatusCode)
			}
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}

		// Parse response
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return PriceResponse{}, fmt.Errorf("failed to read response: %v", err)
		}

		err = json.Unmarshal(body, &response)
		if err != nil {
			return PriceResponse{}, fmt.Errorf("failed to unmarshal response: %v", err)
		}

		break
	}

	return response, nil
}

// generateFallbackPrice generates price using local business logic
func (ps *PricingService) generateFallbackPrice(request PriceRequest) (PriceResponse, error) {
	// Get base fare for route
	baseFare := getBaseFareForRoute(request.Route)
	if baseFare == 0 {
		return PriceResponse{}, errors.New("route not found")
	}

	// Initialize breakdown
	breakdown := PriceBreakdown{
		BaseFare: baseFare,
	}

	finalFare := baseFare
	adjustments := []PriceAdjustment{}

	// Apply demand adjustments
	if demandLevel, ok := request.MarketData["demand_level"].(float64); ok {
		if demandLevel > 0.8 {
			surgeAmount := baseFare * 0.3
			finalFare += surgeAmount
			breakdown.DemandSurge = surgeAmount
			adjustments = append(adjustments, PriceAdjustment{
				Type:        "demand_surge",
				Description: "High demand surge pricing",
				Amount:      surgeAmount,
				Percentage:  30.0,
				Applied:     true,
			})
		} else if demandLevel < 0.3 {
			discountAmount := baseFare * 0.1
			finalFare -= discountAmount
			breakdown.DemandSurge = -discountAmount
			adjustments = append(adjustments, PriceAdjustment{
				Type:        "demand_discount",
				Description: "Low demand discount",
				Amount:      -discountAmount,
				Percentage:  -10.0,
				Applied:     true,
			})
		}
	}

	// Apply seasonal adjustments
	if seasonFactor, ok := request.MarketData["season_factor"].(float64); ok && seasonFactor != 1.0 {
		seasonalAdj := baseFare * (seasonFactor - 1.0)
		finalFare += seasonalAdj
		breakdown.SeasonalAdj = seasonalAdj
		adjustments = append(adjustments, PriceAdjustment{
			Type:        "seasonal",
			Description: "Seasonal pricing adjustment",
			Amount:      seasonalAdj,
			Percentage:  (seasonFactor - 1.0) * 100,
			Applied:     true,
		})
	}

	// Apply route multiplier
	routeMultiplier := getRouteMultiplier(request.Route)
	if routeMultiplier != 1.0 {
		routeAdj := baseFare * (routeMultiplier - 1.0)
		finalFare += routeAdj
		breakdown.RouteMultiplier = routeAdj
		adjustments = append(adjustments, PriceAdjustment{
			Type:        "route_multiplier",
			Description: "Route-specific pricing",
			Amount:      routeAdj,
			Percentage:  (routeMultiplier - 1.0) * 100,
			Applied:     true,
		})
	}

	// Apply class multiplier
	classMultiplier := getClassMultiplier(request.Class)
	if classMultiplier != 1.0 {
		classAdj := baseFare * (classMultiplier - 1.0)
		finalFare += classAdj
		breakdown.ClassMultiplier = classAdj
		adjustments = append(adjustments, PriceAdjustment{
			Type:        "class_multiplier",
			Description: fmt.Sprintf("Class %s pricing", request.Class),
			Amount:      classAdj,
			Percentage:  (classMultiplier - 1.0) * 100,
			Applied:     true,
		})
	}

	// Apply loyalty discount
	if request.LoyaltyTier != "" {
		loyaltyDiscount := getLoyaltyDiscount(request.LoyaltyTier)
		if loyaltyDiscount > 0 {
			discountAmount := finalFare * loyaltyDiscount
			finalFare -= discountAmount
			breakdown.LoyaltyDiscount = -discountAmount
			adjustments = append(adjustments, PriceAdjustment{
				Type:        "loyalty_discount",
				Description: fmt.Sprintf("%s member discount", request.LoyaltyTier),
				Amount:      -discountAmount,
				Percentage:  -loyaltyDiscount * 100,
				Applied:     true,
			})
		}
	}

	// Apply corporate discount
	if request.CorporateID != "" {
		corporateDiscount := 0.1 // 10% corporate discount
		discountAmount := finalFare * corporateDiscount
		finalFare -= discountAmount
		breakdown.CorporateDiscount = -discountAmount
		adjustments = append(adjustments, PriceAdjustment{
			Type:        "corporate_discount",
			Description: "Corporate discount",
			Amount:      -discountAmount,
			Percentage:  -corporateDiscount * 100,
			Applied:     true,
		})
	}

	// Apply channel adjustment
	channelAdj := getChannelAdjustment(request.BookingChannel)
	if channelAdj != 0 {
		channelAmount := finalFare * channelAdj
		finalFare += channelAmount
		breakdown.ChannelAdj = channelAmount
		adjustments = append(adjustments, PriceAdjustment{
			Type:        "channel_adjustment",
			Description: fmt.Sprintf("%s channel adjustment", request.BookingChannel),
			Amount:      channelAmount,
			Percentage:  channelAdj * 100,
			Applied:     true,
		})
	}

	// Calculate taxes and fees
	breakdown.SubTotal = finalFare
	breakdown.Taxes = finalFare * 0.15 // 15% taxes
	breakdown.Fees = 25.0              // Fixed fees
	breakdown.Total = finalFare + breakdown.Taxes + breakdown.Fees

	// Create response
	response := PriceResponse{
		Route:       request.Route,
		RequestID:   request.RequestID,
		BaseFare:    baseFare,
		FinalFare:   finalFare,
		Currency:    "USD",
		Taxes:       breakdown.Taxes,
		Fees:        breakdown.Fees,
		Total:       breakdown.Total,
		Adjustments: adjustments,
		Breakdown:   breakdown,
		ValidUntil:  time.Now().Add(15 * time.Minute),
		Timestamp:   time.Now(),
		ConfidenceScore: 0.85, // Lower confidence for fallback
		Metadata: map[string]interface{}{
			"fallback_reason": "pricing_service_unavailable",
			"adjustments_applied": len(adjustments),
		},
	}

	return response, nil
}

// Helper functions
func getBaseFareForRoute(route string) float64 {
	routeFares := map[string]float64{
		"NYC-LON": 650.0, "NYC-PAR": 700.0, "NYC-FRA": 680.0, "NYC-AMS": 620.0,
		"LON-PAR": 150.0, "LON-FRA": 180.0, "LON-AMS": 140.0, "PAR-FRA": 120.0,
		"PAR-AMS": 100.0, "FRA-AMS": 90.0, "NYC-LAX": 350.0, "NYC-SFO": 380.0,
		"NYC-CHI": 200.0, "NYC-MIA": 280.0, "LAX-SFO": 120.0, "LAX-CHI": 320.0,
		"SFO-CHI": 340.0, "DXB-BOM": 180.0, "DXB-DEL": 200.0, "DXB-CCU": 220.0,
		"BOM-DEL": 80.0, "BOM-CCU": 120.0, "DEL-CCU": 100.0,
	}

	if fare, exists := routeFares[route]; exists {
		return fare
	}
	return 400.0 // Default fare
}

func getRouteMultiplier(route string) float64 {
	// High-demand routes get higher multipliers
	multipliers := map[string]float64{
		"NYC-LON": 1.2, "NYC-PAR": 1.15, "NYC-FRA": 1.1,
		"LON-PAR": 0.9, "LON-FRA": 0.95, "DXB-BOM": 1.05,
		"DXB-DEL": 1.1, "NYC-LAX": 1.05, "NYC-SFO": 1.1,
	}

	if multiplier, exists := multipliers[route]; exists {
		return multiplier
	}
	return 1.0
}

func getClassMultiplier(class string) float64 {
	multipliers := map[string]float64{
		"Y": 1.0,   // Economy
		"W": 1.5,   // Premium Economy
		"J": 3.0,   // Business
		"F": 6.0,   // First
		"P": 1.3,   // Premium
		"C": 2.5,   // Club
	}

	if multiplier, exists := multipliers[class]; exists {
		return multiplier
	}
	return 1.0
}

func getLoyaltyDiscount(tier string) float64 {
	discounts := map[string]float64{
		"Basic":    0.0,
		"Silver":   0.05,
		"Gold":     0.08,
		"Platinum": 0.12,
		"Diamond":  0.15,
	}

	if discount, exists := discounts[tier]; exists {
		return discount
	}
	return 0.0
}

func getChannelAdjustment(channel string) float64 {
	adjustments := map[string]float64{
		"web":      0.0,
		"mobile":   -0.02, // 2% discount for mobile
		"agent":    0.05,  // 5% surcharge for agent
		"gds":      0.03,  // 3% surcharge for GDS
		"api":      -0.01, // 1% discount for API
	}

	if adjustment, exists := adjustments[channel]; exists {
		return adjustment
	}
	return 0.0
}

// Cache management functions
func (ps *PricingService) generateCacheKey(request PriceRequest) string {
	// Generate cache key from request parameters
	key := fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		request.Route,
		request.Class,
		request.DepartureDate,
		request.LoyaltyTier,
		request.CorporateID,
		request.BookingChannel,
	)
	return key
}

func (ps *PricingService) getCachedPrice(key string) (PriceResponse, bool) {
	ps.cache.mu.RLock()
	defer ps.cache.mu.RUnlock()

	// Check if key exists and hasn't expired
	if expiry, exists := ps.cache.expiry[key]; exists {
		if time.Now().Before(expiry) {
			if price, exists := ps.cache.prices[key]; exists {
				ps.cache.hitCount++
				return price, true
			}
		} else {
			// Clean up expired entry
			delete(ps.cache.prices, key)
			delete(ps.cache.expiry, key)
		}
	}

	ps.cache.missCount++
	return PriceResponse{}, false
}

func (ps *PricingService) setCachedPrice(key string, response PriceResponse) {
	ps.cache.mu.Lock()
	defer ps.cache.mu.Unlock()

	ps.cache.prices[key] = response
	ps.cache.expiry[key] = time.Now().Add(ps.config.CacheExpiry)
}

// Rate limiter worker
func (ps *PricingService) rateLimiterWorker() {
	ticker := time.NewTicker(time.Second / time.Duration(ps.config.RateLimitRPS))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			select {
			case <-ps.rateLimiter:
				// Rate limit slot released
			default:
				// No goroutine waiting
			}
		}
	}
}

// Legacy functions for backward compatibility
func getBasicPrice(route string) (float64, error) {
	baseFare := getBaseFareForRoute(route)
	if baseFare == 0 {
		return 0, errors.New("route not found")
	}
	return baseFare * 1.3, nil // Basic markup
}

// Additional service functions
func GetDetailedPrice(route string, passengers int, class string) (PriceResponse, error) {
	if err := Initialize(); err != nil {
		return PriceResponse{}, err
	}

	request := PriceRequest{
		Route:          route,
		Passengers:     passengers,
		Class:          class,
		DepartureDate:  time.Now().AddDate(0, 0, 30).Format("2006-01-02"),
		BookingChannel: "web",
		RequestID:      fmt.Sprintf("detailed-%d", time.Now().UnixNano()),
		Timestamp:      time.Now(),
	}

	return pricingService.GetComprehensivePrice(request)
}

func GetPricingMetrics() map[string]interface{} {
	if pricingService == nil {
		return map[string]interface{}{"error": "service not initialized"}
	}

	pricingService.metrics.mu.RLock()
	defer pricingService.metrics.mu.RUnlock()

	hitRate := 0.0
	if pricingService.metrics.CacheHits+pricingService.metrics.CacheMisses > 0 {
		hitRate = float64(pricingService.metrics.CacheHits) / float64(pricingService.metrics.CacheHits+pricingService.metrics.CacheMisses)
	}

	return map[string]interface{}{
		"total_requests":   pricingService.metrics.TotalRequests,
		"cache_hits":       pricingService.metrics.CacheHits,
		"cache_misses":     pricingService.metrics.CacheMisses,
		"cache_hit_rate":   hitRate,
		"service_calls":    pricingService.metrics.ServiceCalls,
		"fallback_calls":   pricingService.metrics.FallbackCalls,
		"error_count":      pricingService.metrics.ErrorCount,
		"avg_response_time": pricingService.metrics.AvgResponseTime.String(),
		"last_updated":     pricingService.metrics.LastUpdated,
	}
}

func ClearCache() {
	if pricingService == nil {
		return
	}

	pricingService.cache.mu.Lock()
	defer pricingService.cache.mu.Unlock()

	pricingService.cache.prices = make(map[string]PriceResponse)
	pricingService.cache.expiry = make(map[string]time.Time)
	pricingService.cache.hitCount = 0
	pricingService.cache.missCount = 0
}

func GetCacheStatus() map[string]interface{} {
	if pricingService == nil {
		return map[string]interface{}{"error": "service not initialized"}
	}

	pricingService.cache.mu.RLock()
	defer pricingService.cache.mu.RUnlock()

	return map[string]interface{}{
		"cached_prices": len(pricingService.cache.prices),
		"hit_count":     pricingService.cache.hitCount,
		"miss_count":    pricingService.cache.missCount,
		"cache_size":    len(pricingService.cache.prices),
	}
} 