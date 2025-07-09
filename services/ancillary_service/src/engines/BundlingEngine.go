package engines

import (
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"time"
	"strings"

	"iaros/ancillary_service/src/models"
)

// BundlingEngine implements advanced AI-powered bundling logic
type BundlingEngine struct {
	ancillaryItems    []models.AncillaryItem
	bundles           []models.AncillaryBundle
	analytics         map[string]models.AncillaryAnalytics
	bundleAnalytics   map[string]models.BundleAnalytics
	customerProfiles  map[string]models.Customer
	mlModel           *MLRecommendationModel
}

// MLRecommendationModel represents a simplified ML model for recommendations
type MLRecommendationModel struct {
	CategoryWeights      map[string]float64
	SegmentPreferences   map[string][]string
	PopularityThreshold  float64
	PriceElasticityFactors map[string]float64
	SeasonalFactors      map[string]float64
	RouteFactors         map[string]float64
}

// NewBundlingEngine creates a new bundling engine instance
func NewBundlingEngine() *BundlingEngine {
	engine := &BundlingEngine{
		ancillaryItems:   models.GetDefaultAncillaryItems(),
		bundles:          models.GetDefaultBundles(),
		analytics:        make(map[string]models.AncillaryAnalytics),
		bundleAnalytics:  make(map[string]models.BundleAnalytics),
		customerProfiles: make(map[string]models.Customer),
		mlModel:          initializeMLModel(),
	}
	
	// Initialize analytics with mock data
	engine.initializeAnalytics()
	
	return engine
}

// initializeMLModel creates and initializes the ML recommendation model
func initializeMLModel() *MLRecommendationModel {
	return &MLRecommendationModel{
		CategoryWeights: map[string]float64{
			"connectivity": 0.9,
			"comfort":      0.8,
			"convenience":  0.7,
			"dining":       0.6,
			"baggage":      0.8,
			"protection":   0.4,
			"entertainment": 0.5,
			"ground_service": 0.6,
		},
		SegmentPreferences: map[string][]string{
			"Business Elite":     {"comfort", "convenience", "connectivity", "ground_service"},
			"Family Traveler":    {"baggage", "dining", "entertainment", "protection"},
			"Frequent Flyer":     {"convenience", "comfort", "connectivity"},
			"Leisure Traveler":   {"entertainment", "dining", "comfort"},
			"Budget Conscious":   {"baggage", "convenience"},
			"Tech Savvy":         {"connectivity", "entertainment"},
			"Luxury Traveler":    {"comfort", "dining", "ground_service"},
			"International":      {"protection", "comfort", "convenience"},
		},
		PopularityThreshold: 0.5,
		PriceElasticityFactors: map[string]float64{
			"high":   0.7,
			"medium": 0.85,
			"low":    1.2,
		},
		SeasonalFactors: map[string]float64{
			"peak":     1.3,
			"shoulder": 1.1,
			"off":      0.9,
		},
		RouteFactors: map[string]float64{
			"NYC-LON": 1.2,
			"LAX-TOK": 1.3,
			"SFO-FRA": 1.1,
			"MIA-MAD": 1.0,
		},
	}
}

// initializeAnalytics populates analytics with mock data
func (be *BundlingEngine) initializeAnalytics() {
	for _, item := range be.ancillaryItems {
		be.analytics[item.ID] = models.AncillaryAnalytics{
			ItemID:               item.ID,
			TotalSales:           rand.Intn(5000) + 1000,
			TotalRevenue:         float64(rand.Intn(100000) + 20000),
			ConversionRate:       item.ConversionRate,
			AveragePrice:         item.BasePrice,
			PopularityTrend:      []string{"increasing", "stable", "decreasing"}[rand.Intn(3)],
			PerformanceScore:     item.PopularityScore,
			CustomerSatisfaction: 3.5 + rand.Float64()*1.5,
			ReturnRate:          rand.Float64() * 0.1,
			LastUpdated:         time.Now(),
		}
	}

	for _, bundle := range be.bundles {
		be.bundleAnalytics[bundle.ID] = models.BundleAnalytics{
			BundleID:             bundle.ID,
			TotalSales:           rand.Intn(2000) + 500,
			TotalRevenue:         float64(rand.Intn(200000) + 50000),
			ConversionRate:       0.15 + rand.Float64()*0.25,
			AverageDiscount:      bundle.DiscountPercentage,
			PopularityTrend:      []string{"increasing", "stable", "decreasing"}[rand.Intn(3)],
			PerformanceScore:     bundle.PopularityScore,
			CustomerSatisfaction: 3.8 + rand.Float64()*1.2,
			AttachRate:          0.3 + rand.Float64()*0.4,
			LastUpdated:         time.Now(),
		}
	}
}

