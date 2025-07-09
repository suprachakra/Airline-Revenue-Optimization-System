package services

import (
	"errors"
	"fmt"
	"log"
	"time"
	"math"
	"sort"
	"sync"

	"iaros/ancillary_service/src/models"
)

// AncillaryService provides business logic for ancillary operations
type AncillaryService struct {
	ancillaryItems map[string]models.AncillaryItem
	bundles        map[string]models.AncillaryBundle
	customers      map[string]models.Customer
	purchases      map[string]models.Purchase
	preferences    map[string]models.CustomerPreferences
	mutex          sync.RWMutex
}

// NewAncillaryService creates a new ancillary service instance
func NewAncillaryService() *AncillaryService {
	service := &AncillaryService{
		ancillaryItems: make(map[string]models.AncillaryItem),
		bundles:        make(map[string]models.AncillaryBundle),
		customers:      make(map[string]models.Customer),
		purchases:      make(map[string]models.Purchase),
		preferences:    make(map[string]models.CustomerPreferences),
	}
	
	// Initialize with default data
	service.initializeData()
	
	return service
}

// initializeData populates the service with default ancillary items and bundles
func (as *AncillaryService) initializeData() {
	// Load default ancillary items
	for _, item := range models.GetDefaultAncillaryItems() {
		as.ancillaryItems[item.ID] = item
	}
	
	// Load default bundles
	for _, bundle := range models.GetDefaultBundles() {
		as.bundles[bundle.ID] = bundle
	}
	
	// Initialize sample customers
	as.initializeSampleCustomers()
	
	log.Printf("Initialized ancillary service with %d items and %d bundles", 
		len(as.ancillaryItems), len(as.bundles))
}

// initializeSampleCustomers creates sample customer profiles
func (as *AncillaryService) initializeSampleCustomers() {
	sampleCustomers := []models.Customer{
		{
			ID:      "cust-001",
			Segment: "Business Elite",
			Tier:    "Platinum",
			PreviousPurchases: []string{"wifi-premium", "seat-premium"},
			PreferredCategories: []models.AncillaryCategory{
				models.CategoryComfort,
				models.CategoryConnectivity,
				models.CategoryGroundService,
			},
			SpendingProfile: models.SpendingProfile{
				AverageAncillarySpend: 85.0,
				MaxAncillarySpend:     200.0,
				Pricesensitivity:      "low",
				PreferredPriceRange:   models.PriceRange{Min: 20.0, Max: 150.0},
			},
			TravelFrequency: "frequent",
			Route:           "NYC-LON",
			BookingClass:    "Business",
			TripType:        "business",
			CompanionCount:  0,
			Age:             &[]int{42}[0],
			LastUpdate:      time.Now(),
		},
		{
			ID:      "cust-002",
			Segment: "Family Traveler",
			Tier:    "Gold",
			PreviousPurchases: []string{"baggage-20kg", "meal-premium"},
			PreferredCategories: []models.AncillaryCategory{
				models.CategoryBaggage,
				models.CategoryDining,
				models.CategoryEntertainment,
			},
			SpendingProfile: models.SpendingProfile{
				AverageAncillarySpend: 65.0,
				MaxAncillarySpend:     120.0,
				Pricesensitivity:      "medium",
				PreferredPriceRange:   models.PriceRange{Min: 10.0, Max: 80.0},
			},
			TravelFrequency: "occasional",
			Route:           "LAX-TOK",
			BookingClass:    "Economy",
			TripType:        "leisure",
			CompanionCount:  3,
			Age:             &[]int{35}[0],
			LastUpdate:      time.Now(),
		},
		{
			ID:      "cust-003",
			Segment: "Budget Conscious",
			Tier:    "Silver",
			PreviousPurchases: []string{"priority-boarding"},
			PreferredCategories: []models.AncillaryCategory{
				models.CategoryConvenience,
				models.CategoryBaggage,
			},
			SpendingProfile: models.SpendingProfile{
				AverageAncillarySpend: 25.0,
				MaxAncillarySpend:     50.0,
				Pricesensitivity:      "high",
				PreferredPriceRange:   models.PriceRange{Min: 5.0, Max: 40.0},
			},
			TravelFrequency: "infrequent",
			Route:           "SFO-FRA",
			BookingClass:    "Economy",
			TripType:        "leisure",
			CompanionCount:  1,
			Age:             &[]int{28}[0],
			LastUpdate:      time.Now(),
		},
	}
	
	for _, customer := range sampleCustomers {
		as.customers[customer.ID] = customer
	}
}

