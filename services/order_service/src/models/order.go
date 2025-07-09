package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// OrderStatus represents order status following IATA ONE Order standard
type OrderStatus string

const (
	OrderStatusCreated   OrderStatus = "CREATED"
	OrderStatusConfirmed OrderStatus = "CONFIRMED"
	OrderStatusTicketed  OrderStatus = "TICKETED"
	OrderStatusFulfilled OrderStatus = "FULFILLED"
	OrderStatusModified  OrderStatus = "MODIFIED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
	OrderStatusRefunded  OrderStatus = "REFUNDED"
	OrderStatusExpired   OrderStatus = "EXPIRED"
)

// PaymentStatus represents payment status for order financial tracking
type PaymentStatus string

const (
	PaymentStatusPending           PaymentStatus = "PENDING"
	PaymentStatusAuthorized        PaymentStatus = "AUTHORIZED"
	PaymentStatusCaptured          PaymentStatus = "CAPTURED"
	PaymentStatusFailed            PaymentStatus = "FAILED"
	PaymentStatusRefunded          PaymentStatus = "REFUNDED"
	PaymentStatusPartiallyRefunded PaymentStatus = "PARTIALLY_REFUNDED"
)

// CurrencyCode represents ISO 4217 currency codes
type CurrencyCode string

const (
	CurrencyUSD CurrencyCode = "USD"
	CurrencyEUR CurrencyCode = "EUR"
	CurrencyGBP CurrencyCode = "GBP"
	CurrencyAED CurrencyCode = "AED"
	CurrencySGD CurrencyCode = "SGD"
	CurrencyJPY CurrencyCode = "JPY"
	CurrencyCAD CurrencyCode = "CAD"
	CurrencyAUD CurrencyCode = "AUD"
)

// ServiceType represents service types within an order
type ServiceType string

const (
	ServiceTypeFlight           ServiceType = "FLIGHT"
	ServiceTypeSeat             ServiceType = "SEAT"
	ServiceTypeBaggage          ServiceType = "BAGGAGE"
	ServiceTypeMeal             ServiceType = "MEAL"
	ServiceTypeLounge           ServiceType = "LOUNGE"
	ServiceTypeUpgrade          ServiceType = "UPGRADE"
	ServiceTypeInsurance        ServiceType = "INSURANCE"
	ServiceTypeWiFi             ServiceType = "WIFI"
	ServiceTypePriorityBoarding ServiceType = "PRIORITY_BOARDING"
	ServiceTypeFastTrack        ServiceType = "FAST_TRACK"
)

// PassengerType represents different passenger categories
type PassengerType string

const (
	PassengerTypeAdult  PassengerType = "ADULT"
	PassengerTypeChild  PassengerType = "CHILD"
	PassengerTypeInfant PassengerType = "INFANT"
)

// OrderItem represents an individual item within an order
type OrderItem struct {
	ID          uint            `gorm:"primaryKey" json:"id"`
	ItemID      string          `gorm:"uniqueIndex;size:36" json:"item_id"`
	OrderID     string          `gorm:"index;size:36" json:"order_id"`
	ServiceType ServiceType     `gorm:"size:50" json:"service_type"`
	ServiceID   *string         `gorm:"size:100" json:"service_id,omitempty"`
	Description string          `gorm:"size:500" json:"description"`
	Quantity    int             `json:"quantity"`
	UnitPrice   decimal.Decimal `gorm:"type:decimal(10,2)" json:"unit_price"`
	Currency    CurrencyCode    `gorm:"size:3" json:"currency"`
	TotalPrice  decimal.Decimal `gorm:"type:decimal(10,2)" json:"total_price"`
	Metadata    string          `gorm:"type:text" json:"metadata"` // JSON stored as string
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// GetMetadata returns the metadata as a map
func (oi *OrderItem) GetMetadata() (map[string]interface{}, error) {
	if oi.Metadata == "" {
		return make(map[string]interface{}), nil
	}
	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(oi.Metadata), &metadata)
	return metadata, err
}

// SetMetadata sets the metadata from a map
func (oi *OrderItem) SetMetadata(metadata map[string]interface{}) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	oi.Metadata = string(data)
	return nil
}

