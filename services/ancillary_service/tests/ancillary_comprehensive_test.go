package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"iaros/ancillary_service/src/controllers"
	"iaros/ancillary_service/src/engines"
	"iaros/ancillary_service/src/models"
	"iaros/ancillary_service/src/services"
)

// AncillaryServiceTestSuite defines the test suite
type AncillaryServiceTestSuite struct {
	suite.Suite
	controller    *controllers.AncillaryController
	router        *mux.Router
	bundlingEngine *engines.BundlingEngine
	ancillaryService *services.AncillaryService
	testCustomer  models.Customer
}

// SetupSuite initializes the test suite
func (suite *AncillaryServiceTestSuite) SetupSuite() {
	suite.controller = controllers.NewAncillaryController()
	suite.router = mux.NewRouter()
	suite.controller.RegisterRoutes(suite.router)
	
	suite.bundlingEngine = engines.NewBundlingEngine()
	suite.ancillaryService = services.NewAncillaryService()
	
	// Create test customer
	suite.testCustomer = models.Customer{
		ID:      "test-customer-001",
		Segment: "Business Elite",
		Tier:    "Platinum",
		PreviousPurchases: []string{"wifi-premium", "seat-premium"},
		PreferredCategories: []models.AncillaryCategory{
			models.CategoryComfort,
			models.CategoryConnectivity,
			models.CategoryGroundService,
		},
		SpendingProfile: models.SpendingProfile{
			AverageAncillarySpend: 120.0,
			MaxAncillarySpend:     250.0,
			Pricesensitivity:      "low",
			PreferredPriceRange:   models.PriceRange{Min: 20.0, Max: 200.0},
		},
		TravelFrequency: "frequent",
		Route:           "NYC-LON",
		BookingClass:    "Business",
		TripType:        "business",
		CompanionCount:  0,
		Age:             &[]int{42}[0],
		LastUpdate:      time.Now(),
	}
}

// Test Health Check
func (suite *AncillaryServiceTestSuite) TestHealthCheck() {
	req, _ := http.NewRequest("GET", "/ancillary/health", nil)
	rr := httptest.NewRecorder()
	
	suite.router.ServeHTTP(rr, req)
	
	assert.Equal(suite.T(), http.StatusOK, rr.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "healthy", response["status"])
}

// Test Recommendation Generation
func (suite *AncillaryServiceTestSuite) TestGenerateRecommendations() {
	customerJSON, _ := json.Marshal(suite.testCustomer)
	
	req, _ := http.NewRequest("POST", "/ancillary/recommendations", bytes.NewBuffer(customerJSON))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	
	suite.router.ServeHTTP(rr, req)
	
	assert.Equal(suite.T(), http.StatusOK, rr.Code)
	
	var response models.BundleRecommendation
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), suite.testCustomer.ID, response.CustomerID)
	assert.Greater(suite.T(), len(response.RecommendedBundles), 0)
	assert.Greater(suite.T(), len(response.IndividualItems), 0)
	assert.Greater(suite.T(), response.ConfidenceScore, 0.0)
}

// Test Ancillary Items Management
func (suite *AncillaryServiceTestSuite) TestAncillaryItemsManagement() {
	// Test GET all items
	req, _ := http.NewRequest("GET", "/ancillary/items", nil)
	rr := httptest.NewRecorder()
	
	suite.router.ServeHTTP(rr, req)
	
	assert.Equal(suite.T(), http.StatusOK, rr.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Greater(suite.T(), response["count"], 0.0)
	
	// Test GET specific item
	req, _ = http.NewRequest("GET", "/ancillary/items/wifi-premium", nil)
	rr = httptest.NewRecorder()
	
	suite.router.ServeHTTP(rr, req)
	
	assert.Equal(suite.T(), http.StatusOK, rr.Code)
	
	var item models.AncillaryItem
	err = json.Unmarshal(rr.Body.Bytes(), &item)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "wifi-premium", item.ID)
	assert.Equal(suite.T(), "Premium WiFi", item.Name)
}

// Test Bundle Management
func (suite *AncillaryServiceTestSuite) TestBundleManagement() {
	// Test GET all bundles
	req, _ := http.NewRequest("GET", "/ancillary/bundles", nil)
	rr := httptest.NewRecorder()
	
	suite.router.ServeHTTP(rr, req)
	
	assert.Equal(suite.T(), http.StatusOK, rr.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Greater(suite.T(), response["count"], 0.0)
	
	// Test GET specific bundle
	req, _ = http.NewRequest("GET", "/ancillary/bundles/comfort-plus", nil)
	rr = httptest.NewRecorder()
	
	suite.router.ServeHTTP(rr, req)
	
	assert.Equal(suite.T(), http.StatusOK, rr.Code)
	
	var bundle models.AncillaryBundle
	err = json.Unmarshal(rr.Body.Bytes(), &bundle)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "comfort-plus", bundle.ID)
	assert.Equal(suite.T(), "Comfort Plus Bundle", bundle.Name)
	assert.Greater(suite.T(), len(bundle.Items), 0)
}