// Ancillary Item Management

// GetAncillaryItems retrieves ancillary items with optional filtering
func (as *AncillaryService) GetAncillaryItems(filter models.AncillaryFilter) ([]models.AncillaryItem, error) {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	
	var items []models.AncillaryItem
	
	for _, item := range as.ancillaryItems {
		if as.matchesFilter(item, filter) {
			items = append(items, item)
		}
	}
	
	// Sort by popularity score (highest first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].PopularityScore > items[j].PopularityScore
	})
	
	return items, nil
}

// GetAncillaryItem retrieves a specific ancillary item
func (as *AncillaryService) GetAncillaryItem(itemID string) (models.AncillaryItem, error) {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	
	item, exists := as.ancillaryItems[itemID]
	if !exists {
		return models.AncillaryItem{}, errors.New("item not found")
	}
	
	return item, nil
}

// CreateAncillaryItem creates a new ancillary item
func (as *AncillaryService) CreateAncillaryItem(item models.AncillaryItem) (models.AncillaryItem, error) {
	as.mutex.Lock()
	defer as.mutex.Unlock()
	
	// Generate ID if not provided
	if item.ID == "" {
		item.ID = as.generateItemID(item.Name)
	}
	
	// Check if item already exists
	if _, exists := as.ancillaryItems[item.ID]; exists {
		return models.AncillaryItem{}, errors.New("item already exists")
	}
	
	// Set default values
	if item.Currency == "" {
		item.Currency = "USD"
	}
	if item.MaxQuantity == 0 {
		item.MaxQuantity = 1
	}
	if item.SeasonalMultiplier == 0 {
		item.SeasonalMultiplier = 1.0
	}
	
	// Initialize collections
	if item.RouteMultipliers == nil {
		item.RouteMultipliers = make(map[string]float64)
	}
	if item.CustomerSegmentPrice == nil {
		item.CustomerSegmentPrice = make(map[string]float64)
	}
	if item.BundleCompatibility == nil {
		item.BundleCompatibility = []string{}
	}
	
	as.ancillaryItems[item.ID] = item
	
	log.Printf("Created ancillary item: %s (ID: %s)", item.Name, item.ID)
	return item, nil
}

// UpdateAncillaryItem updates an existing ancillary item
func (as *AncillaryService) UpdateAncillaryItem(updateData models.AncillaryItem) (models.AncillaryItem, error) {
	as.mutex.Lock()
	defer as.mutex.Unlock()
	
	existingItem, exists := as.ancillaryItems[updateData.ID]
	if !exists {
		return models.AncillaryItem{}, errors.New("item not found")
	}
	
	// Preserve creation time
	updateData.CreatedAt = existingItem.CreatedAt
	
	as.ancillaryItems[updateData.ID] = updateData
	
	log.Printf("Updated ancillary item: %s (ID: %s)", updateData.Name, updateData.ID)
	return updateData, nil
}

// DeleteAncillaryItem deletes an ancillary item
func (as *AncillaryService) DeleteAncillaryItem(itemID string) error {
	as.mutex.Lock()
	defer as.mutex.Unlock()
	
	if _, exists := as.ancillaryItems[itemID]; !exists {
		return errors.New("item not found")
	}
	
	// Check if item is used in any bundles
	for _, bundle := range as.bundles {
		for _, bundleItemID := range bundle.Items {
			if bundleItemID == itemID {
				return errors.New("cannot delete item - it is used in bundle: " + bundle.Name)
			}
		}
	}
	
	delete(as.ancillaryItems, itemID)
	
	log.Printf("Deleted ancillary item: %s", itemID)
	return nil
}

// Bundle Management

// GetBundles retrieves bundles with optional filtering
func (as *AncillaryService) GetBundles(filter models.BundleFilter) ([]models.AncillaryBundle, error) {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	
	var bundles []models.AncillaryBundle
	
	for _, bundle := range as.bundles {
		if as.matchesBundleFilter(bundle, filter) {
			bundles = append(bundles, bundle)
		}
	}
	
	// Sort by popularity score (highest first)
	sort.Slice(bundles, func(i, j int) bool {
		return bundles[i].PopularityScore > bundles[j].PopularityScore
	})
	
	return bundles, nil
}

