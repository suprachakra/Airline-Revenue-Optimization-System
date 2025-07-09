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

type AirportServicesModule struct {
	db               *mongo.Database
	loungeAPI        *LoungeAccessAPI
	securityAPI      *FastTrackAPI
	transportAPI     *GroundTransportAPI
	parkingAPI       *ParkingAPI
	baggageAPI       *BaggageServicesAPI
	conciergeAPI     *ConciergeAPI
	availabilityCache map[string]*ServiceAvailability
}

type AirportService struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ServiceType     string            `bson:"serviceType" json:"serviceType"`
	AirportCode     string            `bson:"airportCode" json:"airportCode"`
	Terminal        string            `bson:"terminal" json:"terminal"`
	Name            string            `bson:"name" json:"name"`
	Description     string            `bson:"description" json:"description"`
	Provider        string            `bson:"provider" json:"provider"`
	BasePrice       float64           `bson:"basePrice" json:"basePrice"`
	Currency        string            `bson:"currency" json:"currency"`
	Duration        int               `bson:"duration" json:"duration"` // minutes
	Capacity        int               `bson:"capacity" json:"capacity"`
	Features        []string          `bson:"features" json:"features"`
	OpeningHours    OpeningHours      `bson:"openingHours" json:"openingHours"`
	Location        Location          `bson:"location" json:"location"`
	BookingPolicy   BookingPolicy     `bson:"bookingPolicy" json:"bookingPolicy"`
	CancellationPolicy CancellationPolicy `bson:"cancellationPolicy" json:"cancellationPolicy"`
	Status          string            `bson:"status" json:"status"`
	CreatedAt       time.Time         `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time         `bson:"updatedAt" json:"updatedAt"`
}

type ServiceBooking struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BookingReference string            `bson:"bookingReference" json:"bookingReference"`
	ServiceID       primitive.ObjectID `bson:"serviceId" json:"serviceId"`
	CustomerID      string            `bson:"customerId" json:"customerId"`
	FlightNumber    string            `bson:"flightNumber" json:"flightNumber"`
	PNR             string            `bson:"pnr" json:"pnr"`
	ServiceType     string            `bson:"serviceType" json:"serviceType"`
	AirportCode     string            `bson:"airportCode" json:"airportCode"`
	Terminal        string            `bson:"terminal" json:"terminal"`
	ServiceDate     time.Time         `bson:"serviceDate" json:"serviceDate"`
	ServiceTime     string            `bson:"serviceTime" json:"serviceTime"`
	Duration        int               `bson:"duration" json:"duration"`
	GuestCount      int               `bson:"guestCount" json:"guestCount"`
	TotalPrice      float64           `bson:"totalPrice" json:"totalPrice"`
	Currency        string            `bson:"currency" json:"currency"`
	PaymentStatus   string            `bson:"paymentStatus" json:"paymentStatus"`
	BookingStatus   string            `bson:"bookingStatus" json:"bookingStatus"`
	SpecialRequests []string          `bson:"specialRequests" json:"specialRequests"`
	ContactInfo     ContactInfo       `bson:"contactInfo" json:"contactInfo"`
	CheckInDetails  CheckInDetails    `bson:"checkInDetails" json:"checkInDetails"`
	CreatedAt       time.Time         `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time         `bson:"updatedAt" json:"updatedAt"`
}

type ServiceAvailability struct {
	ServiceID     primitive.ObjectID `json:"serviceId"`
	Date          time.Time         `json:"date"`
	TimeSlots     []TimeSlot        `json:"timeSlots"`
	TotalCapacity int               `json:"totalCapacity"`
	Available     int               `json:"available"`
	WaitList      int               `json:"waitList"`
	LastUpdated   time.Time         `json:"lastUpdated"`
}

