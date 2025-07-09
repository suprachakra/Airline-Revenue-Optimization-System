package repository

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"iaros/order_service/src/database"
	"iaros/order_service/src/models"
)

// OrderRepository provides data access methods for orders
type OrderRepository struct {
	db *gorm.DB
}

// NewOrderRepository creates a new order repository
func NewOrderRepository() *OrderRepository {
	return &OrderRepository{
		db: database.GetDB(),
	}
}

// OrderQueryParams represents query parameters for order searches
type OrderQueryParams struct {
	CustomerID    *string
	Status        *models.OrderStatus
	Channel       *string
	Route         *string
	BookingClass  *string
	DateFrom      *time.Time
	DateTo        *time.Time
	PaymentStatus *models.PaymentStatus
	Page          int
	PageSize      int
	SortBy        string
	SortOrder     string
}

// OrderSearchResult represents paginated order search results
type OrderSearchResult struct {
	Orders     []models.Order `json:"orders"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// Create creates a new order
func (r *OrderRepository) Create(order *models.Order) error {
	// Start transaction
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create the order
	if err := tx.Create(order).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create order: %v", err)
	}

	// Create contact info
	order.ContactInfo.OrderID = order.OrderID
	if err := tx.Create(&order.ContactInfo).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create contact info: %v", err)
	}

	// Create passengers
	for i := range order.Passengers {
		order.Passengers[i].OrderID = order.OrderID
		if err := tx.Create(&order.Passengers[i]).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create passenger %d: %v", i, err)
		}
	}

	// Create order items
	for i := range order.Items {
		order.Items[i].OrderID = order.OrderID
		if err := tx.Create(&order.Items[i]).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create order item %d: %v", i, err)
		}
	}

	// Create payment methods
	for i := range order.PaymentMethods {
		order.PaymentMethods[i].OrderID = order.OrderID
		if err := tx.Create(&order.PaymentMethods[i]).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create payment method %d: %v", i, err)
		}
	}

	// Create audit trail entries
	for i := range order.AuditTrail {
		order.AuditTrail[i].OrderID = order.OrderID
		if err := tx.Create(&order.AuditTrail[i]).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create audit entry %d: %v", i, err)
		}
	}

	return tx.Commit().Error
}

// GetByID retrieves an order by ID
func (r *OrderRepository) GetByID(orderID string) (*models.Order, error) {
	var order models.Order
	
	err := r.db.Preload("ContactInfo").
		Preload("Passengers").
		Preload("Items").
		Preload("PaymentMethods").
		Preload("AuditTrail", func(db *gorm.DB) *gorm.DB {
			return db.Order("timestamp DESC")
		}).
		Where("order_id = ?", orderID).
		First(&order).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("order not found: %s", orderID)
		}
		return nil, fmt.Errorf("failed to get order: %v", err)
	}

	return &order, nil
}

// GetByReference retrieves an order by reference
func (r *OrderRepository) GetByReference(orderReference string) (*models.Order, error) {
	var order models.Order
	
	err := r.db.Preload("ContactInfo").
		Preload("Passengers").
		Preload("Items").
		Preload("PaymentMethods").
		Preload("AuditTrail", func(db *gorm.DB) *gorm.DB {
			return db.Order("timestamp DESC")
		}).
		Where("order_reference = ?", orderReference).
		First(&order).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("order not found: %s", orderReference)
		}
		return nil, fmt.Errorf("failed to get order: %v", err)
	}

	return &order, nil
}

// Update updates an existing order
func (r *OrderRepository) Update(order *models.Order) error {
	// Start transaction
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update the main order record
	if err := tx.Save(order).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update order: %v", err)
	}

	// Update contact info
	if err := tx.Save(&order.ContactInfo).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update contact info: %v", err)
	}

	return tx.Commit().Error
}

// Delete soft deletes an order (marks as cancelled)
func (r *OrderRepository) Delete(orderID string, reason string, userID *string) error {
	// Get the order first
	order, err := r.GetByID(orderID)
	if err != nil {
		return err
	}

	// Cancel the order
	if err := order.CancelOrder(reason, userID); err != nil {
		return err
	}

	// Update in database
	return r.Update(order)
}

// Search searches orders with filters and pagination
func (r *OrderRepository) Search(params OrderQueryParams) (*OrderSearchResult, error) {
	query := r.db.Model(&models.Order{})

	// Apply filters
	if params.CustomerID != nil {
		query = query.Where("customer_id = ?", *params.CustomerID)
	}

	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	if params.Channel != nil {
		query = query.Where("channel = ?", *params.Channel)
	}

	if params.Route != nil {
		query = query.Where("route = ?", *params.Route)
	}

	if params.BookingClass != nil {
		query = query.Where("booking_class = ?", *params.BookingClass)
	}

	if params.PaymentStatus != nil {
		query = query.Where("payment_status = ?", *params.PaymentStatus)
	}

	if params.DateFrom != nil {
		query = query.Where("created_at >= ?", *params.DateFrom)
	}

	if params.DateTo != nil {
		query = query.Where("created_at <= ?", *params.DateTo)
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count orders: %v", err)
	}

	// Apply sorting
	sortBy := "created_at"
	if params.SortBy != "" {
		sortBy = params.SortBy
	}

	sortOrder := "DESC"
	if params.SortOrder != "" {
		sortOrder = params.SortOrder
	}

	query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

	// Apply pagination
	pageSize := 10
	if params.PageSize > 0 {
		pageSize = params.PageSize
	}

	page := 1
	if params.Page > 0 {
		page = params.Page
	}

	offset := (page - 1) * pageSize
	query = query.Offset(offset).Limit(pageSize)

	// Execute query with preloads
	var orders []models.Order
	err := query.Preload("ContactInfo").
		Preload("Passengers").
		Preload("Items").
		Find(&orders).Error

	if err != nil {
		return nil, fmt.Errorf("failed to search orders: %v", err)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &OrderSearchResult{
		Orders:     orders,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetByCustomerID retrieves all orders for a customer
func (r *OrderRepository) GetByCustomerID(customerID string, limit int) ([]models.Order, error) {
	var orders []models.Order
	
	query := r.db.Preload("ContactInfo").
		Preload("Passengers").
		Preload("Items").
		Where("customer_id = ?", customerID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&orders).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get orders for customer %s: %v", customerID, err)
	}

	return orders, nil
}

// GetOrdersByStatus retrieves orders by status
func (r *OrderRepository) GetOrdersByStatus(status models.OrderStatus, limit int) ([]models.Order, error) {
	var orders []models.Order
	
	query := r.db.Preload("ContactInfo").
		Preload("Passengers").
		Preload("Items").
		Where("status = ?", status).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&orders).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get orders with status %s: %v", status, err)
	}

	return orders, nil
}

// GetRecentOrders retrieves recent orders
func (r *OrderRepository) GetRecentOrders(hours int, limit int) ([]models.Order, error) {
	var orders []models.Order
	
	cutoffTime := time.Now().Add(-time.Duration(hours) * time.Hour)
	
	query := r.db.Preload("ContactInfo").
		Preload("Passengers").
		Preload("Items").
		Where("created_at >= ?", cutoffTime).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&orders).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get recent orders: %v", err)
	}

	return orders, nil
}

// GetOrdersNeedingAction retrieves orders that need action (expiring, pending payment, etc.)
func (r *OrderRepository) GetOrdersNeedingAction() ([]models.Order, error) {
	var orders []models.Order
	
	// Orders that are created but not confirmed for more than 24 hours
	cutoffTime := time.Now().Add(-24 * time.Hour)
	
	err := r.db.Preload("ContactInfo").
		Preload("Passengers").
		Preload("Items").
		Where("status IN ? AND created_at < ?", 
			[]models.OrderStatus{models.OrderStatusCreated}, cutoffTime).
		Or("payment_status = ? AND created_at < ?", 
			models.PaymentStatusPending, cutoffTime).
		Order("created_at ASC").
		Find(&orders).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get orders needing action: %v", err)
	}

	return orders, nil
}

// AddOrderItem adds an item to an existing order
func (r *OrderRepository) AddOrderItem(orderID string, item *models.OrderItem) error {
	// Start transaction
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get the order
	var order models.Order
	if err := tx.Where("order_id = ?", orderID).First(&order).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("order not found: %s", orderID)
	}

	// Check if order is modifiable
	if !order.IsModifiable() {
		tx.Rollback()
		return fmt.Errorf("order cannot be modified in status: %s", order.Status)
	}

	// Add the item
	item.OrderID = orderID
	if err := tx.Create(item).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to add order item: %v", err)
	}

	// Recalculate totals
	var items []models.OrderItem
	if err := tx.Where("order_id = ?", orderID).Find(&items).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to get order items for recalculation: %v", err)
	}

	order.Items = items
	order.RecalculateTotals()
	order.Status = models.OrderStatusModified
	order.UpdatedAt = time.Now()

	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update order totals: %v", err)
	}

	return tx.Commit().Error
}

// RemoveOrderItem removes an item from an existing order
func (r *OrderRepository) RemoveOrderItem(orderID, itemID string) error {
	// Start transaction
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get the order
	var order models.Order
	if err := tx.Where("order_id = ?", orderID).First(&order).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("order not found: %s", orderID)
	}

	// Check if order is modifiable
	if !order.IsModifiable() {
		tx.Rollback()
		return fmt.Errorf("order cannot be modified in status: %s", order.Status)
	}

	// Remove the item
	if err := tx.Where("order_id = ? AND item_id = ?", orderID, itemID).Delete(&models.OrderItem{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to remove order item: %v", err)
	}

	// Recalculate totals
	var items []models.OrderItem
	if err := tx.Where("order_id = ?", orderID).Find(&items).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to get order items for recalculation: %v", err)
	}

	order.Items = items
	order.RecalculateTotals()
	order.Status = models.OrderStatusModified
	order.UpdatedAt = time.Now()

	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update order totals: %v", err)
	}

	return tx.Commit().Error
}

// AddAuditEntry adds an audit entry to an order
func (r *OrderRepository) AddAuditEntry(orderID string, entry *models.AuditEntry) error {
	entry.OrderID = orderID
	entry.Timestamp = time.Now()
	
	if err := r.db.Create(entry).Error; err != nil {
		return fmt.Errorf("failed to add audit entry: %v", err)
	}

	return nil
}

// GetOrderSummary returns order statistics
func (r *OrderRepository) GetOrderSummary(customerID *string, dateFrom, dateTo *time.Time) (map[string]interface{}, error) {
	query := r.db.Model(&models.Order{})

	if customerID != nil {
		query = query.Where("customer_id = ?", *customerID)
	}

	if dateFrom != nil {
		query = query.Where("created_at >= ?", *dateFrom)
	}

	if dateTo != nil {
		query = query.Where("created_at <= ?", *dateTo)
	}

	// Count by status
	var statusCounts []struct {
		Status models.OrderStatus `json:"status"`
		Count  int64              `json:"count"`
	}

	err := query.Select("status, count(*) as count").
		Group("status").
		Find(&statusCounts).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get status counts: %v", err)
	}

	// Total orders
	var totalOrders int64
	if err := query.Count(&totalOrders).Error; err != nil {
		return nil, fmt.Errorf("failed to get total orders: %v", err)
	}

	// Total revenue
	var totalRevenue struct {
		Total float64 `json:"total"`
	}

	err = query.Select("COALESCE(SUM(total_amount), 0) as total").
		Where("status NOT IN ?", []models.OrderStatus{models.OrderStatusCancelled, models.OrderStatusRefunded}).
		Scan(&totalRevenue).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get total revenue: %v", err)
	}

	return map[string]interface{}{
		"total_orders":   totalOrders,
		"total_revenue":  totalRevenue.Total,
		"status_counts":  statusCounts,
		"query_date_from": dateFrom,
		"query_date_to":   dateTo,
	}, nil
}

// GetOrderMetrics returns comprehensive order metrics
func (r *OrderRepository) GetOrderMetrics() (map[string]interface{}, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	thisWeek := today.AddDate(0, 0, -int(today.Weekday()))
	thisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	metrics := make(map[string]interface{})

	// Orders today
	var ordersToday int64
	r.db.Model(&models.Order{}).Where("created_at >= ?", today).Count(&ordersToday)
	metrics["orders_today"] = ordersToday

	// Orders this week
	var ordersThisWeek int64
	r.db.Model(&models.Order{}).Where("created_at >= ?", thisWeek).Count(&ordersThisWeek)
	metrics["orders_this_week"] = ordersThisWeek

	// Orders this month
	var ordersThisMonth int64
	r.db.Model(&models.Order{}).Where("created_at >= ?", thisMonth).Count(&ordersThisMonth)
	metrics["orders_this_month"] = ordersThisMonth

	// Revenue today
	var revenueToday float64
	r.db.Model(&models.Order{}).
		Select("COALESCE(SUM(total_amount), 0)").
		Where("created_at >= ? AND status NOT IN ?", today, 
			[]models.OrderStatus{models.OrderStatusCancelled, models.OrderStatusRefunded}).
		Scan(&revenueToday)
	metrics["revenue_today"] = revenueToday

	// Average order value
	var avgOrderValue float64
	r.db.Model(&models.Order{}).
		Select("COALESCE(AVG(total_amount), 0)").
		Where("status NOT IN ?", 
			[]models.OrderStatus{models.OrderStatusCancelled, models.OrderStatusRefunded}).
		Scan(&avgOrderValue)
	metrics["avg_order_value"] = avgOrderValue

	// Order status distribution
	var statusDist []struct {
		Status models.OrderStatus `json:"status"`
		Count  int64              `json:"count"`
	}
	r.db.Model(&models.Order{}).
		Select("status, count(*) as count").
		Group("status").
		Find(&statusDist)
	metrics["status_distribution"] = statusDist

	return metrics, nil
}

// ExpireOldOrders marks old pending orders as expired
func (r *OrderRepository) ExpireOldOrders(cutoffHours int) (int64, error) {
	cutoffTime := time.Now().Add(-time.Duration(cutoffHours) * time.Hour)
	
	result := r.db.Model(&models.Order{}).
		Where("status = ? AND created_at < ?", models.OrderStatusCreated, cutoffTime).
		Updates(map[string]interface{}{
			"status":     models.OrderStatusExpired,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to expire old orders: %v", result.Error)
	}

	return result.RowsAffected, nil
} 