// Test Dynamic Pricing
func (suite *AncillaryServiceTestSuite) TestDynamicPricing() {
	customerJSON, _ := json.Marshal(suite.testCustomer)
	
	req, _ := http.NewRequest("POST", "/ancillary/items/wifi-premium/price", bytes.NewBuffer(customerJSON))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	
	suite.router.ServeHTTP(rr, req)
	
	assert.Equal(suite.T(), http.StatusOK, rr.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "wifi-premium", response["item_id"])
	assert.Greater(suite.T(), response["dynamic_price"], 0.0)
	assert.Equal(suite.T(), suite.testCustomer.ID, response["customer_id"])
}

// Test Analytics
func (suite *AncillaryServiceTestSuite) TestAnalytics() {
	// Test item analytics
	req, _ := http.NewRequest("GET", "/ancillary/analytics/items", nil)
	rr := httptest.NewRecorder()
	
	suite.router.ServeHTTP(rr, req)
	
	assert.Equal(suite.T(), http.StatusOK, rr.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Greater(suite.T(), response["count"], 0.0)
	
	// Test bundle analytics
	req, _ = http.NewRequest("GET", "/ancillary/analytics/bundles", nil)
	rr = httptest.NewRecorder()
	
	suite.router.ServeHTTP(rr, req)
	
	assert.Equal(suite.T(), http.StatusOK, rr.Code)
	
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Greater(suite.T(), response["count"], 0.0)
	
	// Test performance metrics
	req, _ = http.NewRequest("GET", "/ancillary/analytics/performance", nil)
	rr = httptest.NewRecorder()
	
	suite.router.ServeHTTP(rr, req)
	
	assert.Equal(suite.T(), http.StatusOK, rr.Code)
	
	var perfMetrics map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &perfMetrics)
	assert.NoError(suite.T(), err)
	assert.Greater(suite.T(), perfMetrics["total_revenue"], 0.0)
	assert.Greater(suite.T(), perfMetrics["conversion_rate"], 0.0)
}

// Test Purchase Recording
func (suite *AncillaryServiceTestSuite) TestPurchaseRecording() {
	purchase := models.Purchase{
		CustomerID: suite.testCustomer.ID,
		ItemID:     "wifi-premium",
		Amount:     15.99,
		Currency:   "USD",
		Status:     "completed",
		Timestamp:  time.Now(),
	}
	
	purchaseJSON, _ := json.Marshal(purchase)
	
	req, _ := http.NewRequest("POST", "/ancillary/purchase", bytes.NewBuffer(purchaseJSON))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	
	suite.router.ServeHTTP(rr, req)
	
	assert.Equal(suite.T(), http.StatusCreated, rr.Code)
	
	var response models.Purchase
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), suite.testCustomer.ID, response.CustomerID)
	assert.Equal(suite.T(), "wifi-premium", response.ItemID)
	assert.Equal(suite.T(), 15.99, response.Amount)
}

// Test Bundling Engine
func (suite *AncillaryServiceTestSuite) TestBundlingEngine() {
	recommendations, err := suite.bundlingEngine.GenerateRecommendations(suite.testCustomer)
	
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), suite.testCustomer.ID, recommendations.CustomerID)
	assert.Greater(suite.T(), len(recommendations.RecommendedBundles), 0)
	assert.Greater(suite.T(), len(recommendations.IndividualItems), 0)
	assert.Greater(suite.T(), recommendations.ConfidenceScore, 0.0)
	
	// Test bundle scoring
	for _, bundle := range recommendations.RecommendedBundles {
		assert.Greater(suite.T(), bundle.OverallScore, 0.0)
		assert.Greater(suite.T(), bundle.RelevanceScore, 0.0)
		assert.Greater(suite.T(), bundle.PriceScore, 0.0)
		assert.Greater(suite.T(), bundle.PopularityScore, 0.0)
		assert.NotEmpty(suite.T(), bundle.Reasoning)
	}
	
	// Test item scoring
	for _, item := range recommendations.IndividualItems {
		assert.Greater(suite.T(), item.OverallScore, 0.0)
		assert.Greater(suite.T(), item.RelevanceScore, 0.0)
		assert.Greater(suite.T(), item.PriceScore, 0.0)
		assert.Greater(suite.T(), item.PopularityScore, 0.0)
		assert.NotEmpty(suite.T(), item.Reasoning)
	}
}

