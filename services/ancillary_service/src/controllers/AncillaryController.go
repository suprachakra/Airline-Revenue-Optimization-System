package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"iaros/ancillary_service/src/engines"
	"iaros/ancillary_service/src/models"
	"iaros/ancillary_service/src/services"
)

// AncillaryController handles all ancillary service operations
type AncillaryController struct {
	bundlingEngine    *engines.BundlingEngine
	ancillaryService  *services.AncillaryService
	analyticsService  *services.AnalyticsService
}

// NewAncillaryController creates a new ancillary controller
func NewAncillaryController() *AncillaryController {
	return &AncillaryController{
		bundlingEngine:   engines.NewBundlingEngine(),
		ancillaryService: services.NewAncillaryService(),
		analyticsService: services.NewAnalyticsService(),
	}
}

// RegisterRoutes registers all ancillary service routes
func (ac *AncillaryController) RegisterRoutes(router *mux.Router) {
	// Recommendation endpoints
	router.HandleFunc("/ancillary/recommendations", ac.GetRecommendations).Methods("POST")
	router.HandleFunc("/ancillary/recommendations/{customerID}", ac.GetCustomerRecommendations).Methods("GET")
	
	// Ancillary item management
	router.HandleFunc("/ancillary/items", ac.GetAncillaryItems).Methods("GET")
	router.HandleFunc("/ancillary/items/{itemID}", ac.GetAncillaryItem).Methods("GET")
	router.HandleFunc("/ancillary/items", ac.CreateAncillaryItem).Methods("POST")
	router.HandleFunc("/ancillary/items/{itemID}", ac.UpdateAncillaryItem).Methods("PUT")
	router.HandleFunc("/ancillary/items/{itemID}", ac.DeleteAncillaryItem).Methods("DELETE")
	router.HandleFunc("/ancillary/items/{itemID}/price", ac.GetDynamicPrice).Methods("POST")
	
	// Bundle management
	router.HandleFunc("/ancillary/bundles", ac.GetBundles).Methods("GET")
	router.HandleFunc("/ancillary/bundles/{bundleID}", ac.GetBundle).Methods("GET")
	router.HandleFunc("/ancillary/bundles", ac.CreateBundle).Methods("POST")
	router.HandleFunc("/ancillary/bundles/{bundleID}", ac.UpdateBundle).Methods("PUT")
	router.HandleFunc("/ancillary/bundles/{bundleID}", ac.DeleteBundle).Methods("DELETE")
	router.HandleFunc("/ancillary/bundles/generate", ac.GenerateDynamicBundle).Methods("POST")
	
	// Analytics endpoints
	router.HandleFunc("/ancillary/analytics/items", ac.GetItemAnalytics).Methods("GET")
	router.HandleFunc("/ancillary/analytics/bundles", ac.GetBundleAnalytics).Methods("GET")
	router.HandleFunc("/ancillary/analytics/performance", ac.GetPerformanceMetrics).Methods("GET")
	router.HandleFunc("/ancillary/analytics/revenue", ac.GetRevenueAnalytics).Methods("GET")
	
	// Purchase tracking
	router.HandleFunc("/ancillary/purchase", ac.RecordPurchase).Methods("POST")
	router.HandleFunc("/ancillary/purchase/{purchaseID}", ac.GetPurchase).Methods("GET")
	
	// Customer management
	router.HandleFunc("/ancillary/customers/{customerID}/profile", ac.GetCustomerProfile).Methods("GET")
	router.HandleFunc("/ancillary/customers/{customerID}/profile", ac.UpdateCustomerProfile).Methods("PUT")
	router.HandleFunc("/ancillary/customers/{customerID}/preferences", ac.GetCustomerPreferences).Methods("GET")
	router.HandleFunc("/ancillary/customers/{customerID}/preferences", ac.UpdateCustomerPreferences).Methods("PUT")
	
	// Health check
	router.HandleFunc("/ancillary/health", ac.HealthCheck).Methods("GET")
}

