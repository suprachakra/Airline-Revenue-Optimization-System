package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type InFlightRetailPlatform struct {
	db               *mongo.Database
	catalogManager   *CatalogManager
	inventoryManager *InventoryManager
	orderManager     *OrderManager
	paymentProcessor *PaymentProcessor
	deliveryManager  *DeliveryManager
	ifeIntegration   *IFEIntegration
	dutyFreeAPI      *DutyFreeAPI
}

type Product struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SKU             string            `bson:"sku" json:"sku"`
	Name            string            `bson:"name" json:"name"`
	Description     string            `bson:"description" json:"description"`
	Category        string            `bson:"category" json:"category"`
	Subcategory     string            `bson:"subcategory" json:"subcategory"`
	Brand           string            `bson:"brand" json:"brand"`
	Price           float64           `bson:"price" json:"price"`
	Currency        string            `bson:"currency" json:"currency"`
	Images          []string          `bson:"images" json:"images"`
	Videos          []string          `bson:"videos" json:"videos"`
	Specifications  map[string]string `bson:"specifications" json:"specifications"`
	Dimensions      Dimensions        `bson:"dimensions" json:"dimensions"`
	Weight          float64           `bson:"weight" json:"weight"`
	AvailableRoutes []string          `bson:"availableRoutes" json:"availableRoutes"`
	DeliveryOptions []DeliveryOption  `bson:"deliveryOptions" json:"deliveryOptions"`
	TaxInfo         TaxInfo           `bson:"taxInfo" json:"taxInfo"`
	Restrictions    []string          `bson:"restrictions" json:"restrictions"`
	Status          string            `bson:"status" json:"status"`
	CreatedAt       time.Time         `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time         `bson:"updatedAt" json:"updatedAt"`
}

type FlightCatalog struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FlightNumber      string            `bson:"flightNumber" json:"flightNumber"`
	Route             string            `bson:"route" json:"route"`
	AircraftType      string            `bson:"aircraftType" json:"aircraftType"`
	Date              time.Time         `bson:"date" json:"date"`
	Categories        []ProductCategory `bson:"categories" json:"categories"`
	FeaturedProducts  []string          `bson:"featuredProducts" json:"featuredProducts"`
	SpecialOffers     []SpecialOffer    `bson:"specialOffers" json:"specialOffers"`
	LocalCuration     LocalCuration     `bson:"localCuration" json:"localCuration"`
	InventorySnapshot map[string]int    `bson:"inventorySnapshot" json:"inventorySnapshot"`
	LastUpdated       time.Time         `bson:"lastUpdated" json:"lastUpdated"`
}

type ProductCategory struct {
	Name        string   `bson:"name" json:"name"`
	DisplayName string   `bson:"displayName" json:"displayName"`
	Description string   `bson:"description" json:"description"`
	Icon        string   `bson:"icon" json:"icon"`
	Products    []string `bson:"products" json:"products"`
	SortOrder   int      `bson:"sortOrder" json:"sortOrder"`
}

type SpecialOffer struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name            string            `bson:"name" json:"name"`
	Description     string            `bson:"description" json:"description"`
	Type            string            `bson:"type" json:"type"` // discount, bundle, bogo, etc.
	Value           float64           `bson:"value" json:"value"`
	MinPurchase     float64           `bson:"minPurchase" json:"minPurchase"`
	ApplicableItems []string          `bson:"applicableItems" json:"applicableItems"`
	ValidFrom       time.Time         `bson:"validFrom" json:"validFrom"`
	ValidUntil      time.Time         `bson:"validUntil" json:"validUntil"`
	UsageCount      int               `bson:"usageCount" json:"usageCount"`
	MaxUsage        int               `bson:"maxUsage" json:"maxUsage"`
	Status          string            `bson:"status" json:"status"`
}

type LocalCuration struct {
	OriginCity      string           `bson:"originCity" json:"originCity"`
	DestinationCity string           `bson:"destinationCity" json:"destinationCity"`
	LocalProducts   []LocalProduct   `bson:"localProducts" json:"localProducts"`
	CulturalItems   []CulturalItem   `bson:"culturalItems" json:"culturalItems"`
	Recommendations []Recommendation `bson:"recommendations" json:"recommendations"`
}

type LocalProduct struct {
	ProductID   string `bson:"productId" json:"productId"`
	LocalStory  string `bson:"localStory" json:"localStory"`
	Origin      string `bson:"origin" json:"origin"`
	Artisan     string `bson:"artisan" json:"artisan"`
	Authenticity string `bson:"authenticity" json:"authenticity"`
}

type CulturalItem struct {
	Name        string `bson:"name" json:"name"`
	Description string `bson:"description" json:"description"`
	Category    string `bson:"category" json:"category"`
	ProductIDs  []string `bson:"productIds" json:"productIds"`
}

type Recommendation struct {
	Title       string   `bson:"title" json:"title"`
	Description string   `bson:"description" json:"description"`
	ProductIDs  []string `bson:"productIds" json:"productIds"`
	Reason      string   `bson:"reason" json:"reason"`
}

type InFlightOrder struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrderNumber       string            `bson:"orderNumber" json:"orderNumber"`
	FlightNumber      string            `bson:"flightNumber" json:"flightNumber"`
	SeatNumber        string            `bson:"seatNumber" json:"seatNumber"`
	PassengerName     string            `bson:"passengerName" json:"passengerName"`
	PassengerID       string            `bson:"passengerId" json:"passengerId"`
	Items             []OrderItem       `bson:"items" json:"items"`
	Subtotal          float64           `bson:"subtotal" json:"subtotal"`
	Tax               float64           `bson:"tax" json:"tax"`
	Discount          float64           `bson:"discount" json:"discount"`
	Total             float64           `bson:"total" json:"total"`
	Currency          string            `bson:"currency" json:"currency"`
	PaymentMethod     string            `bson:"paymentMethod" json:"paymentMethod"`
	PaymentStatus     string            `bson:"paymentStatus" json:"paymentStatus"`
	OrderStatus       string            `bson:"orderStatus" json:"orderStatus"`
	DeliveryMethod    string            `bson:"deliveryMethod" json:"deliveryMethod"`
	DeliveryAddress   DeliveryAddress   `bson:"deliveryAddress" json:"deliveryAddress"`
	SpecialInstructions string          `bson:"specialInstructions" json:"specialInstructions"`
	CreatedAt         time.Time         `bson:"createdAt" json:"createdAt"`
	UpdatedAt         time.Time         `bson:"updatedAt" json:"updatedAt"`
	DeliveredAt       *time.Time        `bson:"deliveredAt" json:"deliveredAt"`
}

type OrderItem struct {
	ProductID    string            `bson:"productId" json:"productId"`
	SKU          string            `bson:"sku" json:"sku"`
	Name         string            `bson:"name" json:"name"`
	Quantity     int               `bson:"quantity" json:"quantity"`
	UnitPrice    float64           `bson:"unitPrice" json:"unitPrice"`
	TotalPrice   float64           `bson:"totalPrice" json:"totalPrice"`
	Customization map[string]string `bson:"customization" json:"customization"`
	GiftWrap     bool              `bson:"giftWrap" json:"giftWrap"`
	GiftMessage  string            `bson:"giftMessage" json:"giftMessage"`
}

type DeliveryOption struct {
	Type         string  `bson:"type" json:"type"` // seat, gate, home, hotel
	Name         string  `bson:"name" json:"name"`
	Description  string  `bson:"description" json:"description"`
	Fee          float64 `bson:"fee" json:"fee"`
	EstimatedTime string `bson:"estimatedTime" json:"estimatedTime"`
	Available    bool    `bson:"available" json:"available"`
}

type DeliveryAddress struct {
	Type        string `bson:"type" json:"type"`
	Name        string `bson:"name" json:"name"`
	AddressLine1 string `bson:"addressLine1" json:"addressLine1"`
	AddressLine2 string `bson:"addressLine2" json:"addressLine2"`
	City        string `bson:"city" json:"city"`
	State       string `bson:"state" json:"state"`
	Country     string `bson:"country" json:"country"`
	PostalCode  string `bson:"postalCode" json:"postalCode"`
	Phone       string `bson:"phone" json:"phone"`
}

type Dimensions struct {
	Length float64 `bson:"length" json:"length"`
	Width  float64 `bson:"width" json:"width"`
	Height float64 `bson:"height" json:"height"`
	Unit   string  `bson:"unit" json:"unit"`
}

type TaxInfo struct {
	TaxRate     float64 `bson:"taxRate" json:"taxRate"`
	TaxType     string  `bson:"taxType" json:"taxType"`
	DutyFree    bool    `bson:"dutyFree" json:"dutyFree"`
	Jurisdiction string `bson:"jurisdiction" json:"jurisdiction"`
}

type InventoryItem struct {
	ProductID     string    `bson:"productId" json:"productId"`
	FlightNumber  string    `bson:"flightNumber" json:"flightNumber"`
	Quantity      int       `bson:"quantity" json:"quantity"`
	Reserved      int       `bson:"reserved" json:"reserved"`
	Available     int       `bson:"available" json:"available"`
	Location      string    `bson:"location" json:"location"` // galley, cargo, etc.
	LastUpdated   time.Time `bson:"lastUpdated" json:"lastUpdated"`
}

// Service components
type CatalogManager struct {
	db *mongo.Database
}

type InventoryManager struct {
	db *mongo.Database
}

type OrderManager struct {
	db *mongo.Database
}

type PaymentProcessor struct {
	db *mongo.Database
}

type DeliveryManager struct {
	db *mongo.Database
}

type IFEIntegration struct {
	db *mongo.Database
}

type DutyFreeAPI struct {
	db *mongo.Database
}

func NewInFlightRetailPlatform(db *mongo.Database) *InFlightRetailPlatform {
	return &InFlightRetailPlatform{
		db:               db,
		catalogManager:   &CatalogManager{db: db},
		inventoryManager: &InventoryManager{db: db},
		orderManager:     &OrderManager{db: db},
		paymentProcessor: &PaymentProcessor{db: db},
		deliveryManager:  &DeliveryManager{db: db},
		ifeIntegration:   &IFEIntegration{db: db},
		dutyFreeAPI:      &DutyFreeAPI{db: db},
	}
}

func (ifrp *InFlightRetailPlatform) GetFlightCatalog(c *gin.Context) {
	flightNumber := c.Query("flight")
	date := c.Query("date")
	seatClass := c.Query("class")
	passengerID := c.Query("passenger")

	if flightNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Flight number is required"})
		return
	}

	// Parse date
	var flightDate time.Time
	var err error
	if date != "" {
		flightDate, err = time.Parse("2006-01-02", date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
			return
		}
	} else {
		flightDate = time.Now()
	}

	// Get flight catalog
	catalog, err := ifrp.catalogManager.GetFlightCatalog(flightNumber, flightDate)
	if err != nil {
		log.Printf("Error getting flight catalog: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get catalog"})
		return
	}

	// Get personalized recommendations if passenger ID provided
	var recommendations []Recommendation
	if passengerID != "" {
		recommendations, err = ifrp.getPersonalizedRecommendations(passengerID, flightNumber)
		if err != nil {
			log.Printf("Error getting recommendations: %v", err)
		}
	}

	// Get real-time inventory
	inventory, err := ifrp.inventoryManager.GetFlightInventory(flightNumber, flightDate)
	if err != nil {
		log.Printf("Error getting inventory: %v", err)
		inventory = make(map[string]int) // Default empty inventory
	}

	// Filter catalog based on seat class if provided
	if seatClass != "" {
		catalog = ifrp.filterCatalogByClass(catalog, seatClass)
	}

	c.JSON(http.StatusOK, gin.H{
		"catalog":         catalog,
		"inventory":       inventory,
		"recommendations": recommendations,
		"flight":          flightNumber,
		"date":           flightDate.Format("2006-01-02"),
		"lastUpdated":    catalog.LastUpdated,
	})
}

func (ifrp *InFlightRetailPlatform) BrowseProducts(c *gin.Context) {
	flightNumber := c.Query("flight")
	category := c.Query("category")
	search := c.Query("search")
	priceMin := c.Query("price_min")
	priceMax := c.Query("price_max")
	brand := c.Query("brand")
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "20")

	filter := bson.M{}
	
	if flightNumber != "" {
		filter["availableRoutes"] = bson.M{"$in": []string{flightNumber, "all"}}
	}

	if category != "" {
		filter["category"] = category
	}

	if brand != "" {
		filter["brand"] = brand
	}

	if search != "" {
		filter["$or"] = []bson.M{
			{"name": bson.M{"$regex": search, "$options": "i"}},
			{"description": bson.M{"$regex": search, "$options": "i"}},
			{"brand": bson.M{"$regex": search, "$options": "i"}},
		}
	}

	// Price filtering
	if priceMin != "" || priceMax != "" {
		priceFilter := bson.M{}
		if priceMin != "" {
			if minPrice, err := parseFloat(priceMin); err == nil {
				priceFilter["$gte"] = minPrice
			}
		}
		if priceMax != "" {
			if maxPrice, err := parseFloat(priceMax); err == nil {
				priceFilter["$lte"] = maxPrice
			}
		}
		if len(priceFilter) > 0 {
			filter["price"] = priceFilter
		}
	}

	filter["status"] = "active"

	collection := ifrp.db.Collection("retail_products")
	
	// Count total
	total, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count products"})
		return
	}

	// Parse pagination
	pageNum := parseInt(page, 1)
	limitNum := parseInt(limit, 20)
	skip := (pageNum - 1) * limitNum

	// Find products
	cursor, err := collection.Find(context.Background(), filter, 
		&options.FindOptions{
			Skip:  &skip,
			Limit: &limitNum,
			Sort:  bson.M{"name": 1},
		})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}
	defer cursor.Close(context.Background())

	var products []Product
	if err = cursor.All(context.Background(), &products); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode products"})
		return
	}

	// Get inventory for these products
	inventory, _ := ifrp.inventoryManager.GetProductsInventory(flightNumber, extractProductIDs(products))

	// Enrich products with inventory data
	enrichedProducts := ifrp.enrichProductsWithInventory(products, inventory)

	c.JSON(http.StatusOK, gin.H{
		"products":    enrichedProducts,
		"total":       total,
		"page":        pageNum,
		"limit":       limitNum,
		"totalPages":  (total + int64(limitNum) - 1) / int64(limitNum),
		"filters":     filter,
	})
}

func (ifrp *InFlightRetailPlatform) CreateOrder(c *gin.Context) {
	var orderRequest struct {
		FlightNumber        string            `json:"flightNumber" binding:"required"`
		SeatNumber          string            `json:"seatNumber" binding:"required"`
		PassengerName       string            `json:"passengerName" binding:"required"`
		PassengerID         string            `json:"passengerId" binding:"required"`
		Items               []OrderItem       `json:"items" binding:"required"`
		DeliveryMethod      string            `json:"deliveryMethod" binding:"required"`
		DeliveryAddress     *DeliveryAddress  `json:"deliveryAddress"`
		PaymentMethod       string            `json:"paymentMethod" binding:"required"`
		SpecialInstructions string            `json:"specialInstructions"`
		PromoCode           string            `json:"promoCode"`
	}

	if err := c.ShouldBindJSON(&orderRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate inventory availability
	for _, item := range orderRequest.Items {
		available, err := ifrp.inventoryManager.CheckAvailability(orderRequest.FlightNumber, item.ProductID, item.Quantity)
		if err != nil || !available {
			c.JSON(http.StatusConflict, gin.H{
				"error": fmt.Sprintf("Product %s is not available in requested quantity", item.Name),
				"item":  item.ProductID,
			})
			return
		}
	}

	// Calculate pricing
	subtotal := 0.0
	for i, item := range orderRequest.Items {
		product, err := ifrp.getProductByID(item.ProductID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Product %s not found", item.ProductID)})
			return
		}
		
		orderRequest.Items[i].UnitPrice = product.Price
		orderRequest.Items[i].TotalPrice = product.Price * float64(item.Quantity)
		subtotal += orderRequest.Items[i].TotalPrice
	}

	// Apply discounts
	discount := 0.0
	if orderRequest.PromoCode != "" {
		discount, _ = ifrp.calculateDiscount(orderRequest.PromoCode, subtotal, orderRequest.Items)
	}

	// Calculate tax
	tax := ifrp.calculateTax(subtotal-discount, orderRequest.FlightNumber)
	total := subtotal + tax - discount

	// Create order
	order := InFlightOrder{
		ID:              primitive.NewObjectID(),
		OrderNumber:     ifrp.generateOrderNumber(),
		FlightNumber:    orderRequest.FlightNumber,
		SeatNumber:      orderRequest.SeatNumber,
		PassengerName:   orderRequest.PassengerName,
		PassengerID:     orderRequest.PassengerID,
		Items:           orderRequest.Items,
		Subtotal:        subtotal,
		Tax:             tax,
		Discount:        discount,
		Total:           total,
		Currency:        "USD",
		PaymentMethod:   orderRequest.PaymentMethod,
		PaymentStatus:   "pending",
		OrderStatus:     "confirmed",
		DeliveryMethod:  orderRequest.DeliveryMethod,
		SpecialInstructions: orderRequest.SpecialInstructions,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if orderRequest.DeliveryAddress != nil {
		order.DeliveryAddress = *orderRequest.DeliveryAddress
	}

	// Save order
	collection := ifrp.db.Collection("inflight_orders")
	_, err := collection.InsertOne(context.Background(), order)
	if err != nil {
		log.Printf("Error creating order: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	// Reserve inventory
	for _, item := range orderRequest.Items {
		err := ifrp.inventoryManager.ReserveItem(orderRequest.FlightNumber, item.ProductID, item.Quantity)
		if err != nil {
			log.Printf("Error reserving inventory for %s: %v", item.ProductID, err)
		}
	}

	// Process payment
	go ifrp.paymentProcessor.ProcessPayment(order)

	// Send confirmation to IFE system
	go ifrp.ifeIntegration.SendOrderConfirmation(order)

	c.JSON(http.StatusCreated, gin.H{
		"order":          order,
		"paymentStatus":  "processing",
		"estimatedTime":  ifrp.getDeliveryEstimate(orderRequest.DeliveryMethod),
		"trackingInfo":   ifrp.generateTrackingInfo(order),
	})
}

func (ifrp *InFlightRetailPlatform) GetOrder(c *gin.Context) {
	orderNumber := c.Param("orderNumber")
	
	collection := ifrp.db.Collection("inflight_orders")
	var order InFlightOrder
	
	err := collection.FindOne(context.Background(), bson.M{
		"orderNumber": orderNumber,
	}).Decode(&order)
	
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve order"})
		}
		return
	}

	// Get delivery status
	deliveryStatus := ifrp.deliveryManager.GetDeliveryStatus(order.OrderNumber)

	c.JSON(http.StatusOK, gin.H{
		"order":          order,
		"deliveryStatus": deliveryStatus,
		"timeline":       ifrp.getOrderTimeline(order),
	})
}

func (ifrp *InFlightRetailPlatform) UpdateOrder(c *gin.Context) {
	orderNumber := c.Param("orderNumber")
	
	var updateRequest struct {
		DeliveryAddress     *DeliveryAddress `json:"deliveryAddress"`
		SpecialInstructions string           `json:"specialInstructions"`
		AddItems            []OrderItem      `json:"addItems"`
		RemoveItems         []string         `json:"removeItems"`
	}

	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	collection := ifrp.db.Collection("inflight_orders")
	
	// Get current order
	var order InFlightOrder
	err := collection.FindOne(context.Background(), bson.M{
		"orderNumber": orderNumber,
	}).Decode(&order)
	
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Check if order can be modified
	if order.OrderStatus == "delivered" || order.OrderStatus == "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order cannot be modified in current status"})
		return
	}

	// Update fields
	update := bson.M{"$set": bson.M{"updatedAt": time.Now()}}

	if updateRequest.DeliveryAddress != nil {
		update["$set"].(bson.M)["deliveryAddress"] = *updateRequest.DeliveryAddress
	}

	if updateRequest.SpecialInstructions != "" {
		update["$set"].(bson.M)["specialInstructions"] = updateRequest.SpecialInstructions
	}

	// Handle item modifications
	if len(updateRequest.AddItems) > 0 || len(updateRequest.RemoveItems) > 0 {
		// This would require complex logic to add/remove items and recalculate pricing
		// For now, return success but log for manual processing
		log.Printf("Order modification requested for %s: add %d items, remove %d items", 
			orderNumber, len(updateRequest.AddItems), len(updateRequest.RemoveItems))
	}

	// Update in database
	_, err = collection.UpdateOne(context.Background(), bson.M{
		"orderNumber": orderNumber,
	}, update)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "updated",
		"message": "Order updated successfully",
	})
}

func (ifrp *InFlightRetailPlatform) CancelOrder(c *gin.Context) {
	orderNumber := c.Param("orderNumber")
	
	collection := ifrp.db.Collection("inflight_orders")
	
	// Get order
	var order InFlightOrder
	err := collection.FindOne(context.Background(), bson.M{
		"orderNumber": orderNumber,
	}).Decode(&order)
	
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Check if cancellation is allowed
	if order.OrderStatus == "delivered" || order.OrderStatus == "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order cannot be cancelled"})
		return
	}

	// Calculate refund amount
	refundAmount := order.Total
	cancellationFee := 0.0

	// Apply cancellation policy based on delivery method
	if order.DeliveryMethod == "home" || order.DeliveryMethod == "hotel" {
		// No cancellation fee for post-flight delivery
	} else {
		// Small fee for in-flight cancellations
		cancellationFee = 5.0
		refundAmount -= cancellationFee
	}

	// Update order status
	_, err = collection.UpdateOne(context.Background(), bson.M{
		"orderNumber": orderNumber,
	}, bson.M{
		"$set": bson.M{
			"orderStatus": "cancelled",
			"updatedAt":   time.Now(),
		},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel order"})
		return
	}

	// Release reserved inventory
	for _, item := range order.Items {
		err := ifrp.inventoryManager.ReleaseReservation(order.FlightNumber, item.ProductID, item.Quantity)
		if err != nil {
			log.Printf("Error releasing inventory for %s: %v", item.ProductID, err)
		}
	}

	// Process refund
	go ifrp.paymentProcessor.ProcessRefund(order, refundAmount)

	c.JSON(http.StatusOK, gin.H{
		"status":           "cancelled",
		"refundAmount":     refundAmount,
		"cancellationFee":  cancellationFee,
		"refundTimeline":   "3-5 business days",
	})
}

// Catalog Manager methods
func (cm *CatalogManager) GetFlightCatalog(flightNumber string, date time.Time) (*FlightCatalog, error) {
	collection := cm.db.Collection("flight_catalogs")
	var catalog FlightCatalog
	
	err := collection.FindOne(context.Background(), bson.M{
		"flightNumber": flightNumber,
		"date": bson.M{
			"$gte": date.Truncate(24 * time.Hour),
			"$lt":  date.Truncate(24 * time.Hour).Add(24 * time.Hour),
		},
	}).Decode(&catalog)
	
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Create default catalog
			return cm.createDefaultCatalog(flightNumber, date)
		}
		return nil, err
	}
	
	return &catalog, nil
}

func (cm *CatalogManager) createDefaultCatalog(flightNumber string, date time.Time) (*FlightCatalog, error) {
	// Create a default catalog structure
	catalog := &FlightCatalog{
		ID:           primitive.NewObjectID(),
		FlightNumber: flightNumber,
		Route:        "Default Route",
		AircraftType: "B737",
		Date:         date,
		Categories: []ProductCategory{
			{Name: "food", DisplayName: "Food & Beverages", SortOrder: 1},
			{Name: "duty_free", DisplayName: "Duty Free", SortOrder: 2},
			{Name: "electronics", DisplayName: "Electronics", SortOrder: 3},
			{Name: "fashion", DisplayName: "Fashion & Accessories", SortOrder: 4},
			{Name: "travel", DisplayName: "Travel Essentials", SortOrder: 5},
		},
		FeaturedProducts: []string{},
		SpecialOffers:    []SpecialOffer{},
		LocalCuration:    LocalCuration{},
		LastUpdated:      time.Now(),
	}
	
	return catalog, nil
}

// Inventory Manager methods
func (im *InventoryManager) GetFlightInventory(flightNumber string, date time.Time) (map[string]int, error) {
	collection := im.db.Collection("flight_inventory")
	
	cursor, err := collection.Find(context.Background(), bson.M{
		"flightNumber": flightNumber,
		"date": bson.M{
			"$gte": date.Truncate(24 * time.Hour),
			"$lt":  date.Truncate(24 * time.Hour).Add(24 * time.Hour),
		},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	inventory := make(map[string]int)
	for cursor.Next(context.Background()) {
		var item InventoryItem
		if err := cursor.Decode(&item); err != nil {
			continue
		}
		inventory[item.ProductID] = item.Available
	}

	return inventory, nil
}

func (im *InventoryManager) CheckAvailability(flightNumber, productID string, quantity int) (bool, error) {
	collection := im.db.Collection("flight_inventory")
	var item InventoryItem
	
	err := collection.FindOne(context.Background(), bson.M{
		"flightNumber": flightNumber,
		"productId":    productID,
	}).Decode(&item)
	
	if err != nil {
		return false, err
	}
	
	return item.Available >= quantity, nil
}

func (im *InventoryManager) ReserveItem(flightNumber, productID string, quantity int) error {
	collection := im.db.Collection("flight_inventory")
	
	_, err := collection.UpdateOne(context.Background(), bson.M{
		"flightNumber": flightNumber,
		"productId":    productID,
		"available":    bson.M{"$gte": quantity},
	}, bson.M{
		"$inc": bson.M{
			"available": -quantity,
			"reserved":  quantity,
		},
		"$set": bson.M{"lastUpdated": time.Now()},
	})
	
	return err
}

func (im *InventoryManager) ReleaseReservation(flightNumber, productID string, quantity int) error {
	collection := im.db.Collection("flight_inventory")
	
	_, err := collection.UpdateOne(context.Background(), bson.M{
		"flightNumber": flightNumber,
		"productId":    productID,
	}, bson.M{
		"$inc": bson.M{
			"available": quantity,
			"reserved":  -quantity,
		},
		"$set": bson.M{"lastUpdated": time.Now()},
	})
	
	return err
}

func (im *InventoryManager) GetProductsInventory(flightNumber string, productIDs []string) (map[string]int, error) {
	collection := im.db.Collection("flight_inventory")
	
	cursor, err := collection.Find(context.Background(), bson.M{
		"flightNumber": flightNumber,
		"productId":    bson.M{"$in": productIDs},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	inventory := make(map[string]int)
	for cursor.Next(context.Background()) {
		var item InventoryItem
		if err := cursor.Decode(&item); err != nil {
			continue
		}
		inventory[item.ProductID] = item.Available
	}

	return inventory, nil
}

// Helper methods
func (ifrp *InFlightRetailPlatform) getPersonalizedRecommendations(passengerID, flightNumber string) ([]Recommendation, error) {
	// Get passenger purchase history and preferences
	// Return personalized recommendations
	return []Recommendation{
		{
			Title:       "Based on your previous purchases",
			Description: "You might enjoy these premium chocolates",
			ProductIDs:  []string{"chocolate_001", "chocolate_002"},
			Reason:      "purchase_history",
		},
	}, nil
}

func (ifrp *InFlightRetailPlatform) filterCatalogByClass(catalog *FlightCatalog, seatClass string) *FlightCatalog {
	// Filter products based on seat class availability
	return catalog
}

func (ifrp *InFlightRetailPlatform) getProductByID(productID string) (*Product, error) {
	collection := ifrp.db.Collection("retail_products")
	var product Product
	
	objID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return nil, err
	}
	
	err = collection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&product)
	if err != nil {
		return nil, err
	}
	
	return &product, nil
}

func (ifrp *InFlightRetailPlatform) calculateDiscount(promoCode string, subtotal float64, items []OrderItem) (float64, error) {
	// Calculate discount based on promo code
	// This would involve looking up the promo code and applying relevant discounts
	return subtotal * 0.1, nil // 10% discount for demo
}

func (ifrp *InFlightRetailPlatform) calculateTax(amount float64, flightNumber string) float64 {
	// Calculate tax based on flight route and applicable jurisdictions
	return amount * 0.08 // 8% tax rate for demo
}

func (ifrp *InFlightRetailPlatform) generateOrderNumber() string {
	return fmt.Sprintf("IFR%d", time.Now().Unix())
}

func (ifrp *InFlightRetailPlatform) getDeliveryEstimate(deliveryMethod string) string {
	switch deliveryMethod {
	case "seat":
		return "15-30 minutes"
	case "gate":
		return "Upon arrival"
	case "home":
		return "5-7 business days"
	case "hotel":
		return "Next day delivery"
	default:
		return "TBD"
	}
}

func (ifrp *InFlightRetailPlatform) generateTrackingInfo(order InFlightOrder) map[string]string {
	return map[string]string{
		"trackingNumber": fmt.Sprintf("TRK%s", order.OrderNumber),
		"carrier":        "In-Flight Retail",
		"estimatedDelivery": time.Now().Add(24 * time.Hour).Format("2006-01-02"),
	}
}

func (ifrp *InFlightRetailPlatform) getOrderTimeline(order InFlightOrder) []map[string]interface{} {
	timeline := []map[string]interface{}{
		{
			"status":    "Order Placed",
			"timestamp": order.CreatedAt,
			"completed": true,
		},
		{
			"status":    "Payment Processed",
			"timestamp": order.CreatedAt.Add(2 * time.Minute),
			"completed": order.PaymentStatus == "completed",
		},
		{
			"status":    "Preparing Order",
			"timestamp": order.CreatedAt.Add(5 * time.Minute),
			"completed": order.OrderStatus != "pending",
		},
	}
	
	if order.DeliveredAt != nil {
		timeline = append(timeline, map[string]interface{}{
			"status":    "Delivered",
			"timestamp": *order.DeliveredAt,
			"completed": true,
		})
	}
	
	return timeline
}

func (ifrp *InFlightRetailPlatform) enrichProductsWithInventory(products []Product, inventory map[string]int) []map[string]interface{} {
	var enriched []map[string]interface{}
	
	for _, product := range products {
		productMap := map[string]interface{}{
			"product":     product,
			"available":   inventory[product.ID.Hex()],
			"inStock":     inventory[product.ID.Hex()] > 0,
		}
		enriched = append(enriched, productMap)
	}
	
	return enriched
}

func extractProductIDs(products []Product) []string {
	var ids []string
	for _, product := range products {
		ids = append(ids, product.ID.Hex())
	}
	return ids
}

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func parseInt(s string, defaultVal int) int {
	if val, err := strconv.Atoi(s); err == nil {
		return val
	}
	return defaultVal
}

// Payment Processor methods
func (pp *PaymentProcessor) ProcessPayment(order InFlightOrder) {
	log.Printf("Processing payment for order %s", order.OrderNumber)
	// Implementation for payment processing
}

func (pp *PaymentProcessor) ProcessRefund(order InFlightOrder, amount float64) {
	log.Printf("Processing refund for order %s: $%.2f", order.OrderNumber, amount)
	// Implementation for refund processing
}

// IFE Integration methods
func (ife *IFEIntegration) SendOrderConfirmation(order InFlightOrder) {
	log.Printf("Sending order confirmation to IFE for seat %s", order.SeatNumber)
	// Implementation for IFE system integration
}

// Delivery Manager methods
func (dm *DeliveryManager) GetDeliveryStatus(orderNumber string) map[string]interface{} {
	return map[string]interface{}{
		"status":          "in_transit",
		"estimatedTime":   "25 minutes",
		"currentLocation": "Galley 2",
		"deliveryPerson":  "Cabin Crew Member #3",
	}
}

// RegisterRoutes registers all in-flight retail routes
func (ifrp *InFlightRetailPlatform) RegisterRoutes(router *gin.Engine) {
	retailRoutes := router.Group("/api/v1/inflight-retail")
	{
		retailRoutes.GET("/catalog", ifrp.GetFlightCatalog)
		retailRoutes.GET("/products", ifrp.BrowseProducts)
		retailRoutes.POST("/orders", ifrp.CreateOrder)
		retailRoutes.GET("/orders/:orderNumber", ifrp.GetOrder)
		retailRoutes.PUT("/orders/:orderNumber", ifrp.UpdateOrder)
		retailRoutes.DELETE("/orders/:orderNumber", ifrp.CancelOrder)
	}
} 