package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"iaros/distribution_service/src/models"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/sony/gobreaker"
	"gorm.io/gorm"
)

// GDSService handles GDS integrations
type GDSService struct {
	db             *gorm.DB
	sessionManager *SessionManager
	httpClient     *resty.Client
	circuitBreaker *gobreaker.CircuitBreaker
	configurations map[models.GDSProvider]*GDSConfiguration
}

// GDSConfiguration holds GDS-specific configuration
type GDSConfiguration struct {
	Provider    models.GDSProvider
	BaseURL     string
	Username    string
	Password    string
	PseudoCity  string
	OfficeID    string
	Timeout     time.Duration
	RetryCount  int
	Endpoints   map[string]string
	Headers     map[string]string
	Features    map[string]bool
}

// NewGDSService creates a new GDS service
func NewGDSService(db *gorm.DB, sessionManager *SessionManager) *GDSService {
	// Configure circuit breaker
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "GDS_Service",
		MaxRequests: 3,
		Interval:    30 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Printf("Circuit breaker '%s' changed from '%s' to '%s'", name, from, to)
		},
	})

	// Initialize HTTP client
	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetRetryCount(3)
	client.SetRetryWaitTime(1 * time.Second)
	client.SetRetryMaxWaitTime(5 * time.Second)

	return &GDSService{
		db:             db,
		sessionManager: sessionManager,
		httpClient:     client,
		circuitBreaker: cb,
		configurations: make(map[models.GDSProvider]*GDSConfiguration),
	}
}

// InitializeGDSConfigurations sets up GDS configurations
func (s *GDSService) InitializeGDSConfigurations() {
	// Amadeus configuration
	s.configurations[models.AmadeusGDS] = &GDSConfiguration{
		Provider:    models.AmadeusGDS,
		BaseURL:     "https://test.api.amadeus.com/v2",
		Timeout:     30 * time.Second,
		RetryCount:  3,
		Endpoints: map[string]string{
			"token":           "/security/oauth2/token",
			"flight_search":   "/shopping/flight-offers",
			"flight_price":    "/shopping/flight-offers/pricing",
			"flight_booking":  "/booking/flight-orders",
			"order_retrieve":  "/booking/flight-orders/{orderId}",
			"order_cancel":    "/booking/flight-orders/{orderId}/cancel",
		},
		Headers: map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		},
		Features: map[string]bool{
			"branded_fares":      true,
			"ancillary_services": true,
			"seat_maps":          true,
			"fare_rules":         true,
			"schedule_change":    true,
		},
	}

	// Sabre configuration
	s.configurations[models.SabreGDS] = &GDSConfiguration{
		Provider:    models.SabreGDS,
		BaseURL:     "https://api.test.sabre.com/v2",
		Timeout:     30 * time.Second,
		RetryCount:  3,
		Endpoints: map[string]string{
			"token":           "/auth/token",
			"bargain_finder":  "/shop/flights/search",
			"enhanced_search": "/shop/flights/search/enhanced",
			"price_quote":     "/shop/flights/price",
			"create_booking":  "/passenger/reservation",
			"get_booking":     "/passenger/reservation/{recordLocator}",
			"cancel_booking":  "/passenger/reservation/{recordLocator}/cancel",
		},
		Headers: map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		},
		Features: map[string]bool{
			"bargain_finder":     true,
			"enhanced_search":    true,
			"calendar_search":    true,
			"multi_city":         true,
			"corporate_discounts": true,
		},
	}

	// Travelport configuration
	s.configurations[models.TravelportGDS] = &GDSConfiguration{
		Provider:    models.TravelportGDS,
		BaseURL:     "https://api.travelport.com/v1",
		Timeout:     30 * time.Second,
		RetryCount:  3,
		Endpoints: map[string]string{
			"authenticate":    "/system/authenticate",
			"air_search":      "/air/search",
			"air_price":       "/air/price",
			"air_book":        "/air/book",
			"universal_record": "/universal/record",
			"air_cancel":      "/air/cancel",
		},
		Headers: map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		},
		Features: map[string]bool{
			"universal_record": true,
			"air_merchandising": true,
			"hotel_search":     true,
			"car_search":       true,
			"rail_search":      true,
		},
	}
}