// GetBundle retrieves a specific bundle
func (as *AncillaryService) GetBundle(bundleID string) (models.AncillaryBundle, error) {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	
	bundle, exists := as.bundles[bundleID]
	if !exists {
		return models.AncillaryBundle{}, errors.New("bundle not found")
	}
	
	return bundle, nil
}

// CreateBundle creates a new bundle
func (as *AncillaryService) CreateBundle(bundle models.AncillaryBundle) (models.AncillaryBundle, error) {
	as.mutex.Lock()
	defer as.mutex.Unlock()
	
	// Generate ID if not provided
	if bundle.ID == "" {
		bundle.ID = as.generateBundleID(bundle.Name)
	}
	
	// Check if bundle already exists
	if _, exists := as.bundles[bundle.ID]; exists {
		return models.AncillaryBundle{}, errors.New("bundle already exists")
	}
	
	// Validate bundle items exist
	for _, itemID := range bundle.Items {
		if _, exists := as.ancillaryItems[itemID]; !exists {
			return models.AncillaryBundle{}, errors.New("invalid item ID: " + itemID)
		}
	}
	
	// Calculate prices if not provided
	if bundle.OriginalPrice == 0 {
		bundle.OriginalPrice = as.calculateBundleOriginalPrice(bundle.Items)
	}
	
	if bundle.BundlePrice == 0 && bundle.DiscountPercentage > 0 {
		bundle.BundlePrice = bundle.OriginalPrice * (1 - bundle.DiscountPercentage/100)
	}
	
	// Set default values
	if bundle.Currency == "" {
		bundle.Currency = "USD"
	}
	if bundle.MaxItemsPerBundle == 0 {
		bundle.MaxItemsPerBundle = 6
	}
	
	as.bundles[bundle.ID] = bundle
	
	log.Printf("Created bundle: %s (ID: %s)", bundle.Name, bundle.ID)
	return bundle, nil
}

// UpdateBundle updates an existing bundle
func (as *AncillaryService) UpdateBundle(updateData models.AncillaryBundle) (models.AncillaryBundle, error) {
	as.mutex.Lock()
	defer as.mutex.Unlock()
	
	existingBundle, exists := as.bundles[updateData.ID]
	if !exists {
		return models.AncillaryBundle{}, errors.New("bundle not found")
	}
	
	// Validate bundle items exist
	for _, itemID := range updateData.Items {
		if _, exists := as.ancillaryItems[itemID]; !exists {
			return models.AncillaryBundle{}, errors.New("invalid item ID: " + itemID)
		}
	}
	
	// Preserve creation time
	updateData.CreatedAt = existingBundle.CreatedAt
	
	as.bundles[updateData.ID] = updateData
	
	log.Printf("Updated bundle: %s (ID: %s)", updateData.Name, updateData.ID)
	return updateData, nil
}

// DeleteBundle deletes a bundle
func (as *AncillaryService) DeleteBundle(bundleID string) error {
	as.mutex.Lock()
	defer as.mutex.Unlock()
	
	if _, exists := as.bundles[bundleID]; !exists {
		return errors.New("bundle not found")
	}
	
	delete(as.bundles, bundleID)
	
	log.Printf("Deleted bundle: %s", bundleID)
	return nil
}

