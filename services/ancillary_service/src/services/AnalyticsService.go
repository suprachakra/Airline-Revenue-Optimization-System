package services

import (
	"fmt"
	"log"
	"math"
	"time"
	"sort"

	"iaros/ancillary_service/src/models"
)

// AnalyticsService provides analytics and reporting for ancillary services
type AnalyticsService struct {
	performanceMetrics *PerformanceMetrics
	revenueData        *RevenueData
	customerInsights   *CustomerInsights
	marketTrends       *MarketTrends
}

// PerformanceMetrics contains overall performance data
type PerformanceMetrics struct {
	TotalRevenue        float64                        `json:"total_revenue"`
	TotalSales          int                            `json:"total_sales"`
	AverageOrderValue   float64                        `json:"average_order_value"`
	ConversionRate      float64                        `json:"conversion_rate"`
	AttachRate          float64                        `json:"attach_rate"`
	CustomerSatisfaction float64                       `json:"customer_satisfaction"`
	RevenueGrowth       float64                        `json:"revenue_growth"`
	BundlePerformance   map[string]BundlePerformance   `json:"bundle_performance"`
	CategoryPerformance map[string]CategoryPerformance `json:"category_performance"`
	RoutePerformance    map[string]RoutePerformance    `json:"route_performance"`
	LastUpdated         time.Time                      `json:"last_updated"`
}

// BundlePerformance contains bundle-specific performance metrics
type BundlePerformance struct {
	BundleID            string  `json:"bundle_id"`
	BundleName          string  `json:"bundle_name"`
	TotalSales          int     `json:"total_sales"`
	Revenue             float64 `json:"revenue"`
	ConversionRate      float64 `json:"conversion_rate"`
	AverageDiscount     float64 `json:"average_discount"`
	CustomerSatisfaction float64 `json:"customer_satisfaction"`
	PopularityScore     float64 `json:"popularity_score"`
}

// CategoryPerformance contains category-specific performance metrics
type CategoryPerformance struct {
	Category            string  `json:"category"`
	TotalSales          int     `json:"total_sales"`
	Revenue             float64 `json:"revenue"`
	ConversionRate      float64 `json:"conversion_rate"`
	AveragePrice        float64 `json:"average_price"`
	MarketShare         float64 `json:"market_share"`
	GrowthRate          float64 `json:"growth_rate"`
}

// RoutePerformance contains route-specific performance metrics
type RoutePerformance struct {
	Route              string  `json:"route"`
	TotalSales         int     `json:"total_sales"`
	Revenue            float64 `json:"revenue"`
	AverageOrderValue  float64 `json:"average_order_value"`
	PopularItems       []string `json:"popular_items"`
	ConversionRate     float64 `json:"conversion_rate"`
}

// RevenueData contains revenue analytics
type RevenueData struct {
	CurrentPeriod       PeriodRevenue            `json:"current_period"`
	PreviousPeriod      PeriodRevenue            `json:"previous_period"`
	YearToDate          PeriodRevenue            `json:"year_to_date"`
	MonthlyTrend        []MonthlyRevenue         `json:"monthly_trend"`
	CategoryBreakdown   map[string]float64       `json:"category_breakdown"`
	SegmentBreakdown    map[string]float64       `json:"segment_breakdown"`
	RouteBreakdown      map[string]float64       `json:"route_breakdown"`
	Forecasts           RevenueForecast          `json:"forecasts"`
	LastUpdated         time.Time                `json:"last_updated"`
}

// PeriodRevenue contains revenue data for a specific period
type PeriodRevenue struct {
	Revenue             float64 `json:"revenue"`
	Sales               int     `json:"sales"`
	Growth              float64 `json:"growth"`
	AverageOrderValue   float64 `json:"average_order_value"`
	ConversionRate      float64 `json:"conversion_rate"`
	Period              string  `json:"period"`
}

// MonthlyRevenue contains monthly revenue data
type MonthlyRevenue struct {
	Month     string  `json:"month"`
	Revenue   float64 `json:"revenue"`
	Sales     int     `json:"sales"`
	Growth    float64 `json:"growth"`
}