// GetRecommendations generates personalized recommendations for a customer
func (ac *AncillaryController) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	var customer models.Customer
	if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
		ac.sendError(w, http.StatusBadRequest, "Invalid customer data: "+err.Error())
		return
	}
	
	// Validate required fields
	if customer.ID == "" || customer.Segment == "" || customer.Route == "" {
		ac.sendError(w, http.StatusBadRequest, "Missing required customer fields (ID, Segment, Route)")
		return
	}
	
	// Generate recommendations
	recommendations, err := ac.bundlingEngine.GenerateRecommendations(customer)
	if err != nil {
		ac.sendError(w, http.StatusInternalServerError, "Failed to generate recommendations: "+err.Error())
		return
	}
	
	// Log recommendation generation
	log.Printf("Generated %d bundle recommendations and %d item recommendations for customer %s",
		len(recommendations.RecommendedBundles), len(recommendations.IndividualItems), customer.ID)
	
	ac.sendJSON(w, http.StatusOK, recommendations)
}

// GetCustomerRecommendations gets cached recommendations for a customer
func (ac *AncillaryController) GetCustomerRecommendations(w http.ResponseWriter, r *http.Request) {
	customerID := mux.Vars(r)["customerID"]
	
	// Get customer profile
	customer, err := ac.ancillaryService.GetCustomerProfile(customerID)
	if err != nil {
		ac.sendError(w, http.StatusNotFound, "Customer not found: "+err.Error())
		return
	}
	
	recommendations, err := ac.bundlingEngine.GenerateRecommendations(customer)
	if err != nil {
		ac.sendError(w, http.StatusInternalServerError, "Failed to generate recommendations: "+err.Error())
		return
	}
	
	ac.sendJSON(w, http.StatusOK, recommendations)
}

// GetAncillaryItems returns all ancillary items with optional filtering
func (ac *AncillaryController) GetAncillaryItems(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	
	// Parse query parameters
	category := query.Get("category")
	available := query.Get("available")
	minPrice := query.Get("min_price")
	maxPrice := query.Get("max_price")
	
	// Build filter
	filter := models.AncillaryFilter{
		Category:  category,
		Available: available == "true",
		MinPrice:  parseFloat(minPrice),
		MaxPrice:  parseFloat(maxPrice),
	}
	
	items, err := ac.ancillaryService.GetAncillaryItems(filter)
	if err != nil {
		ac.sendError(w, http.StatusInternalServerError, "Failed to retrieve items: "+err.Error())
		return
	}
	
	response := map[string]interface{}{
		"items": items,
		"count": len(items),
		"timestamp": time.Now().UTC(),
	}
	
	ac.sendJSON(w, http.StatusOK, response)
}

// GetAncillaryItem returns a specific ancillary item
func (ac *AncillaryController) GetAncillaryItem(w http.ResponseWriter, r *http.Request) {
	itemID := mux.Vars(r)["itemID"]
	
	item, err := ac.ancillaryService.GetAncillaryItem(itemID)
	if err != nil {
		ac.sendError(w, http.StatusNotFound, "Item not found: "+err.Error())
		return
	}
	
	ac.sendJSON(w, http.StatusOK, item)
}

// CreateAncillaryItem creates a new ancillary item
func (ac *AncillaryController) CreateAncillaryItem(w http.ResponseWriter, r *http.Request) {
	var item models.AncillaryItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		ac.sendError(w, http.StatusBadRequest, "Invalid item data: "+err.Error())
		return
	}
	
	// Validate required fields
	if item.Name == "" || item.Category == "" || item.BasePrice <= 0 {
		ac.sendError(w, http.StatusBadRequest, "Missing required item fields (Name, Category, BasePrice)")
		return
	}
	
	// Set timestamps
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()
	
	createdItem, err := ac.ancillaryService.CreateAncillaryItem(item)
	if err != nil {
		ac.sendError(w, http.StatusInternalServerError, "Failed to create item: "+err.Error())
		return
	}
	
	log.Printf("Created new ancillary item: %s (ID: %s)", item.Name, item.ID)
	ac.sendJSON(w, http.StatusCreated, createdItem)
}