type TimeSlot struct {
	StartTime   string  `json:"startTime"`
	EndTime     string  `json:"endTime"`
	Capacity    int     `json:"capacity"`
	Available   int     `json:"available"`
	Price       float64 `json:"price"`
	Status      string  `json:"status"` // available, limited, full, closed
}

type OpeningHours struct {
	Monday    []string `bson:"monday" json:"monday"`
	Tuesday   []string `bson:"tuesday" json:"tuesday"`
	Wednesday []string `bson:"wednesday" json:"wednesday"`
	Thursday  []string `bson:"thursday" json:"thursday"`
	Friday    []string `bson:"friday" json:"friday"`
	Saturday  []string `bson:"saturday" json:"saturday"`
	Sunday    []string `bson:"sunday" json:"sunday"`
}

type Location struct {
	Level       string  `bson:"level" json:"level"`
	Zone        string  `bson:"zone" json:"zone"`
	Gate        string  `bson:"gate" json:"gate"`
	Latitude    float64 `bson:"latitude" json:"latitude"`
	Longitude   float64 `bson:"longitude" json:"longitude"`
	Directions  string  `bson:"directions" json:"directions"`
}

type BookingPolicy struct {
	AdvanceBooking    int      `bson:"advanceBooking" json:"advanceBooking"` // hours
	MaxBookingWindow  int      `bson:"maxBookingWindow" json:"maxBookingWindow"` // days
	RequiredDocuments []string `bson:"requiredDocuments" json:"requiredDocuments"`
	AgeRestrictions   string   `bson:"ageRestrictions" json:"ageRestrictions"`
	DressCode         string   `bson:"dressCode" json:"dressCode"`
}

type CancellationPolicy struct {
	FreeCancellation int     `bson:"freeCancellation" json:"freeCancellation"` // hours before
	CancellationFee  float64 `bson:"cancellationFee" json:"cancellationFee"`
	RefundPolicy     string  `bson:"refundPolicy" json:"refundPolicy"`
}

type ContactInfo struct {
	Name    string `bson:"name" json:"name"`
	Email   string `bson:"email" json:"email"`
	Phone   string `bson:"phone" json:"phone"`
	Country string `bson:"country" json:"country"`
}

type CheckInDetails struct {
	QRCode         string    `bson:"qrCode" json:"qrCode"`
	AccessCode     string    `bson:"accessCode" json:"accessCode"`
	CheckInTime    time.Time `bson:"checkInTime" json:"checkInTime"`
	CheckOutTime   time.Time `bson:"checkOutTime" json:"checkOutTime"`
	ActualDuration int       `bson:"actualDuration" json:"actualDuration"`
	UsageNotes     string    `bson:"usageNotes" json:"usageNotes"`
}

// Service-specific APIs
type LoungeAccessAPI struct {
	db *mongo.Database
}

type FastTrackAPI struct {
	db *mongo.Database
}

type GroundTransportAPI struct {
	db *mongo.Database
}

type ParkingAPI struct {
	db *mongo.Database
}

type BaggageServicesAPI struct {
	db *mongo.Database
}

type ConciergeAPI struct {
	db *mongo.Database
}

func NewAirportServicesModule(db *mongo.Database) *AirportServicesModule {
	return &AirportServicesModule{
		db:               db,
		loungeAPI:        &LoungeAccessAPI{db: db},
		securityAPI:      &FastTrackAPI{db: db},
		transportAPI:     &GroundTransportAPI{db: db},
		parkingAPI:       &ParkingAPI{db: db},
		baggageAPI:       &BaggageServicesAPI{db: db},
		conciergeAPI:     &ConciergeAPI{db: db},
		availabilityCache: make(map[string]*ServiceAvailability),
	}
}