// BeforeCreate sets the ItemID and calculates TotalPrice
func (oi *OrderItem) BeforeCreate(tx *gorm.DB) error {
	if oi.ItemID == "" {
		oi.ItemID = uuid.New().String()
	}
	oi.TotalPrice = oi.UnitPrice.Mul(decimal.NewFromInt(int64(oi.Quantity)))
	return nil
}

// PaymentMethod represents payment method information
type PaymentMethod struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	OrderID        string    `gorm:"index;size:36" json:"order_id"`
	PaymentType    string    `gorm:"size:50" json:"payment_type"` // VISA, MASTERCARD, PAYPAL
	LastFour       string    `gorm:"size:4" json:"last_four"`
	ExpiryMonth    *int      `json:"expiry_month,omitempty"`
	ExpiryYear     *int      `json:"expiry_year,omitempty"`
	CardholderName *string   `gorm:"size:100" json:"cardholder_name,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ContactInfo represents customer contact information
type ContactInfo struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	OrderID   string    `gorm:"index;size:36" json:"order_id"`
	Email     string    `gorm:"size:255" json:"email"`
	Phone     *string   `gorm:"size:20" json:"phone,omitempty"`
	Address   string    `gorm:"type:text" json:"address"` // JSON stored as string
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetAddress returns the address as a map
func (ci *ContactInfo) GetAddress() (map[string]interface{}, error) {
	if ci.Address == "" {
		return make(map[string]interface{}), nil
	}
	var address map[string]interface{}
	err := json.Unmarshal([]byte(ci.Address), &address)
	return address, err
}

// SetAddress sets the address from a map
func (ci *ContactInfo) SetAddress(address map[string]interface{}) error {
	data, err := json.Marshal(address)
	if err != nil {
		return err
	}
	ci.Address = string(data)
	return nil
}

// Validate validates contact information
func (ci *ContactInfo) Validate() error {
	// Email validation
	emailPattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	if matched, _ := regexp.MatchString(emailPattern, ci.Email); !matched {
		return fmt.Errorf("invalid email format: %s", ci.Email)
	}
	
	// Phone validation (basic)
	if ci.Phone != nil && *ci.Phone != "" {
		phonePattern := `^\+?[\d\s\-\(\)]{10,}$`
		if matched, _ := regexp.MatchString(phonePattern, *ci.Phone); !matched {
			return fmt.Errorf("invalid phone format: %s", *ci.Phone)
		}
	}
	
	return nil
}

// PassengerInfo represents passenger information within an order
type PassengerInfo struct {
	ID             uint          `gorm:"primaryKey" json:"id"`
	PassengerID    string        `gorm:"uniqueIndex;size:36" json:"passenger_id"`
	OrderID        string        `gorm:"index;size:36" json:"order_id"`
	FirstName      string        `gorm:"size:100" json:"first_name"`
	LastName       string        `gorm:"size:100" json:"last_name"`
	DateOfBirth    string        `gorm:"size:10" json:"date_of_birth"` // YYYY-MM-DD
	Gender         string        `gorm:"size:10" json:"gender"`
	Nationality    string        `gorm:"size:3" json:"nationality"`    // ISO 3166-1 alpha-3
	PassportNumber *string       `gorm:"size:20" json:"passport_number,omitempty"`
	PassengerType  PassengerType `gorm:"size:10" json:"passenger_type"`
	LoyaltyNumber  *string       `gorm:"size:50" json:"loyalty_number,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// BeforeCreate sets the PassengerID
func (pi *PassengerInfo) BeforeCreate(tx *gorm.DB) error {
	if pi.PassengerID == "" {
		pi.PassengerID = uuid.New().String()
	}
	return nil
}

// Validate validates passenger information
func (pi *PassengerInfo) Validate() error {
	if pi.FirstName == "" || pi.LastName == "" {
		return errors.New("first name and last name are required")
	}
	
	if pi.PassengerType != PassengerTypeAdult && 
	   pi.PassengerType != PassengerTypeChild && 
	   pi.PassengerType != PassengerTypeInfant {
		return fmt.Errorf("invalid passenger type: %s", pi.PassengerType)
	}
	
	return nil
}

