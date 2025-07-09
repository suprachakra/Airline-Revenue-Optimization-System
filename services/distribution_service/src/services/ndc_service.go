package services

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"time"

	"iaros/distribution_service/src/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// NDCService handles NDC Level 4 operations
type NDCService struct {
	db              *gorm.DB
	sessionManager  *SessionManager
	orderServiceURL string
	offerServiceURL string
}

// NewNDCService creates a new NDC service instance
func NewNDCService(db *gorm.DB, sessionManager *SessionManager, orderServiceURL, offerServiceURL string) *NDCService {
	return &NDCService{
		db:              db,
		sessionManager:  sessionManager,
		orderServiceURL: orderServiceURL,
		offerServiceURL: offerServiceURL,
	}
}

// ProcessAirShopping handles NDC AirShopping requests
func (s *NDCService) ProcessAirShopping(ctx context.Context, request *models.AirShoppingRQ) (*models.AirShoppingRS, error) {
	log.Printf("Processing AirShopping request: %s", request.MessageID)

	// Create NDC session
	session, err := s.sessionManager.CreateNDCSession(ctx, request.Source.RequestorID, "AA")
	if err != nil {
		return nil, fmt.Errorf("failed to create NDC session: %w", err)
	}

	// Validate request
	if err := s.validateAirShoppingRequest(request); err != nil {
		return s.createErrorResponse(request, "VALIDATION_ERROR", err.Error()), nil
	}

	// Get offers from offer service
	offers, err := s.getOffersFromService(ctx, request)
	if err != nil {
		log.Printf("Error getting offers: %v", err)
		return s.createErrorResponse(request, "OFFER_SERVICE_ERROR", err.Error()), nil
	}

	// Transform offers to NDC format
	ndcOffers := s.transformOffersToNDC(offers)

	// Create shopping response
	response := &models.AirShoppingRS{
		Version:   "20.3",
		MessageID: uuid.New().String(),
		Timestamp: time.Now().UTC(),
		Source:    request.Source,
		Success: models.Success{
			Code:        "SUCCESS",
			Description: "Air shopping completed successfully",
		},
		OffersGroup: models.OffersGroup{
			CarrierOffers: []models.CarrierOffers{
				{
					AirlineCode: "AA",
					Offers:      ndcOffers,
				},
			},
		},
		DataLists: s.createDataLists(request, ndcOffers),
		Metadata:  s.createMetadata(request, session),
	}

	// Store shopping context in session
	s.sessionManager.UpdateNDCShoppingContext(ctx, session.SessionID, response)

	return response, nil
}

// ProcessOfferPrice handles NDC OfferPrice requests
func (s *NDCService) ProcessOfferPrice(ctx context.Context, request interface{}) (interface{}, error) {
	// Implementation for offer pricing
	return nil, fmt.Errorf("OfferPrice not implemented yet")
}

// ProcessOrderCreate handles NDC OrderCreate requests
func (s *NDCService) ProcessOrderCreate(ctx context.Context, request interface{}) (interface{}, error) {
	// Implementation for order creation
	return nil, fmt.Errorf("OrderCreate not implemented yet")
}

// ProcessOrderRetrieve handles NDC OrderRetrieve requests
func (s *NDCService) ProcessOrderRetrieve(ctx context.Context, request interface{}) (interface{}, error) {
	// Implementation for order retrieval
	return nil, fmt.Errorf("OrderRetrieve not implemented yet")
}

// ProcessOrderCancel handles NDC OrderCancel requests
func (s *NDCService) ProcessOrderCancel(ctx context.Context, request interface{}) (interface{}, error) {
	// Implementation for order cancellation
	return nil, fmt.Errorf("OrderCancel not implemented yet")
}

// validateAirShoppingRequest validates the incoming request
func (s *NDCService) validateAirShoppingRequest(request *models.AirShoppingRQ) error {
	if request.MessageID == "" {
		return fmt.Errorf("MessageID is required")
	}
	if len(request.CoreQuery.OriginDestinations) == 0 {
		return fmt.Errorf("at least one origin-destination is required")
	}
	if len(request.Travelers) == 0 {
		return fmt.Errorf("at least one traveler is required")
	}
	
	// Validate each origin-destination
	for i, od := range request.CoreQuery.OriginDestinations {
		if od.Origin == "" || od.Destination == "" {
			return fmt.Errorf("origin and destination are required for segment %d", i)
		}
		if od.DepartureDate.IsZero() {
			return fmt.Errorf("departure date is required for segment %d", i)
		}
	}

	return nil
}