func (asm *AirportServicesModule) SearchServices(c *gin.Context) {
	airportCode := c.Query("airport")
	serviceType := c.Query("type")
	date := c.Query("date")
	terminal := c.Query("terminal")
	
	if airportCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Airport code is required"})
		return
	}

	// Parse date
	var serviceDate time.Time
	var err error
	if date != "" {
		serviceDate, err = time.Parse("2006-01-02", date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
			return
		}
	} else {
		serviceDate = time.Now()
	}

	// Build search filter
	filter := bson.M{
		"airportCode": airportCode,
		"status":      "active",
	}

	if serviceType != "" {
		filter["serviceType"] = serviceType
	}

	if terminal != "" {
		filter["terminal"] = terminal
	}

	// Search services
	collection := asm.db.Collection("airport_services")
	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		log.Printf("Error searching airport services: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search services"})
		return
	}
	defer cursor.Close(context.Background())

	var services []AirportService
	if err = cursor.All(context.Background(), &services); err != nil {
		log.Printf("Error decoding services: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process services"})
		return
	}

	// Get availability for each service
	var enrichedServices []map[string]interface{}
	for _, service := range services {
		availability, err := asm.getServiceAvailability(service.ID, serviceDate)
		if err != nil {
			log.Printf("Error getting availability for service %s: %v", service.ID.Hex(), err)
			continue
		}

		enrichedService := map[string]interface{}{
			"service":      service,
			"availability": availability,
		}
		enrichedServices = append(enrichedServices, enrichedService)
	}

	c.JSON(http.StatusOK, gin.H{
		"services": enrichedServices,
		"total":    len(enrichedServices),
		"airport":  airportCode,
		"date":     serviceDate.Format("2006-01-02"),
	})
}

func (asm *AirportServicesModule) BookService(c *gin.Context) {
	var bookingRequest struct {
		ServiceID       string            `json:"serviceId" binding:"required"`
		CustomerID      string            `json:"customerId" binding:"required"`
		FlightNumber    string            `json:"flightNumber"`
		PNR             string            `json:"pnr"`
		ServiceDate     string            `json:"serviceDate" binding:"required"`
		ServiceTime     string            `json:"serviceTime"`
		GuestCount      int               `json:"guestCount"`
		SpecialRequests []string          `json:"specialRequests"`
		ContactInfo     ContactInfo       `json:"contactInfo" binding:"required"`
	}

	if err := c.ShouldBindJSON(&bookingRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert service ID
	serviceObjID, err := primitive.ObjectIDFromHex(bookingRequest.ServiceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service ID"})
		return
	}

	// Parse service date
	serviceDate, err := time.Parse("2006-01-02", bookingRequest.ServiceDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service date format"})
		return
	}

	// Get service details
	service, err := asm.getServiceByID(serviceObjID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Check availability
	availability, err := asm.getServiceAvailability(serviceObjID, serviceDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check availability"})
		return
	}

	if availability.Available < bookingRequest.GuestCount {
		c.JSON(http.StatusConflict, gin.H{"error": "Insufficient availability"})
		return
	}

	// Calculate pricing
	totalPrice := service.BasePrice * float64(bookingRequest.GuestCount)
	
	// Apply dynamic pricing if applicable
	if bookingRequest.ServiceTime != "" {
		for _, slot := range availability.TimeSlots {
			if slot.StartTime == bookingRequest.ServiceTime {
				totalPrice = slot.Price * float64(bookingRequest.GuestCount)
				break
			}
		}
	}

	// Create booking
	booking := ServiceBooking{
		ID:              primitive.NewObjectID(),
		BookingReference: asm.generateBookingReference(),
		ServiceID:       serviceObjID,
		CustomerID:      bookingRequest.CustomerID,
		FlightNumber:    bookingRequest.FlightNumber,
		PNR:             bookingRequest.PNR,
		ServiceType:     service.ServiceType,
		AirportCode:     service.AirportCode,
		Terminal:        service.Terminal,
		ServiceDate:     serviceDate,
		ServiceTime:     bookingRequest.ServiceTime,
		Duration:        service.Duration,
		GuestCount:      bookingRequest.GuestCount,
		TotalPrice:      totalPrice,
		Currency:        service.Currency,
		PaymentStatus:   "pending",
		BookingStatus:   "confirmed",
		SpecialRequests: bookingRequest.SpecialRequests,
		ContactInfo:     bookingRequest.ContactInfo,
		CheckInDetails:  CheckInDetails{
			QRCode:     asm.generateQRCode(booking.ID),
			AccessCode: asm.generateAccessCode(),
		},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Save booking
	collection := asm.db.Collection("service_bookings")
	_, err = collection.InsertOne(context.Background(), booking)
	if err != nil {
		log.Printf("Error creating booking: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking"})
		return
	}

	// Update availability
	err = asm.updateServiceAvailability(serviceObjID, serviceDate, bookingRequest.ServiceTime, bookingRequest.GuestCount)
	if err != nil {
		log.Printf("Error updating availability: %v", err)
		// Note: In production, this should trigger a compensation workflow
	}

	// Send confirmation notifications
	go asm.sendBookingConfirmation(booking)

	c.JSON(http.StatusCreated, gin.H{
		"booking":           booking,
		"paymentRequired":   totalPrice > 0,
		"confirmationSent":  true,
	})
}

func (asm *AirportServicesModule) GetBooking(c *gin.Context) {
	bookingRef := c.Param("reference")
	
	collection := asm.db.Collection("service_bookings")
	var booking ServiceBooking
	
	err := collection.FindOne(context.Background(), bson.M{
		"bookingReference": bookingRef,
	}).Decode(&booking)
	
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve booking"})
		}
		return
	}

	// Get service details
	service, err := asm.getServiceByID(booking.ServiceID)
	if err != nil {
		log.Printf("Error getting service details: %v", err)
	}

	response := map[string]interface{}{
		"booking": booking,
	}
	
	if service != nil {
		response["service"] = service
	}

	c.JSON(http.StatusOK, response)
}