// RevenueForecast contains revenue forecasting data
type RevenueForecast struct {
	NextMonth    float64 `json:"next_month"`
	NextQuarter  float64 `json:"next_quarter"`
	NextYear     float64 `json:"next_year"`
	Confidence   float64 `json:"confidence"`
	TrendDirection string `json:"trend_direction"`
}

// CustomerInsights contains customer behavior analytics
type CustomerInsights struct {
	SegmentAnalysis        map[string]SegmentInsight   `json:"segment_analysis"`
	CustomerLifetimeValue  map[string]float64          `json:"customer_lifetime_value"`
	ChurnRisk             map[string]float64          `json:"churn_risk"`
	PreferenceAnalysis    map[string][]string         `json:"preference_analysis"`
	BehaviorPatterns      map[string]BehaviorPattern  `json:"behavior_patterns"`
	SatisfactionScores    map[string]float64          `json:"satisfaction_scores"`
	LastUpdated           time.Time                   `json:"last_updated"`
}

// SegmentInsight contains insights for a customer segment
type SegmentInsight struct {
	Segment            string    `json:"segment"`
	TotalCustomers     int       `json:"total_customers"`
	AverageSpend       float64   `json:"average_spend"`
	ConversionRate     float64   `json:"conversion_rate"`
	PreferredCategories []string  `json:"preferred_categories"`
	SatisfactionScore  float64   `json:"satisfaction_score"`
	GrowthRate         float64   `json:"growth_rate"`
}

// BehaviorPattern contains customer behavior pattern data
type BehaviorPattern struct {
	Pattern            string    `json:"pattern"`
	Frequency          int       `json:"frequency"`
	AverageSpend       float64   `json:"average_spend"`
	PreferredItems     []string  `json:"preferred_items"`
	SeasonalTrends     []string  `json:"seasonal_trends"`
}

// MarketTrends contains market trend analysis
type MarketTrends struct {
	TrendingItems       []TrendingItem      `json:"trending_items"`
	SeasonalPatterns    []SeasonalPattern   `json:"seasonal_patterns"`
	CompetitorAnalysis  CompetitorAnalysis  `json:"competitor_analysis"`
	PricingTrends       PricingTrends       `json:"pricing_trends"`
	FutureOpportunities []Opportunity       `json:"future_opportunities"`
	LastUpdated         time.Time           `json:"last_updated"`
}

// TrendingItem contains trending item data
type TrendingItem struct {
	ItemID         string  `json:"item_id"`
	ItemName       string  `json:"item_name"`
	GrowthRate     float64 `json:"growth_rate"`
	PopularityScore float64 `json:"popularity_score"`
	RevenueImpact  float64 `json:"revenue_impact"`
}

// SeasonalPattern contains seasonal pattern data
type SeasonalPattern struct {
	Pattern        string             `json:"pattern"`
	Season         string             `json:"season"`
	ItemCategories []string           `json:"item_categories"`
	RevenueImpact  float64           `json:"revenue_impact"`
	Recommendations []string          `json:"recommendations"`
}

// CompetitorAnalysis contains competitor analysis data
type CompetitorAnalysis struct {
	MarketPosition    string             `json:"market_position"`
	CompetitivenessScore float64         `json:"competitiveness_score"`
	PriceComparison   map[string]float64 `json:"price_comparison"`
	FeatureGaps       []string           `json:"feature_gaps"`
	Opportunities     []string           `json:"opportunities"`
}

// PricingTrends contains pricing trend analysis
type PricingTrends struct {
	AveragePriceChange float64            `json:"average_price_change"`
	CategoryTrends     map[string]float64 `json:"category_trends"`
	OptimalPricing     map[string]float64 `json:"optimal_pricing"`
	PriceElasticity    map[string]float64 `json:"price_elasticity"`
}