// getOffersFromService retrieves offers from the offer service
func (s *NDCService) getOffersFromService(ctx context.Context, request *models.AirShoppingRQ) ([]map[string]interface{}, error) {
	// Mock implementation - in real scenario, this would call the offer service
	offers := []map[string]interface{}{
		{
			"offer_id":    "OFFER_001",
			"price":       decimal.NewFromFloat(299.99),
			"currency":    "USD",
			"valid_from":  time.Now(),
			"valid_to":    time.Now().Add(24 * time.Hour),
			"segments": []map[string]interface{}{
				{
					"origin":      request.CoreQuery.OriginDestinations[0].Origin,
					"destination": request.CoreQuery.OriginDestinations[0].Destination,
					"departure":   request.CoreQuery.OriginDestinations[0].DepartureDate,
					"airline":     "AA",
					"flight":      "1234",
					"class":       "Y",
				},
			},
		},
		{
			"offer_id":    "OFFER_002",
			"price":       decimal.NewFromFloat(399.99),
			"currency":    "USD",
			"valid_from":  time.Now(),
			"valid_to":    time.Now().Add(24 * time.Hour),
			"segments": []map[string]interface{}{
				{
					"origin":      request.CoreQuery.OriginDestinations[0].Origin,
					"destination": request.CoreQuery.OriginDestinations[0].Destination,
					"departure":   request.CoreQuery.OriginDestinations[0].DepartureDate,
					"airline":     "AA",
					"flight":      "5678",
					"class":       "J",
				},
			},
		},
	}

	return offers, nil
}

// transformOffersToNDC transforms internal offers to NDC format
func (s *NDCService) transformOffersToNDC(offers []map[string]interface{}) []models.Offer {
	var ndcOffers []models.Offer

	for _, offer := range offers {
		price := offer["price"].(decimal.Decimal)
		currency := offer["currency"].(string)
		validFrom := offer["valid_from"].(time.Time)
		validTo := offer["valid_to"].(time.Time)

		ndcOffer := models.Offer{
			OfferID:   offer["offer_id"].(string),
			Owner:     "AA",
			ValidFrom: validFrom,
			ValidTo:   validTo,
			TotalPrice: models.TotalPrice{
				DetailCurrencyPrice: models.DetailCurrencyPrice{
					Total: models.PriceDetail{
						Amount:   price,
						Currency: currency,
					},
					Base: models.PriceDetail{
						Amount:   price.Mul(decimal.NewFromFloat(0.85)),
						Currency: currency,
					},
					Taxes: models.PriceDetail{
						Amount:   price.Mul(decimal.NewFromFloat(0.10)),
						Currency: currency,
					},
					Fees: models.PriceDetail{
						Amount:   price.Mul(decimal.NewFromFloat(0.05)),
						Currency: currency,
					},
				},
				TaxSummary: []models.TaxSummary{
					{
						TaxCode:     "US",
						Amount:      price.Mul(decimal.NewFromFloat(0.08)),
						Currency:    currency,
						Description: "US Transportation Tax",
					},
					{
						TaxCode:     "XF",
						Amount:      price.Mul(decimal.NewFromFloat(0.02)),
						Currency:    currency,
						Description: "Passenger Facility Charge",
					},
				},
				FeeSummary: []models.FeeSummary{
					{
						FeeCode:     "YC",
						Amount:      price.Mul(decimal.NewFromFloat(0.03)),
						Currency:    currency,
						Description: "Security Fee",
					},
					{
						FeeCode:     "YQ",
						Amount:      price.Mul(decimal.NewFromFloat(0.02)),
						Currency:    currency,
						Description: "Fuel Surcharge",
					},
				},
			},
			TimeLimits: models.TimeLimits{
				OfferExpiration:    validTo,
				PaymentTimeLimit:   time.Now().Add(2 * time.Hour),
				TicketingTimeLimit: time.Now().Add(24 * time.Hour),
				PriceGuaranteeTime: time.Now().Add(30 * time.Minute),
			},
			BaggageAllowance: []models.BaggageAllowance{
				{
					BaggageCategory: "CHECKED",
					AllowanceType:   "WEIGHT",
					MaxWeight:       23,
					WeightUnit:      "KG",
					MaxPieces:       1,
					MaxSize:         "158CM",
				},
			},
			OfferItems: []models.OfferItem{
				{
					OfferItemID: fmt.Sprintf("%s_ITEM_001", offer["offer_id"].(string)),
					Service: models.Service{
						ServiceID:   "FLIGHT_001",
						ServiceType: "FLIGHT",
						ServiceDefinition: models.ServiceDefinition{
							Name:            "Economy Flight",
							Description:     "Standard economy class flight",
							Code:            "ECON",
							ServiceCategory: "FLIGHT",
							Encoding:        "NDC",
						},
					},
					FareDetail: models.FareDetail{
						FareBasis: "Y26",
						FareCalculation: fmt.Sprintf("%.2f%s", price, currency),
						PriceClass: models.PriceClass{
							ClassCode:   "Y",
							ClassName:   "Economy",
							DisplayName: "Economy Class",
						},
					},
				},
			},
		}

		ndcOffers = append(ndcOffers, ndcOffer)
	}

	return ndcOffers
}