func (asm *AirportServicesModule) CancelBooking(c *gin.Context) {
	bookingRef := c.Param("reference")
	
	collection := asm.db.Collection("service_bookings")
	
	// Get booking first
	var booking ServiceBooking
	err := collection.FindOne(context.Background(), bson.M{
		"bookingReference": bookingRef,
	}).Decode(&booking)
	
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve booking"})
		}
		return
	}

	// Check if cancellation is allowed
	if booking.BookingStatus == "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Booking already cancelled"})
		return
	}

	// Check cancellation policy
	service, err := asm.getServiceByID(booking.ServiceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get service details"})
		return
	}

	hoursUntilService := time.Until(booking.ServiceDate).Hours()
	cancellationFee := 0.0
	
	if hoursUntilService < float64(service.CancellationPolicy.FreeCancellation) {
		cancellationFee = service.CancellationPolicy.CancellationFee
	}

	// Update booking status
	update := bson.M{
		"$set": bson.M{
			"bookingStatus": "cancelled",
			"updatedAt":     time.Now(),
		},
	}

	_, err = collection.UpdateOne(context.Background(), bson.M{
		"bookingReference": bookingRef,
	}, update)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel booking"})
		return
	}

	// Restore availability
	err = asm.restoreServiceAvailability(booking.ServiceID, booking.ServiceDate, booking.ServiceTime, booking.GuestCount)
	if err != nil {
		log.Printf("Error restoring availability: %v", err)
	}

	// Calculate refund amount
	refundAmount := booking.TotalPrice - cancellationFee

	// Send cancellation confirmation
	go asm.sendCancellationConfirmation(booking, refundAmount)

	c.JSON(http.StatusOK, gin.H{
		"status":           "cancelled",
		"cancellationFee":  cancellationFee,
		"refundAmount":     refundAmount,
		"refundTimeline":   "5-7 business days",
	})
}