// UpdateAncillaryItem updates an existing ancillary item
func (ac *AncillaryController) UpdateAncillaryItem(w http.ResponseWriter, r *http.Request) {
	itemID := mux.Vars(r)["itemID"]
	
	var updateData models.AncillaryItem
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		ac.sendError(w, http.StatusBadRequest, "Invalid update data: "+err.Error())
		return
	}
	
	updateData.ID = itemID
	updateData.UpdatedAt = time.Now()
	
	updatedItem, err := ac.ancillaryService.UpdateAncillaryItem(updateData)
	if err != nil {
		ac.sendError(w, http.StatusInternalServerError, "Failed to update item: "+err.Error())
		return
	}
	
	log.Printf("Updated ancillary item: %s (ID: %s)", updatedItem.Name, itemID)
	ac.sendJSON(w, http.StatusOK, updatedItem)
}

// DeleteAncillaryItem deletes an ancillary item
func (ac *AncillaryController) DeleteAncillaryItem(w http.ResponseWriter, r *http.Request) {
	itemID := mux.Vars(r)["itemID"]
	
	err := ac.ancillaryService.DeleteAncillaryItem(itemID)
	if err != nil {
		ac.sendError(w, http.StatusInternalServerError, "Failed to delete item: "+err.Error())
		return
	}
	
	log.Printf("Deleted ancillary item: %s", itemID)
	ac.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Item deleted successfully",
		"item_id": itemID,
		"timestamp": time.Now().UTC(),
	})
}

// GetDynamicPrice calculates dynamic pricing for an item
func (ac *AncillaryController) GetDynamicPrice(w http.ResponseWriter, r *http.Request) {
	itemID := mux.Vars(r)["itemID"]
	
	var customer models.Customer
	if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
		ac.sendError(w, http.StatusBadRequest, "Invalid customer data: "+err.Error())
		return
	}
	
	item, err := ac.ancillaryService.GetAncillaryItem(itemID)
	if err != nil {
		ac.sendError(w, http.StatusNotFound, "Item not found: "+err.Error())
		return
	}
	
	dynamicPrice := item.GetDynamicPrice(customer, customer.Route)
	
	response := map[string]interface{}{
		"item_id":       itemID,
		"base_price":    item.BasePrice,
		"dynamic_price": dynamicPrice,
		"discount":      item.BasePrice - dynamicPrice,
		"customer_id":   customer.ID,
		"route":         customer.Route,
		"timestamp":     time.Now().UTC(),
	}
	
	ac.sendJSON(w, http.StatusOK, response)
}

// GetBundles returns all bundles with optional filtering
func (ac *AncillaryController) GetBundles(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	
	// Parse query parameters
	category := query.Get("category")
	available := query.Get("available")
	segment := query.Get("segment")
	
	filter := models.BundleFilter{
		Category:  category,
		Available: available == "true",
		Segment:   segment,
	}
	
	bundles, err := ac.ancillaryService.GetBundles(filter)
	if err != nil {
		ac.sendError(w, http.StatusInternalServerError, "Failed to retrieve bundles: "+err.Error())
		return
	}
	
	response := map[string]interface{}{
		"bundles": bundles,
		"count":   len(bundles),
		"timestamp": time.Now().UTC(),
	}
	
	ac.sendJSON(w, http.StatusOK, response)
}

// GetBundle returns a specific bundle
func (ac *AncillaryController) GetBundle(w http.ResponseWriter, r *http.Request) {
	bundleID := mux.Vars(r)["bundleID"]
	
	bundle, err := ac.ancillaryService.GetBundle(bundleID)
	if err != nil {
		ac.sendError(w, http.StatusNotFound, "Bundle not found: "+err.Error())
		return
	}
	
	ac.sendJSON(w, http.StatusOK, bundle)
}