// AuditEntry represents an audit trail entry for order changes
type AuditEntry struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	OrderID     string    `gorm:"index;size:36" json:"order_id"`
	Action      string    `gorm:"size:100" json:"action"`
	Description string    `gorm:"size:500" json:"description"`
	UserID      *string   `gorm:"size:36" json:"user_id,omitempty"`
	Metadata    string    `gorm:"type:text" json:"metadata"` // JSON stored as string
	Timestamp   time.Time `json:"timestamp"`
}

// GetMetadata returns the metadata as a map
func (ae *AuditEntry) GetMetadata() (map[string]interface{}, error) {
	if ae.Metadata == "" {
		return make(map[string]interface{}), nil
	}
	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(ae.Metadata), &metadata)
	return metadata, err
}

// SetMetadata sets the metadata from a map
func (ae *AuditEntry) SetMetadata(metadata map[string]interface{}) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	ae.Metadata = string(data)
	return nil
}

// Order represents the IATA ONE Order Implementation
type Order struct {
	// Primary key and basic identifiers
	ID               uint            `gorm:"primaryKey" json:"id"`
	OrderID          string          `gorm:"uniqueIndex;size:36" json:"order_id"`
	CustomerID       string          `gorm:"index;size:36" json:"customer_id"`
	OrderReference   string          `gorm:"uniqueIndex;size:10" json:"order_reference"`
	
	// Order metadata
	Status           OrderStatus     `gorm:"size:20" json:"status"`
	Channel          string          `gorm:"size:20" json:"channel"` // DIRECT, GDS, OTA, NDC
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	ModifiedAt       *time.Time      `json:"modified_at,omitempty"`
	ExpiresAt        *time.Time      `json:"expires_at,omitempty"`
	
	// Financial information
	SubtotalAmount   decimal.Decimal `gorm:"type:decimal(10,2)" json:"subtotal_amount"`
	TaxAmount        decimal.Decimal `gorm:"type:decimal(10,2)" json:"tax_amount"`
	TotalAmount      decimal.Decimal `gorm:"type:decimal(10,2)" json:"total_amount"`
	Currency         CurrencyCode    `gorm:"size:3" json:"currency"`
	
	// Payment information
	PaymentStatus    PaymentStatus   `gorm:"size:30" json:"payment_status"`
	PaymentReference *string         `gorm:"size:100" json:"payment_reference,omitempty"`
	
	// Fulfillment information
	PNRReference     *string         `gorm:"size:10" json:"pnr_reference,omitempty"`
	TicketNumbers    string          `gorm:"type:text" json:"ticket_numbers"` // JSON array as string
	
	// Business metadata
	BookingClass     string          `gorm:"size:10" json:"booking_class"`
	Route            string          `gorm:"size:20" json:"route"`
	DepartureDate    *time.Time      `json:"departure_date,omitempty"`
	ReturnDate       *time.Time      `json:"return_date,omitempty"`
	
	// Version control for modifications
	Version          int             `gorm:"default:1" json:"version"`
	PreviousOrderID  *string         `gorm:"size:36" json:"previous_order_id,omitempty"`
	
	// Additional metadata
	Metadata         string          `gorm:"type:text" json:"metadata"` // JSON stored as string
	
	// Relationships
	ContactInfo      ContactInfo     `gorm:"foreignKey:OrderID;references:OrderID" json:"contact_info"`
	Passengers       []PassengerInfo `gorm:"foreignKey:OrderID;references:OrderID" json:"passengers"`
	Items            []OrderItem     `gorm:"foreignKey:OrderID;references:OrderID" json:"items"`
	PaymentMethods   []PaymentMethod `gorm:"foreignKey:OrderID;references:OrderID" json:"payment_methods"`
	AuditTrail       []AuditEntry    `gorm:"foreignKey:OrderID;references:OrderID" json:"audit_trail"`
}

// BeforeCreate sets the OrderID and OrderReference
func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.OrderID == "" {
		o.OrderID = uuid.New().String()
	}
	if o.OrderReference == "" {
		o.OrderReference = o.generateOrderReference()
	}
	return nil
}

// generateOrderReference generates a unique order reference
func (o *Order) generateOrderReference() string {
	// Generate a 6-character alphanumeric reference
	timestamp := time.Now().Unix()
	ref := fmt.Sprintf("OR%s", strings.ToUpper(uuid.New().String()[:6]))
	
	// Add timestamp-based suffix for uniqueness
	suffix := strconv.FormatInt(timestamp%10000, 10)
	if len(suffix) < 4 {
		suffix = fmt.Sprintf("%04s", suffix)
	}
	
	return ref[:8] // Ensure 8 characters total
}