func (asm *AirportServicesModule) CheckInService(c *gin.Context) {
	var checkInRequest struct {
		BookingReference string `json:"bookingReference" binding:"required"`
		QRCode          string `json:"qrCode"`
		AccessCode      string `json:"accessCode"`
	}

	if err := c.ShouldBindJSON(&checkInRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	collection := asm.db.Collection("service_bookings")
	
	// Find and update booking with check-in details
	filter := bson.M{"bookingReference": checkInRequest.BookingReference}
	
	// Verify QR code or access code
	if checkInRequest.QRCode != "" {
		filter["checkInDetails.qrCode"] = checkInRequest.QRCode
	} else if checkInRequest.AccessCode != "" {
		filter["checkInDetails.accessCode"] = checkInRequest.AccessCode
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "QR code or access code required"})
		return
	}

	update := bson.M{
		"$set": bson.M{
			"checkInDetails.checkInTime": time.Now(),
			"bookingStatus": "checked-in",
			"updatedAt":     time.Now(),
		},
	}

	result, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Check-in failed"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid booking or access credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "checked-in",
		"checkInTime":  time.Now(),
		"message":      "Check-in successful. Welcome to the service!",
	})
}

// Lounge-specific methods
func (la *LoungeAccessAPI) GetLoungesByAirport(airportCode string) ([]AirportService, error) {
	// Implementation for getting lounges
	return nil, nil
}

func (la *LoungeAccessAPI) CheckLoungeEligibility(customerID, loungeID string) (bool, error) {
	// Implementation for checking lounge eligibility based on ticket class, status, etc.
	return true, nil
}

// Fast Track Security methods
func (ft *FastTrackAPI) GetSecurityTracks(airportCode, terminal string) ([]AirportService, error) {
	// Implementation for getting fast track security options
	return nil, nil
}

// Ground Transport methods
func (gt *GroundTransportAPI) GetTransportOptions(airportCode string, transportType string) ([]AirportService, error) {
	// Implementation for getting transport options (taxi, bus, train, etc.)
	return nil, nil
}

// Parking methods
func (pa *ParkingAPI) GetParkingOptions(airportCode string, duration int) ([]AirportService, error) {
	// Implementation for getting parking options
	return nil, nil
}

// Helper methods
func (asm *AirportServicesModule) getServiceByID(serviceID primitive.ObjectID) (*AirportService, error) {
	collection := asm.db.Collection("airport_services")
	var service AirportService
	
	err := collection.FindOne(context.Background(), bson.M{"_id": serviceID}).Decode(&service)
	if err != nil {
		return nil, err
	}
	
	return &service, nil
}

func (asm *AirportServicesModule) getServiceAvailability(serviceID primitive.ObjectID, date time.Time) (*ServiceAvailability, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s_%s", serviceID.Hex(), date.Format("2006-01-02"))
	if cached, exists := asm.availabilityCache[cacheKey]; exists && 
		time.Since(cached.LastUpdated) < 10*time.Minute {
		return cached, nil
	}

	// Get from database
	collection := asm.db.Collection("service_availability")
	var availability ServiceAvailability
	
	err := collection.FindOne(context.Background(), bson.M{
		"serviceId": serviceID,
		"date":      date,
	}).Decode(&availability)
	
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Create default availability
			availability = asm.createDefaultAvailability(serviceID, date)
		} else {
			return nil, err
		}
	}

	// Cache the result
	asm.availabilityCache[cacheKey] = &availability
	
	return &availability, nil
}