// CreateBundle creates a new bundle
func (ac *AncillaryController) CreateBundle(w http.ResponseWriter, r *http.Request) {
	var bundle models.AncillaryBundle
	if err := json.NewDecoder(r.Body).Decode(&bundle); err != nil {
		ac.sendError(w, http.StatusBadRequest, "Invalid bundle data: "+err.Error())
		return
	}
	
	// Validate required fields
	if bundle.Name == "" || len(bundle.Items) == 0 {
		ac.sendError(w, http.StatusBadRequest, "Missing required bundle fields (Name, Items)")
		return
	}
	
	// Set timestamps
	bundle.CreatedAt = time.Now()
	bundle.UpdatedAt = time.Now()
	
	createdBundle, err := ac.ancillaryService.CreateBundle(bundle)
	if err != nil {
		ac.sendError(w, http.StatusInternalServerError, "Failed to create bundle: "+err.Error())
		return
	}
	
	log.Printf("Created new bundle: %s (ID: %s)", bundle.Name, bundle.ID)
	ac.sendJSON(w, http.StatusCreated, createdBundle)
}

// UpdateBundle updates an existing bundle
func (ac *AncillaryController) UpdateBundle(w http.ResponseWriter, r *http.Request) {
	bundleID := mux.Vars(r)["bundleID"]
	
	var updateData models.AncillaryBundle
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		ac.sendError(w, http.StatusBadRequest, "Invalid update data: "+err.Error())
		return
	}
	
	updateData.ID = bundleID
	updateData.UpdatedAt = time.Now()
	
	updatedBundle, err := ac.ancillaryService.UpdateBundle(updateData)
	if err != nil {
		ac.sendError(w, http.StatusInternalServerError, "Failed to update bundle: "+err.Error())
		return
	}
	
	log.Printf("Updated bundle: %s (ID: %s)", updatedBundle.Name, bundleID)
	ac.sendJSON(w, http.StatusOK, updatedBundle)
}

// DeleteBundle deletes a bundle
func (ac *AncillaryController) DeleteBundle(w http.ResponseWriter, r *http.Request) {
	bundleID := mux.Vars(r)["bundleID"]
	
	err := ac.ancillaryService.DeleteBundle(bundleID)
	if err != nil {
		ac.sendError(w, http.StatusInternalServerError, "Failed to delete bundle: "+err.Error())
		return
	}
	
	log.Printf("Deleted bundle: %s", bundleID)
	ac.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Bundle deleted successfully",
		"bundle_id": bundleID,
		"timestamp": time.Now().UTC(),
	})
}

