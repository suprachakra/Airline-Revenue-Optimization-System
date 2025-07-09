package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// ============= Common GDS Models =============

type GDSProvider string

const (
	AmadeusGDS    GDSProvider = "AMADEUS"
	SabreGDS      GDSProvider = "SABRE"
	TravelportGDS GDSProvider = "TRAVELPORT"
)

type GDSRequest struct {
	Provider     GDSProvider            `json:"provider"`
	RequestType  string                 `json:"request_type"`
	PseudoCity   string                 `json:"pseudo_city"`
	UserID       string                 `json:"user_id"`
	SessionID    string                 `json:"session_id"`
	RequestData  map[string]interface{} `json:"request_data"`
	Headers      map[string]string      `json:"headers"`
	Timeout      time.Duration          `json:"timeout"`
}

type GDSResponse struct {
	Provider     GDSProvider            `json:"provider"`
	ResponseType string                 `json:"response_type"`
	Success      bool                   `json:"success"`
	Data         map[string]interface{} `json:"data"`
	Errors       []GDSError            `json:"errors,omitempty"`
	Warnings     []GDSWarning          `json:"warnings,omitempty"`
	SessionID    string                 `json:"session_id"`
	TransactionID string                `json:"transaction_id"`
	ProcessingTime time.Duration        `json:"processing_time"`
}