// GenerateRecommendations creates personalized ancillary recommendations for a customer
func (be *BundlingEngine) GenerateRecommendations(customer models.Customer) (models.BundleRecommendation, error) {
	if time.Since(customer.LastUpdate) > time.Hour*24 {
		log.Printf("Customer data stale for customer %s; using general recommendations", customer.ID)
	}

	// Filter available items for the customer
	availableItems := be.filterAvailableItems(customer)
	if len(availableItems) == 0 {
		return models.BundleRecommendation{}, errors.New("no available ancillary items for customer")
	}

	// Generate bundle recommendations
	recommendedBundles := be.generateBundleRecommendations(customer, availableItems)
	
	// Generate individual item recommendations
	recommendedItems := be.generateItemRecommendations(customer, availableItems)

	// Calculate overall confidence score
	confidenceScore := be.calculateConfidenceScore(customer, recommendedBundles, recommendedItems)

	recommendation := models.BundleRecommendation{
		CustomerID:          customer.ID,
		RecommendedBundles:  recommendedBundles,
		IndividualItems:     recommendedItems,
		ConfidenceScore:     confidenceScore,
		RecommendationLogic: be.generateRecommendationLogic(customer),
		GeneratedAt:         time.Now(),
	}

	return recommendation, nil
}

// filterAvailableItems filters ancillary items available for the customer
func (be *BundlingEngine) filterAvailableItems(customer models.Customer) []models.AncillaryItem {
	var availableItems []models.AncillaryItem
	
	for _, item := range be.ancillaryItems {
		if item.IsAvailableForCustomer(customer) {
			availableItems = append(availableItems, item)
		}
	}
	
	return availableItems
}

// generateBundleRecommendations creates scored bundle recommendations
func (be *BundlingEngine) generateBundleRecommendations(customer models.Customer, availableItems []models.AncillaryItem) []models.RecommendedBundle {
	var recommendations []models.RecommendedBundle
	
	// Check existing bundles
	for _, bundle := range be.bundles {
		if be.isBundleApplicable(bundle, customer) {
			score := be.calculateBundleScore(bundle, customer, availableItems)
			
			recommendation := models.RecommendedBundle{
				Bundle:          bundle,
				RelevanceScore:  score.relevance,
				PriceScore:      score.price,
				PopularityScore: score.popularity,
				OverallScore:    score.overall,
				Reasoning:       be.generateBundleReasoning(bundle, customer, score),
			}
			
			recommendations = append(recommendations, recommendation)
		}
	}
	
	// Generate dynamic bundles using AI
	dynamicBundles := be.generateDynamicBundles(customer, availableItems)
	recommendations = append(recommendations, dynamicBundles...)
	
	// Sort by overall score
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].OverallScore > recommendations[j].OverallScore
	})
	
	// Return top 5 recommendations
	if len(recommendations) > 5 {
		recommendations = recommendations[:5]
	}
	
	return recommendations
}

// generateItemRecommendations creates scored individual item recommendations
func (be *BundlingEngine) generateItemRecommendations(customer models.Customer, availableItems []models.AncillaryItem) []models.RecommendedItem {
	var recommendations []models.RecommendedItem
	
	for _, item := range availableItems {
		score := be.calculateItemScore(item, customer)
		
		recommendation := models.RecommendedItem{
			Item:            item,
			RelevanceScore:  score.relevance,
			PriceScore:      score.price,
			PopularityScore: score.popularity,
			OverallScore:    score.overall,
			Reasoning:       be.generateItemReasoning(item, customer, score),
		}
		
		recommendations = append(recommendations, recommendation)
	}
	
	// Sort by overall score
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].OverallScore > recommendations[j].OverallScore
	})
	
	// Return top 8 recommendations
	if len(recommendations) > 8 {
		recommendations = recommendations[:8]
	}
	
	return recommendations
}

