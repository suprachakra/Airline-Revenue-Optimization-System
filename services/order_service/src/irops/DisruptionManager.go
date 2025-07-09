package irops

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/iaros/common/logging"
	"github.com/iaros/services/order_service/models"
)

// DisruptionManager handles irregular operations and flight disruptions
type DisruptionManager struct {
	// Core services
	orderService       *models.OrderService
	flightService      FlightService
	notificationService NotificationService
	inventoryService   InventoryService
	
	// Disruption management
	disruptionDetector *DisruptionDetector
	rebookingEngine    *RebookingEngine
	compensationEngine *CompensationEngine
	
	// External integrations
	weatherService     WeatherService
	airportService     AirportService
	aircraftService    AircraftService
	
	// Automation and ML
	mlOptimizer        *MLDisruptionOptimizer
	automationRules    *AutomationRuleEngine
	
	// Storage and metrics
	storage            DisruptionStorage
	metricsCollector   MetricsCollector
	logger             logging.Logger
}

// Disruption represents a flight disruption event
type Disruption struct {
	ID               string                 `json:"id"`
	Type             DisruptionType         `json:"type"`
	Severity         SeverityLevel          `json:"severity"`
	Status           DisruptionStatus       `json:"status"`
	
	// Flight details
	FlightID         string                 `json:"flight_id"`
	FlightNumber     string                 `json:"flight_number"`
	Route            Route                  `json:"route"`
	ScheduledDeparture time.Time            `json:"scheduled_departure"`
	ScheduledArrival   time.Time            `json:"scheduled_arrival"`
	
	// Disruption details
	Cause            DisruptionCause        `json:"cause"`
	Description      string                 `json:"description"`
	Impact           DisruptionImpact       `json:"impact"`
	
	// Timing
	DetectedAt       time.Time              `json:"detected_at"`
	EstimatedDelay   time.Duration          `json:"estimated_delay"`
	NewDeparture     *time.Time             `json:"new_departure,omitempty"`
	NewArrival       *time.Time             `json:"new_arrival,omitempty"`
	
	// Affected passengers
	AffectedBookings []string               `json:"affected_bookings"`
	PassengerCount   int                    `json:"passenger_count"`
	
	// Response actions
	AutoActions      []AutomatedAction      `json:"auto_actions"`
	ManualActions    []ManualAction         `json:"manual_actions"`
	
	// Resolution
	ResolutionPlan   *ResolutionPlan        `json:"resolution_plan,omitempty"`
	ResolvedAt       *time.Time             `json:"resolved_at,omitempty"`
	
	// Metadata
	Metadata         map[string]interface{} `json:"metadata"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// PassengerReaccommodation represents rebooking options for affected passengers
type PassengerReaccommodation struct {
	BookingID        string                 `json:"booking_id"`
	PassengerID      string                 `json:"passenger_id"`
	Status           ReaccommodationStatus  `json:"status"`
	
	// Original booking
	OriginalFlight   FlightDetails          `json:"original_flight"`
	
	// Rebooking options
	RebookingOptions []RebookingOption      `json:"rebooking_options"`
	SelectedOption   *RebookingOption       `json:"selected_option,omitempty"`
	
	// Compensation
	Compensation     *CompensationPackage   `json:"compensation,omitempty"`
	
	// Communication
	NotificationsSent []NotificationRecord  `json:"notifications_sent"`
	
	// Timing
	ProcessedAt      time.Time              `json:"processed_at"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
}

// RebookingOption represents an alternative flight option
type RebookingOption struct {
	ID               string                 `json:"id"`
	Type             RebookingType          `json:"type"`
	Priority         int                    `json:"priority"`
	
	// Flight details
	FlightID         string                 `json:"flight_id"`
	FlightNumber     string                 `json:"flight_number"`
	Departure        time.Time              `json:"departure"`
	Arrival          time.Time              `json:"arrival"`
	Route            Route                  `json:"route"`
	Aircraft         string                 `json:"aircraft"`
	
	// Booking details
	SeatAvailability int                    `json:"seat_availability"`
	CabinClass       string                 `json:"cabin_class"`
	FareClass        string                 `json:"fare_class"`
	
	// Cost and policies
	AdditionalCost   float64                `json:"additional_cost"`
	RefundAmount     float64                `json:"refund_amount"`
	WaiverApplied    bool                   `json:"waiver_applied"`
	
	// Experience impact
	DelayImpact      time.Duration          `json:"delay_impact"`
	ConnectionCount  int                    `json:"connection_count"`
	ComfortScore     float64                `json:"comfort_score"`
	
	// Acceptance tracking
	OfferExpiresAt   time.Time              `json:"offer_expires_at"`
	AcceptedAt       *time.Time             `json:"accepted_at,omitempty"`
}