// createDataLists creates the data lists for the response
func (s *NDCService) createDataLists(request *models.AirShoppingRQ, offers []models.Offer) models.DataLists {
	return models.DataLists{
		PassengerList: models.PassengerList{
			Passenger: s.createPassengerList(request.Travelers),
		},
		FlightSegmentList: models.FlightSegmentList{
			FlightSegment: s.createFlightSegmentList(request.CoreQuery.OriginDestinations),
		},
		FlightList: models.FlightList{
			Flight: s.createFlightList(offers),
		},
		PriceClassList: models.PriceClassList{
			PriceClass: s.createPriceClassList(),
		},
	}
}

// createPassengerList creates passenger data list
func (s *NDCService) createPassengerList(travelers []models.Traveler) []models.PassengerData {
	var passengers []models.PassengerData

	for _, traveler := range travelers {
		passenger := models.PassengerData{
			PassengerID: traveler.TravelerID,
			ObjectKey:   fmt.Sprintf("PAX_%s", traveler.TravelerID),
			PTC:         traveler.PassengerType,
			Individual: models.Individual{
				Gender:    traveler.Gender,
				ProfileID: traveler.TravelerID,
			},
		}

		if traveler.LoyaltyProgram.ProgramID != "" {
			passenger.LoyaltyPrograms = []models.LoyaltyProgram{traveler.LoyaltyProgram}
		}

		passengers = append(passengers, passenger)
	}

	return passengers
}

// createFlightSegmentList creates flight segment data list
func (s *NDCService) createFlightSegmentList(ods []models.OriginDestination) []models.FlightSegment {
	var segments []models.FlightSegment

	for i, od := range ods {
		segment := models.FlightSegment{
			SegmentKey: fmt.Sprintf("SEG_%d", i+1),
			Departure: models.Departure{
				AirportCode:  od.Origin,
				AirportName:  s.getAirportName(od.Origin),
				Date:         od.DepartureDate,
				Time:         od.DepartureDate.Format("15:04"),
				TimezoneCode: s.getTimezone(od.Origin),
			},
			Arrival: models.Arrival{
				AirportCode:  od.Destination,
				AirportName:  s.getAirportName(od.Destination),
				Date:         od.DepartureDate.Add(2 * time.Hour),
				Time:         od.DepartureDate.Add(2 * time.Hour).Format("15:04"),
				TimezoneCode: s.getTimezone(od.Destination),
			},
			MarketingCarrier: models.MarketingCarrier{
				AirlineCode:  "AA",
				Name:         "American Airlines",
				FlightNumber: "1234",
			},
			OperatingCarrier: models.OperatingCarrier{
				AirlineCode:  "AA",
				Name:         "American Airlines",
				FlightNumber: "1234",
			},
			Equipment: models.Equipment{
				AircraftCode: "321",
				Name:         "Airbus A321",
			},
			ClassOfService: []models.ClassOfService{
				{
					Code:          "Y",
					MarketingName: "Economy",
					Availability:  9,
					Cabin:         "Economy",
				},
				{
					Code:          "J",
					MarketingName: "Business",
					Availability:  4,
					Cabin:         "Business",
				},
			},
		}

		segments = append(segments, segment)
	}

	return segments
}