// Score represents scoring components
type Score struct {
	relevance  float64
	price      float64
	popularity float64
	overall    float64
}

// calculateBundleScore calculates scoring for a bundle
func (be *BundlingEngine) calculateBundleScore(bundle models.AncillaryBundle, customer models.Customer, availableItems []models.AncillaryItem) Score {
	// Relevance score based on customer segment and preferences
	relevanceScore := be.calculateBundleRelevance(bundle, customer)
	
	// Price score based on customer spending profile
	priceScore := be.calculateBundlePriceScore(bundle, customer)
	
	// Popularity score from analytics
	popularityScore := bundle.PopularityScore
	if analytics, exists := be.bundleAnalytics[bundle.ID]; exists {
		popularityScore = analytics.PerformanceScore
	}
	
	// Overall score with weighted components
	overallScore := (relevanceScore * 0.4) + (priceScore * 0.35) + (popularityScore * 0.25)
	
	return Score{
		relevance:  relevanceScore,
		price:      priceScore,
		popularity: popularityScore,
		overall:    overallScore,
	}
}

// calculateItemScore calculates scoring for an individual item
func (be *BundlingEngine) calculateItemScore(item models.AncillaryItem, customer models.Customer) Score {
	// Relevance score based on customer preferences and category weights
	relevanceScore := be.calculateItemRelevance(item, customer)
	
	// Price score based on customer spending profile
	priceScore := be.calculateItemPriceScore(item, customer)
	
	// Popularity score from analytics
	popularityScore := item.PopularityScore
	if analytics, exists := be.analytics[item.ID]; exists {
		popularityScore = analytics.PerformanceScore
	}
	
	// Overall score with weighted components
	overallScore := (relevanceScore * 0.4) + (priceScore * 0.35) + (popularityScore * 0.25)
	
	return Score{
		relevance:  relevanceScore,
		price:      priceScore,
		popularity: popularityScore,
		overall:    overallScore,
	}
}

// calculateBundleRelevance calculates how relevant a bundle is to the customer
func (be *BundlingEngine) calculateBundleRelevance(bundle models.AncillaryBundle, customer models.Customer) float64 {
	score := 0.0
	
	// Segment matching
	for _, segment := range bundle.TargetSegments {
		if segment == customer.Segment {
			score += 0.4
			break
		}
	}
	
	// Route applicability
	for _, route := range bundle.RouteApplicability {
		if route == customer.Route {
			score += 0.2
			break
		}
	}
	
	// Category preferences
	if preferences, exists := be.mlModel.SegmentPreferences[customer.Segment]; exists {
		for _, category := range preferences {
			if strings.Contains(strings.ToLower(bundle.Name), category) || 
			   strings.Contains(strings.ToLower(bundle.Description), category) {
				score += 0.3
				break
			}
		}
	}
	
	// Previous purchase history
	for _, purchase := range customer.PreviousPurchases {
		for _, item := range bundle.Items {
			if purchase == item {
				score += 0.1
				break
			}
		}
	}
	
	return math.Min(score, 1.0)
}

// calculateItemRelevance calculates how relevant an item is to the customer
func (be *BundlingEngine) calculateItemRelevance(item models.AncillaryItem, customer models.Customer) float64 {
	score := 0.0
	
	// Category weight from ML model
	if weight, exists := be.mlModel.CategoryWeights[string(item.Category)]; exists {
		score += weight * 0.3
	}
	
	// Segment preferences
	if preferences, exists := be.mlModel.SegmentPreferences[customer.Segment]; exists {
		for _, category := range preferences {
			if category == string(item.Category) {
				score += 0.4
				break
			}
		}
	}
	
	// Customer preferred categories
	for _, category := range customer.PreferredCategories {
		if category == item.Category {
			score += 0.2
			break
		}
	}
	
	// Previous purchase history
	for _, purchase := range customer.PreviousPurchases {
		if purchase == item.ID {
			score += 0.1
			break
		}
	}
	
	return math.Min(score, 1.0)
}

