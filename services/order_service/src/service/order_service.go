package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"iaros/order_service/src/models"
	"iaros/order_service/src/repository"
)

// OrderService provides business logic for order management
type OrderService struct {
	repo         *repository.OrderRepository
	redisClient  *redis.Client
	offerService OfferServiceClient
	paymentService PaymentServiceClient
	notificationService NotificationServiceClient
}

// ServiceClients interfaces for external services
type OfferServiceClient interface {
	ValidateOffer(ctx context.Context, offerID string) (*OfferValidationResult, error)
	GetOfferPricing(ctx context.Context, offerID string) (*OfferPricing, error)
}

type PaymentServiceClient interface {
	ProcessPayment(ctx context.Context, request *PaymentRequest) (*PaymentResponse, error)
	RefundPayment(ctx context.Context, request *RefundRequest) (*RefundResponse, error)
}

type NotificationServiceClient interface {
	SendOrderConfirmation(ctx context.Context, order *models.Order) error
	SendOrderCancellation(ctx context.Context, order *models.Order) error
}

// External service models
type OfferValidationResult struct {
	Valid       bool                   `json:"valid"`
	ExpiresAt   time.Time             `json:"expires_at"`
	Pricing     OfferPricing          `json:"pricing"`
	Availability map[string]interface{} `json:"availability"`
}

type OfferPricing struct {
	BasePrice    decimal.Decimal `json:"base_price"`
	Taxes        decimal.Decimal `json:"taxes"`
	Fees         decimal.Decimal `json:"fees"`
	TotalPrice   decimal.Decimal `json:"total_price"`
	Currency     string          `json:"currency"`
}