// Enums
type DisruptionType string
const (
	DisruptionDelay       DisruptionType = "delay"
	DisruptionCancellation DisruptionType = "cancellation"
	DisruptionDiversion    DisruptionType = "diversion"
	DisruptionGateChange   DisruptionType = "gate_change"
	DisruptionAircraftChange DisruptionType = "aircraft_change"
	DisruptionCrewIssue    DisruptionType = "crew_issue"
)

type SeverityLevel string
const (
	SeverityLow        SeverityLevel = "low"       // <30 min delay
	SeverityMedium     SeverityLevel = "medium"    // 30-120 min delay
	SeverityHigh       SeverityLevel = "high"      // >120 min delay
	SeverityCritical   SeverityLevel = "critical"  // Cancellation
)

type DisruptionCause string
const (
	CauseWeather       DisruptionCause = "weather"
	CauseATC           DisruptionCause = "air_traffic_control"
	CauseMechanical    DisruptionCause = "mechanical"
	CauseCrew          DisruptionCause = "crew"
	CauseAirport       DisruptionCause = "airport"
	CauseSecurity      DisruptionCause = "security"
	CauseConnectivity  DisruptionCause = "connectivity"
	CauseOther         DisruptionCause = "other"
)

type DisruptionStatus string
const (
	StatusDetected     DisruptionStatus = "detected"
	StatusAnalyzing    DisruptionStatus = "analyzing"
	StatusProcessing   DisruptionStatus = "processing"
	StatusResolved     DisruptionStatus = "resolved"
	StatusEscalated    DisruptionStatus = "escalated"
)

type ReaccommodationStatus string
const (
	ReaccommodationPending   ReaccommodationStatus = "pending"
	ReaccommodationOffered   ReaccommodationStatus = "offered"
	ReaccommodationAccepted  ReaccommodationStatus = "accepted"
	ReaccommodationCompleted ReaccommodationStatus = "completed"
	ReaccommodationDeclined  ReaccommodationStatus = "declined"
)

type RebookingType string
const (
	RebookingSameDay     RebookingType = "same_day"
	RebookingNextDay     RebookingType = "next_day"
	RebookingAlternateRoute RebookingType = "alternate_route"
	RebookingPartner     RebookingType = "partner_airline"
	RebookingRefund      RebookingType = "refund"
)

// NewDisruptionManager creates a new disruption management system
func NewDisruptionManager(orderService *models.OrderService, storage DisruptionStorage) *DisruptionManager {
	return &DisruptionManager{
		orderService:       orderService,
		disruptionDetector: NewDisruptionDetector(),
		rebookingEngine:    NewRebookingEngine(),
		compensationEngine: NewCompensationEngine(),
		mlOptimizer:        NewMLDisruptionOptimizer(),
		automationRules:    NewAutomationRuleEngine(),
		storage:           storage,
		metricsCollector:  NewMetricsCollector(),
		logger:            logging.GetLogger("disruption-manager"),
	}
}