// calculateBundlePriceScore calculates price attractiveness for a bundle
func (be *BundlingEngine) calculateBundlePriceScore(bundle models.AncillaryBundle, customer models.Customer) float64 {
	// Check if price is within customer's preferred range
	priceRange := customer.SpendingProfile.PreferredPriceRange
	if bundle.BundlePrice >= priceRange.Min && bundle.BundlePrice <= priceRange.Max {
		// Calculate position within range (closer to min = higher score)
		position := (priceRange.Max - bundle.BundlePrice) / (priceRange.Max - priceRange.Min)
		return 0.7 + (position * 0.3) // 0.7 to 1.0
	}
	
	// Apply price elasticity
	elasticity := 1.0
	if factor, exists := be.mlModel.PriceElasticityFactors[customer.SpendingProfile.Pricesensitivity]; exists {
		elasticity = factor
	}
	
	// Calculate value perception based on discount
	discountScore := bundle.DiscountPercentage / 100.0
	
	// Combine factors
	score := (elasticity * 0.6) + (discountScore * 0.4)
	
	return math.Min(score, 1.0)
}

// calculateItemPriceScore calculates price attractiveness for an item
func (be *BundlingEngine) calculateItemPriceScore(item models.AncillaryItem, customer models.Customer) float64 {
	dynamicPrice := item.GetDynamicPrice(customer, customer.Route)
	
	// Check if price is within customer's preferred range
	priceRange := customer.SpendingProfile.PreferredPriceRange
	if dynamicPrice >= priceRange.Min && dynamicPrice <= priceRange.Max {
		// Calculate position within range (closer to min = higher score)
		position := (priceRange.Max - dynamicPrice) / (priceRange.Max - priceRange.Min)
		return 0.7 + (position * 0.3) // 0.7 to 1.0
	}
	
	// Apply price elasticity
	elasticity := 1.0
	if factor, exists := be.mlModel.PriceElasticityFactors[customer.SpendingProfile.Pricesensitivity]; exists {
		elasticity = factor
	}
	
	// Calculate value based on revenue impact
	valueScore := item.RevenueImpact / 5.0 // Normalize to 0-1 scale
	
	// Combine factors
	score := (elasticity * 0.7) + (valueScore * 0.3)
	
	return math.Min(score, 1.0)
}

// generateDynamicBundles creates AI-generated dynamic bundles
func (be *BundlingEngine) generateDynamicBundles(customer models.Customer, availableItems []models.AncillaryItem) []models.RecommendedBundle {
	var dynamicBundles []models.RecommendedBundle
	
	// Group items by category
	categoryItems := make(map[models.AncillaryCategory][]models.AncillaryItem)
	for _, item := range availableItems {
		categoryItems[item.Category] = append(categoryItems[item.Category], item)
	}
	
	// Generate smart combinations based on customer segment
	combinations := be.generateSmartCombinations(customer, categoryItems)
	
	for i, combination := range combinations {
		if len(combination) < 2 || len(combination) > 4 {
			continue // Skip single items or too large bundles
		}
		
		bundle := be.createDynamicBundle(fmt.Sprintf("ai-generated-%d", i), combination, customer)
		score := be.calculateDynamicBundleScore(bundle, customer, combination)
		
		recommendation := models.RecommendedBundle{
			Bundle:          bundle,
			RelevanceScore:  score.relevance,
			PriceScore:      score.price,
			PopularityScore: score.popularity,
			OverallScore:    score.overall,
			Reasoning:       be.generateDynamicBundleReasoning(bundle, customer),
		}
		
		dynamicBundles = append(dynamicBundles, recommendation)
	}
	
	return dynamicBundles
}