// ProcessGDSRequest processes a GDS request
func (s *GDSService) ProcessGDSRequest(ctx context.Context, request *models.GDSRequest) (*models.GDSResponse, error) {
	startTime := time.Now()

	// Get GDS configuration
	config, exists := s.configurations[request.Provider]
	if !exists {
		return nil, fmt.Errorf("unsupported GDS provider: %s", request.Provider)
	}

	// Create or get GDS session
	session, err := s.sessionManager.GetOrCreateGDSSession(ctx, request.Provider, request.UserID, request.PseudoCity)
	if err != nil {
		return nil, fmt.Errorf("failed to manage GDS session: %w", err)
	}

	// Execute request through circuit breaker
	result, err := s.circuitBreaker.Execute(func() (interface{}, error) {
		return s.executeGDSRequest(ctx, request, config, session)
	})

	if err != nil {
		return s.createErrorResponse(request, "EXECUTION_ERROR", err.Error(), time.Since(startTime)), nil
	}

	response := result.(*models.GDSResponse)
	response.ProcessingTime = time.Since(startTime)

	return response, nil
}

// executeGDSRequest executes the actual GDS request
func (s *GDSService) executeGDSRequest(ctx context.Context, request *models.GDSRequest, config *GDSConfiguration, session *models.GDSSession) (*models.GDSResponse, error) {
	switch request.Provider {
	case models.AmadeusGDS:
		return s.executeAmadeusRequest(ctx, request, config, session)
	case models.SabreGDS:
		return s.executeSabreRequest(ctx, request, config, session)
	case models.TravelportGDS:
		return s.executeTravelportRequest(ctx, request, config, session)
	default:
		return nil, fmt.Errorf("unsupported GDS provider: %s", request.Provider)
	}
}

// executeAmadeusRequest handles Amadeus-specific requests
func (s *GDSService) executeAmadeusRequest(ctx context.Context, request *models.GDSRequest, config *GDSConfiguration, session *models.GDSSession) (*models.GDSResponse, error) {
	switch request.RequestType {
	case "FLIGHT_SEARCH":
		return s.amadeusFlightSearch(ctx, request, config, session)
	case "FLIGHT_PRICE":
		return s.amadeusFlightPrice(ctx, request, config, session)
	case "FLIGHT_BOOKING":
		return s.amadeusFlightBooking(ctx, request, config, session)
	default:
		return nil, fmt.Errorf("unsupported Amadeus request type: %s", request.RequestType)
	}
}

// executeSabreRequest handles Sabre-specific requests
func (s *GDSService) executeSabreRequest(ctx context.Context, request *models.GDSRequest, config *GDSConfiguration, session *models.GDSSession) (*models.GDSResponse, error) {
	switch request.RequestType {
	case "BARGAIN_FINDER":
		return s.sabreBargainFinder(ctx, request, config, session)
	case "ENHANCED_SEARCH":
		return s.sabreEnhancedSearch(ctx, request, config, session)
	case "PRICE_QUOTE":
		return s.sabrePriceQuote(ctx, request, config, session)
	default:
		return nil, fmt.Errorf("unsupported Sabre request type: %s", request.RequestType)
	}
}

// executeTravelportRequest handles Travelport-specific requests
func (s *GDSService) executeTravelportRequest(ctx context.Context, request *models.GDSRequest, config *GDSConfiguration, session *models.GDSSession) (*models.GDSResponse, error) {
	switch request.RequestType {
	case "AIR_SEARCH":
		return s.travelportAirSearch(ctx, request, config, session)
	case "AIR_PRICE":
		return s.travelportAirPrice(ctx, request, config, session)
	case "AIR_BOOK":
		return s.travelportAirBook(ctx, request, config, session)
	default:
		return nil, fmt.Errorf("unsupported Travelport request type: %s", request.RequestType)
	}
}