type PaymentRequest struct {
	OrderID       string                 `json:"order_id"`
	Amount        decimal.Decimal        `json:"amount"`
	Currency      string                 `json:"currency"`
	PaymentMethod map[string]interface{} `json:"payment_method"`
	CustomerID    string                 `json:"customer_id"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type PaymentResponse struct {
	TransactionID string            `json:"transaction_id"`
	Status        string            `json:"status"`
	AuthCode      string            `json:"auth_code"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type RefundRequest struct {
	OrderID       string          `json:"order_id"`
	TransactionID string          `json:"transaction_id"`
	Amount        decimal.Decimal `json:"amount"`
	Reason        string          `json:"reason"`
}

type RefundResponse struct {
	RefundID   string `json:"refund_id"`
	Status     string `json:"status"`
	ProcessedAt time.Time `json:"processed_at"`
}

// Order creation request
type CreateOrderRequest struct {
	CustomerID       string                   `json:"customer_id"`
	Channel          string                   `json:"channel"`
	ContactInfo      ContactInfoRequest       `json:"contact_info"`
	Passengers       []PassengerRequest       `json:"passengers"`
	Items            []OrderItemRequest       `json:"items"`
	PaymentMethod    *PaymentMethodRequest    `json:"payment_method,omitempty"`
	Route            string                   `json:"route"`
	BookingClass     string                   `json:"booking_class"`
	DepartureDate    *time.Time               `json:"departure_date,omitempty"`
	ReturnDate       *time.Time               `json:"return_date,omitempty"`
	Metadata         map[string]interface{}   `json:"metadata,omitempty"`
}

type ContactInfoRequest struct {
	Email   string                 `json:"email"`
	Phone   *string                `json:"phone,omitempty"`
	Address map[string]interface{} `json:"address,omitempty"`
}

type PassengerRequest struct {
	FirstName      string  `json:"first_name"`
	LastName       string  `json:"last_name"`
	DateOfBirth    string  `json:"date_of_birth"`
	Gender         string  `json:"gender"`
	Nationality    string  `json:"nationality"`
	PassportNumber *string `json:"passport_number,omitempty"`
	PassengerType  string  `json:"passenger_type"`
	LoyaltyNumber  *string `json:"loyalty_number,omitempty"`
}

type OrderItemRequest struct {
	ServiceType string                 `json:"service_type"`
	ServiceID   *string                `json:"service_id,omitempty"`
	Description string                 `json:"description"`
	Quantity    int                    `json:"quantity"`
	UnitPrice   decimal.Decimal        `json:"unit_price"`
	Currency    string                 `json:"currency"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type PaymentMethodRequest struct {
	PaymentType    string  `json:"payment_type"`
	LastFour       string  `json:"last_four"`
	ExpiryMonth    *int    `json:"expiry_month,omitempty"`
	ExpiryYear     *int    `json:"expiry_year,omitempty"`
	CardholderName *string `json:"cardholder_name,omitempty"`
}

// Order modification request
type ModifyOrderRequest struct {
	OrderID       string                   `json:"order_id"`
	AddItems      []OrderItemRequest       `json:"add_items,omitempty"`
	RemoveItems   []string                 `json:"remove_items,omitempty"`
	UpdateContact *ContactInfoRequest      `json:"update_contact,omitempty"`
	Metadata      map[string]interface{}   `json:"metadata,omitempty"`
	UserID        *string                  `json:"user_id,omitempty"`
}

// NewOrderService creates a new order service
func NewOrderService(redisClient *redis.Client, offerService OfferServiceClient, 
	paymentService PaymentServiceClient, notificationService NotificationServiceClient) *OrderService {
	return &OrderService{
		repo:                repository.NewOrderRepository(),
		redisClient:         redisClient,
		offerService:        offerService,
		paymentService:      paymentService,
		notificationService: notificationService,
	}
}

// CreateOrder creates a new order with business logic validation
func (s *OrderService) CreateOrder(ctx context.Context, request *CreateOrderRequest) (*models.Order, error) {
	// Validate request
	if err := s.validateCreateOrderRequest(request); err != nil {
		return nil, fmt.Errorf("validation failed: %v", err)
	}

	// Create order model
	order := &models.Order{
		CustomerID:    request.CustomerID,
		Channel:       request.Channel,
		Status:        models.OrderStatusCreated,
		PaymentStatus: models.PaymentStatusPending,
		Route:         request.Route,
		BookingClass:  request.BookingClass,
		DepartureDate: request.DepartureDate,
		ReturnDate:    request.ReturnDate,
		Currency:      models.CurrencyUSD, // Default currency
		Version:       1,
	}

	// Set metadata
	if request.Metadata != nil {
		if err := order.SetMetadata(request.Metadata); err != nil {
			return nil, fmt.Errorf("failed to set metadata: %v", err)
		}
	}

	// Create contact info
	contactInfo := models.ContactInfo{
		Email: request.ContactInfo.Email,
		Phone: request.ContactInfo.Phone,
	}

	if request.ContactInfo.Address != nil {
		if err := contactInfo.SetAddress(request.ContactInfo.Address); err != nil {
			return nil, fmt.Errorf("failed to set address: %v", err)
		}
	}

	order.ContactInfo = contactInfo

	// Create passengers
	for _, passengerReq := range request.Passengers {
		passenger := models.PassengerInfo{
			FirstName:      passengerReq.FirstName,
			LastName:       passengerReq.LastName,
			DateOfBirth:    passengerReq.DateOfBirth,
			Gender:         passengerReq.Gender,
			Nationality:    passengerReq.Nationality,
			PassportNumber: passengerReq.PassportNumber,
			PassengerType:  models.PassengerType(passengerReq.PassengerType),
			LoyaltyNumber:  passengerReq.LoyaltyNumber,
		}

		order.Passengers = append(order.Passengers, passenger)
	}

	// Create order items with validation
	for _, itemReq := range request.Items {
		// Validate service type
		if !s.isValidServiceType(itemReq.ServiceType) {
			return nil, fmt.Errorf("invalid service type: %s", itemReq.ServiceType)
		}

		item := models.OrderItem{
			ServiceType: models.ServiceType(itemReq.ServiceType),
			ServiceID:   itemReq.ServiceID,
			Description: itemReq.Description,
			Quantity:    itemReq.Quantity,
			UnitPrice:   itemReq.UnitPrice,
			Currency:    models.CurrencyCode(itemReq.Currency),
		}

		if itemReq.Metadata != nil {
			if err := item.SetMetadata(itemReq.Metadata); err != nil {
				return nil, fmt.Errorf("failed to set item metadata: %v", err)
			}
		}

		order.Items = append(order.Items, item)
	}

	// Add payment method if provided
	if request.PaymentMethod != nil {
		paymentMethod := models.PaymentMethod{
			PaymentType:    request.PaymentMethod.PaymentType,
			LastFour:       request.PaymentMethod.LastFour,
			ExpiryMonth:    request.PaymentMethod.ExpiryMonth,
			ExpiryYear:     request.PaymentMethod.ExpiryYear,
			CardholderName: request.PaymentMethod.CardholderName,
		}

		order.PaymentMethods = append(order.PaymentMethods, paymentMethod)
	}

	// Calculate totals
	order.RecalculateTotals()

	// Set expiration time (24 hours from creation)
	expiresAt := time.Now().Add(24 * time.Hour)
	order.ExpiresAt = &expiresAt

	// Add audit entry
	if err := order.AddAuditEntry("ORDER_CREATED", "Order created via API", nil, nil); err != nil {
		return nil, fmt.Errorf("failed to add audit entry: %v", err)
	}

	// Validate the complete order
	if err := order.Validate(); err != nil {
		return nil, fmt.Errorf("order validation failed: %v", err)
	}

	// Create in database
	if err := s.repo.Create(order); err != nil {
		return nil, fmt.Errorf("failed to create order: %v", err)
	}

	// Cache order for quick access
	s.cacheOrder(ctx, order)

	log.Printf("Order created successfully: %s", order.OrderID)

	return order, nil
}

// GetOrder retrieves an order by ID with caching
func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*models.Order, error) {
	// Try cache first
	if order, err := s.getCachedOrder(ctx, orderID); err == nil && order != nil {
		return order, nil
	}

	// Get from database
	order, err := s.repo.GetByID(orderID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cacheOrder(ctx, order)

	return order, nil
}

// GetOrderByReference retrieves an order by reference
func (s *OrderService) GetOrderByReference(ctx context.Context, orderReference string) (*models.Order, error) {
	return s.repo.GetByReference(orderReference)
}

// ModifyOrder modifies an existing order
func (s *OrderService) ModifyOrder(ctx context.Context, request *ModifyOrderRequest) (*models.Order, error) {
	// Get existing order
	order, err := s.GetOrder(ctx, request.OrderID)
	if err != nil {
		return nil, err
	}

	// Check if order can be modified
	if !order.IsModifiable() {
		return nil, fmt.Errorf("order cannot be modified in status: %s", order.Status)
	}

	// Start modification process
	modifications := make(map[string]interface{})

	// Add items
	for _, itemReq := range request.AddItems {
		if !s.isValidServiceType(itemReq.ServiceType) {
			return nil, fmt.Errorf("invalid service type: %s", itemReq.ServiceType)
		}

		item := &models.OrderItem{
			ServiceType: models.ServiceType(itemReq.ServiceType),
			ServiceID:   itemReq.ServiceID,
			Description: itemReq.Description,
			Quantity:    itemReq.Quantity,
			UnitPrice:   itemReq.UnitPrice,
			Currency:    models.CurrencyCode(itemReq.Currency),
		}

		if itemReq.Metadata != nil {
			if err := item.SetMetadata(itemReq.Metadata); err != nil {
				return nil, fmt.Errorf("failed to set item metadata: %v", err)
			}
		}

		if err := s.repo.AddOrderItem(order.OrderID, item); err != nil {
			return nil, fmt.Errorf("failed to add item: %v", err)
		}

		modifications["added_items"] = append(modifications["added_items"].([]string), item.ItemID)
	}

	// Remove items
	for _, itemID := range request.RemoveItems {
		if err := s.repo.RemoveOrderItem(order.OrderID, itemID); err != nil {
			return nil, fmt.Errorf("failed to remove item: %v", err)
		}

		modifications["removed_items"] = append(modifications["removed_items"].([]string), itemID)
	}

	// Update contact info
	if request.UpdateContact != nil {
		order.ContactInfo.Email = request.UpdateContact.Email
		order.ContactInfo.Phone = request.UpdateContact.Phone

		if request.UpdateContact.Address != nil {
			if err := order.ContactInfo.SetAddress(request.UpdateContact.Address); err != nil {
				return nil, fmt.Errorf("failed to set address: %v", err)
			}
		}

		modifications["updated_contact"] = true
	}

	// Add modification metadata
	if request.Metadata != nil {
		modifications["metadata"] = request.Metadata
	}

	// Apply modifications
	if err := order.ModifyOrder(modifications, request.UserID); err != nil {
		return nil, fmt.Errorf("failed to modify order: %v", err)
	}

	// Update in database
	if err := s.repo.Update(order); err != nil {
		return nil, fmt.Errorf("failed to update order: %v", err)
	}

	// Clear cache
	s.clearOrderCache(ctx, order.OrderID)

	log.Printf("Order modified successfully: %s", order.OrderID)

	return order, nil
}

// ConfirmOrder confirms an order and processes payment
func (s *OrderService) ConfirmOrder(ctx context.Context, orderID string) (*models.Order, error) {
	order, err := s.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order.Status != models.OrderStatusCreated {
		return nil, fmt.Errorf("order cannot be confirmed in status: %s", order.Status)
	}

	// Process payment if payment method exists
	if len(order.PaymentMethods) > 0 {
		paymentReq := &PaymentRequest{
			OrderID:    order.OrderID,
			Amount:     order.TotalAmount,
			Currency:   string(order.Currency),
			CustomerID: order.CustomerID,
			PaymentMethod: map[string]interface{}{
				"type":      order.PaymentMethods[0].PaymentType,
				"last_four": order.PaymentMethods[0].LastFour,
			},
		}

		paymentResp, err := s.paymentService.ProcessPayment(ctx, paymentReq)
		if err != nil {
			return nil, fmt.Errorf("payment processing failed: %v", err)
		}

		// Update payment reference
		order.PaymentReference = &paymentResp.TransactionID
		order.PaymentStatus = models.PaymentStatusAuthorized

		// Add audit entry
		metadata := map[string]interface{}{
			"transaction_id": paymentResp.TransactionID,
			"auth_code":      paymentResp.AuthCode,
		}
		order.AddAuditEntry("PAYMENT_PROCESSED", "Payment authorized", nil, metadata)
	}

	// Confirm payment
	if err := order.ConfirmPayment(); err != nil {
		return nil, fmt.Errorf("failed to confirm payment: %v", err)
	}

	// Update in database
	if err := s.repo.Update(order); err != nil {
		return nil, fmt.Errorf("failed to update order: %v", err)
	}

	// Send confirmation notification
	if s.notificationService != nil {
		if err := s.notificationService.SendOrderConfirmation(ctx, order); err != nil {
			log.Printf("Failed to send order confirmation: %v", err)
		}
	}

	// Clear cache
	s.clearOrderCache(ctx, order.OrderID)

	log.Printf("Order confirmed successfully: %s", order.OrderID)

	return order, nil
}

// CancelOrder cancels an order
func (s *OrderService) CancelOrder(ctx context.Context, orderID, reason string, userID *string) (*models.Order, error) {
	order, err := s.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if !order.IsCancellable() {
		return nil, fmt.Errorf("order cannot be cancelled in status: %s", order.Status)
	}

	// Cancel the order
	if err := order.CancelOrder(reason, userID); err != nil {
		return nil, fmt.Errorf("failed to cancel order: %v", err)
	}

	// Update in database
	if err := s.repo.Update(order); err != nil {
		return nil, fmt.Errorf("failed to update order: %v", err)
	}

	// Send cancellation notification
	if s.notificationService != nil {
		if err := s.notificationService.SendOrderCancellation(ctx, order); err != nil {
			log.Printf("Failed to send order cancellation: %v", err)
		}
	}

	// Clear cache
	s.clearOrderCache(ctx, order.OrderID)

	log.Printf("Order cancelled successfully: %s", order.OrderID)

	return order, nil
}

// RefundOrder processes a refund for an order
func (s *OrderService) RefundOrder(ctx context.Context, orderID string, refundAmount decimal.Decimal, reason string, userID *string) (*models.Order, error) {
	order, err := s.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if !order.IsRefundable() {
		return nil, fmt.Errorf("order cannot be refunded in status: %s", order.Status)
	}

	// Process refund through payment service
	if order.PaymentReference != nil && s.paymentService != nil {
		refundReq := &RefundRequest{
			OrderID:       order.OrderID,
			TransactionID: *order.PaymentReference,
			Amount:        refundAmount,
			Reason:        reason,
		}

		refundResp, err := s.paymentService.RefundPayment(ctx, refundReq)
		if err != nil {
			return nil, fmt.Errorf("refund processing failed: %v", err)
		}

		// Add audit entry for refund
		metadata := map[string]interface{}{
			"refund_id":     refundResp.RefundID,
			"processed_at":  refundResp.ProcessedAt,
		}
		order.AddAuditEntry("REFUND_PROCESSED", "Refund processed via payment service", userID, metadata)
	}

	// Update order status
	if err := order.RefundOrder(refundAmount, reason, userID); err != nil {
		return nil, fmt.Errorf("failed to refund order: %v", err)
	}

	// Update in database
	if err := s.repo.Update(order); err != nil {
		return nil, fmt.Errorf("failed to update order: %v", err)
	}

	// Clear cache
	s.clearOrderCache(ctx, order.OrderID)

	log.Printf("Order refunded successfully: %s", order.OrderID)

	return order, nil
}

// SearchOrders searches orders with filters
func (s *OrderService) SearchOrders(ctx context.Context, params repository.OrderQueryParams) (*repository.OrderSearchResult, error) {
	return s.repo.Search(params)
}

// GetOrdersByCustomer retrieves orders for a customer
func (s *OrderService) GetOrdersByCustomer(ctx context.Context, customerID string, limit int) ([]models.Order, error) {
	return s.repo.GetByCustomerID(customerID, limit)
}

// GetOrderMetrics returns order metrics
func (s *OrderService) GetOrderMetrics(ctx context.Context) (map[string]interface{}, error) {
	return s.repo.GetOrderMetrics()
}

// ExpireOldOrders expires old pending orders
func (s *OrderService) ExpireOldOrders(ctx context.Context, cutoffHours int) (int64, error) {
	return s.repo.ExpireOldOrders(cutoffHours)
}

// Validation methods
func (s *OrderService) validateCreateOrderRequest(request *CreateOrderRequest) error {
	if request.CustomerID == "" {
		return errors.New("customer ID is required")
	}

	if request.ContactInfo.Email == "" {
		return errors.New("email is required")
	}

	if len(request.Passengers) == 0 {
		return errors.New("at least one passenger is required")
	}

	if len(request.Items) == 0 {
		return errors.New("at least one order item is required")
	}

	// Validate passengers
	for i, passenger := range request.Passengers {
		if passenger.FirstName == "" || passenger.LastName == "" {
			return fmt.Errorf("passenger %d: first name and last name are required", i)
		}

		if passenger.PassengerType != "ADULT" && passenger.PassengerType != "CHILD" && passenger.PassengerType != "INFANT" {
			return fmt.Errorf("passenger %d: invalid passenger type: %s", i, passenger.PassengerType)
		}
	}

	// Validate items
	for i, item := range request.Items {
		if item.ServiceType == "" {
			return fmt.Errorf("item %d: service type is required", i)
		}

		if item.Quantity <= 0 {
			return fmt.Errorf("item %d: quantity must be positive", i)
		}

		if item.UnitPrice.LessThan(decimal.Zero) {
			return fmt.Errorf("item %d: unit price cannot be negative", i)
		}
	}

	return nil
}

func (s *OrderService) isValidServiceType(serviceType string) bool {
	validTypes := []string{
		"FLIGHT", "SEAT", "BAGGAGE", "MEAL", "LOUNGE", 
		"UPGRADE", "INSURANCE", "WIFI", "PRIORITY_BOARDING", "FAST_TRACK",
	}

	for _, validType := range validTypes {
		if serviceType == validType {
			return true
		}
	}

	return false
}

// Cache methods
func (s *OrderService) cacheOrder(ctx context.Context, order *models.Order) {
	if s.redisClient == nil {
		return
	}

	orderJSON, err := json.Marshal(order)
	if err != nil {
		log.Printf("Failed to marshal order for caching: %v", err)
		return
	}

	key := fmt.Sprintf("order:%s", order.OrderID)
	s.redisClient.Set(ctx, key, orderJSON, 30*time.Minute)
}

func (s *OrderService) getCachedOrder(ctx context.Context, orderID string) (*models.Order, error) {
	if s.redisClient == nil {
		return nil, errors.New("redis client not available")
	}

	key := fmt.Sprintf("order:%s", orderID)
	orderJSON, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var order models.Order
	if err := json.Unmarshal([]byte(orderJSON), &order); err != nil {
		return nil, err
	}

	return &order, nil
}

func (s *OrderService) clearOrderCache(ctx context.Context, orderID string) {
	if s.redisClient == nil {
		return
	}

	key := fmt.Sprintf("order:%s", orderID)
	s.redisClient.Del(ctx, key)
} 