// generateSmartCombinations creates smart item combinations based on customer segment
func (be *BundlingEngine) generateSmartCombinations(customer models.Customer, categoryItems map[models.AncillaryCategory][]models.AncillaryItem) [][]models.AncillaryItem {
	var combinations [][]models.AncillaryItem
	
	// Get preferred categories for customer segment
	preferredCategories, exists := be.mlModel.SegmentPreferences[customer.Segment]
	if !exists {
		preferredCategories = []string{"comfort", "convenience"} // Default
	}
	
	// Generate combinations based on preferences
	for i, category1 := range preferredCategories {
		if i >= 3 { break } // Limit to top 3 categories
		
		cat1 := models.AncillaryCategory(category1)
		if items1, exists := categoryItems[cat1]; exists && len(items1) > 0 {
			// Single category bundle
			if len(items1) > 1 {
				combinations = append(combinations, items1[:2])
			}
			
			// Cross-category bundles
			for j, category2 := range preferredCategories {
				if i != j && j < 2 { // Combine with top 2 other categories
					cat2 := models.AncillaryCategory(category2)
					if items2, exists := categoryItems[cat2]; exists && len(items2) > 0 {
						combo := []models.AncillaryItem{items1[0]}
						if len(items2) > 0 {
							combo = append(combo, items2[0])
						}
						combinations = append(combinations, combo)
					}
				}
			}
		}
	}
	
	return combinations
}

// createDynamicBundle creates a dynamic bundle from a combination of items
func (be *BundlingEngine) createDynamicBundle(id string, items []models.AncillaryItem, customer models.Customer) models.AncillaryBundle {
	var itemIDs []string
	var originalPrice float64
	var categories []string
	
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
		originalPrice += item.GetDynamicPrice(customer, customer.Route)
		categories = append(categories, string(item.Category))
	}
	
	// Calculate intelligent discount based on bundle size and customer segment
	discountPercentage := be.calculateIntelligentDiscount(len(items), customer)
	bundlePrice := originalPrice * (1 - discountPercentage/100)
	
	// Generate bundle name
	name := be.generateBundleName(categories, customer.Segment)
	
	return models.AncillaryBundle{
		ID:                 id,
		Name:               name,
		Description:        fmt.Sprintf("AI-curated bundle for %s", customer.Segment),
		Items:              itemIDs,
		OriginalPrice:      originalPrice,
		BundlePrice:        bundlePrice,
		DiscountPercentage: discountPercentage,
		Currency:           "USD",
		Available:          true,
		PopularityScore:    0.7, // Default for new bundles
		Category:           "ai-generated",
		TargetSegments:     []string{customer.Segment},
		RouteApplicability: []string{customer.Route},
		ValidFrom:          time.Now(),
		ValidTo:            time.Now().AddDate(0, 1, 0), // Valid for 1 month
		MaxItemsPerBundle:  4,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
}

// calculateIntelligentDiscount calculates discount based on bundle characteristics
func (be *BundlingEngine) calculateIntelligentDiscount(itemCount int, customer models.Customer) float64 {
	baseDiscount := 10.0 // Base 10% discount
	
	// More items = higher discount
	sizeDiscount := float64(itemCount-1) * 3.0
	
	// Customer tier bonus
	tierBonus := 0.0
	switch customer.Tier {
	case "Diamond":
		tierBonus = 5.0
	case "Platinum":
		tierBonus = 3.0
	case "Gold":
		tierBonus = 2.0
	}
	
	// Segment-based discount
	segmentBonus := 0.0
	switch customer.Segment {
	case "Business Elite":
		segmentBonus = 3.0
	case "Family Traveler":
		segmentBonus = 5.0
	case "Frequent Flyer":
		segmentBonus = 4.0
	}
	
	totalDiscount := baseDiscount + sizeDiscount + tierBonus + segmentBonus
	
	// Cap at 25%
	return math.Min(totalDiscount, 25.0)
}

// generateBundleName creates intelligent bundle names
func (be *BundlingEngine) generateBundleName(categories []string, segment string) string {
	uniqueCategories := make(map[string]bool)
	for _, cat := range categories {
		uniqueCategories[cat] = true
	}
	
	if len(uniqueCategories) == 1 {
		// Single category bundle
		for cat := range uniqueCategories {
			switch cat {
			case "comfort":
				return "Smart Comfort Bundle"
			case "convenience":
				return "Time Saver Bundle"
			case "connectivity":
				return "Stay Connected Bundle"
			case "dining":
				return "Gourmet Experience Bundle"
			case "baggage":
				return "Extra Space Bundle"
			default:
				return fmt.Sprintf("Smart %s Bundle", strings.Title(cat))
			}
		}
	}
	
	// Multi-category bundle
	switch segment {
	case "Business Elite":
		return "Executive Experience Bundle"
	case "Family Traveler":
		return "Family Perfect Bundle"
	case "Frequent Flyer":
		return "Traveler's Choice Bundle"
	case "Leisure Traveler":
		return "Vacation Enhancer Bundle"
	default:
		return "Personalized Travel Bundle"
	}
}

