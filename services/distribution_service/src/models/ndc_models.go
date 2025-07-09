package models

import (
	"encoding/xml"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// NDC Level 4 Message Structure
type NDCMessage struct {
	XMLName   xml.Name `xml:"NDCMessage"`
	Version   string   `xml:"version,attr"`
	Timestamp time.Time `xml:"timestamp,attr"`
	MessageID string   `xml:"messageId,attr"`
	Source    Source   `xml:"Source"`
	Document  Document `xml:"Document"`
}

type Source struct {
	Name         string `xml:"name,attr"`
	AirlineCode  string `xml:"airlineCode,attr"`
	RequestID    string `xml:"requestId,attr"`
	SessionID    string `xml:"sessionId,attr"`
}

type Document struct {
	Name    string      `xml:"name,attr"`
	Content interface{} `xml:",any"`
}

// =========== NDC Shopping Messages ===========

// AirShoppingRQ - NDC Air Shopping Request
type AirShoppingRQ struct {
	XMLName     xml.Name    `xml:"AirShoppingRQ"`
	Version     string      `xml:"Version"`
	MessageID   string      `xml:"MessageID"`
	Timestamp   time.Time   `xml:"Timestamp"`
	Source      SourceType  `xml:"Source"`
	Preference  Preference  `xml:"Preference"`
	Travelers   []Traveler  `xml:"Travelers>Traveler"`
	CoreQuery   CoreQuery   `xml:"CoreQuery"`
	Qualifiers  Qualifiers  `xml:"Qualifiers"`
}

type SourceType struct {
	RequestorID string `xml:"RequestorID"`
	PseudoCity  string `xml:"PseudoCity"`
	ISOCountry  string `xml:"ISOCountry"`
	POSCode     string `xml:"POSCode"`
}

type Preference struct {
	CurrencyCode    string   `xml:"CurrencyCode"`
	LanguageCode    string   `xml:"LanguageCode"`
	AirlinePrefs    []string `xml:"AirlinePrefs>AirlineCode"`
	CabinTypePrefs  []string `xml:"CabinTypePrefs>CabinType"`
	FarePrefs       FarePrefs `xml:"FarePrefs"`
}

type FarePrefs struct {
	Types    []string `xml:"Types>Type"`
	Private  bool     `xml:"Private"`
	Public   bool     `xml:"Public"`
}

type Traveler struct {
	TravelerID   string    `xml:"TravelerID,attr"`
	PassengerType string   `xml:"PassengerType"`
	Age          int       `xml:"Age,omitempty"`
	Gender       string    `xml:"Gender,omitempty"`
	Nationality  string    `xml:"Nationality,omitempty"`
	Residence    string    `xml:"Residence,omitempty"`
	LoyaltyProgram LoyaltyProgram `xml:"LoyaltyProgram,omitempty"`
}

type LoyaltyProgram struct {
	ProgramID   string `xml:"ProgramID"`
	MemberID    string `xml:"MemberID"`
	TierLevel   string `xml:"TierLevel"`
}

type CoreQuery struct {
	OriginDestinations []OriginDestination `xml:"OriginDestinations>OriginDestination"`
	DateFlexibility    DateFlexibility     `xml:"DateFlexibility"`
}

type OriginDestination struct {
	SegmentKey      string    `xml:"SegmentKey,attr"`
	Origin          string    `xml:"Departure>AirportCode"`
	Destination     string    `xml:"Arrival>AirportCode"`
	DepartureDate   time.Time `xml:"Departure>Date"`
	ArrivalDate     time.Time `xml:"Arrival>Date,omitempty"`
	FlightNumber    string    `xml:"MarketingCarrier>FlightNumber,omitempty"`
	AirlineCode     string    `xml:"MarketingCarrier>AirlineCode,omitempty"`
}

type DateFlexibility struct {
	DaysAfter  int `xml:"DaysAfter"`
	DaysBefore int `xml:"DaysBefore"`
}

type Qualifiers struct {
	FlightFilters    FlightFilters    `xml:"FlightFilters"`
	PricingQualifiers PricingQualifiers `xml:"PricingQualifiers"`
	ServiceFilters   ServiceFilters   `xml:"ServiceFilters"`
}

type FlightFilters struct {
	ConnectionTypes []string `xml:"ConnectionTypes>Type"`
	StopQuantity    int      `xml:"StopQuantity"`
	CarrierFilters  []string `xml:"CarrierFilters>AirlineCode"`
	TimeWindows     []TimeWindow `xml:"TimeWindows>TimeWindow"`
}

type TimeWindow struct {
	Departure string `xml:"Departure"`
	Arrival   string `xml:"Arrival"`
	Duration  string `xml:"Duration"`
}

type PricingQualifiers struct {
	PriceRanges     []PriceRange    `xml:"PriceRanges>PriceRange"`
	FareTypes       []string        `xml:"FareTypes>Type"`
	PaymentMethods  []PaymentMethod `xml:"PaymentMethods>Method"`
}

type PriceRange struct {
	MinAmount decimal.Decimal `xml:"MinAmount"`
	MaxAmount decimal.Decimal `xml:"MaxAmount"`
	Currency  string          `xml:"Currency,attr"`
}

type PaymentMethod struct {
	Type        string `xml:"Type"`
	CardType    string `xml:"CardType,omitempty"`
	InstallmentOptions bool `xml:"InstallmentOptions,omitempty"`
}

type ServiceFilters struct {
	SeatPreferences     []SeatPreference     `xml:"SeatPreferences>Preference"`
	MealPreferences     []MealPreference     `xml:"MealPreferences>Preference"`
	BaggagePreferences  []BaggagePreference  `xml:"BaggagePreferences>Preference"`
	ServiceCategories   []string             `xml:"ServiceCategories>Category"`
}

type SeatPreference struct {
	Type        string `xml:"Type"`
	Location    string `xml:"Location"`
	Pitch       string `xml:"Pitch"`
	Width       string `xml:"Width"`
	Features    []string `xml:"Features>Feature"`
}

type MealPreference struct {
	Type        string   `xml:"Type"`
	Dietary     []string `xml:"Dietary>Restriction"`
	Special     bool     `xml:"Special"`
}

type BaggagePreference struct {
	Type        string `xml:"Type"`
	WeightLimit int    `xml:"WeightLimit"`
	Dimensions  string `xml:"Dimensions"`
	Pieces      int    `xml:"Pieces"`
}

// AirShoppingRS - NDC Air Shopping Response
type AirShoppingRS struct {
	XMLName       xml.Name      `xml:"AirShoppingRS"`
	Version       string        `xml:"Version"`
	MessageID     string        `xml:"MessageID"`
	Timestamp     time.Time     `xml:"Timestamp"`
	Source        SourceType    `xml:"Source"`
	Success       Success       `xml:"Success,omitempty"`
	Warnings      []Warning     `xml:"Warnings>Warning,omitempty"`
	Errors        []Error       `xml:"Errors>Error,omitempty"`
	OffersGroup   OffersGroup   `xml:"OffersGroup"`
	DataLists     DataLists     `xml:"DataLists"`
	Metadata      Metadata      `xml:"Metadata"`
}

type Success struct {
	Code        string `xml:"Code,attr"`
	Description string `xml:",chardata"`
}

type Warning struct {
	Code        string `xml:"Code,attr"`
	Type        string `xml:"Type,attr"`
	Description string `xml:",chardata"`
}

type Error struct {
	Code        string `xml:"Code,attr"`
	Type        string `xml:"Type,attr"`
	Description string `xml:",chardata"`
	Owner       string `xml:"Owner,attr"`
}

type OffersGroup struct {
	CarrierOffers []CarrierOffers `xml:"CarrierOffers"`
	AirlineOffers []AirlineOffers `xml:"AirlineOffers"`
}

type CarrierOffers struct {
	AirlineCode string  `xml:"AirlineCode,attr"`
	Offers      []Offer `xml:"Offer"`
}

type AirlineOffers struct {
	AirlineCode string  `xml:"AirlineCode,attr"`
	Offers      []Offer `xml:"Offer"`
}

type Offer struct {
	OfferID       string        `xml:"OfferID,attr"`
	Owner         string        `xml:"Owner,attr"`
	ValidFrom     time.Time     `xml:"ValidFrom,attr"`
	ValidTo       time.Time     `xml:"ValidTo,attr"`
	TotalPrice    TotalPrice    `xml:"TotalPrice"`
	PricedOffer   PricedOffer   `xml:"PricedOffer"`
	OfferItems    []OfferItem   `xml:"OfferItems>OfferItem"`
	TimeLimits    TimeLimits    `xml:"TimeLimits"`
	BaggageAllowance []BaggageAllowance `xml:"BaggageAllowance"`
}

type TotalPrice struct {
	DetailCurrencyPrice DetailCurrencyPrice `xml:"DetailCurrencyPrice"`
	TaxSummary          []TaxSummary        `xml:"TaxSummary>Tax"`
	FeeSummary          []FeeSummary        `xml:"FeeSummary>Fee"`
}

type DetailCurrencyPrice struct {
	Total    PriceDetail `xml:"Total"`
	Base     PriceDetail `xml:"Base"`
	Taxes    PriceDetail `xml:"Taxes"`
	Fees     PriceDetail `xml:"Fees"`
}

type PriceDetail struct {
	Amount   decimal.Decimal `xml:",chardata"`
	Currency string          `xml:"Code,attr"`
}

type TaxSummary struct {
	TaxCode     string          `xml:"TaxCode,attr"`
	Amount      decimal.Decimal `xml:"Amount"`
	Currency    string          `xml:"Currency,attr"`
	Description string          `xml:"Description"`
}

type FeeSummary struct {
	FeeCode     string          `xml:"FeeCode,attr"`
	Amount      decimal.Decimal `xml:"Amount"`
	Currency    string          `xml:"Currency,attr"`
	Description string          `xml:"Description"`
}

type PricedOffer struct {
	AssociatedAdults   int             `xml:"AssociatedAdults"`
	AssociatedChildren int             `xml:"AssociatedChildren"`
	AssociatedInfants  int             `xml:"AssociatedInfants"`
	RequestedDate      time.Time       `xml:"RequestedDate"`
	Associations       Associations    `xml:"Associations"`
}

type Associations struct {
	AssociatedTraveler []AssociatedTraveler `xml:"AssociatedTraveler"`
	OtherAssociation   []OtherAssociation   `xml:"OtherAssociation"`
}

type AssociatedTraveler struct {
	TravelerReferences []string `xml:"TravelerReferences>TravelerReference"`
	PriceClass         string   `xml:"PriceClass"`
}

type OtherAssociation struct {
	Type      string   `xml:"Type,attr"`
	Reference []string `xml:"Reference"`
}

type OfferItem struct {
	OfferItemID           string                `xml:"OfferItemID,attr"`
	TotalPriceDetail      TotalPriceDetail      `xml:"TotalPriceDetail"`
	Service               Service               `xml:"Service"`
	FareDetail            FareDetail            `xml:"FareDetail"`
	OfferItemType         OfferItemType         `xml:"OfferItemType"`
}

type TotalPriceDetail struct {
	TotalAmount  PriceDetail     `xml:"TotalAmount"`
	BaseAmount   PriceDetail     `xml:"BaseAmount"`
	Taxes        []TaxDetail     `xml:"Taxes>Tax"`
	Fees         []FeeDetail     `xml:"Fees>Fee"`
	Discounts    []DiscountDetail `xml:"Discounts>Discount"`
}

type TaxDetail struct {
	TaxCode     string          `xml:"TaxCode,attr"`
	Amount      decimal.Decimal `xml:"Amount"`
	Currency    string          `xml:"Currency,attr"`
	Description string          `xml:"Description"`
	Country     string          `xml:"Country,attr"`
}

type FeeDetail struct {
	FeeCode     string          `xml:"FeeCode,attr"`
	Amount      decimal.Decimal `xml:"Amount"`
	Currency    string          `xml:"Currency,attr"`
	Description string          `xml:"Description"`
	Type        string          `xml:"Type,attr"`
}

type DiscountDetail struct {
	DiscountCode string          `xml:"DiscountCode,attr"`
	Amount       decimal.Decimal `xml:"Amount"`
	Currency     string          `xml:"Currency,attr"`
	Description  string          `xml:"Description"`
	Type         string          `xml:"Type,attr"`
}

type Service struct {
	ServiceID       string          `xml:"ServiceID,attr"`
	ServiceType     string          `xml:"ServiceType,attr"`
	FlightRefs      []string        `xml:"FlightRefs>FlightRef"`
	PassengerRefs   []string        `xml:"PassengerRefs>PassengerRef"`
	ServiceDefinition ServiceDefinition `xml:"ServiceDefinition"`
}

type ServiceDefinition struct {
	Name           string         `xml:"Name"`
	Description    string         `xml:"Description"`
	Code           string         `xml:"Code,attr"`
	ServiceCategory string        `xml:"ServiceCategory"`
	Encoding       string         `xml:"Encoding"`
	Media          []MediaItem    `xml:"Media>MediaItem"`
	Terms          TermsConditions `xml:"Terms"`
}

type MediaItem struct {
	Type         string `xml:"Type,attr"`
	URL          string `xml:"URL"`
	Description  string `xml:"Description"`
	AltText      string `xml:"AltText"`
}

type TermsConditions struct {
	Language    string   `xml:"Language,attr"`
	Text        string   `xml:"Text"`
	URL         string   `xml:"URL,omitempty"`
	Categories  []string `xml:"Categories>Category"`
}

type FareDetail struct {
	FareComponent     []FareComponent     `xml:"FareComponent"`
	FareBasis         string              `xml:"FareBasis"`
	FareRules         FareRules           `xml:"FareRules"`
	FareCalculation   string              `xml:"FareCalculation"`
	PriceClass        PriceClass          `xml:"PriceClass"`
}

type FareComponent struct {
	FareBasisCode string          `xml:"FareBasisCode"`
	Amount        decimal.Decimal `xml:"Amount"`
	Currency      string          `xml:"Currency,attr"`
	CabinType     string          `xml:"CabinType"`
	ClassOfService string         `xml:"ClassOfService"`
	FareType      string          `xml:"FareType"`
	SegmentRefs   []string        `xml:"SegmentRefs>SegmentRef"`
}

type FareRules struct {
	Penalty         Penalty         `xml:"Penalty"`
	Restrictions    Restrictions    `xml:"Restrictions"`
	Applicability   Applicability   `xml:"Applicability"`
}

type Penalty struct {
	CancellationPenalty PenaltyDetail `xml:"CancellationPenalty"`
	ChangePenalty       PenaltyDetail `xml:"ChangePenalty"`
	NoShowPenalty       PenaltyDetail `xml:"NoShowPenalty"`
}

type PenaltyDetail struct {
	Application string          `xml:"Application,attr"`
	Amount      decimal.Decimal `xml:"Amount"`
	Currency    string          `xml:"Currency,attr"`
	Percentage  decimal.Decimal `xml:"Percentage,omitempty"`
	Condition   string          `xml:"Condition"`
}

type Restrictions struct {
	AdvancePurchase   AdvancePurchase   `xml:"AdvancePurchase"`
	MinStay           MinStay           `xml:"MinStay"`
	MaxStay           MaxStay           `xml:"MaxStay"`
	Refundability     string            `xml:"Refundability"`
	Exchangeability   string            `xml:"Exchangeability"`
}

type AdvancePurchase struct {
	Required bool   `xml:"Required,attr"`
	Days     int    `xml:"Days"`
	Hours    int    `xml:"Hours"`
}

type MinStay struct {
	Required bool   `xml:"Required,attr"`
	Days     int    `xml:"Days"`
	Pattern  string `xml:"Pattern"`
}

type MaxStay struct {
	Required bool   `xml:"Required,attr"`
	Days     int    `xml:"Days"`
	Months   int    `xml:"Months"`
}

type Applicability struct {
	DateRange   DateRange   `xml:"DateRange"`
	RouteScope  RouteScope  `xml:"RouteScope"`
	FlightScope FlightScope `xml:"FlightScope"`
}

type DateRange struct {
	StartDate time.Time `xml:"StartDate"`
	EndDate   time.Time `xml:"EndDate"`
}

type RouteScope struct {
	Origins      []string `xml:"Origins>Airport"`
	Destinations []string `xml:"Destinations>Airport"`
	International bool    `xml:"International"`
	Domestic      bool    `xml:"Domestic"`
}

type FlightScope struct {
	Airlines     []string `xml:"Airlines>AirlineCode"`
	FlightTypes  []string `xml:"FlightTypes>Type"`
	CabinTypes   []string `xml:"CabinTypes>Type"`
}

type PriceClass struct {
	ClassCode    string   `xml:"ClassCode"`
	ClassName    string   `xml:"ClassName"`
	DisplayName  string   `xml:"DisplayName"`
	Descriptions []string `xml:"Descriptions>Description"`
}

type OfferItemType struct {
	Code        string `xml:"Code,attr"`
	Definition  string `xml:"Definition"`
	SubCode     string `xml:"SubCode,omitempty"`
}

type TimeLimits struct {
	OfferExpiration     time.Time `xml:"OfferExpiration"`
	PaymentTimeLimit    time.Time `xml:"PaymentTimeLimit"`
	TicketingTimeLimit  time.Time `xml:"TicketingTimeLimit"`
	PriceGuaranteeTime  time.Time `xml:"PriceGuaranteeTime"`
}

type BaggageAllowance struct {
	BaggageCategory    string `xml:"BaggageCategory,attr"`
	AllowanceType      string `xml:"AllowanceType"`
	MaxWeight          int    `xml:"MaxWeight"`
	WeightUnit         string `xml:"WeightUnit"`
	MaxPieces          int    `xml:"MaxPieces"`
	MaxSize            string `xml:"MaxSize"`
	Applicability      BaggageApplicability `xml:"Applicability"`
}

type BaggageApplicability struct {
	SegmentRefs    []string `xml:"SegmentRefs>SegmentRef"`
	PassengerRefs  []string `xml:"PassengerRefs>PassengerRef"`
	ServiceRefs    []string `xml:"ServiceRefs>ServiceRef"`
}

// DataLists contain reference data used in offers
type DataLists struct {
	PassengerList       PassengerList       `xml:"PassengerList"`
	ContactList         ContactList         `xml:"ContactList"`
	FlightSegmentList   FlightSegmentList   `xml:"FlightSegmentList"`
	FlightList          FlightList          `xml:"FlightList"`
	OriginDestList      OriginDestList      `xml:"OriginDestList"`
	PriceClassList      PriceClassList      `xml:"PriceClassList"`
	ServiceDefinitionList ServiceDefinitionList `xml:"ServiceDefinitionList"`
}

type PassengerList struct {
	Passenger []PassengerData `xml:"Passenger"`
}

type PassengerData struct {
	PassengerID   string `xml:"PassengerID,attr"`
	ObjectKey     string `xml:"ObjectKey,attr"`
	PTC           string `xml:"PTC"`
	CitizenshipCountryCode string `xml:"CitizenshipCountryCode"`
	Individual    Individual `xml:"Individual"`
	LoyaltyPrograms []LoyaltyProgram `xml:"LoyaltyPrograms>LoyaltyProgram"`
}

type Individual struct {
	Birthdate      time.Time     `xml:"Birthdate"`
	Gender         string        `xml:"Gender"`
	NameTitle      string        `xml:"NameTitle"`
	GivenName      []string      `xml:"GivenName"`
	MiddleName     []string      `xml:"MiddleName"`
	Surname        string        `xml:"Surname"`
	ProfileID      string        `xml:"ProfileID"`
	IdentityDoc    IdentityDoc   `xml:"IdentityDoc"`
}

type IdentityDoc struct {
	IdentityDocNumber string    `xml:"IdentityDocNumber"`
	IdentityDocType   string    `xml:"IdentityDocType"`
	ExpiryDate        time.Time `xml:"ExpiryDate"`
	IssuingCountry    string    `xml:"IssuingCountry"`
}

type ContactList struct {
	ContactInformation []ContactInformation `xml:"ContactInformation"`
}

type ContactInformation struct {
	ContactID       string          `xml:"ContactID,attr"`
	ContactType     string          `xml:"ContactType,attr"`
	EmailAddress    EmailAddress    `xml:"EmailAddress"`
	Phone           []PhoneContact  `xml:"Phone"`
	PostalAddress   PostalAddress   `xml:"PostalAddress"`
	OtherAddress    []OtherAddress  `xml:"OtherAddress"`
}

type EmailAddress struct {
	EmailAddressValue string `xml:"EmailAddressValue"`
	Comment           string `xml:"Comment,omitempty"`
}

type PhoneContact struct {
	Label       string `xml:"Label,attr"`
	PhoneNumber string `xml:"PhoneNumber"`
	Extension   string `xml:"Extension,omitempty"`
	CountryCode string `xml:"CountryCode,omitempty"`
}

type PostalAddress struct {
	Street       []string `xml:"Street"`
	CityName     string   `xml:"CityName"`
	PostalCode   string   `xml:"PostalCode"`
	CountrySubdivision string `xml:"CountrySubdivision"`
	CountryCode  string   `xml:"CountryCode"`
}

type OtherAddress struct {
	Type    string `xml:"Type,attr"`
	Address string `xml:"Address"`
}

type FlightSegmentList struct {
	FlightSegment []FlightSegment `xml:"FlightSegment"`
}

type FlightSegment struct {
	SegmentKey      string    `xml:"SegmentKey,attr"`
	Departure       Departure `xml:"Departure"`
	Arrival         Arrival   `xml:"Arrival"`
	MarketingCarrier MarketingCarrier `xml:"MarketingCarrier"`
	OperatingCarrier OperatingCarrier `xml:"OperatingCarrier"`
	Equipment       Equipment `xml:"Equipment"`
	ClassOfService  []ClassOfService `xml:"ClassOfService"`
	FlightDetail    FlightDetail `xml:"FlightDetail"`
}

type Departure struct {
	AirportCode  string    `xml:"AirportCode"`
	AirportName  string    `xml:"AirportName"`
	Date         time.Time `xml:"Date"`
	Time         string    `xml:"Time"`
	Terminal     string    `xml:"Terminal,omitempty"`
	TimezoneCode string    `xml:"TimezoneCode"`
}

type Arrival struct {
	AirportCode  string    `xml:"AirportCode"`
	AirportName  string    `xml:"AirportName"`
	Date         time.Time `xml:"Date"`
	Time         string    `xml:"Time"`
	Terminal     string    `xml:"Terminal,omitempty"`
	TimezoneCode string    `xml:"TimezoneCode"`
	ChangeOfDay  int       `xml:"ChangeOfDay,omitempty"`
}

type MarketingCarrier struct {
	AirlineCode string `xml:"AirlineCode"`
	Name        string `xml:"Name"`
	FlightNumber string `xml:"FlightNumber"`
}

type OperatingCarrier struct {
	AirlineCode  string `xml:"AirlineCode"`
	Name         string `xml:"Name"`
	FlightNumber string `xml:"FlightNumber"`
	Disclosures  []string `xml:"Disclosures>Disclosure"`
}

type Equipment struct {
	AircraftCode string `xml:"AircraftCode"`
	Name         string `xml:"Name"`
	AircraftConfiguration AircraftConfiguration `xml:"AircraftConfiguration"`
}

type AircraftConfiguration struct {
	CabinType []CabinType `xml:"CabinType"`
}

type CabinType struct {
	CabinTypeCode string `xml:"CabinTypeCode,attr"`
	CabinTypeName string `xml:"CabinTypeName"`
	SeatCount     int    `xml:"SeatCount"`
	ColumnCount   int    `xml:"ColumnCount"`
	RowCount      int    `xml:"RowCount"`
}

type ClassOfService struct {
	Code        string    `xml:"Code,attr"`
	MarketingName string  `xml:"MarketingName"`
	Availability int      `xml:"Availability"`
	Cabin       string    `xml:"Cabin"`
}

type FlightDetail struct {
	FlightDuration string     `xml:"FlightDuration"`
	Stops          StopDetail `xml:"Stops"`
	Distance       Distance   `xml:"Distance"`
}

type StopDetail struct {
	StopQuantity int    `xml:"StopQuantity"`
	StopLocations []StopLocation `xml:"StopLocations>StopLocation"`
}

type StopLocation struct {
	LocationCode string `xml:"LocationCode"`
	Duration     string `xml:"Duration"`
	Equipment    string `xml:"Equipment,omitempty"`
}

type Distance struct {
	Value decimal.Decimal `xml:"Value"`
	UOM   string          `xml:"UOM,attr"`
}

type FlightList struct {
	Flight []Flight `xml:"Flight"`
}

type Flight struct {
	FlightKey     string         `xml:"FlightKey,attr"`
	Journey       Journey        `xml:"Journey"`
	SegmentReferences []string     `xml:"SegmentReferences>SegmentRef"`
	Settlement    Settlement     `xml:"Settlement"`
}

type Journey struct {
	JourneyDistance Distance `xml:"JourneyDistance"`
	JourneyTime     string   `xml:"JourneyTime"`
}

type Settlement struct {
	Method     string `xml:"Method,attr"`
	Interline  bool   `xml:"Interline"`
	BSPCountry string `xml:"BSPCountry"`
}

type OriginDestList struct {
	OriginDestination []OriginDestData `xml:"OriginDestination"`
}

type OriginDestData struct {
	OriginDestKey     string   `xml:"OriginDestKey,attr"`
	DepartureCode     string   `xml:"DepartureCode"`
	ArrivalCode       string   `xml:"ArrivalCode"`
	FlightReferences  []string `xml:"FlightReferences>FlightRef"`
}

type PriceClassList struct {
	PriceClass []PriceClassData `xml:"PriceClass"`
}

type PriceClassData struct {
	PriceClassID string      `xml:"PriceClassID,attr"`
	Name         string      `xml:"Name"`
	Code         string      `xml:"Code"`
	Descriptions []Description `xml:"Descriptions>Description"`
}

type Description struct {
	Text     string `xml:"Text"`
	Language string `xml:"Language,attr"`
	Application string `xml:"Application,attr"`
}

type ServiceDefinitionList struct {
	ServiceDefinition []ServiceDefinitionData `xml:"ServiceDefinition"`
}

type ServiceDefinitionData struct {
	ServiceDefinitionID string        `xml:"ServiceDefinitionID,attr"`
	Name                string        `xml:"Name"`
	Code                string        `xml:"Code"`
	ServiceCategory     string        `xml:"ServiceCategory"`
	ServiceType         string        `xml:"ServiceType"`
	Descriptions        []Description `xml:"Descriptions>Description"`
	BookingInstructions BookingInstructions `xml:"BookingInstructions"`
}

type BookingInstructions struct {
	Method      string   `xml:"Method,attr"`
	Instruction []string `xml:"Instruction"`
}

// Metadata contains processing information
type Metadata struct {
	Other      OtherMetadata      `xml:"Other"`
	Policies   PolicyMetadata     `xml:"Policies"`
	Shopping   ShoppingMetadata   `xml:"Shopping"`
}

type OtherMetadata struct {
	OtherMetadata []MetadataItem `xml:"OtherMetadata"`
}

type MetadataItem struct {
	Context     string `xml:"Context,attr"`
	Description string `xml:"Description"`
	Value       string `xml:"Value"`
}

type PolicyMetadata struct {
	PriceGuaranteePolicy PriceGuaranteePolicy `xml:"PriceGuaranteePolicy"`
	PaymentTimeLimitPolicy PaymentTimeLimitPolicy `xml:"PaymentTimeLimitPolicy"`
	PenaltyPolicy        PenaltyPolicy          `xml:"PenaltyPolicy"`
}

type PriceGuaranteePolicy struct {
	GuaranteeTime string `xml:"GuaranteeTime"`
	Application   string `xml:"Application"`
}

type PaymentTimeLimitPolicy struct {
	TimeLimit   string `xml:"TimeLimit"`
	Application string `xml:"Application"`
}

type PenaltyPolicy struct {
	CancellationPenalty string `xml:"CancellationPenalty"`
	ChangePenalty       string `xml:"ChangePenalty"`
	Application         string `xml:"Application"`
}

type ShoppingMetadata struct {
	ShoppingResponseID    ShoppingResponseID    `xml:"ShoppingResponseID"`
	AugmentationPoint     AugmentationPoint     `xml:"AugmentationPoint"`
}

type ShoppingResponseID struct {
	ResponseID string    `xml:"ResponseID"`
	Owner      string    `xml:"Owner,attr"`
	Timestamp  time.Time `xml:"Timestamp"`
}

type AugmentationPoint struct {
	Extensions []Extension `xml:"Extensions>Extension"`
}

type Extension struct {
	Name  string `xml:"Name,attr"`
	Value string `xml:"Value"`
} 