// GenerateDynamicBundle generates a dynamic bundle for a customer
func (as *AncillaryService) GenerateDynamicBundle(customer models.Customer, itemIDs []string) (models.AncillaryBundle, error) {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	
	if len(itemIDs) < 2 {
		return models.AncillaryBundle{}, errors.New("bundle must contain at least 2 items")
	}
	
	// Validate all items exist and are available for customer
	var items []models.AncillaryItem
	for _, itemID := range itemIDs {
		item, exists := as.ancillaryItems[itemID]
		if !exists {
			return models.AncillaryBundle{}, errors.New("invalid item ID: " + itemID)
		}
		
		if !item.IsAvailableForCustomer(customer) {
			return models.AncillaryBundle{}, errors.New("item not available for customer: " + itemID)
		}
		
		items = append(items, item)
	}
	
	// Calculate pricing
	originalPrice := 0.0
	for _, item := range items {
		originalPrice += item.GetDynamicPrice(customer, customer.Route)
	}
	
	// Calculate intelligent discount
	discountPercentage := as.calculateDynamicDiscount(len(items), customer)
	bundlePrice := originalPrice * (1 - discountPercentage/100)
	
	// Generate bundle
	bundle := models.AncillaryBundle{
		ID:                 fmt.Sprintf("dynamic-%s-%d", customer.ID, time.Now().Unix()),
		Name:               as.generateDynamicBundleName(items, customer),
		Description:        fmt.Sprintf("Personalized bundle for %s", customer.Segment),
		Items:              itemIDs,
		OriginalPrice:      originalPrice,
		BundlePrice:        bundlePrice,
		DiscountPercentage: discountPercentage,
		Currency:           "USD",
		Available:          true,
		PopularityScore:    0.7,
		Category:           "dynamic",
		TargetSegments:     []string{customer.Segment},
		RouteApplicability: []string{customer.Route},
		ValidFrom:          time.Now(),
		ValidTo:            time.Now().AddDate(0, 0, 7), // Valid for 7 days
		MaxItemsPerBundle:  6,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	
	return bundle, nil
}

// Customer Management

// GetCustomerProfile retrieves a customer profile
func (as *AncillaryService) GetCustomerProfile(customerID string) (models.Customer, error) {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	
	customer, exists := as.customers[customerID]
	if !exists {
		return models.Customer{}, errors.New("customer not found")
	}
	
	return customer, nil
}

// UpdateCustomerProfile updates a customer profile
func (as *AncillaryService) UpdateCustomerProfile(profile models.Customer) (models.Customer, error) {
	as.mutex.Lock()
	defer as.mutex.Unlock()
	
	as.customers[profile.ID] = profile
	
	log.Printf("Updated customer profile: %s", profile.ID)
	return profile, nil
}

// GetCustomerPreferences retrieves customer preferences
func (as *AncillaryService) GetCustomerPreferences(customerID string) (models.CustomerPreferences, error) {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	
	preferences, exists := as.preferences[customerID]
	if !exists {
		// Return default preferences
		return models.CustomerPreferences{
			CustomerID: customerID,
			PreferredCategories: []models.AncillaryCategory{
				models.CategoryComfort,
				models.CategoryConvenience,
			},
			PriceRange: models.PriceRange{Min: 10.0, Max: 100.0},
			Notifications: models.NotificationPreferences{
				Email: true,
				SMS:   false,
				Push:  true,
			},
			UpdatedAt: time.Now(),
		}, nil
	}
	
	return preferences, nil
}

// UpdateCustomerPreferences updates customer preferences
func (as *AncillaryService) UpdateCustomerPreferences(preferences models.CustomerPreferences) (models.CustomerPreferences, error) {
	as.mutex.Lock()
	defer as.mutex.Unlock()
	
	as.preferences[preferences.CustomerID] = preferences
	
	log.Printf("Updated customer preferences: %s", preferences.CustomerID)
	return preferences, nil
}

// Purchase Management

// RecordPurchase records a purchase
func (as *AncillaryService) RecordPurchase(purchase models.Purchase) (models.Purchase, error) {
	as.mutex.Lock()
	defer as.mutex.Unlock()
	
	// Generate ID if not provided
	if purchase.ID == "" {
		purchase.ID = fmt.Sprintf("purchase-%d", time.Now().UnixNano())
	}
	
	// Set default currency
	if purchase.Currency == "" {
		purchase.Currency = "USD"
	}
	
	as.purchases[purchase.ID] = purchase
	
	// Update customer purchase history
	if customer, exists := as.customers[purchase.CustomerID]; exists {
		customer.PreviousPurchases = append(customer.PreviousPurchases, purchase.ItemID)
		customer.LastUpdate = time.Now()
		as.customers[purchase.CustomerID] = customer
	}
	
	log.Printf("Recorded purchase: %s (Customer: %s, Amount: $%.2f)", 
		purchase.ID, purchase.CustomerID, purchase.Amount)
	
	return purchase, nil
}

// GetPurchase retrieves a purchase
func (as *AncillaryService) GetPurchase(purchaseID string) (models.Purchase, error) {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	
	purchase, exists := as.purchases[purchaseID]
	if !exists {
		return models.Purchase{}, errors.New("purchase not found")
	}
	
	return purchase, nil
}

// Helper Methods

// matchesFilter checks if an item matches the filter criteria
func (as *AncillaryService) matchesFilter(item models.AncillaryItem, filter models.AncillaryFilter) bool {
	if filter.Category != "" && string(item.Category) != filter.Category {
		return false
	}
	
	if filter.Available && !item.Available {
		return false
	}
	
	if filter.MinPrice > 0 && item.BasePrice < filter.MinPrice {
		return false
	}
	
	if filter.MaxPrice > 0 && item.BasePrice > filter.MaxPrice {
		return false
	}
	
	return true
}

// matchesBundleFilter checks if a bundle matches the filter criteria
func (as *AncillaryService) matchesBundleFilter(bundle models.AncillaryBundle, filter models.BundleFilter) bool {
	if filter.Category != "" && bundle.Category != filter.Category {
		return false
	}
	
	if filter.Available && !bundle.Available {
		return false
	}
	
	if filter.Segment != "" {
		segmentMatch := false
		for _, segment := range bundle.TargetSegments {
			if segment == filter.Segment {
				segmentMatch = true
				break
			}
		}
		if !segmentMatch {
			return false
		}
	}
	
	return true
}

// generateItemID generates a unique ID for an ancillary item
func (as *AncillaryService) generateItemID(name string) string {
	// Simple ID generation based on name
	id := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	id = strings.ReplaceAll(id, "(", "")
	id = strings.ReplaceAll(id, ")", "")
	
	// Ensure uniqueness
	counter := 1
	originalID := id
	for {
		if _, exists := as.ancillaryItems[id]; !exists {
			break
		}
		id = fmt.Sprintf("%s-%d", originalID, counter)
		counter++
	}
	
	return id
}

// generateBundleID generates a unique ID for a bundle
func (as *AncillaryService) generateBundleID(name string) string {
	// Simple ID generation based on name
	id := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	id = strings.ReplaceAll(id, "(", "")
	id = strings.ReplaceAll(id, ")", "")
	
	// Ensure uniqueness
	counter := 1
	originalID := id
	for {
		if _, exists := as.bundles[id]; !exists {
			break
		}
		id = fmt.Sprintf("%s-%d", originalID, counter)
		counter++
	}
	
	return id
}

// calculateBundleOriginalPrice calculates the original price of a bundle
func (as *AncillaryService) calculateBundleOriginalPrice(itemIDs []string) float64 {
	var total float64
	
	for _, itemID := range itemIDs {
		if item, exists := as.ancillaryItems[itemID]; exists {
			total += item.BasePrice
		}
	}
	
	return total
}

// calculateDynamicDiscount calculates discount for dynamic bundles
func (as *AncillaryService) calculateDynamicDiscount(itemCount int, customer models.Customer) float64 {
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

// generateDynamicBundleName generates a name for dynamic bundles
func (as *AncillaryService) generateDynamicBundleName(items []models.AncillaryItem, customer models.Customer) string {
	if len(items) == 0 {
		return "Custom Bundle"
	}
	
	// Group by category
	categoryCount := make(map[models.AncillaryCategory]int)
	for _, item := range items {
		categoryCount[item.Category]++
	}
	
	// Find dominant category
	var dominantCategory models.AncillaryCategory
	maxCount := 0
	for category, count := range categoryCount {
		if count > maxCount {
			maxCount = count
			dominantCategory = category
		}
	}
	
	// Generate name based on dominant category and customer segment
	switch dominantCategory {
	case models.CategoryComfort:
		return fmt.Sprintf("Comfort Plus Bundle for %s", customer.Segment)
	case models.CategoryConvenience:
		return fmt.Sprintf("Convenience Bundle for %s", customer.Segment)
	case models.CategoryConnectivity:
		return fmt.Sprintf("Stay Connected Bundle for %s", customer.Segment)
	case models.CategoryDining:
		return fmt.Sprintf("Dining Experience Bundle for %s", customer.Segment)
	case models.CategoryBaggage:
		return fmt.Sprintf("Extra Space Bundle for %s", customer.Segment)
	default:
		return fmt.Sprintf("Personalized Bundle for %s", customer.Segment)
	}
} 