// Opportunity contains future opportunity data
type Opportunity struct {
	OpportunityID   string  `json:"opportunity_id"`
	Description     string  `json:"description"`
	RevenueImpact   float64 `json:"revenue_impact"`
	Implementation  string  `json:"implementation"`
	Priority        string  `json:"priority"`
	Timeline        string  `json:"timeline"`
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService() *AnalyticsService {
	service := &AnalyticsService{}
	
	// Initialize with mock data
	service.initializeMockData()
	
	return service
}

// initializeMockData initializes the service with realistic mock data
func (as *AnalyticsService) initializeMockData() {
	// Initialize performance metrics
	as.performanceMetrics = &PerformanceMetrics{
		TotalRevenue:        1234567.89,
		TotalSales:          15672,
		AverageOrderValue:   78.76,
		ConversionRate:      0.234,
		AttachRate:          0.456,
		CustomerSatisfaction: 4.2,
		RevenueGrowth:       18.3,
		BundlePerformance: map[string]BundlePerformance{
			"comfort-plus": {
				BundleID:            "comfort-plus",
				BundleName:          "Comfort Plus Bundle",
				TotalSales:          2843,
				Revenue:             170580.00,
				ConversionRate:      0.312,
				AverageDiscount:     18.0,
				CustomerSatisfaction: 4.5,
				PopularityScore:     0.85,
			},
			"business-traveler": {
				BundleID:            "business-traveler",
				BundleName:          "Business Traveler Bundle",
				TotalSales:          1967,
				Revenue:             127836.00,
				ConversionRate:      0.267,
				AverageDiscount:     14.0,
				CustomerSatisfaction: 4.3,
				PopularityScore:     0.72,
			},
			"family-pack": {
				BundleID:            "family-pack",
				BundleName:          "Family Pack Bundle",
				TotalSales:          1534,
				Revenue:             101250.00,
				ConversionRate:      0.189,
				AverageDiscount:     14.0,
				CustomerSatisfaction: 4.1,
				PopularityScore:     0.69,
			},
		},
		CategoryPerformance: map[string]CategoryPerformance{
			"connectivity": {
				Category:       "connectivity",
				TotalSales:     4567,
				Revenue:        59371.00,
				ConversionRate: 0.456,
				AveragePrice:   12.99,
				MarketShare:    0.28,
				GrowthRate:     15.2,
			},
			"comfort": {
				Category:       "comfort",
				TotalSales:     3234,
				Revenue:        145530.00,
				ConversionRate: 0.234,
				AveragePrice:   45.00,
				MarketShare:    0.22,
				GrowthRate:     12.8,
			},
			"baggage": {
				Category:       "baggage",
				TotalSales:     2876,
				Revenue:        100660.00,
				ConversionRate: 0.198,
				AveragePrice:   35.00,
				MarketShare:    0.18,
				GrowthRate:     8.5,
			},
		},
		RoutePerformance: map[string]RoutePerformance{
			"NYC-LON": {
				Route:             "NYC-LON",
				TotalSales:        3456,
				Revenue:           285432.00,
				AverageOrderValue: 82.50,
				PopularItems:      []string{"wifi-premium", "seat-premium", "lounge-access"},
				ConversionRate:    0.267,
			},
			"LAX-TOK": {
				Route:             "LAX-TOK",
				TotalSales:        2897,
				Revenue:           256789.00,
				AverageOrderValue: 88.65,
				PopularItems:      []string{"baggage-20kg", "meal-premium", "wifi-premium"},
				ConversionRate:    0.234,
			},
		},
		LastUpdated: time.Now(),
	}
	
	// Initialize revenue data
	as.revenueData = &RevenueData{
		CurrentPeriod: PeriodRevenue{
			Revenue:           1234567.89,
			Sales:             15672,
			Growth:            18.3,
			AverageOrderValue: 78.76,
			ConversionRate:    0.234,
			Period:            "Current Month",
		},
		PreviousPeriod: PeriodRevenue{
			Revenue:           1043892.34,
			Sales:             13254,
			Growth:            12.1,
			AverageOrderValue: 78.76,
			ConversionRate:    0.219,
			Period:            "Previous Month",
		},
		YearToDate: PeriodRevenue{
			Revenue:           12876543.21,
			Sales:             163478,
			Growth:            22.5,
			AverageOrderValue: 78.76,
			ConversionRate:    0.241,
			Period:            "Year to Date",
		},
		MonthlyTrend: []MonthlyRevenue{
			{Month: "Jan", Revenue: 980234.56, Sales: 12456, Growth: 8.2},
			{Month: "Feb", Revenue: 1123456.78, Sales: 14267, Growth: 14.6},
			{Month: "Mar", Revenue: 1234567.89, Sales: 15672, Growth: 9.9},
			{Month: "Apr", Revenue: 1345678.90, Sales: 17083, Growth: 9.0},
			{Month: "May", Revenue: 1456789.01, Sales: 18494, Growth: 8.2},
			{Month: "Jun", Revenue: 1567890.12, Sales: 19905, Growth: 7.6},
		},
		CategoryBreakdown: map[string]float64{
			"connectivity":    295678.45,
			"comfort":         356789.12,
			"baggage":         234567.89,
			"dining":          187654.32,
			"convenience":     123456.78,
			"ground_service":  98765.43,
			"protection":      65432.10,
			"entertainment":   54321.09,
		},
		SegmentBreakdown: map[string]float64{
			"Business Elite":   456789.12,
			"Family Traveler":  345678.90,
			"Frequent Flyer":   234567.89,
			"Leisure Traveler": 123456.78,
			"Budget Conscious": 74075.20,
		},
		RouteBreakdown: map[string]float64{
			"NYC-LON": 285432.00,
			"LAX-TOK": 256789.00,
			"SFO-FRA": 198765.43,
			"MIA-MAD": 165432.10,
			"ORD-CDG": 132109.87,
		},
		Forecasts: RevenueForecast{
			NextMonth:      1523456.78,
			NextQuarter:    4567890.12,
			NextYear:       15234567.89,
			Confidence:     0.87,
			TrendDirection: "upward",
		},
		LastUpdated: time.Now(),
	}
	
	// Initialize customer insights
	as.customerInsights = &CustomerInsights{
		SegmentAnalysis: map[string]SegmentInsight{
			"Business Elite": {
				Segment:            "Business Elite",
				TotalCustomers:     5678,
				AverageSpend:       156.78,
				ConversionRate:     0.312,
				PreferredCategories: []string{"comfort", "connectivity", "ground_service"},
				SatisfactionScore:  4.5,
				GrowthRate:         22.3,
			},
			"Family Traveler": {
				Segment:            "Family Traveler",
				TotalCustomers:     8765,
				AverageSpend:       98.45,
				ConversionRate:     0.234,
				PreferredCategories: []string{"baggage", "dining", "entertainment"},
				SatisfactionScore:  4.2,
				GrowthRate:         18.7,
			},
		},
		CustomerLifetimeValue: map[string]float64{
			"Business Elite":   2456.78,
			"Family Traveler":  1345.67,
			"Frequent Flyer":   1876.54,
			"Leisure Traveler": 876.54,
			"Budget Conscious": 456.78,
		},
		ChurnRisk: map[string]float64{
			"Business Elite":   0.12,
			"Family Traveler":  0.18,
			"Frequent Flyer":   0.15,
			"Leisure Traveler": 0.25,
			"Budget Conscious": 0.32,
		},
		PreferenceAnalysis: map[string][]string{
			"Business Elite":   {"comfort", "connectivity", "ground_service"},
			"Family Traveler":  {"baggage", "dining", "entertainment"},
			"Frequent Flyer":   {"convenience", "comfort", "connectivity"},
			"Leisure Traveler": {"entertainment", "dining", "comfort"},
			"Budget Conscious": {"baggage", "convenience"},
		},
		BehaviorPatterns: map[string]BehaviorPattern{
			"Early Booker": {
				Pattern:        "Early Booker",
				Frequency:      3456,
				AverageSpend:   125.67,
				PreferredItems: []string{"seat-premium", "wifi-premium"},
				SeasonalTrends: []string{"summer", "winter holidays"},
			},
			"Last Minute": {
				Pattern:        "Last Minute",
				Frequency:      2345,
				AverageSpend:   78.90,
				PreferredItems: []string{"baggage-20kg", "priority-boarding"},
				SeasonalTrends: []string{"business travel peaks"},
			},
		},
		SatisfactionScores: map[string]float64{
			"Business Elite":   4.5,
			"Family Traveler":  4.2,
			"Frequent Flyer":   4.3,
			"Leisure Traveler": 4.0,
			"Budget Conscious": 3.8,
		},
		LastUpdated: time.Now(),
	}
	
	// Initialize market trends
	as.marketTrends = &MarketTrends{
		TrendingItems: []TrendingItem{
			{
				ItemID:         "wifi-premium",
				ItemName:       "Premium WiFi",
				GrowthRate:     25.6,
				PopularityScore: 0.91,
				RevenueImpact:  1.8,
			},
			{
				ItemID:         "seat-premium",
				ItemName:       "Premium Seat Selection",
				GrowthRate:     18.3,
				PopularityScore: 0.76,
				RevenueImpact:  3.2,
			},
		},
		SeasonalPatterns: []SeasonalPattern{
			{
				Pattern:        "Summer Travel Peak",
				Season:         "Summer",
				ItemCategories: []string{"entertainment", "comfort"},
				RevenueImpact:  1.35,
				Recommendations: []string{"Increase entertainment bundles", "Promote comfort upgrades"},
			},
			{
				Pattern:        "Business Travel Peak",
				Season:         "Fall",
				ItemCategories: []string{"connectivity", "convenience"},
				RevenueImpact:  1.22,
				Recommendations: []string{"Focus on business bundles", "Promote connectivity services"},
			},
		},
		CompetitorAnalysis: CompetitorAnalysis{
			MarketPosition:      "Leading",
			CompetitivenessScore: 0.84,
			PriceComparison: map[string]float64{
				"wifi-premium": 0.95,  // 5% below market
				"seat-premium": 1.02,  // 2% above market
				"baggage-20kg": 0.98,  // 2% below market
			},
			FeatureGaps:   []string{"Mobile app integration", "Voice booking"},
			Opportunities: []string{"AI personalization", "Dynamic bundling", "Subscription models"},
		},
		PricingTrends: PricingTrends{
			AveragePriceChange: 3.2,
			CategoryTrends: map[string]float64{
				"connectivity": 8.5,
				"comfort":      4.2,
				"baggage":      2.1,
			},
			OptimalPricing: map[string]float64{
				"wifi-premium": 14.99,
				"seat-premium": 48.00,
				"baggage-20kg": 37.50,
			},
			PriceElasticity: map[string]float64{
				"wifi-premium": -0.8,
				"seat-premium": -1.2,
				"baggage-20kg": -0.6,
			},
		},
		FutureOpportunities: []Opportunity{
			{
				OpportunityID:   "ai-bundling",
				Description:     "AI-powered dynamic bundling based on real-time customer behavior",
				RevenueImpact:   2.5, // Million USD
				Implementation:  "Machine learning model with customer segmentation",
				Priority:        "High",
				Timeline:        "6 months",
			},
			{
				OpportunityID:   "subscription-model",
				Description:     "Monthly subscription for frequent travelers",
				RevenueImpact:   1.8,
				Implementation:  "Tiered subscription with usage analytics",
				Priority:        "Medium",
				Timeline:        "9 months",
			},
		},
		LastUpdated: time.Now(),
	}
	
	log.Println("Initialized analytics service with comprehensive mock data")
}

// GetPerformanceMetrics returns overall performance metrics
func (as *AnalyticsService) GetPerformanceMetrics() (*PerformanceMetrics, error) {
	// Update with latest calculations
	as.refreshPerformanceMetrics()
	
	return as.performanceMetrics, nil
}

// GetRevenueAnalytics returns revenue analytics for a specific period
func (as *AnalyticsService) GetRevenueAnalytics(period string) (*RevenueData, error) {
	// Update with latest calculations
	as.refreshRevenueData(period)
	
	return as.revenueData, nil
}

// GetCustomerInsights returns customer behavior insights
func (as *AnalyticsService) GetCustomerInsights() (*CustomerInsights, error) {
	// Update with latest calculations
	as.refreshCustomerInsights()
	
	return as.customerInsights, nil
}

// GetMarketTrends returns market trend analysis
func (as *AnalyticsService) GetMarketTrends() (*MarketTrends, error) {
	// Update with latest calculations
	as.refreshMarketTrends()
	
	return as.marketTrends, nil
}

// GetTopPerformingItems returns top performing ancillary items
func (as *AnalyticsService) GetTopPerformingItems(limit int) ([]TrendingItem, error) {
	if limit <= 0 {
		limit = 10
	}
	
	items := as.marketTrends.TrendingItems
	
	// Sort by revenue impact
	sort.Slice(items, func(i, j int) bool {
		return items[i].RevenueImpact > items[j].RevenueImpact
	})
	
	if len(items) > limit {
		items = items[:limit]
	}
	
	return items, nil
}

// GetRevenueByCategory returns revenue breakdown by category
func (as *AnalyticsService) GetRevenueByCategory() (map[string]float64, error) {
	return as.revenueData.CategoryBreakdown, nil
}

// GetRevenueBySegment returns revenue breakdown by customer segment
func (as *AnalyticsService) GetRevenueBySegment() (map[string]float64, error) {
	return as.revenueData.SegmentBreakdown, nil
}

// GetRevenueForecast returns revenue forecast
func (as *AnalyticsService) GetRevenueForecast() (*RevenueForecast, error) {
	return &as.revenueData.Forecasts, nil
}

// GetCustomerSegmentAnalysis returns analysis for a specific customer segment
func (as *AnalyticsService) GetCustomerSegmentAnalysis(segment string) (*SegmentInsight, error) {
	insight, exists := as.customerInsights.SegmentAnalysis[segment]
	if !exists {
		return nil, fmt.Errorf("segment analysis not found for: %s", segment)
	}
	
	return &insight, nil
}

// GetBundlePerformance returns performance metrics for a specific bundle
func (as *AnalyticsService) GetBundlePerformance(bundleID string) (*BundlePerformance, error) {
	performance, exists := as.performanceMetrics.BundlePerformance[bundleID]
	if !exists {
		return nil, fmt.Errorf("bundle performance not found for: %s", bundleID)
	}
	
	return &performance, nil
}

// GetRoutePerformance returns performance metrics for a specific route
func (as *AnalyticsService) GetRoutePerformance(route string) (*RoutePerformance, error) {
	performance, exists := as.performanceMetrics.RoutePerformance[route]
	if !exists {
		return nil, fmt.Errorf("route performance not found for: %s", route)
	}
	
	return &performance, nil
}

// Helper methods for data refresh

// refreshPerformanceMetrics updates performance metrics with latest data
func (as *AnalyticsService) refreshPerformanceMetrics() {
	// In a real implementation, this would fetch latest data from database
	// For now, we'll simulate small changes
	
	// Add some realistic variation
	as.performanceMetrics.TotalRevenue *= (1 + (time.Now().Unix()%10-5)/1000.0)
	as.performanceMetrics.ConversionRate *= (1 + (time.Now().Unix()%6-3)/1000.0)
	as.performanceMetrics.LastUpdated = time.Now()
}

// refreshRevenueData updates revenue data with latest information
func (as *AnalyticsService) refreshRevenueData(period string) {
	// In a real implementation, this would fetch data for the specific period
	// For now, we'll adjust based on period
	
	multiplier := 1.0
	switch period {
	case "week":
		multiplier = 0.25
	case "month":
		multiplier = 1.0
	case "quarter":
		multiplier = 3.0
	case "year":
		multiplier = 12.0
	}
	
	as.revenueData.CurrentPeriod.Revenue *= multiplier
	as.revenueData.CurrentPeriod.Sales = int(float64(as.revenueData.CurrentPeriod.Sales) * multiplier)
	as.revenueData.LastUpdated = time.Now()
}

// refreshCustomerInsights updates customer insights with latest data
func (as *AnalyticsService) refreshCustomerInsights() {
	// In a real implementation, this would recalculate customer insights
	// For now, we'll simulate minor changes
	
	for segment, insight := range as.customerInsights.SegmentAnalysis {
		// Simulate small changes in satisfaction and growth
		insight.SatisfactionScore *= (1 + (time.Now().Unix()%10-5)/1000.0)
		insight.GrowthRate *= (1 + (time.Now().Unix()%8-4)/1000.0)
		as.customerInsights.SegmentAnalysis[segment] = insight
	}
	
	as.customerInsights.LastUpdated = time.Now()
}

// refreshMarketTrends updates market trends with latest analysis
func (as *AnalyticsService) refreshMarketTrends() {
	// In a real implementation, this would analyze latest market data
	// For now, we'll simulate trend changes
	
	for i, item := range as.marketTrends.TrendingItems {
		// Simulate growth rate changes
		item.GrowthRate *= (1 + (time.Now().Unix()%12-6)/1000.0)
		as.marketTrends.TrendingItems[i] = item
	}
	
	as.marketTrends.LastUpdated = time.Now()
}

// CalculateROI calculates return on investment for ancillary services
func (as *AnalyticsService) CalculateROI(investment float64) (float64, error) {
	if investment <= 0 {
		return 0, fmt.Errorf("investment must be positive")
	}
	
	totalRevenue := as.revenueData.CurrentPeriod.Revenue
	roi := ((totalRevenue - investment) / investment) * 100
	
	return roi, nil
}

// GenerateRecommendations generates business recommendations based on analytics
func (as *AnalyticsService) GenerateRecommendations() ([]string, error) {
	var recommendations []string
	
	// Analyze conversion rates
	if as.performanceMetrics.ConversionRate < 0.20 {
		recommendations = append(recommendations, "Consider improving product positioning to increase conversion rates")
	}
	
	// Analyze revenue growth
	if as.performanceMetrics.RevenueGrowth < 15.0 {
		recommendations = append(recommendations, "Focus on high-performing categories to accelerate growth")
	}
	
	// Analyze customer satisfaction
	if as.performanceMetrics.CustomerSatisfaction < 4.0 {
		recommendations = append(recommendations, "Implement customer feedback program to improve satisfaction")
	}
	
	// Analyze bundle performance
	for _, bundle := range as.performanceMetrics.BundlePerformance {
		if bundle.ConversionRate < 0.15 {
			recommendations = append(recommendations, 
				fmt.Sprintf("Review and optimize '%s' bundle - low conversion rate", bundle.BundleName))
		}
	}
	
	// Analyze market trends
	for _, opportunity := range as.marketTrends.FutureOpportunities {
		if opportunity.Priority == "High" && opportunity.RevenueImpact > 2.0 {
			recommendations = append(recommendations, 
				fmt.Sprintf("Prioritize implementation of '%s' - high revenue impact", opportunity.Description))
		}
	}
	
	return recommendations, nil
}

// GetKPIDashboard returns key performance indicators for dashboard
func (as *AnalyticsService) GetKPIDashboard() (map[string]interface{}, error) {
	dashboard := map[string]interface{}{
		"total_revenue":        as.performanceMetrics.TotalRevenue,
		"revenue_growth":       as.performanceMetrics.RevenueGrowth,
		"conversion_rate":      as.performanceMetrics.ConversionRate,
		"customer_satisfaction": as.performanceMetrics.CustomerSatisfaction,
		"attach_rate":          as.performanceMetrics.AttachRate,
		"average_order_value":  as.performanceMetrics.AverageOrderValue,
		"top_category":         as.getTopCategory(),
		"top_bundle":           as.getTopBundle(),
		"growth_trend":         as.revenueData.Forecasts.TrendDirection,
		"market_position":      as.marketTrends.CompetitorAnalysis.MarketPosition,
		"last_updated":         time.Now().UTC(),
	}
	
	return dashboard, nil
}

// Helper methods

func (as *AnalyticsService) getTopCategory() string {
	var topCategory string
	var maxRevenue float64
	
	for category, revenue := range as.revenueData.CategoryBreakdown {
		if revenue > maxRevenue {
			maxRevenue = revenue
			topCategory = category
		}
	}
	
	return topCategory
}

func (as *AnalyticsService) getTopBundle() string {
	var topBundle string
	var maxRevenue float64
	
	for _, bundle := range as.performanceMetrics.BundlePerformance {
		if bundle.Revenue > maxRevenue {
			maxRevenue = bundle.Revenue
			topBundle = bundle.BundleName
		}
	}
	
	return topBundle
} 