type GDSError struct {
	Code        string `json:"code"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

type GDSWarning struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// ============= Amadeus Models =============

type AmadeusSearchRequest struct {
	OriginDestinationInformation []AmadeusOriginDestination `json:"origin_destination_information"`
	TravelPreferences           AmadeusTravelPreferences   `json:"travel_preferences"`
	TravelerInformation         AmadeusTravelerInformation `json:"traveler_information"`
	SpecificFlightInfo          *AmadeusSpecificFlightInfo `json:"specific_flight_info,omitempty"`
}

type AmadeusOriginDestination struct {
	DepartureDateTime     string `json:"departure_date_time"`
	DepartureLocation     string `json:"departure_location"`
	ArrivalLocation       string `json:"arrival_location"`
	ConnectionTime        int    `json:"connection_time,omitempty"`
	AlternateLocationInfo []AmadeusAlternateLocation `json:"alternate_location_info,omitempty"`
}

type AmadeusAlternateLocation struct {
	LocationCode string `json:"location_code"`
	Distance     int    `json:"distance"`
	Include      bool   `json:"include"`
}

type AmadeusTravelPreferences struct {
	VendorPreference       []string                `json:"vendor_preference,omitempty"`
	CabinPreference        string                  `json:"cabin_preference,omitempty"`
	FlightTypePreference   string                  `json:"flight_type_preference,omitempty"`
	EquipmentPreference    []string                `json:"equipment_preference,omitempty"`
	FareRestrictPreference AmadeusFareRestrictions `json:"fare_restrict_preference"`
	TicketDistribType      string                  `json:"ticket_distrib_type,omitempty"`
}

type AmadeusFareRestrictions struct {
	AdvResTicketing   AmadeusAdvanceRestriction `json:"adv_res_ticketing"`
	StayRestrictions  AmadeusStayRestriction    `json:"stay_restrictions"`
	VoluntaryChanges  AmadeusVoluntaryChanges   `json:"voluntary_changes"`
}

type AmadeusAdvanceRestriction struct {
	AdvReservation bool `json:"adv_reservation"`
	AdvTicketing   bool `json:"adv_ticketing"`
}

type AmadeusStayRestriction struct {
	MinStay bool `json:"min_stay"`
	MaxStay bool `json:"max_stay"`
}

type AmadeusVoluntaryChanges struct {
	Penalty bool `json:"penalty"`
}

type AmadeusTravelerInformation struct {
	AirTraveler []AmadeusAirTraveler `json:"air_traveler"`
}

type AmadeusAirTraveler struct {
	PassengerTypeQuantity AmadeusPassengerTypeQuantity `json:"passenger_type_quantity"`
	TravelerRefNumber     string                       `json:"traveler_ref_number"`
}

type AmadeusPassengerTypeQuantity struct {
	Code     string `json:"code"`
	Quantity int    `json:"quantity"`
}

type AmadeusSpecificFlightInfo struct {
	BookingClassPref string                       `json:"booking_class_pref"`
	FlightNumber     string                       `json:"flight_number"`
	Airline          string                       `json:"airline"`
	FlightRefNumber  string                       `json:"flight_ref_number"`
}

type AmadeusSearchResponse struct {
	PricedItineraries []AmadeusPricedItinerary `json:"priced_itineraries"`
	TPAExtensions     AmadeusTPAExtensions     `json:"tpa_extensions"`
	Success           AmadeusSuccess           `json:"success"`
	Warnings          []AmadeusWarning         `json:"warnings,omitempty"`
	Errors            []AmadeusError           `json:"errors,omitempty"`
}

type AmadeusPricedItinerary struct {
	SequenceNumber          int                           `json:"sequence_number"`
	AirItinerary            AmadeusAirItinerary          `json:"air_itinerary"`
	AirItineraryPricingInfo AmadeusAirItineraryPricingInfo `json:"air_itinerary_pricing_info"`
	TicketingInfo           AmadeusTicketingInfo         `json:"ticketing_info"`
	TPAExtensions           AmadeusItineraryExtensions   `json:"tpa_extensions"`
}

type AmadeusAirItinerary struct {
	OriginDestinationOptions []AmadeusOriginDestinationOption `json:"origin_destination_options"`
}

type AmadeusOriginDestinationOption struct {
	FlightSegment []AmadeusFlightSegment `json:"flight_segment"`
}

type AmadeusFlightSegment struct {
	DepartureAirport         string                      `json:"departure_airport"`
	ArrivalAirport           string                      `json:"arrival_airport"`
	OperatingAirline         AmadeusOperatingAirline     `json:"operating_airline"`
	Equipment                string                      `json:"equipment"`
	MarketingAirline         AmadeusMarketingAirline     `json:"marketing_airline"`
	DepartureDateTime        string                      `json:"departure_date_time"`
	ArrivalDateTime          string                      `json:"arrival_date_time"`
	StopQuantity             int                         `json:"stop_quantity"`
	FlightNumber             string                      `json:"flight_number"`
	ResBookDesigCode         string                      `json:"res_book_desig_code"`
	BookingClassAvails       []AmadeusBookingClassAvail  `json:"booking_class_avails"`
	Comment                  []AmadeusComment            `json:"comment,omitempty"`
}

type AmadeusOperatingAirline struct {
	Code     string `json:"code"`
	FlightID string `json:"flight_id"`
}

type AmadeusMarketingAirline struct {
	Code string `json:"code"`
}

type AmadeusBookingClassAvail struct {
	ResBookDesigCode     string `json:"res_book_desig_code"`
	ResBookDesigQuantity string `json:"res_book_desig_quantity"`
	RPH                  string `json:"rph"`
}

type AmadeusComment struct {
	Name string `json:"name"`
	Text string `json:"text"`
}

type AmadeusAirItineraryPricingInfo struct {
	ItinTotalFare    AmadeusItinTotalFare    `json:"itin_total_fare"`
	PTCFareBreakdowns []AmadeusPTCFareBreakdown `json:"ptc_fare_breakdowns"`
	FareInfos        []AmadeusFareInfo       `json:"fare_infos"`
}

type AmadeusItinTotalFare struct {
	BaseFare    AmadeusBaseFare    `json:"base_fare"`
	FareConstruction string         `json:"fare_construction"`
	EquivFare   AmadeusEquivFare   `json:"equiv_fare"`
	Taxes       AmadeusTaxes       `json:"taxes"`
	TotalFare   AmadeusTotalFare   `json:"total_fare"`
}

type AmadeusBaseFare struct {
	Amount       decimal.Decimal `json:"amount"`
	CurrencyCode string          `json:"currency_code"`
}

type AmadeusEquivFare struct {
	Amount       decimal.Decimal `json:"amount"`
	CurrencyCode string          `json:"currency_code"`
}

type AmadeusTaxes struct {
	Tax         []AmadeusTax `json:"tax"`
	TotalAmount decimal.Decimal `json:"total_amount"`
	CurrencyCode string        `json:"currency_code"`
}

type AmadeusTax struct {
	TaxCode      string          `json:"tax_code"`
	Amount       decimal.Decimal `json:"amount"`
	CurrencyCode string          `json:"currency_code"`
	Description  string          `json:"description"`
}

type AmadeusTotalFare struct {
	Amount       decimal.Decimal `json:"amount"`
	CurrencyCode string          `json:"currency_code"`
}

type AmadeusPTCFareBreakdown struct {
	PassengerTypeQuantity AmadeusPassengerTypeQuantity `json:"passenger_type_quantity"`
	FareBasisCodes        []AmadeusFareBasisCode       `json:"fare_basis_codes"`
	PassengerFare         AmadeusPassengerFare         `json:"passenger_fare"`
}

type AmadeusFareBasisCode struct {
	BookingCode        string `json:"booking_code"`
	AvailabilityBreak  bool   `json:"availability_break"`
	FareBasisCode      string `json:"fare_basis_code"`
}

type AmadeusPassengerFare struct {
	BaseFare     AmadeusBaseFare     `json:"base_fare"`
	FareConstruction string          `json:"fare_construction"`
	EquivFare    AmadeusEquivFare    `json:"equiv_fare"`
	Taxes        AmadeusTaxes        `json:"taxes"`
	TotalFare    AmadeusTotalFare    `json:"total_fare"`
}

type AmadeusFareInfo struct {
	FareReference []AmadeusFareReference `json:"fare_reference"`
}

type AmadeusFareReference struct {
	ResBookDesigCode string `json:"res_book_desig_code"`
	AccountCode      string `json:"account_code"`
}

type AmadeusTicketingInfo struct {
	TicketType      string `json:"ticket_type"`
	ValidatingCarrier string `json:"validating_carrier"`
}

type AmadeusItineraryExtensions struct {
	DivideInParty AmadeusDivideInParty `json:"divide_in_party"`
}

type AmadeusDivideInParty struct {
	Indicator bool `json:"indicator"`
}

type AmadeusTPAExtensions struct {
	IntelliSellTransaction AmadeusIntelliSellTransaction `json:"intelli_sell_transaction"`
}

type AmadeusIntelliSellTransaction struct {
	RequestType string `json:"request_type"`
}

type AmadeusSuccess struct {
	TimeStamp string `json:"time_stamp"`
}

type AmadeusWarning struct {
	Type        string `json:"type"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

type AmadeusError struct {
	Type        string `json:"type"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

// ============= Sabre Models =============

type SabreSearchRequest struct {
	OTA_AirLowFareSearchRQ SabreOTAAirLowFareSearchRQ `json:"OTA_AirLowFareSearchRQ"`
}

type SabreOTAAirLowFareSearchRQ struct {
	OriginDestinationInformation []SabreOriginDestinationInfo `json:"OriginDestinationInformation"`
	TravelPreferences           SabreTravelPreferences       `json:"TravelPreferences"`
	TravelerInfoSummary         SabreTravelerInfoSummary     `json:"TravelerInfoSummary"`
	TPA_Extensions              SabreTPAExtensions           `json:"TPA_Extensions"`
}

type SabreOriginDestinationInfo struct {
	RPH               string                    `json:"RPH"`
	DepartureDateTime string                    `json:"DepartureDateTime"`
	OriginLocation    SabreLocationInfo         `json:"OriginLocation"`
	DestinationLocation SabreLocationInfo       `json:"DestinationLocation"`
	TPA_Extensions    SabreODTPAExtensions      `json:"TPA_Extensions"`
}

type SabreLocationInfo struct {
	LocationCode string `json:"LocationCode"`
}

type SabreODTPAExtensions struct {
	SegmentType SabreSegmentType `json:"SegmentType"`
}

type SabreSegmentType struct {
	Code string `json:"Code"`
}

type SabreTravelPreferences struct {
	MaxStopsQuantity    int                          `json:"MaxStopsQuantity"`
	CabinPref           []SabreCabinPref             `json:"CabinPref"`
	VendorPref          []SabreVendorPref            `json:"VendorPref"`
	TPA_Extensions      SabreTravelPrefExtensions    `json:"TPA_Extensions"`
}

type SabreCabinPref struct {
	Cabin          string `json:"Cabin"`
	PreferLevel    string `json:"PreferLevel"`
}

type SabreVendorPref struct {
	Code        string `json:"Code"`
	PreferLevel string `json:"PreferLevel"`
}

type SabreTravelPrefExtensions struct {
	NumTrips           SabreNumTrips           `json:"NumTrips"`
	DataSources        SabreDataSources        `json:"DataSources"`
	SeatStatusSim      SabreSeatStatusSim      `json:"SeatStatusSim"`
}

type SabreNumTrips struct {
	Number int `json:"Number"`
}

type SabreDataSources struct {
	NDC     string `json:"NDC"`
	ATPCO   string `json:"ATPCO"`
	LCC     string `json:"LCC"`
}

type SabreSeatStatusSim struct {
	RequestType string `json:"RequestType"`
}

type SabreTravelerInfoSummary struct {
	SeatsRequested      []int                        `json:"SeatsRequested"`
	AirTravelerAvail    []SabreAirTravelerAvail      `json:"AirTravelerAvail"`
	PriceRequestInformation SabrePriceRequestInfo    `json:"PriceRequestInformation"`
}

type SabreAirTravelerAvail struct {
	PassengerTypeQuantity SabrePassengerTypeQuantity `json:"PassengerTypeQuantity"`
}

type SabrePassengerTypeQuantity struct {
	Code     string `json:"Code"`
	Quantity int    `json:"Quantity"`
}

type SabrePriceRequestInfo struct {
	CurrencyCode          string                    `json:"CurrencyCode"`
	PricingSource         string                    `json:"PricingSource"`
	TPA_Extensions        SabrePriceReqExtensions   `json:"TPA_Extensions"`
}

type SabrePriceReqExtensions struct {
	PublicFare      SabrePublicFare      `json:"PublicFare"`
	PrivateFare     SabrePrivateFare     `json:"PrivateFare"`
	iataNumber      string               `json:"iataNumber"`
}

type SabrePublicFare struct {
	Ind bool `json:"Ind"`
}

type SabrePrivateFare struct {
	Ind bool `json:"Ind"`
}

type SabreTPAExtensions struct {
	IntelliSellTransaction SabreIntelliSellTransaction `json:"IntelliSellTransaction"`
}

type SabreIntelliSellTransaction struct {
	RequestType string `json:"RequestType"`
}

// ============= Travelport Models =============

type TravelportSearchRequest struct {
	LowFareSearchReq TravelportLowFareSearchReq `json:"LowFareSearchReq"`
}

type TravelportLowFareSearchReq struct {
	BillingPointOfSaleInfo TravelportBillingPOSInfo     `json:"BillingPointOfSaleInfo"`
	SearchAirLeg           []TravelportSearchAirLeg     `json:"SearchAirLeg"`
	SearchPassenger        []TravelportSearchPassenger  `json:"SearchPassenger"`
	AirSearchModifiers     TravelportAirSearchModifiers `json:"AirSearchModifiers"`
}

type TravelportBillingPOSInfo struct {
	OriginApplication string `json:"OriginApplication"`
}

type TravelportSearchAirLeg struct {
	SearchOrigin      []TravelportSearchOrigin      `json:"SearchOrigin"`
	SearchDestination []TravelportSearchDestination `json:"SearchDestination"`
	SearchDepTime     []TravelportSearchDepTime     `json:"SearchDepTime"`
}

type TravelportSearchOrigin struct {
	Airport []TravelportAirport `json:"Airport"`
}

type TravelportSearchDestination struct {
	Airport []TravelportAirport `json:"Airport"`
}

type TravelportAirport struct {
	Code string `json:"Code"`
}

type TravelportSearchDepTime struct {
	PreferredTime string `json:"PreferredTime"`
}

type TravelportSearchPassenger struct {
	Code string `json:"Code"`
	Age  int    `json:"Age,omitempty"`
}

type TravelportAirSearchModifiers struct {
	MaxJourneyTime      int                               `json:"MaxJourneyTime"`
	JourneyType         string                            `json:"JourneyType"`
	PreferredCabins     []TravelportPreferredCabin        `json:"PreferredCabins"`
	PermittedCarriers   []TravelportPermittedCarrier      `json:"PermittedCarriers"`
	ProhibitedCarriers  []TravelportProhibitedCarrier     `json:"ProhibitedCarriers"`
	FlightType          TravelportFlightType              `json:"FlightType"`
}

type TravelportPreferredCabin struct {
	CabinClass string `json:"CabinClass"`
}

type TravelportPermittedCarrier struct {
	Code string `json:"Code"`
}

type TravelportProhibitedCarrier struct {
	Code string `json:"Code"`
}

type TravelportFlightType struct {
	RequireSingleCarrier bool   `json:"RequireSingleCarrier"`
	MaxConnections       int    `json:"MaxConnections"`
	MaxStops             int    `json:"MaxStops"`
	NonStopDirects       bool   `json:"NonStopDirects"`
}

// ============= Distribution Channel Models =============

type DistributionChannel string

const (
	DirectChannel   DistributionChannel = "DIRECT"
	GDSChannel      DistributionChannel = "GDS"
	OTAChannel      DistributionChannel = "OTA"
	TMCChannel      DistributionChannel = "TMC"
	MetaSearchChannel DistributionChannel = "METASEARCH"
	NDCChannel      DistributionChannel = "NDC"
	PSSSelfService  DistributionChannel = "PSS_SELF_SERVICE"
	CallCenter      DistributionChannel = "CALL_CENTER"
	TravelAgent     DistributionChannel = "TRAVEL_AGENT"
	CorporatePortal DistributionChannel = "CORPORATE_PORTAL"
)

type ChannelConfiguration struct {
	ChannelID       string              `json:"channel_id"`
	ChannelType     DistributionChannel `json:"channel_type"`
	Provider        string              `json:"provider"`
	Enabled         bool                `json:"enabled"`
	Configuration   map[string]interface{} `json:"configuration"`
	Authentication  ChannelAuth         `json:"authentication"`
	RateLimits      ChannelRateLimits   `json:"rate_limits"`
	Features        ChannelFeatures     `json:"features"`
}

type ChannelAuth struct {
	Type        string                 `json:"type"` // API_KEY, OAUTH2, BASIC, CERTIFICATE
	Credentials map[string]interface{} `json:"credentials"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
}

type ChannelRateLimits struct {
	RequestsPerSecond int `json:"requests_per_second"`
	RequestsPerMinute int `json:"requests_per_minute"`
	RequestsPerHour   int `json:"requests_per_hour"`
	RequestsPerDay    int `json:"requests_per_day"`
}

type ChannelFeatures struct {
	SupportsRealTimeInventory bool     `json:"supports_real_time_inventory"`
	SupportsAncillaries       bool     `json:"supports_ancillaries"`
	SupportsSeatMaps          bool     `json:"supports_seat_maps"`
	SupportsLounge            bool     `json:"supports_lounge"`
	SupportedCabinClasses     []string `json:"supported_cabin_classes"`
	SupportedPaymentMethods   []string `json:"supported_payment_methods"`
}

// ============= Multi-Channel Distribution Models =============

type DistributionRequest struct {
	RequestID       string                 `json:"request_id"`
	Channel         DistributionChannel    `json:"channel"`
	RequestType     string                 `json:"request_type"`
	SourceData      map[string]interface{} `json:"source_data"`
	TargetChannels  []DistributionChannel  `json:"target_channels"`
	TransformRules  []TransformRule        `json:"transform_rules"`
	Metadata        map[string]interface{} `json:"metadata"`
	Priority        int                    `json:"priority"`
	Timeout         time.Duration          `json:"timeout"`
}

type TransformRule struct {
	SourceChannel DistributionChannel    `json:"source_channel"`
	TargetChannel DistributionChannel    `json:"target_channel"`
	FieldMappings []FieldMapping         `json:"field_mappings"`
	Conditions    []TransformCondition   `json:"conditions"`
}

type FieldMapping struct {
	SourceField    string      `json:"source_field"`
	TargetField    string      `json:"target_field"`
	TransformType  string      `json:"transform_type"`
	DefaultValue   interface{} `json:"default_value,omitempty"`
	Required       bool        `json:"required"`
}

type TransformCondition struct {
	Field     string      `json:"field"`
	Operator  string      `json:"operator"`
	Value     interface{} `json:"value"`
	Action    string      `json:"action"`
}

type DistributionResponse struct {
	RequestID     string                 `json:"request_id"`
	Channel       DistributionChannel    `json:"channel"`
	Success       bool                   `json:"success"`
	Data          map[string]interface{} `json:"data"`
	Errors        []DistributionError    `json:"errors,omitempty"`
	Warnings      []DistributionWarning  `json:"warnings,omitempty"`
	ProcessingTime time.Duration         `json:"processing_time"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type DistributionError struct {
	Code        string                 `json:"code"`
	Message     string                 `json:"message"`
	Field       string                 `json:"field,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

type DistributionWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
} 