// Test Ancillary Item Availability
func (suite *AncillaryServiceTestSuite) TestAncillaryItemAvailability() {
	items := models.GetDefaultAncillaryItems()
	
	for _, item := range items {
		isAvailable := item.IsAvailableForCustomer(suite.testCustomer)
		
		// For our test customer (Business Elite, NYC-LON route), most items should be available
		if item.ID == "wifi-premium" || item.ID == "seat-premium" {
			assert.True(suite.T(), isAvailable, fmt.Sprintf("Item %s should be available for test customer", item.ID))
		}
	}
}

// Test Dynamic Pricing Calculations
func (suite *AncillaryServiceTestSuite) TestDynamicPricingCalculations() {
	items := models.GetDefaultAncillaryItems()
	
	for _, item := range items {
		if item.IsAvailableForCustomer(suite.testCustomer) {
			dynamicPrice := item.GetDynamicPrice(suite.testCustomer, suite.testCustomer.Route)
			
			assert.Greater(suite.T(), dynamicPrice, 0.0)
			
			// For Business Elite customers, prices might be higher or include different multipliers
			if item.ID == "wifi-premium" {
				// Should have customer segment pricing
				if segmentPrice, exists := item.CustomerSegmentPrice[suite.testCustomer.Segment]; exists {
					assert.Equal(suite.T(), segmentPrice*0.9, dynamicPrice) // 10% tier discount for Platinum
				}
			}
		}
	}
}

// Test Customer Segmentation
func (suite *AncillaryServiceTestSuite) TestCustomerSegmentation() {
	// Test different customer segments
	segments := []string{"Business Elite", "Family Traveler", "Budget Conscious", "Frequent Flyer"}
	
	for _, segment := range segments {
		testCustomer := suite.testCustomer
		testCustomer.Segment = segment
		testCustomer.ID = fmt.Sprintf("test-customer-%s", segment)
		
		recommendations, err := suite.bundlingEngine.GenerateRecommendations(testCustomer)
		
		assert.NoError(suite.T(), err)
		assert.Greater(suite.T(), len(recommendations.RecommendedBundles), 0)
		assert.Greater(suite.T(), len(recommendations.IndividualItems), 0)
		
		// Check that recommendations are tailored to the segment
		assert.Contains(suite.T(), recommendations.RecommendationLogic, segment)
	}
}

// Test Bundle Generation
func (suite *AncillaryServiceTestSuite) TestBundleGeneration() {
	itemIDs := []string{"wifi-premium", "seat-premium", "priority-boarding"}
	
	bundle, err := suite.ancillaryService.GenerateDynamicBundle(suite.testCustomer, itemIDs)
	
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), len(itemIDs), len(bundle.Items))
	assert.Greater(suite.T(), bundle.OriginalPrice, 0.0)
	assert.Greater(suite.T(), bundle.BundlePrice, 0.0)
	assert.Greater(suite.T(), bundle.DiscountPercentage, 0.0)
	assert.Less(suite.T(), bundle.BundlePrice, bundle.OriginalPrice)
	assert.Contains(suite.T(), bundle.Name, suite.testCustomer.Segment)
}

// Test Error Handling
func (suite *AncillaryServiceTestSuite) TestErrorHandling() {
	// Test invalid customer data
	invalidCustomer := models.Customer{} // Missing required fields
	customerJSON, _ := json.Marshal(invalidCustomer)
	
	req, _ := http.NewRequest("POST", "/ancillary/recommendations", bytes.NewBuffer(customerJSON))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	
	suite.router.ServeHTTP(rr, req)
	
	assert.Equal(suite.T(), http.StatusBadRequest, rr.Code)
	
	// Test non-existent item
	req, _ = http.NewRequest("GET", "/ancillary/items/non-existent-item", nil)
	rr = httptest.NewRecorder()
	
	suite.router.ServeHTTP(rr, req)
	
	assert.Equal(suite.T(), http.StatusNotFound, rr.Code)
	
	// Test non-existent bundle
	req, _ = http.NewRequest("GET", "/ancillary/bundles/non-existent-bundle", nil)
	rr = httptest.NewRecorder()
	
	suite.router.ServeHTTP(rr, req)
	
	assert.Equal(suite.T(), http.StatusNotFound, rr.Code)
}