// Helper methods for scoring and reasoning

func (be *BundlingEngine) calculateDynamicBundleScore(bundle models.AncillaryBundle, customer models.Customer, items []models.AncillaryItem) Score {
	// Calculate average item scores
	var totalRelevance, totalPrice, totalPopularity float64
	
	for _, item := range items {
		itemScore := be.calculateItemScore(item, customer)
		totalRelevance += itemScore.relevance
		totalPrice += itemScore.price
		totalPopularity += itemScore.popularity
	}
	
	count := float64(len(items))
	avgRelevance := totalRelevance / count
	avgPrice := totalPrice / count
	avgPopularity := totalPopularity / count
	
	// Bundle-specific adjustments
	bundleBonus := 0.1 // Bonus for being a bundle
	discountBonus := bundle.DiscountPercentage / 100.0 * 0.2
	
	relevanceScore := math.Min(avgRelevance + bundleBonus, 1.0)
	priceScore := math.Min(avgPrice + discountBonus, 1.0)
	popularityScore := avgPopularity
	
	overallScore := (relevanceScore * 0.4) + (priceScore * 0.35) + (popularityScore * 0.25)
	
	return Score{
		relevance:  relevanceScore,
		price:      priceScore,
		popularity: popularityScore,
		overall:    overallScore,
	}
}

func (be *BundlingEngine) isBundleApplicable(bundle models.AncillaryBundle, customer models.Customer) bool {
	// Check if bundle is currently valid
	now := time.Now()
	if now.Before(bundle.ValidFrom) || now.After(bundle.ValidTo) {
		return false
	}
	
	// Check target segments
	if len(bundle.TargetSegments) > 0 {
		segmentMatch := false
		for _, segment := range bundle.TargetSegments {
			if segment == customer.Segment {
				segmentMatch = true
				break
			}
		}
		if !segmentMatch {
			return false
		}
	}
	
	// Check route applicability
	if len(bundle.RouteApplicability) > 0 {
		routeMatch := false
		for _, route := range bundle.RouteApplicability {
			if route == customer.Route {
				routeMatch = true
				break
			}
		}
		if !routeMatch {
			return false
		}
	}
	
	return bundle.Available
}

func (be *BundlingEngine) calculateConfidenceScore(customer models.Customer, bundles []models.RecommendedBundle, items []models.RecommendedItem) float64 {
	// Base confidence from data freshness
	dataFreshness := 1.0 - (time.Since(customer.LastUpdate).Hours() / 24.0)
	if dataFreshness < 0 {
		dataFreshness = 0.3 // Minimum confidence for stale data
	}
	
	// Confidence from recommendation quality
	bundleConfidence := 0.5
	if len(bundles) > 0 {
		totalScore := 0.0
		for _, bundle := range bundles {
			totalScore += bundle.OverallScore
		}
		bundleConfidence = totalScore / float64(len(bundles))
	}
	
	itemConfidence := 0.5
	if len(items) > 0 {
		totalScore := 0.0
		for _, item := range items {
			totalScore += item.OverallScore
		}
		itemConfidence = totalScore / float64(len(items))
	}
	
	// Customer profile completeness
	profileCompleteness := be.calculateProfileCompleteness(customer)
	
	// Overall confidence
	confidence := (dataFreshness * 0.3) + (bundleConfidence * 0.3) + (itemConfidence * 0.3) + (profileCompleteness * 0.1)
	
	return math.Min(confidence, 1.0)
}