// GetMetadata returns the metadata as a map
func (o *Order) GetMetadata() (map[string]interface{}, error) {
	if o.Metadata == "" {
		return make(map[string]interface{}), nil
	}
	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(o.Metadata), &metadata)
	return metadata, err
}

// SetMetadata sets the metadata from a map
func (o *Order) SetMetadata(metadata map[string]interface{}) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	o.Metadata = string(data)
	return nil
}

// GetTicketNumbers returns ticket numbers as a slice
func (o *Order) GetTicketNumbers() ([]string, error) {
	if o.TicketNumbers == "" {
		return []string{}, nil
	}
	var tickets []string
	err := json.Unmarshal([]byte(o.TicketNumbers), &tickets)
	return tickets, err
}

// SetTicketNumbers sets ticket numbers from a slice
func (o *Order) SetTicketNumbers(tickets []string) error {
	data, err := json.Marshal(tickets)
	if err != nil {
		return err
	}
	o.TicketNumbers = string(data)
	return nil
}

// AddAuditEntry adds an audit trail entry
func (o *Order) AddAuditEntry(action, description string, userID *string, metadata map[string]interface{}) error {
	entry := AuditEntry{
		OrderID:     o.OrderID,
		Action:      action,
		Description: description,
		UserID:      userID,
		Timestamp:   time.Now(),
	}
	
	if metadata != nil {
		if err := entry.SetMetadata(metadata); err != nil {
			return err
		}
	}
	
	o.AuditTrail = append(o.AuditTrail, entry)
	return nil
}

// AddItem adds an order item
func (o *Order) AddItem(item *OrderItem) {
	item.OrderID = o.OrderID
	o.Items = append(o.Items, *item)
	o.RecalculateTotals()
}

// RemoveItem removes an order item by ItemID
func (o *Order) RemoveItem(itemID string) bool {
	for i, item := range o.Items {
		if item.ItemID == itemID {
			o.Items = append(o.Items[:i], o.Items[i+1:]...)
			o.RecalculateTotals()
			return true
		}
	}
	return false
}

// RecalculateTotals recalculates order totals
func (o *Order) RecalculateTotals() {
	subtotal := decimal.Zero
	for _, item := range o.Items {
		subtotal = subtotal.Add(item.TotalPrice)
	}
	
	o.SubtotalAmount = subtotal
	// Simple tax calculation (15%)
	o.TaxAmount = subtotal.Mul(decimal.NewFromFloat(0.15))
	o.TotalAmount = o.SubtotalAmount.Add(o.TaxAmount)
}

// ConfirmPayment confirms payment and updates status
func (o *Order) ConfirmPayment() error {
	if o.PaymentStatus != PaymentStatusAuthorized {
		return errors.New("payment not authorized")
	}
	
	o.PaymentStatus = PaymentStatusCaptured
	o.Status = OrderStatusConfirmed
	o.UpdatedAt = time.Now()
	
	return o.AddAuditEntry("PAYMENT_CONFIRMED", "Payment successfully captured", nil, nil)
}

// AddFulfillmentInfo adds fulfillment information
func (o *Order) AddFulfillmentInfo(pnrReference string, ticketNumbers []string) error {
	o.PNRReference = &pnrReference
	
	if err := o.SetTicketNumbers(ticketNumbers); err != nil {
		return err
	}
	
	o.Status = OrderStatusTicketed
	o.UpdatedAt = time.Now()
	
	metadata := map[string]interface{}{
		"pnr_reference":  pnrReference,
		"ticket_numbers": ticketNumbers,
	}
	
	return o.AddAuditEntry("FULFILLMENT_INFO_ADDED", "PNR and ticket information added", nil, metadata)
}

// CancelOrder cancels the order
func (o *Order) CancelOrder(reason string, userID *string) error {
	if o.Status == OrderStatusCancelled || o.Status == OrderStatusRefunded {
		return errors.New("order already cancelled or refunded")
	}
	
	o.Status = OrderStatusCancelled
	o.UpdatedAt = time.Now()
	
	metadata := map[string]interface{}{
		"cancellation_reason": reason,
	}
	
	return o.AddAuditEntry("ORDER_CANCELLED", reason, userID, metadata)
}