// createFlightList creates flight data list
func (s *NDCService) createFlightList(offers []models.Offer) []models.Flight {
	var flights []models.Flight

	for i, offer := range offers {
		flight := models.Flight{
			FlightKey: fmt.Sprintf("FLT_%d", i+1),
			Journey: models.Journey{
				JourneyDistance: models.Distance{
					Value: decimal.NewFromFloat(1200.5),
					UOM:   "MI",
				},
				JourneyTime: "PT2H15M",
			},
			SegmentReferences: []string{"SEG_1"},
			Settlement: models.Settlement{
				Method:     "BSP",
				Interline:  false,
				BSPCountry: "US",
			},
		}

		flights = append(flights, flight)
	}

	return flights
}

// createPriceClassList creates price class data list
func (s *NDCService) createPriceClassList() []models.PriceClassData {
	return []models.PriceClassData{
		{
			PriceClassID: "PC_ECONOMY",
			Name:         "Economy",
			Code:         "Y",
			Descriptions: []models.Description{
				{
					Text:        "Standard economy class with basic amenities",
					Language:    "EN",
					Application: "CUSTOMER",
				},
			},
		},
		{
			PriceClassID: "PC_BUSINESS",
			Name:         "Business",
			Code:         "J",
			Descriptions: []models.Description{
				{
					Text:        "Premium business class with enhanced services",
					Language:    "EN",
					Application: "CUSTOMER",
				},
			},
		},
	}
}

// createMetadata creates metadata for the response
func (s *NDCService) createMetadata(request *models.AirShoppingRQ, session *models.NDCSession) models.Metadata {
	return models.Metadata{
		Shopping: models.ShoppingMetadata{
			ShoppingResponseID: models.ShoppingResponseID{
				ResponseID: uuid.New().String(),
				Owner:      "AA",
				Timestamp:  time.Now().UTC(),
			},
		},
		Policies: models.PolicyMetadata{
			PriceGuaranteePolicy: models.PriceGuaranteePolicy{
				GuaranteeTime: "PT30M",
				Application:   "ALL_OFFERS",
			},
			PaymentTimeLimitPolicy: models.PaymentTimeLimitPolicy{
				TimeLimit:   "PT2H",
				Application: "ALL_OFFERS",
			},
		},
	}
}

// createErrorResponse creates an error response
func (s *NDCService) createErrorResponse(request *models.AirShoppingRQ, errorCode, errorMessage string) *models.AirShoppingRS {
	return &models.AirShoppingRS{
		Version:   "20.3",
		MessageID: uuid.New().String(),
		Timestamp: time.Now().UTC(),
		Source:    request.Source,
		Errors: []models.Error{
			{
				Code:        errorCode,
				Type:        "BUSINESS_ERROR",
				Description: errorMessage,
				Owner:       "AA",
			},
		},
	}
}

// Helper functions
func (s *NDCService) getAirportName(code string) string {
	airports := map[string]string{
		"JFK": "John F. Kennedy International Airport",
		"LAX": "Los Angeles International Airport",
		"ORD": "Chicago O'Hare International Airport",
		"DFW": "Dallas/Fort Worth International Airport",
		"DEN": "Denver International Airport",
		"SFO": "San Francisco International Airport",
		"LAS": "McCarran International Airport",
		"SEA": "Seattle-Tacoma International Airport",
		"MIA": "Miami International Airport",
		"PHX": "Phoenix Sky Harbor International Airport",
	}

	if name, exists := airports[code]; exists {
		return name
	}
	return fmt.Sprintf("%s Airport", code)
}

func (s *NDCService) getTimezone(code string) string {
	timezones := map[string]string{
		"JFK": "America/New_York",
		"LAX": "America/Los_Angeles",
		"ORD": "America/Chicago",
		"DFW": "America/Chicago",
		"DEN": "America/Denver",
		"SFO": "America/Los_Angeles",
		"LAS": "America/Los_Angeles",
		"SEA": "America/Los_Angeles",
		"MIA": "America/New_York",
		"PHX": "America/Phoenix",
	}

	if tz, exists := timezones[code]; exists {
		return tz
	}
	return "UTC"
}

// SerializeNDCMessage converts NDC message to XML
func (s *NDCService) SerializeNDCMessage(message interface{}) ([]byte, error) {
	return xml.MarshalIndent(message, "", "  ")
}

// DeserializeNDCMessage converts XML to NDC message
func (s *NDCService) DeserializeNDCMessage(data []byte, message interface{}) error {
	return xml.Unmarshal(data, message)
} 