func (be *BundlingEngine) calculateProfileCompleteness(customer models.Customer) float64 {
	score := 0.0
	maxScore := 8.0
	
	if customer.Segment != "" { score += 1.0 }
	if customer.Tier != "" { score += 1.0 }
	if len(customer.PreviousPurchases) > 0 { score += 1.0 }
	if len(customer.PreferredCategories) > 0 { score += 1.0 }
	if customer.SpendingProfile.AverageAncillarySpend > 0 { score += 1.0 }
	if customer.TravelFrequency != "" { score += 1.0 }
	if customer.Route != "" { score += 1.0 }
	if customer.BookingClass != "" { score += 1.0 }
	
	return score / maxScore
}

func (be *BundlingEngine) generateRecommendationLogic(customer models.Customer) string {
	logic := fmt.Sprintf("Recommendations based on customer segment: %s", customer.Segment)
	
	if customer.Tier != "" {
		logic += fmt.Sprintf(", tier: %s", customer.Tier)
	}
	
	if len(customer.PreviousPurchases) > 0 {
		logic += fmt.Sprintf(", %d previous purchases analyzed", len(customer.PreviousPurchases))
	}
	
	logic += fmt.Sprintf(", route: %s, class: %s", customer.Route, customer.BookingClass)
	
	return logic
}

func (be *BundlingEngine) generateBundleReasoning(bundle models.AncillaryBundle, customer models.Customer, score Score) string {
	reasons := []string{}
	
	if score.relevance > 0.7 {
		reasons = append(reasons, "highly relevant to your travel profile")
	}
	
	if score.price > 0.7 {
		reasons = append(reasons, "excellent value for money")
	}
	
	if score.popularity > 0.7 {
		reasons = append(reasons, "popular choice among similar travelers")
	}
	
	if bundle.DiscountPercentage > 15 {
		reasons = append(reasons, fmt.Sprintf("significant savings (%.0f%% off)", bundle.DiscountPercentage))
	}
	
	if len(reasons) == 0 {
		return "Good match for your travel needs"
	}
	
	return "Recommended because it offers " + strings.Join(reasons, ", ")
}

func (be *BundlingEngine) generateItemReasoning(item models.AncillaryItem, customer models.Customer, score Score) string {
	reasons := []string{}
	
	if score.relevance > 0.7 {
		reasons = append(reasons, "matches your preferences")
	}
	
	if score.price > 0.7 {
		reasons = append(reasons, "great value")
	}
	
	if score.popularity > 0.7 {
		reasons = append(reasons, "popular choice")
	}
	
	// Check if customer has bought similar items
	for _, purchase := range customer.PreviousPurchases {
		if strings.Contains(purchase, string(item.Category)) {
			reasons = append(reasons, "based on your purchase history")
			break
		}
	}
	
	if len(reasons) == 0 {
		return "Good addition to your journey"
	}
	
	return "Recommended because it " + strings.Join(reasons, ", ")
}

func (be *BundlingEngine) generateDynamicBundleReasoning(bundle models.AncillaryBundle, customer models.Customer) string {
	return fmt.Sprintf("AI-curated specifically for %s travelers on the %s route with %.0f%% savings", 
		customer.Segment, customer.Route, bundle.DiscountPercentage)
}

// GetAnalytics returns analytics for all ancillary items
func (be *BundlingEngine) GetAnalytics() map[string]models.AncillaryAnalytics {
	return be.analytics
}

// GetBundleAnalytics returns analytics for all bundles
func (be *BundlingEngine) GetBundleAnalytics() map[string]models.BundleAnalytics {
	return be.bundleAnalytics
}

// UpdateAnalytics updates analytics based on sales data
func (be *BundlingEngine) UpdateAnalytics(itemID string, salePrice float64, converted bool) {
	analytics, exists := be.analytics[itemID]
	if !exists {
		return
	}
	
	// Update sales and revenue
	if converted {
		analytics.TotalSales++
		analytics.TotalRevenue += salePrice
		analytics.AveragePrice = analytics.TotalRevenue / float64(analytics.TotalSales)
	}
	
	// Recalculate conversion rate (simplified)
	analytics.ConversionRate = float64(analytics.TotalSales) / float64(analytics.TotalSales + 100) // Assuming 100 impressions per sale
	
	analytics.LastUpdated = time.Now()
	be.analytics[itemID] = analytics
} 