// Test Item Filtering
func (suite *AncillaryServiceTestSuite) TestItemFiltering() {
	// Test filtering by category
	req, _ := http.NewRequest("GET", "/ancillary/items?category=connectivity", nil)
	rr := httptest.NewRecorder()
	
	suite.router.ServeHTTP(rr, req)
	
	assert.Equal(suite.T(), http.StatusOK, rr.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Greater(suite.T(), response["count"], 0.0)
	
	// Test filtering by price range
	req, _ = http.NewRequest("GET", "/ancillary/items?min_price=10&max_price=50", nil)
	rr = httptest.NewRecorder()
	
	suite.router.ServeHTTP(rr, req)
	
	assert.Equal(suite.T(), http.StatusOK, rr.Code)
	
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Greater(suite.T(), response["count"], 0.0)
}

// Test Bundle Filtering
func (suite *AncillaryServiceTestSuite) TestBundleFiltering() {
	// Test filtering by segment
	req, _ := http.NewRequest("GET", "/ancillary/bundles?segment=Business Elite", nil)
	rr := httptest.NewRecorder()
	
	suite.router.ServeHTTP(rr, req)
	
	assert.Equal(suite.T(), http.StatusOK, rr.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Greater(suite.T(), response["count"], 0.0)
}

// Test Performance Benchmarks
func (suite *AncillaryServiceTestSuite) TestPerformanceBenchmarks() {
	// Test recommendation generation performance
	start := time.Now()
	
	for i := 0; i < 100; i++ {
		testCustomer := suite.testCustomer
		testCustomer.ID = fmt.Sprintf("perf-test-%d", i)
		
		_, err := suite.bundlingEngine.GenerateRecommendations(testCustomer)
		assert.NoError(suite.T(), err)
	}
	
	duration := time.Since(start)
	averageTime := duration / 100
	
	// Should be able to generate recommendations in under 100ms per request
	assert.Less(suite.T(), averageTime, 100*time.Millisecond, "Recommendation generation should be fast")
}

// Test Concurrent Access
func (suite *AncillaryServiceTestSuite) TestConcurrentAccess() {
	// Test concurrent recommendation generation
	numGoroutines := 10
	done := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			testCustomer := suite.testCustomer
			testCustomer.ID = fmt.Sprintf("concurrent-test-%d", id)
			
			_, err := suite.bundlingEngine.GenerateRecommendations(testCustomer)
			assert.NoError(suite.T(), err)
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// Test Data Validation
func (suite *AncillaryServiceTestSuite) TestDataValidation() {
	// Test creating item with invalid data
	invalidItem := models.AncillaryItem{
		Name:      "", // Missing required field
		BasePrice: -10.0, // Invalid price
	}
	
	itemJSON, _ := json.Marshal(invalidItem)
	
	req, _ := http.NewRequest("POST", "/ancillary/items", bytes.NewBuffer(itemJSON))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	
	suite.router.ServeHTTP(rr, req)
	
	assert.Equal(suite.T(), http.StatusBadRequest, rr.Code)
	
	// Test creating bundle with invalid data
	invalidBundle := models.AncillaryBundle{
		Name:  "", // Missing required field
		Items: []string{}, // Empty items
	}
	
	bundleJSON, _ := json.Marshal(invalidBundle)
	
	req, _ = http.NewRequest("POST", "/ancillary/bundles", bytes.NewBuffer(bundleJSON))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	
	suite.router.ServeHTTP(rr, req)
	
	assert.Equal(suite.T(), http.StatusBadRequest, rr.Code)
}

// Run the test suite
func TestAncillaryServiceTestSuite(t *testing.T) {
	suite.Run(t, new(AncillaryServiceTestSuite))
}

// Benchmark tests
func BenchmarkRecommendationGeneration(b *testing.B) {
	controller := controllers.NewAncillaryController()
	bundlingEngine := engines.NewBundlingEngine()
	
	testCustomer := models.Customer{
		ID:      "benchmark-customer",
		Segment: "Business Elite",
		Tier:    "Platinum",
		Route:   "NYC-LON",
		SpendingProfile: models.SpendingProfile{
			AverageAncillarySpend: 100.0,
			Pricesensitivity:      "low",
		},
		LastUpdate: time.Now(),
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := bundlingEngine.GenerateRecommendations(testCustomer)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDynamicPricing(b *testing.B) {
	items := models.GetDefaultAncillaryItems()
	testCustomer := models.Customer{
		ID:      "benchmark-customer",
		Segment: "Business Elite",
		Tier:    "Platinum",
		Route:   "NYC-LON",
		SpendingProfile: models.SpendingProfile{
			Pricesensitivity: "low",
		},
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		for _, item := range items {
			item.GetDynamicPrice(testCustomer, testCustomer.Route)
		}
	}
}

// Example test demonstrating usage
func ExampleAncillaryService() {
	// Create a customer
	customer := models.Customer{
		ID:      "example-customer",
		Segment: "Business Elite",
		Tier:    "Platinum",
		Route:   "NYC-LON",
		SpendingProfile: models.SpendingProfile{
			AverageAncillarySpend: 120.0,
			Pricesensitivity:      "low",
		},
		LastUpdate: time.Now(),
	}
	
	// Create bundling engine
	engine := engines.NewBundlingEngine()
	
	// Generate recommendations
	recommendations, err := engine.GenerateRecommendations(customer)
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("Generated %d bundle recommendations and %d item recommendations\n", 
		len(recommendations.RecommendedBundles), len(recommendations.IndividualItems))
	
	// Output: Generated 4 bundle recommendations and 8 item recommendations
} 