// Amadeus implementation methods
func (s *GDSService) amadeusFlightSearch(ctx context.Context, request *models.GDSRequest, config *GDSConfiguration, session *models.GDSSession) (*models.GDSResponse, error) {
	// Build Amadeus flight search request
	searchRequest := models.AmadeusSearchRequest{
		OriginDestinationInformation: []models.AmadeusOriginDestination{
			{
				DepartureDateTime: request.RequestData["departure_date"].(string),
				DepartureLocation: request.RequestData["origin"].(string),
				ArrivalLocation:   request.RequestData["destination"].(string),
			},
		},
		TravelPreferences: models.AmadeusTravelPreferences{
			CabinPreference: "Y",
			FareRestrictPreference: models.AmadeusFareRestrictions{
				AdvResTicketing: models.AmadeusAdvanceRestriction{
					AdvReservation: false,
					AdvTicketing:   false,
				},
			},
		},
		TravelerInformation: models.AmadeusTravelerInformation{
			AirTraveler: []models.AmadeusAirTraveler{
				{
					PassengerTypeQuantity: models.AmadeusPassengerTypeQuantity{
						Code:     "ADT",
						Quantity: 1,
					},
				},
			},
		},
	}

	// Make API call
	endpoint := config.BaseURL + config.Endpoints["flight_search"]
	
	resp, err := s.httpClient.R().
		SetContext(ctx).
		SetHeaders(config.Headers).
		SetAuthToken(session.AuthToken).
		SetBody(searchRequest).
		Post(endpoint)

	if err != nil {
		return nil, fmt.Errorf("Amadeus API call failed: %w", err)
	}

	// Parse response
	var amadeusResponse models.AmadeusSearchResponse
	if err := json.Unmarshal(resp.Body(), &amadeusResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Amadeus response: %w", err)
	}

	// Transform to standard format
	return &models.GDSResponse{
		Provider:      models.AmadeusGDS,
		ResponseType:  "FLIGHT_SEARCH",
		Success:       resp.StatusCode() == http.StatusOK,
		Data:          s.transformAmadeusResponse(amadeusResponse),
		SessionID:     session.SessionID,
		TransactionID: uuid.New().String(),
	}, nil
}

func (s *GDSService) amadeusFlightPrice(ctx context.Context, request *models.GDSRequest, config *GDSConfiguration, session *models.GDSSession) (*models.GDSResponse, error) {
	// Implementation for Amadeus flight pricing
	return nil, fmt.Errorf("Amadeus flight pricing not implemented yet")
}

func (s *GDSService) amadeusFlightBooking(ctx context.Context, request *models.GDSRequest, config *GDSConfiguration, session *models.GDSSession) (*models.GDSResponse, error) {
	// Implementation for Amadeus flight booking
	return nil, fmt.Errorf("Amadeus flight booking not implemented yet")
}