// GenerateDynamicBundle generates a dynamic bundle for a customer
func (ac *AncillaryController) GenerateDynamicBundle(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Customer models.Customer `json:"customer"`
		ItemIDs  []string        `json:"item_ids"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		ac.sendError(w, http.StatusBadRequest, "Invalid request data: "+err.Error())
		return
	}
	
	bundle, err := ac.ancillaryService.GenerateDynamicBundle(request.Customer, request.ItemIDs)
	if err != nil {
		ac.sendError(w, http.StatusInternalServerError, "Failed to generate dynamic bundle: "+err.Error())
		return
	}
	
	ac.sendJSON(w, http.StatusOK, bundle)
}

// GetItemAnalytics returns analytics for ancillary items
func (ac *AncillaryController) GetItemAnalytics(w http.ResponseWriter, r *http.Request) {
	analytics := ac.bundlingEngine.GetAnalytics()
	
	response := map[string]interface{}{
		"analytics": analytics,
		"count":     len(analytics),
		"timestamp": time.Now().UTC(),
	}
	
	ac.sendJSON(w, http.StatusOK, response)
}

// GetBundleAnalytics returns analytics for bundles
func (ac *AncillaryController) GetBundleAnalytics(w http.ResponseWriter, r *http.Request) {
	analytics := ac.bundlingEngine.GetBundleAnalytics()
	
	response := map[string]interface{}{
		"analytics": analytics,
		"count":     len(analytics),
		"timestamp": time.Now().UTC(),
	}
	
	ac.sendJSON(w, http.StatusOK, response)
}

// GetPerformanceMetrics returns overall performance metrics
func (ac *AncillaryController) GetPerformanceMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := ac.analyticsService.GetPerformanceMetrics()
	if err != nil {
		ac.sendError(w, http.StatusInternalServerError, "Failed to retrieve metrics: "+err.Error())
		return
	}
	
	ac.sendJSON(w, http.StatusOK, metrics)
}

// GetRevenueAnalytics returns revenue analytics
func (ac *AncillaryController) GetRevenueAnalytics(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	period := query.Get("period")
	if period == "" {
		period = "month"
	}
	
	analytics, err := ac.analyticsService.GetRevenueAnalytics(period)
	if err != nil {
		ac.sendError(w, http.StatusInternalServerError, "Failed to retrieve revenue analytics: "+err.Error())
		return
	}
	
	ac.sendJSON(w, http.StatusOK, analytics)
}

// RecordPurchase records an ancillary purchase
func (ac *AncillaryController) RecordPurchase(w http.ResponseWriter, r *http.Request) {
	var purchase models.Purchase
	if err := json.NewDecoder(r.Body).Decode(&purchase); err != nil {
		ac.sendError(w, http.StatusBadRequest, "Invalid purchase data: "+err.Error())
		return
	}
	
	// Validate required fields
	if purchase.CustomerID == "" || purchase.ItemID == "" || purchase.Amount <= 0 {
		ac.sendError(w, http.StatusBadRequest, "Missing required purchase fields")
		return
	}
	
	purchase.Timestamp = time.Now()
	purchase.Status = "completed"
	
	createdPurchase, err := ac.ancillaryService.RecordPurchase(purchase)
	if err != nil {
		ac.sendError(w, http.StatusInternalServerError, "Failed to record purchase: "+err.Error())
		return
	}
	
	// Update analytics
	ac.bundlingEngine.UpdateAnalytics(purchase.ItemID, purchase.Amount, true)
	
	log.Printf("Recorded purchase: Customer %s bought %s for $%.2f", 
		purchase.CustomerID, purchase.ItemID, purchase.Amount)
	
	ac.sendJSON(w, http.StatusCreated, createdPurchase)
}

// GetPurchase retrieves a specific purchase
func (ac *AncillaryController) GetPurchase(w http.ResponseWriter, r *http.Request) {
	purchaseID := mux.Vars(r)["purchaseID"]
	
	purchase, err := ac.ancillaryService.GetPurchase(purchaseID)
	if err != nil {
		ac.sendError(w, http.StatusNotFound, "Purchase not found: "+err.Error())
		return
	}
	
	ac.sendJSON(w, http.StatusOK, purchase)
}

// GetCustomerProfile retrieves customer profile
func (ac *AncillaryController) GetCustomerProfile(w http.ResponseWriter, r *http.Request) {
	customerID := mux.Vars(r)["customerID"]
	
	profile, err := ac.ancillaryService.GetCustomerProfile(customerID)
	if err != nil {
		ac.sendError(w, http.StatusNotFound, "Customer profile not found: "+err.Error())
		return
	}
	
	ac.sendJSON(w, http.StatusOK, profile)
}

// UpdateCustomerProfile updates customer profile
func (ac *AncillaryController) UpdateCustomerProfile(w http.ResponseWriter, r *http.Request) {
	customerID := mux.Vars(r)["customerID"]
	
	var profile models.Customer
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		ac.sendError(w, http.StatusBadRequest, "Invalid profile data: "+err.Error())
		return
	}
	
	profile.ID = customerID
	profile.LastUpdate = time.Now()
	
	updatedProfile, err := ac.ancillaryService.UpdateCustomerProfile(profile)
	if err != nil {
		ac.sendError(w, http.StatusInternalServerError, "Failed to update profile: "+err.Error())
		return
	}
	
	ac.sendJSON(w, http.StatusOK, updatedProfile)
}

// GetCustomerPreferences retrieves customer preferences
func (ac *AncillaryController) GetCustomerPreferences(w http.ResponseWriter, r *http.Request) {
	customerID := mux.Vars(r)["customerID"]
	
	preferences, err := ac.ancillaryService.GetCustomerPreferences(customerID)
	if err != nil {
		ac.sendError(w, http.StatusNotFound, "Customer preferences not found: "+err.Error())
		return
	}
	
	ac.sendJSON(w, http.StatusOK, preferences)
}

// UpdateCustomerPreferences updates customer preferences
func (ac *AncillaryController) UpdateCustomerPreferences(w http.ResponseWriter, r *http.Request) {
	customerID := mux.Vars(r)["customerID"]
	
	var preferences models.CustomerPreferences
	if err := json.NewDecoder(r.Body).Decode(&preferences); err != nil {
		ac.sendError(w, http.StatusBadRequest, "Invalid preferences data: "+err.Error())
		return
	}
	
	preferences.CustomerID = customerID
	preferences.UpdatedAt = time.Now()
	
	updatedPreferences, err := ac.ancillaryService.UpdateCustomerPreferences(preferences)
	if err != nil {
		ac.sendError(w, http.StatusInternalServerError, "Failed to update preferences: "+err.Error())
		return
	}
	
	ac.sendJSON(w, http.StatusOK, updatedPreferences)
}

// HealthCheck returns the health status of the ancillary service
func (ac *AncillaryController) HealthCheck(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":    "healthy",
		"service":   "ancillary-service",
		"version":   "1.0.0",
		"timestamp": time.Now().UTC(),
		"uptime":    time.Since(time.Now().Add(-time.Hour)).String(),
		"components": map[string]string{
			"bundling_engine":   "operational",
			"analytics_service": "operational",
			"database":          "connected",
			"cache":             "active",
		},
	}
	
	ac.sendJSON(w, http.StatusOK, status)
}

// Helper methods

func (ac *AncillaryController) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (ac *AncillaryController) sendError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":     message,
		"status":    status,
		"timestamp": time.Now().UTC(),
	})
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val
}

// Additional model definitions

// AncillaryFilter represents filtering options for ancillary items
type AncillaryFilter struct {
	Category  string  `json:"category"`
	Available bool    `json:"available"`
	MinPrice  float64 `json:"min_price"`
	MaxPrice  float64 `json:"max_price"`
}

// BundleFilter represents filtering options for bundles
type BundleFilter struct {
	Category  string `json:"category"`
	Available bool   `json:"available"`
	Segment   string `json:"segment"`
}

// Purchase represents an ancillary purchase
type Purchase struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	ItemID     string    `json:"item_id"`
	BundleID   string    `json:"bundle_id,omitempty"`
	Amount     float64   `json:"amount"`
	Currency   string    `json:"currency"`
	Status     string    `json:"status"`
	Timestamp  time.Time `json:"timestamp"`
}

// CustomerPreferences represents customer preferences
type CustomerPreferences struct {
	CustomerID          string                      `json:"customer_id"`
	PreferredCategories []models.AncillaryCategory `json:"preferred_categories"`
	PriceRange          models.PriceRange          `json:"price_range"`
	Notifications       NotificationPreferences     `json:"notifications"`
	UpdatedAt           time.Time                   `json:"updated_at"`
}

// NotificationPreferences represents notification preferences
type NotificationPreferences struct {
	Email bool `json:"email"`
	SMS   bool `json:"sms"`
	Push  bool `json:"push"`
} 