func (asm *AirportServicesModule) createDefaultAvailability(serviceID primitive.ObjectID, date time.Time) ServiceAvailability {
	// Create default time slots (can be customized per service)
	timeSlots := []TimeSlot{
		{StartTime: "06:00", EndTime: "09:00", Capacity: 50, Available: 50, Price: 100.0, Status: "available"},
		{StartTime: "09:00", EndTime: "12:00", Capacity: 50, Available: 50, Price: 120.0, Status: "available"},
		{StartTime: "12:00", EndTime: "15:00", Capacity: 50, Available: 50, Price: 110.0, Status: "available"},
		{StartTime: "15:00", EndTime: "18:00", Capacity: 50, Available: 50, Price: 130.0, Status: "available"},
		{StartTime: "18:00", EndTime: "21:00", Capacity: 50, Available: 50, Price: 140.0, Status: "available"},
		{StartTime: "21:00", EndTime: "24:00", Capacity: 30, Available: 30, Price: 100.0, Status: "available"},
	}

	return ServiceAvailability{
		ServiceID:     serviceID,
		Date:          date,
		TimeSlots:     timeSlots,
		TotalCapacity: 280,
		Available:     280,
		WaitList:      0,
		LastUpdated:   time.Now(),
	}
}

func (asm *AirportServicesModule) updateServiceAvailability(serviceID primitive.ObjectID, date time.Time, timeSlot string, guestCount int) error {
	collection := asm.db.Collection("service_availability")
	
	filter := bson.M{
		"serviceId": serviceID,
		"date":      date,
	}

	var update bson.M
	if timeSlot != "" {
		// Update specific time slot
		update = bson.M{
			"$inc": bson.M{
				"timeSlots.$[slot].available": -guestCount,
				"available": -guestCount,
			},
			"$set": bson.M{"lastUpdated": time.Now()},
		}
	} else {
		// Update overall availability
		update = bson.M{
			"$inc": bson.M{"available": -guestCount},
			"$set": bson.M{"lastUpdated": time.Now()},
		}
	}

	arrayFilters := options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"slot.startTime": timeSlot},
		},
	}

	updateOptions := options.Update().SetArrayFilters(arrayFilters)
	
	_, err := collection.UpdateOne(context.Background(), filter, update, updateOptions)
	
	// Clear cache
	cacheKey := fmt.Sprintf("%s_%s", serviceID.Hex(), date.Format("2006-01-02"))
	delete(asm.availabilityCache, cacheKey)
	
	return err
}

func (asm *AirportServicesModule) restoreServiceAvailability(serviceID primitive.ObjectID, date time.Time, timeSlot string, guestCount int) error {
	return asm.updateServiceAvailability(serviceID, date, timeSlot, -guestCount) // Negative to restore
}

func (asm *AirportServicesModule) generateBookingReference() string {
	return fmt.Sprintf("ASB%d", time.Now().Unix())
}

func (asm *AirportServicesModule) generateQRCode(bookingID primitive.ObjectID) string {
	return fmt.Sprintf("QR_%s_%d", bookingID.Hex(), time.Now().Unix())
}

func (asm *AirportServicesModule) generateAccessCode() string {
	return fmt.Sprintf("%06d", time.Now().Unix()%1000000)
}

func (asm *AirportServicesModule) sendBookingConfirmation(booking ServiceBooking) {
	// Implementation for sending booking confirmation
	log.Printf("Sending booking confirmation for %s to %s", booking.BookingReference, booking.ContactInfo.Email)
}

func (asm *AirportServicesModule) sendCancellationConfirmation(booking ServiceBooking, refundAmount float64) {
	// Implementation for sending cancellation confirmation
	log.Printf("Sending cancellation confirmation for %s, refund: %.2f", booking.BookingReference, refundAmount)
}

// RegisterRoutes registers all airport services routes
func (asm *AirportServicesModule) RegisterRoutes(router *gin.Engine) {
	airportRoutes := router.Group("/api/v1/airport-services")
	{
		airportRoutes.GET("/search", asm.SearchServices)
		airportRoutes.POST("/book", asm.BookService)
		airportRoutes.GET("/booking/:reference", asm.GetBooking)
		airportRoutes.POST("/booking/:reference/cancel", asm.CancelBooking)
		airportRoutes.POST("/checkin", asm.CheckInService)
	}
} 