// RefundOrder processes a refund
func (o *Order) RefundOrder(refundAmount decimal.Decimal, reason string, userID *string) error {
	if o.Status != OrderStatusCancelled && o.Status != OrderStatusTicketed {
		return errors.New("order must be cancelled or ticketed to process refund")
	}
	
	if refundAmount.GreaterThan(o.TotalAmount) {
		return errors.New("refund amount cannot exceed total order amount")
	}
	
	if refundAmount.Equal(o.TotalAmount) {
		o.PaymentStatus = PaymentStatusRefunded
	} else {
		o.PaymentStatus = PaymentStatusPartiallyRefunded
	}
	
	o.Status = OrderStatusRefunded
	o.UpdatedAt = time.Now()
	
	metadata := map[string]interface{}{
		"refund_amount": refundAmount.String(),
		"refund_reason": reason,
	}
	
	return o.AddAuditEntry("ORDER_REFUNDED", fmt.Sprintf("Refund processed: %s", refundAmount.String()), userID, metadata)
}

// ModifyOrder creates a new version of the order with modifications
func (o *Order) ModifyOrder(modifications map[string]interface{}, userID *string) error {
	// Increment version
	o.Version++
	o.Status = OrderStatusModified
	o.ModifiedAt = &[]time.Time{time.Now()}[0]
	o.UpdatedAt = time.Now()
	
	metadata := map[string]interface{}{
		"modifications": modifications,
		"previous_version": o.Version - 1,
	}
	
	return o.AddAuditEntry("ORDER_MODIFIED", "Order modified", userID, metadata)
}

// Validate validates the order
func (o *Order) Validate() error {
	if o.CustomerID == "" {
		return errors.New("customer ID is required")
	}
	
	if len(o.Passengers) == 0 {
		return errors.New("at least one passenger is required")
	}
	
	if len(o.Items) == 0 {
		return errors.New("at least one order item is required")
	}
	
	// Validate contact info
	if err := o.ContactInfo.Validate(); err != nil {
		return fmt.Errorf("contact info validation failed: %v", err)
	}
	
	// Validate passengers
	for i, passenger := range o.Passengers {
		if err := passenger.Validate(); err != nil {
			return fmt.Errorf("passenger %d validation failed: %v", i, err)
		}
	}
	
	return nil
}

// GetSummary returns a summary of the order
func (o *Order) GetSummary() map[string]interface{} {
	summary := map[string]interface{}{
		"order_id":        o.OrderID,
		"order_reference": o.OrderReference,
		"customer_id":     o.CustomerID,
		"status":          o.Status,
		"total_amount":    o.TotalAmount.String(),
		"currency":        o.Currency,
		"passenger_count": len(o.Passengers),
		"item_count":      len(o.Items),
		"created_at":      o.CreatedAt,
		"channel":         o.Channel,
	}
	
	if o.Route != "" {
		summary["route"] = o.Route
	}
	
	if o.DepartureDate != nil {
		summary["departure_date"] = o.DepartureDate
	}
	
	return summary
}

// IsModifiable checks if the order can be modified
func (o *Order) IsModifiable() bool {
	return o.Status == OrderStatusCreated || o.Status == OrderStatusConfirmed
}

// IsCancellable checks if the order can be cancelled
func (o *Order) IsCancellable() bool {
	return o.Status != OrderStatusCancelled && 
		   o.Status != OrderStatusRefunded && 
		   o.Status != OrderStatusExpired
}

// IsRefundable checks if the order can be refunded
func (o *Order) IsRefundable() bool {
	return o.Status == OrderStatusCancelled || 
		   o.Status == OrderStatusTicketed || 
		   o.Status == OrderStatusFulfilled
}

// TableName returns the table name for GORM
func (Order) TableName() string {
	return "orders"
}

func (OrderItem) TableName() string {
	return "order_items"
}

func (PaymentMethod) TableName() string {
	return "payment_methods"
}

func (ContactInfo) TableName() string {
	return "contact_info"
}

func (PassengerInfo) TableName() string {
	return "passenger_info"
}

func (AuditEntry) TableName() string {
	return "audit_entries"
} 