// Sabre implementation methods
func (s *GDSService) sabreBargainFinder(ctx context.Context, request *models.GDSRequest, config *GDSConfiguration, session *models.GDSSession) (*models.GDSResponse, error) {
	// Build Sabre bargain finder request
	searchRequest := models.SabreSearchRequest{
		OTA_AirLowFareSearchRQ: models.SabreOTAAirLowFareSearchRQ{
			OriginDestinationInformation: []models.SabreOriginDestinationInfo{
				{
					RPH:               "1",
					DepartureDateTime: request.RequestData["departure_date"].(string),
					OriginLocation: models.SabreLocationInfo{
						LocationCode: request.RequestData["origin"].(string),
					},
					DestinationLocation: models.SabreLocationInfo{
						LocationCode: request.RequestData["destination"].(string),
					},
				},
			},
			TravelPreferences: models.SabreTravelPreferences{
				MaxStopsQuantity: 2,
				CabinPref: []models.SabreCabinPref{
					{
						Cabin:       "Y",
						PreferLevel: "Preferred",
					},
				},
			},
			TravelerInfoSummary: models.SabreTravelerInfoSummary{
				SeatsRequested: []int{1},
				AirTravelerAvail: []models.SabreAirTravelerAvail{
					{
						PassengerTypeQuantity: models.SabrePassengerTypeQuantity{
							Code:     "ADT",
							Quantity: 1,
						},
					},
				},
			},
		},
	}

	// Make API call
	endpoint := config.BaseURL + config.Endpoints["bargain_finder"]
	
	resp, err := s.httpClient.R().
		SetContext(ctx).
		SetHeaders(config.Headers).
		SetAuthToken(session.AuthToken).
		SetBody(searchRequest).
		Post(endpoint)

	if err != nil {
		return nil, fmt.Errorf("Sabre API call failed: %w", err)
	}

	// Parse response (simplified)
	var responseData map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse Sabre response: %w", err)
	}

	return &models.GDSResponse{
		Provider:      models.SabreGDS,
		ResponseType:  "BARGAIN_FINDER",
		Success:       resp.StatusCode() == http.StatusOK,
		Data:          responseData,
		SessionID:     session.SessionID,
		TransactionID: uuid.New().String(),
	}, nil
}

func (s *GDSService) sabreEnhancedSearch(ctx context.Context, request *models.GDSRequest, config *GDSConfiguration, session *models.GDSSession) (*models.GDSResponse, error) {
	// Implementation for Sabre enhanced search
	return nil, fmt.Errorf("Sabre enhanced search not implemented yet")
}

func (s *GDSService) sabrePriceQuote(ctx context.Context, request *models.GDSRequest, config *GDSConfiguration, session *models.GDSSession) (*models.GDSResponse, error) {
	// Implementation for Sabre price quote
	return nil, fmt.Errorf("Sabre price quote not implemented yet")
}

// Travelport implementation methods
func (s *GDSService) travelportAirSearch(ctx context.Context, request *models.GDSRequest, config *GDSConfiguration, session *models.GDSSession) (*models.GDSResponse, error) {
	// Build Travelport air search request
	searchRequest := models.TravelportSearchRequest{
		LowFareSearchReq: models.TravelportLowFareSearchReq{
			BillingPointOfSaleInfo: models.TravelportBillingPOSInfo{
				OriginApplication: "IAROS",
			},
			SearchAirLeg: []models.TravelportSearchAirLeg{
				{
					SearchOrigin: []models.TravelportSearchOrigin{
						{
							Airport: []models.TravelportAirport{
								{
									Code: request.RequestData["origin"].(string),
								},
							},
						},
					},
					SearchDestination: []models.TravelportSearchDestination{
						{
							Airport: []models.TravelportAirport{
								{
									Code: request.RequestData["destination"].(string),
								},
							},
						},
					},
					SearchDepTime: []models.TravelportSearchDepTime{
						{
							PreferredTime: request.RequestData["departure_date"].(string),
						},
					},
				},
			},
			SearchPassenger: []models.TravelportSearchPassenger{
				{
					Code: "ADT",
				},
			},
			AirSearchModifiers: models.TravelportAirSearchModifiers{
				MaxJourneyTime: 720,
				JourneyType:    "SimpleTrip",
			},
		},
	}

	// Make API call
	endpoint := config.BaseURL + config.Endpoints["air_search"]
	
	resp, err := s.httpClient.R().
		SetContext(ctx).
		SetHeaders(config.Headers).
		SetAuthToken(session.AuthToken).
		SetBody(searchRequest).
		Post(endpoint)

	if err != nil {
		return nil, fmt.Errorf("Travelport API call failed: %w", err)
	}

	// Parse response (simplified)
	var responseData map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse Travelport response: %w", err)
	}

	return &models.GDSResponse{
		Provider:      models.TravelportGDS,
		ResponseType:  "AIR_SEARCH",
		Success:       resp.StatusCode() == http.StatusOK,
		Data:          responseData,
		SessionID:     session.SessionID,
		TransactionID: uuid.New().String(),
	}, nil
}