// ProcessDisruption handles a flight disruption from detection to resolution
func (dm *DisruptionManager) ProcessDisruption(ctx context.Context, flightID string, disruptionData DisruptionEvent) (*Disruption, error) {
	dm.logger.Info("Processing flight disruption", "flight_id", flightID, "type", disruptionData.Type)
	
	// Create disruption record
	disruption := &Disruption{
		ID:               fmt.Sprintf("disrupt_%d", time.Now().Unix()),
		Type:             disruptionData.Type,
		Severity:         dm.calculateSeverity(disruptionData),
		Status:           StatusDetected,
		FlightID:         flightID,
		FlightNumber:     disruptionData.FlightNumber,
		Cause:            disruptionData.Cause,
		Description:      disruptionData.Description,
		DetectedAt:       time.Now(),
		EstimatedDelay:   disruptionData.EstimatedDelay,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	
	// Get flight details
	flightDetails, err := dm.flightService.GetFlightDetails(ctx, flightID)
	if err != nil {
		return nil, fmt.Errorf("failed to get flight details: %w", err)
	}
	
	disruption.Route = flightDetails.Route
	disruption.ScheduledDeparture = flightDetails.ScheduledDeparture
	disruption.ScheduledArrival = flightDetails.ScheduledArrival
	
	// Calculate new schedule if delayed
	if disruption.Type == DisruptionDelay && disruption.EstimatedDelay > 0 {
		newDeparture := disruption.ScheduledDeparture.Add(disruption.EstimatedDelay)
		newArrival := disruption.ScheduledArrival.Add(disruption.EstimatedDelay)
		disruption.NewDeparture = &newDeparture
		disruption.NewArrival = &newArrival
	}
	
	// Get affected bookings
	affectedBookings, err := dm.getAffectedBookings(ctx, flightID)
	if err != nil {
		return nil, fmt.Errorf("failed to get affected bookings: %w", err)
	}
	
	disruption.AffectedBookings = affectedBookings
	disruption.PassengerCount = len(affectedBookings)
	
	// Calculate disruption impact
	disruption.Impact = dm.calculateDisruptionImpact(disruption)
	
	// Store disruption
	if err := dm.storage.StoreDisruption(ctx, disruption); err != nil {
		return nil, fmt.Errorf("failed to store disruption: %w", err)
	}
	
	// Start automated processing
	go dm.processDisruptionAsync(ctx, disruption)
	
	dm.logger.Info("Disruption processing initiated", 
		"disruption_id", disruption.ID,
		"affected_passengers", disruption.PassengerCount,
		"severity", disruption.Severity)
	
	return disruption, nil
}

// processDisruptionAsync handles the asynchronous processing of disruption
func (dm *DisruptionManager) processDisruptionAsync(ctx context.Context, disruption *Disruption) {
	dm.logger.Info("Starting async disruption processing", "disruption_id", disruption.ID)
	
	// Update status to analyzing
	disruption.Status = StatusAnalyzing
	disruption.UpdatedAt = time.Now()
	dm.storage.UpdateDisruption(ctx, disruption)
	
	// Generate resolution plan using ML optimization
	resolutionPlan, err := dm.mlOptimizer.GenerateResolutionPlan(ctx, disruption)
	if err != nil {
		dm.logger.Error("ML optimization failed, using fallback", "error", err)
		resolutionPlan = dm.generateFallbackResolutionPlan(disruption)
	}
	
	disruption.ResolutionPlan = resolutionPlan
	disruption.Status = StatusProcessing
	disruption.UpdatedAt = time.Now()
	dm.storage.UpdateDisruption(ctx, disruption)
	
	// Execute automated actions
	dm.executeAutomatedActions(ctx, disruption, resolutionPlan)
	
	// Process passenger reaccommodations
	dm.processPassengerReaccommodations(ctx, disruption)
	
	// Send notifications
	dm.sendDisruptionNotifications(ctx, disruption)
	
	// Update metrics
	dm.updateDisruptionMetrics(ctx, disruption)
	
	// Check if resolution is complete
	if dm.isDisruptionResolved(disruption) {
		disruption.Status = StatusResolved
		now := time.Now()
		disruption.ResolvedAt = &now
		disruption.UpdatedAt = now
		dm.storage.UpdateDisruption(ctx, disruption)
		
		dm.logger.Info("Disruption resolved", "disruption_id", disruption.ID)
	}
}

// GetRebookingOptions generates rebooking options for a passenger
func (dm *DisruptionManager) GetRebookingOptions(ctx context.Context, disruptionID, bookingID string) ([]*RebookingOption, error) {
	disruption, err := dm.storage.GetDisruption(ctx, disruptionID)
	if err != nil {
		return nil, err
	}
	
	// Get booking details
	booking, err := dm.orderService.GetBooking(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get booking: %w", err)
	}
	
	// Generate rebooking options
	options, err := dm.rebookingEngine.GenerateOptions(ctx, disruption, booking)
	if err != nil {
		return nil, fmt.Errorf("failed to generate rebooking options: %w", err)
	}
	
	// Sort options by priority
	sort.Slice(options, func(i, j int) bool {
		return options[i].Priority < options[j].Priority
	})
	
	dm.logger.Info("Generated rebooking options", 
		"disruption_id", disruptionID,
		"booking_id", bookingID,
		"options_count", len(options))
	
	return options, nil
}

// AcceptRebookingOption processes passenger acceptance of a rebooking option
func (dm *DisruptionManager) AcceptRebookingOption(ctx context.Context, disruptionID, bookingID, optionID string) (*PassengerReaccommodation, error) {
	dm.logger.Info("Processing rebooking acceptance", 
		"disruption_id", disruptionID,
		"booking_id", bookingID,
		"option_id", optionID)
	
	// Get rebooking option
	option, err := dm.storage.GetRebookingOption(ctx, optionID)
	if err != nil {
		return nil, err
	}
	
	// Validate option is still available
	if time.Now().After(option.OfferExpiresAt) {
		return nil, fmt.Errorf("rebooking option has expired")
	}
	
	// Check seat availability
	available, err := dm.inventoryService.CheckSeatAvailability(ctx, option.FlightID, option.CabinClass, 1)
	if err != nil || !available {
		return nil, fmt.Errorf("seats no longer available")
	}
	
	// Create new booking
	newBooking, err := dm.createRebookedBooking(ctx, bookingID, option)
	if err != nil {
		return nil, fmt.Errorf("failed to create new booking: %w", err)
	}
	
	// Cancel original booking
	if err := dm.orderService.CancelBooking(ctx, bookingID, "disruption_rebooking"); err != nil {
		dm.logger.Error("Failed to cancel original booking", "error", err)
		// Continue processing but log the error
	}
	
	// Create reaccommodation record
	now := time.Now()
	option.AcceptedAt = &now
	
	reaccommodation := &PassengerReaccommodation{
		BookingID:       bookingID,
		Status:          ReaccommodationAccepted,
		SelectedOption:  option,
		ProcessedAt:     time.Now(),
		CompletedAt:     &now,
	}
	
	// Calculate compensation if applicable
	compensation, err := dm.compensationEngine.CalculateCompensation(ctx, disruptionID, bookingID)
	if err != nil {
		dm.logger.Warn("Compensation calculation failed", "error", err)
	} else {
		reaccommodation.Compensation = compensation
	}
	
	// Store reaccommodation
	if err := dm.storage.StoreReaccommodation(ctx, reaccommodation); err != nil {
		return nil, err
	}
	
	// Send confirmation notification
	dm.sendRebookingConfirmation(ctx, newBooking, reaccommodation)
	
	dm.logger.Info("Rebooking completed successfully", 
		"disruption_id", disruptionID,
		"original_booking", bookingID,
		"new_booking", newBooking.ID)
	
	return reaccommodation, nil
}

// GetDisruptionStatus returns current status and metrics for a disruption
func (dm *DisruptionManager) GetDisruptionStatus(ctx context.Context, disruptionID string) (*DisruptionStatusReport, error) {
	disruption, err := dm.storage.GetDisruption(ctx, disruptionID)
	if err != nil {
		return nil, err
	}
	
	// Get reaccommodation stats
	reaccommodations, err := dm.storage.GetReaccommodationsByDisruption(ctx, disruptionID)
	if err != nil {
		return nil, err
	}
	
	// Calculate metrics
	totalPassengers := len(disruption.AffectedBookings)
	processedCount := 0
	completedCount := 0
	pendingCount := 0
	
	for _, reaccommodation := range reaccommodations {
		switch reaccommodation.Status {
		case ReaccommodationCompleted:
			completedCount++
			processedCount++
		case ReaccommodationAccepted:
			processedCount++
		case ReaccommodationPending:
			pendingCount++
		}
	}
	
	report := &DisruptionStatusReport{
		DisruptionID:      disruptionID,
		Status:           disruption.Status,
		Severity:         disruption.Severity,
		TotalPassengers:  totalPassengers,
		ProcessedCount:   processedCount,
		CompletedCount:   completedCount,
		PendingCount:     pendingCount,
		ProcessingRate:   float64(processedCount) / float64(totalPassengers) * 100,
		CompletionRate:   float64(completedCount) / float64(totalPassengers) * 100,
		EstimatedCompletion: dm.calculateEstimatedCompletion(disruption, reaccommodations),
		LastUpdated:      disruption.UpdatedAt,
	}
	
	return report, nil
}

// Helper methods

func (dm *DisruptionManager) calculateSeverity(event DisruptionEvent) SeverityLevel {
	switch event.Type {
	case DisruptionCancellation:
		return SeverityCritical
	case DisruptionDelay:
		if event.EstimatedDelay >= 2*time.Hour {
			return SeverityHigh
		} else if event.EstimatedDelay >= 30*time.Minute {
			return SeverityMedium
		}
		return SeverityLow
	case DisruptionDiversion:
		return SeverityHigh
	default:
		return SeverityMedium
	}
}

func (dm *DisruptionManager) calculateDisruptionImpact(disruption *Disruption) DisruptionImpact {
	impact := DisruptionImpact{
		PassengerCount:    disruption.PassengerCount,
		EstimatedDelay:    disruption.EstimatedDelay,
		OperationalCost:   dm.estimateOperationalCost(disruption),
		CustomerSatisfactionImpact: dm.estimateCSATImpact(disruption),
	}
	
	// Calculate financial impact
	switch disruption.Severity {
	case SeverityCritical:
		impact.FinancialImpact = float64(disruption.PassengerCount) * 400 // $400 per passenger for cancellation
	case SeverityHigh:
		impact.FinancialImpact = float64(disruption.PassengerCount) * 200 // $200 per passenger for major delay
	case SeverityMedium:
		impact.FinancialImpact = float64(disruption.PassengerCount) * 100 // $100 per passenger for medium delay
	default:
		impact.FinancialImpact = float64(disruption.PassengerCount) * 50 // $50 per passenger for minor delay
	}
	
	return impact
}

func (dm *DisruptionManager) getAffectedBookings(ctx context.Context, flightID string) ([]string, error) {
	// Query order service for bookings on this flight
	bookings, err := dm.orderService.GetBookingsByFlight(ctx, flightID)
	if err != nil {
		return nil, err
	}
	
	var bookingIDs []string
	for _, booking := range bookings {
		bookingIDs = append(bookingIDs, booking.ID)
	}
	
	return bookingIDs, nil
}

func (dm *DisruptionManager) generateFallbackResolutionPlan(disruption *Disruption) *ResolutionPlan {
	plan := &ResolutionPlan{
		Strategy:        "standard_rebooking",
		PriorityActions: []string{"notify_passengers", "generate_rebooking_options"},
		Timeline:        "immediate",
		AutomationLevel: "partial",
		EstimatedResolutionTime: time.Duration(disruption.PassengerCount/10) * time.Minute, // 10 passengers per minute
	}
	
	// Add strategy based on disruption type
	switch disruption.Type {
	case DisruptionCancellation:
		plan.Strategy = "cancellation_recovery"
		plan.PriorityActions = append(plan.PriorityActions, "find_alternate_flights", "process_refunds")
	case DisruptionDelay:
		if disruption.Severity == SeverityHigh {
			plan.Strategy = "delay_mitigation"
			plan.PriorityActions = append(plan.PriorityActions, "provide_amenities", "explore_alternatives")
		}
	}
	
	return plan
}

func (dm *DisruptionManager) executeAutomatedActions(ctx context.Context, disruption *Disruption, plan *ResolutionPlan) {
	for _, action := range plan.PriorityActions {
		switch action {
		case "notify_passengers":
			dm.sendInitialNotifications(ctx, disruption)
		case "generate_rebooking_options":
			dm.generateAllRebookingOptions(ctx, disruption)
		case "provide_amenities":
			dm.arrangePassengerAmenities(ctx, disruption)
		case "find_alternate_flights":
			dm.findAlternateFlights(ctx, disruption)
		}
		
		// Log automated action
		automatedAction := AutomatedAction{
			Action:      action,
			ExecutedAt:  time.Now(),
			Status:      "completed",
		}
		disruption.AutoActions = append(disruption.AutoActions, automatedAction)
	}
	
	dm.storage.UpdateDisruption(ctx, disruption)
}

func (dm *DisruptionManager) processPassengerReaccommodations(ctx context.Context, disruption *Disruption) {
	for _, bookingID := range disruption.AffectedBookings {
		// Generate rebooking options for each passenger
		options, err := dm.GetRebookingOptions(ctx, disruption.ID, bookingID)
		if err != nil {
			dm.logger.Error("Failed to generate rebooking options", "booking_id", bookingID, "error", err)
			continue
		}
		
		// Create reaccommodation record
		reaccommodation := &PassengerReaccommodation{
			BookingID:        bookingID,
			Status:           ReaccommodationOffered,
			RebookingOptions: options,
			ProcessedAt:      time.Now(),
		}
		
		if err := dm.storage.StoreReaccommodation(ctx, reaccommodation); err != nil {
			dm.logger.Error("Failed to store reaccommodation", "booking_id", bookingID, "error", err)
		}
		
		// Send rebooking options to passenger
		dm.sendRebookingOptions(ctx, bookingID, options)
	}
}

func (dm *DisruptionManager) createRebookedBooking(ctx context.Context, originalBookingID string, option *RebookingOption) (*models.Booking, error) {
	// Get original booking details
	originalBooking, err := dm.orderService.GetBooking(ctx, originalBookingID)
	if err != nil {
		return nil, err
	}
	
	// Create new booking with selected option
	newBookingRequest := &models.BookingRequest{
		FlightID:     option.FlightID,
		PassengerIDs: originalBooking.PassengerIDs,
		CabinClass:   option.CabinClass,
		FareClass:    option.FareClass,
		
		// Rebooking metadata
		IsRebooking:        true,
		OriginalBookingID:  originalBookingID,
		RebookingReason:    "flight_disruption",
		WaiverApplied:      option.WaiverApplied,
		AdditionalCost:     option.AdditionalCost,
	}
	
	newBooking, err := dm.orderService.CreateBooking(ctx, newBookingRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create rebooking: %w", err)
	}
	
	return newBooking, nil
}

// Support types and interfaces

type DisruptionEvent struct {
	Type            DisruptionType    `json:"type"`
	FlightNumber    string           `json:"flight_number"`
	Cause           DisruptionCause  `json:"cause"`
	Description     string           `json:"description"`
	EstimatedDelay  time.Duration    `json:"estimated_delay"`
	Metadata        map[string]interface{} `json:"metadata"`
}

type DisruptionImpact struct {
	PassengerCount             int           `json:"passenger_count"`
	FinancialImpact           float64       `json:"financial_impact"`
	OperationalCost           float64       `json:"operational_cost"`
	EstimatedDelay            time.Duration `json:"estimated_delay"`
	CustomerSatisfactionImpact float64      `json:"customer_satisfaction_impact"`
}

type ResolutionPlan struct {
	Strategy                string        `json:"strategy"`
	PriorityActions         []string      `json:"priority_actions"`
	Timeline                string        `json:"timeline"`
	AutomationLevel         string        `json:"automation_level"`
	EstimatedResolutionTime time.Duration `json:"estimated_resolution_time"`
}

type DisruptionStatusReport struct {
	DisruptionID        string        `json:"disruption_id"`
	Status              DisruptionStatus `json:"status"`
	Severity            SeverityLevel `json:"severity"`
	TotalPassengers     int           `json:"total_passengers"`
	ProcessedCount      int           `json:"processed_count"`
	CompletedCount      int           `json:"completed_count"`
	PendingCount        int           `json:"pending_count"`
	ProcessingRate      float64       `json:"processing_rate"`
	CompletionRate      float64       `json:"completion_rate"`
	EstimatedCompletion *time.Time    `json:"estimated_completion,omitempty"`
	LastUpdated         time.Time     `json:"last_updated"`
}

type AutomatedAction struct {
	Action     string    `json:"action"`
	ExecutedAt time.Time `json:"executed_at"`
	Status     string    `json:"status"`
	Result     string    `json:"result,omitempty"`
}

type ManualAction struct {
	Action      string    `json:"action"`
	ExecutedBy  string    `json:"executed_by"`
	ExecutedAt  time.Time `json:"executed_at"`
	Description string    `json:"description"`
}

type Route struct {
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
	Stops       []string `json:"stops,omitempty"`
}

type FlightDetails struct {
	FlightID           string    `json:"flight_id"`
	FlightNumber       string    `json:"flight_number"`
	Route              Route     `json:"route"`
	ScheduledDeparture time.Time `json:"scheduled_departure"`
	ScheduledArrival   time.Time `json:"scheduled_arrival"`
	Aircraft           string    `json:"aircraft"`
}

type CompensationPackage struct {
	Type        string  `json:"type"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
	VoucherCode string  `json:"voucher_code,omitempty"`
}

type NotificationRecord struct {
	Channel   string    `json:"channel"`
	SentAt    time.Time `json:"sent_at"`
	Status    string    `json:"status"`
	MessageID string    `json:"message_id"`
}

// Interface definitions
type DisruptionStorage interface {
	StoreDisruption(ctx context.Context, disruption *Disruption) error
	GetDisruption(ctx context.Context, disruptionID string) (*Disruption, error)
	UpdateDisruption(ctx context.Context, disruption *Disruption) error
	StoreReaccommodation(ctx context.Context, reaccommodation *PassengerReaccommodation) error
	GetReaccommodationsByDisruption(ctx context.Context, disruptionID string) ([]*PassengerReaccommodation, error)
	GetRebookingOption(ctx context.Context, optionID string) (*RebookingOption, error)
}

type FlightService interface {
	GetFlightDetails(ctx context.Context, flightID string) (*FlightDetails, error)
}

type NotificationService interface {
	SendDisruptionNotification(ctx context.Context, bookingID string, disruption *Disruption) error
	SendRebookingOptions(ctx context.Context, bookingID string, options []*RebookingOption) error
}

type InventoryService interface {
	CheckSeatAvailability(ctx context.Context, flightID, cabinClass string, seatCount int) (bool, error)
}

// Additional service interfaces
type WeatherService interface {
	GetWeatherImpact(ctx context.Context, route Route) (*WeatherImpact, error)
}

type AirportService interface {
	GetAirportStatus(ctx context.Context, airportCode string) (*AirportStatus, error)
}

type AircraftService interface {
	GetAircraftStatus(ctx context.Context, aircraftID string) (*AircraftStatus, error)
}

type MetricsCollector interface {
	RecordDisruption(disruption *Disruption)
	RecordResolutionTime(disruptionID string, resolutionTime time.Duration)
}

// Placeholder component constructors
func NewDisruptionDetector() *DisruptionDetector { return &DisruptionDetector{} }
func NewRebookingEngine() *RebookingEngine { return &RebookingEngine{} }
func NewCompensationEngine() *CompensationEngine { return &CompensationEngine{} }
func NewMLDisruptionOptimizer() *MLDisruptionOptimizer { return &MLDisruptionOptimizer{} }
func NewAutomationRuleEngine() *AutomationRuleEngine { return &AutomationRuleEngine{} }
func NewMetricsCollector() MetricsCollector { return &MockMetricsCollector{} }

// Placeholder component types
type DisruptionDetector struct{}
type RebookingEngine struct{}
type CompensationEngine struct{}
type MLDisruptionOptimizer struct{}
type AutomationRuleEngine struct{}
type MockMetricsCollector struct{}

// Additional placeholder support types
type WeatherImpact struct {
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

type AirportStatus struct {
	Code   string `json:"code"`
	Status string `json:"status"`
	Delays string `json:"delays"`
}

type AircraftStatus struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Issues []string `json:"issues"`
}

// Additional placeholder methods
func (dm *DisruptionManager) estimateOperationalCost(disruption *Disruption) float64 {
	// Simplified cost calculation
	return float64(disruption.PassengerCount) * 150 // $150 per passenger operational cost
}

func (dm *DisruptionManager) estimateCSATImpact(disruption *Disruption) float64 {
	// Simplified CSAT impact calculation
	switch disruption.Severity {
	case SeverityCritical:
		return -2.5 // -2.5 point CSAT impact
	case SeverityHigh:
		return -1.5
	case SeverityMedium:
		return -0.8
	default:
		return -0.3
	}
}

func (dm *DisruptionManager) isDisruptionResolved(disruption *Disruption) bool {
	// Check if all passengers have been processed
	return len(disruption.AffectedBookings) > 0 // Simplified check
}

func (dm *DisruptionManager) calculateEstimatedCompletion(disruption *Disruption, reaccommodations []*PassengerReaccommodation) *time.Time {
	// Simplified estimation
	remaining := len(disruption.AffectedBookings) - len(reaccommodations)
	if remaining <= 0 {
		return nil
	}
	
	estimatedMinutes := remaining / 2 // 2 passengers per minute processing rate
	estimated := time.Now().Add(time.Duration(estimatedMinutes) * time.Minute)
	return &estimated
}

// Additional placeholder notification methods
func (dm *DisruptionManager) sendDisruptionNotifications(ctx context.Context, disruption *Disruption) {}
func (dm *DisruptionManager) sendInitialNotifications(ctx context.Context, disruption *Disruption) {}
func (dm *DisruptionManager) sendRebookingOptions(ctx context.Context, bookingID string, options []*RebookingOption) {}
func (dm *DisruptionManager) sendRebookingConfirmation(ctx context.Context, booking *models.Booking, reaccommodation *PassengerReaccommodation) {}

// Additional placeholder processing methods
func (dm *DisruptionManager) generateAllRebookingOptions(ctx context.Context, disruption *Disruption) {}
func (dm *DisruptionManager) arrangePassengerAmenities(ctx context.Context, disruption *Disruption) {}
func (dm *DisruptionManager) findAlternateFlights(ctx context.Context, disruption *Disruption) {}
func (dm *DisruptionManager) updateDisruptionMetrics(ctx context.Context, disruption *Disruption) {}

// Mock metrics collector methods
func (mc *MockMetricsCollector) RecordDisruption(disruption *Disruption) {}
func (mc *MockMetricsCollector) RecordResolutionTime(disruptionID string, resolutionTime time.Duration) {}

// Component method implementations for placeholders
func (re *RebookingEngine) GenerateOptions(ctx context.Context, disruption *Disruption, booking *models.Booking) ([]*RebookingOption, error) {
	// Simplified rebooking option generation
	options := []*RebookingOption{
		{
			ID:           "option_1",
			Type:         RebookingSameDay,
			Priority:     1,
			FlightNumber: "FL1001",
			DelayImpact:  2 * time.Hour,
		},
		{
			ID:           "option_2", 
			Type:         RebookingNextDay,
			Priority:     2,
			FlightNumber: "FL1002",
			DelayImpact:  24 * time.Hour,
		},
	}
	return options, nil
}

func (ce *CompensationEngine) CalculateCompensation(ctx context.Context, disruptionID, bookingID string) (*CompensationPackage, error) {
	// Simplified compensation calculation
	return &CompensationPackage{
		Type:        "monetary",
		Amount:      200.0,
		Currency:    "USD",
		Description: "Flight disruption compensation",
	}, nil
}

func (ml *MLDisruptionOptimizer) GenerateResolutionPlan(ctx context.Context, disruption *Disruption) (*ResolutionPlan, error) {
	// Simplified ML resolution plan
	return &ResolutionPlan{
		Strategy:                "ml_optimized",
		PriorityActions:         []string{"notify_passengers", "generate_rebooking_options"},
		Timeline:                "immediate",
		AutomationLevel:         "high",
		EstimatedResolutionTime: 30 * time.Minute,
	}, nil
} 