func (s *GDSService) travelportAirPrice(ctx context.Context, request *models.GDSRequest, config *GDSConfiguration, session *models.GDSSession) (*models.GDSResponse, error) {
	// Implementation for Travelport air pricing
	return nil, fmt.Errorf("Travelport air pricing not implemented yet")
}

func (s *GDSService) travelportAirBook(ctx context.Context, request *models.GDSRequest, config *GDSConfiguration, session *models.GDSSession) (*models.GDSResponse, error) {
	// Implementation for Travelport air booking
	return nil, fmt.Errorf("Travelport air booking not implemented yet")
}

// transformAmadeusResponse transforms Amadeus response to standard format
func (s *GDSService) transformAmadeusResponse(response models.AmadeusSearchResponse) map[string]interface{} {
	return map[string]interface{}{
		"provider":            "AMADEUS",
		"priced_itineraries": response.PricedItineraries,
		"success":            response.Success,
		"warnings":           response.Warnings,
		"errors":             response.Errors,
		"extensions":         response.TPAExtensions,
	}
}

// createErrorResponse creates a standardized error response
func (s *GDSService) createErrorResponse(request *models.GDSRequest, errorCode, errorMessage string, processingTime time.Duration) *models.GDSResponse {
	return &models.GDSResponse{
		Provider:      request.Provider,
		ResponseType:  request.RequestType,
		Success:       false,
		Data:          make(map[string]interface{}),
		Errors: []models.GDSError{
			{
				Code:        errorCode,
				Type:        "SYSTEM_ERROR",
				Description: errorMessage,
				Severity:    "ERROR",
			},
		},
		SessionID:      request.SessionID,
		TransactionID:  uuid.New().String(),
		ProcessingTime: processingTime,
	}
}

// HealthCheck performs health check on GDS connections
func (s *GDSService) HealthCheck(ctx context.Context) map[string]interface{} {
	results := make(map[string]interface{})

	for provider, config := range s.configurations {
		result := s.checkGDSHealth(ctx, provider, config)
		results[string(provider)] = result
	}

	return results
}

// checkGDSHealth checks health of a specific GDS
func (s *GDSService) checkGDSHealth(ctx context.Context, provider models.GDSProvider, config *GDSConfiguration) map[string]interface{} {
	start := time.Now()
	
	// Simple health check - attempt to connect to the service
	resp, err := s.httpClient.R().
		SetContext(ctx).
		SetHeaders(config.Headers).
		Get(config.BaseURL + "/health")

	duration := time.Since(start)
	
	if err != nil || resp.StatusCode() != http.StatusOK {
		return map[string]interface{}{
			"status":           "UNHEALTHY",
			"error":            err,
			"response_time_ms": duration.Milliseconds(),
			"last_check":       time.Now().UTC(),
		}
	}

	return map[string]interface{}{
		"status":           "HEALTHY",
		"response_time_ms": duration.Milliseconds(),
		"last_check":       time.Now().UTC(),
	}
}

// GetGDSMetrics returns performance metrics for GDS operations
func (s *GDSService) GetGDSMetrics(ctx context.Context) (map[string]interface{}, error) {
	// Query metrics from database
	var metrics []models.DistributionMetric
	
	err := s.db.Where("channel_type IN (?)", []string{"GDS"}).
		Where("timestamp >= ?", time.Now().Add(-24*time.Hour)).
		Find(&metrics).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query GDS metrics: %w", err)
	}

	// Aggregate metrics by provider
	providerMetrics := make(map[string]interface{})
	
	for _, metric := range metrics {
		provider := metric.ChannelID
		if providerMetrics[provider] == nil {
			providerMetrics[provider] = make(map[string]interface{})
		}
		
		providerData := providerMetrics[provider].(map[string]interface{})
		providerData[metric.MetricName] = metric.Value
	}

	return map[string]interface{}{
		"gds_metrics":    providerMetrics,
		"circuit_breaker": s.circuitBreaker.Name,
		"total_metrics":  len(metrics),
	}, nil
} 