package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"iaros/order_service/src/models"
	"iaros/order_service/src/repository"
	"iaros/order_service/src/service"
)

// OrderController handles HTTP requests for order management
type OrderController struct {
	orderService *service.OrderService
}

// NewOrderController creates a new order controller
func NewOrderController(orderService *service.OrderService) *OrderController {
	return &OrderController{
		orderService: orderService,
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// CreateOrder creates a new order
// @Summary Create a new order
// @Description Create a new order with passengers, items, and payment information
// @Tags Orders
// @Accept json
// @Produce json
// @Param order body service.CreateOrderRequest true "Order creation request"
// @Success 201 {object} models.Order
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /orders [post]
func (c *OrderController) CreateOrder(ctx *gin.Context) {
	var request service.CreateOrderRequest
	
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	order, err := c.orderService.CreateOrder(ctx.Request.Context(), &request)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "validation failed") {
			statusCode = http.StatusBadRequest
		}
		
		ctx.JSON(statusCode, ErrorResponse{
			Error:   "Failed to create order",
			Details: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, order)
}

// GetOrder retrieves an order by ID
// @Summary Get an order by ID
// @Description Retrieve a specific order by its unique identifier
// @Tags Orders
// @Produce json
// @Param order_id path string true "Order ID"
// @Success 200 {object} models.Order
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /orders/{order_id} [get]
func (c *OrderController) GetOrder(ctx *gin.Context) {
	orderID := ctx.Param("order_id")
	if orderID == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Order ID is required",
		})
		return
	}

	order, err := c.orderService.GetOrder(ctx.Request.Context(), orderID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}
		
		ctx.JSON(statusCode, ErrorResponse{
			Error:   "Failed to get order",
			Details: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, order)
}

// GetOrderByReference retrieves an order by reference
// @Summary Get an order by reference
// @Description Retrieve a specific order by its reference number
// @Tags Orders
// @Produce json
// @Param order_reference path string true "Order Reference"
// @Success 200 {object} models.Order
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /orders/reference/{order_reference} [get]
func (c *OrderController) GetOrderByReference(ctx *gin.Context) {
	orderReference := ctx.Param("order_reference")
	if orderReference == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Order reference is required",
		})
		return
	}

	order, err := c.orderService.GetOrderByReference(ctx.Request.Context(), orderReference)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}
		
		ctx.JSON(statusCode, ErrorResponse{
			Error:   "Failed to get order",
			Details: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, order)
}

// SearchOrders searches orders with filters
// @Summary Search orders
// @Description Search orders with various filters and pagination
// @Tags Orders
// @Produce json
// @Param customer_id query string false "Customer ID"
// @Param status query string false "Order Status"
// @Param channel query string false "Channel"
// @Param route query string false "Route"
// @Param booking_class query string false "Booking Class"
// @Param payment_status query string false "Payment Status"
// @Param date_from query string false "Date From (YYYY-MM-DD)"
// @Param date_to query string false "Date To (YYYY-MM-DD)"
// @Param page query int false "Page Number" default(1)
// @Param page_size query int false "Page Size" default(10)
// @Param sort_by query string false "Sort By" default("created_at")
// @Param sort_order query string false "Sort Order" default("DESC")
// @Success 200 {object} repository.OrderSearchResult
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /orders/search [get]
func (c *OrderController) SearchOrders(ctx *gin.Context) {
	params := repository.OrderQueryParams{}

	// Parse query parameters
	if customerID := ctx.Query("customer_id"); customerID != "" {
		params.CustomerID = &customerID
	}

	if status := ctx.Query("status"); status != "" {
		orderStatus := models.OrderStatus(status)
		params.Status = &orderStatus
	}

	if channel := ctx.Query("channel"); channel != "" {
		params.Channel = &channel
	}

	if route := ctx.Query("route"); route != "" {
		params.Route = &route
	}

	if bookingClass := ctx.Query("booking_class"); bookingClass != "" {
		params.BookingClass = &bookingClass
	}

	if paymentStatus := ctx.Query("payment_status"); paymentStatus != "" {
		pStatus := models.PaymentStatus(paymentStatus)
		params.PaymentStatus = &pStatus
	}

	if dateFrom := ctx.Query("date_from"); dateFrom != "" {
		if date, err := time.Parse("2006-01-02", dateFrom); err == nil {
			params.DateFrom = &date
		}
	}

	if dateTo := ctx.Query("date_to"); dateTo != "" {
		if date, err := time.Parse("2006-01-02", dateTo); err == nil {
			params.DateTo = &date
		}
	}

	if page := ctx.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			params.Page = p
		}
	}

	if pageSize := ctx.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil && ps > 0 {
			params.PageSize = ps
		}
	}

	params.SortBy = ctx.DefaultQuery("sort_by", "created_at")
	params.SortOrder = ctx.DefaultQuery("sort_order", "DESC")

	result, err := c.orderService.SearchOrders(ctx.Request.Context(), params)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to search orders",
			Details: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, result)
}

// GetOrdersByCustomer retrieves orders for a specific customer
// @Summary Get orders by customer
// @Description Retrieve all orders for a specific customer
// @Tags Orders
// @Produce json
// @Param customer_id path string true "Customer ID"
// @Param limit query int false "Limit" default(10)
// @Success 200 {array} models.Order
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /orders/customer/{customer_id} [get]
func (c *OrderController) GetOrdersByCustomer(ctx *gin.Context) {
	customerID := ctx.Param("customer_id")
	if customerID == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Customer ID is required",
		})
		return
	}

	limit := 10
	if l := ctx.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	orders, err := c.orderService.GetOrdersByCustomer(ctx.Request.Context(), customerID, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get customer orders",
			Details: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, orders)
}

// ModifyOrder modifies an existing order
// @Summary Modify an order
// @Description Modify an existing order by adding/removing items or updating information
// @Tags Orders
// @Accept json
// @Produce json
// @Param order_id path string true "Order ID"
// @Param modifications body service.ModifyOrderRequest true "Order modification request"
// @Success 200 {object} models.Order
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /orders/{order_id}/modify [put]
func (c *OrderController) ModifyOrder(ctx *gin.Context) {
	orderID := ctx.Param("order_id")
	if orderID == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Order ID is required",
		})
		return
	}

	var request service.ModifyOrderRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	request.OrderID = orderID

	order, err := c.orderService.ModifyOrder(ctx.Request.Context(), &request)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "cannot be modified") {
			statusCode = http.StatusBadRequest
		}
		
		ctx.JSON(statusCode, ErrorResponse{
			Error:   "Failed to modify order",
			Details: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, order)
}

// ConfirmOrder confirms an order and processes payment
// @Summary Confirm an order
// @Description Confirm an order and process payment
// @Tags Orders
// @Produce json
// @Param order_id path string true "Order ID"
// @Success 200 {object} models.Order
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /orders/{order_id}/confirm [post]
func (c *OrderController) ConfirmOrder(ctx *gin.Context) {
	orderID := ctx.Param("order_id")
	if orderID == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Order ID is required",
		})
		return
	}

	order, err := c.orderService.ConfirmOrder(ctx.Request.Context(), orderID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "cannot be confirmed") {
			statusCode = http.StatusBadRequest
		}
		
		ctx.JSON(statusCode, ErrorResponse{
			Error:   "Failed to confirm order",
			Details: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, order)
}

// CancelOrder cancels an order
// @Summary Cancel an order
// @Description Cancel an existing order
// @Tags Orders
// @Accept json
// @Produce json
// @Param order_id path string true "Order ID"
// @Param cancellation body CancelOrderRequest true "Cancellation request"
// @Success 200 {object} models.Order
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /orders/{order_id}/cancel [post]
func (c *OrderController) CancelOrder(ctx *gin.Context) {
	orderID := ctx.Param("order_id")
	if orderID == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Order ID is required",
		})
		return
	}

	var request struct {
		Reason string  `json:"reason" binding:"required"`
		UserID *string `json:"user_id,omitempty"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	order, err := c.orderService.CancelOrder(ctx.Request.Context(), orderID, request.Reason, request.UserID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "cannot be cancelled") {
			statusCode = http.StatusBadRequest
		}
		
		ctx.JSON(statusCode, ErrorResponse{
			Error:   "Failed to cancel order",
			Details: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, order)
}

// RefundOrder processes a refund for an order
// @Summary Refund an order
// @Description Process a refund for an existing order
// @Tags Orders
// @Accept json
// @Produce json
// @Param order_id path string true "Order ID"
// @Param refund body RefundOrderRequest true "Refund request"
// @Success 200 {object} models.Order
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /orders/{order_id}/refund [post]
func (c *OrderController) RefundOrder(ctx *gin.Context) {
	orderID := ctx.Param("order_id")
	if orderID == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Order ID is required",
		})
		return
	}

	var request struct {
		Amount decimal.Decimal `json:"amount" binding:"required"`
		Reason string          `json:"reason" binding:"required"`
		UserID *string         `json:"user_id,omitempty"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	order, err := c.orderService.RefundOrder(ctx.Request.Context(), orderID, request.Amount, request.Reason, request.UserID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "cannot be refunded") {
			statusCode = http.StatusBadRequest
		}
		
		ctx.JSON(statusCode, ErrorResponse{
			Error:   "Failed to refund order",
			Details: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, order)
}

// GetOrderMetrics returns order metrics and statistics
// @Summary Get order metrics
// @Description Retrieve comprehensive order metrics and statistics
// @Tags Orders
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} ErrorResponse
// @Router /orders/metrics [get]
func (c *OrderController) GetOrderMetrics(ctx *gin.Context) {
	metrics, err := c.orderService.GetOrderMetrics(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get order metrics",
			Details: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, metrics)
}

// GetOrderSummary returns a summary of an order
// @Summary Get order summary
// @Description Retrieve a summary view of an order
// @Tags Orders
// @Produce json
// @Param order_id path string true "Order ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /orders/{order_id}/summary [get]
func (c *OrderController) GetOrderSummary(ctx *gin.Context) {
	orderID := ctx.Param("order_id")
	if orderID == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Order ID is required",
		})
		return
	}

	order, err := c.orderService.GetOrder(ctx.Request.Context(), orderID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}
		
		ctx.JSON(statusCode, ErrorResponse{
			Error:   "Failed to get order",
			Details: err.Error(),
		})
		return
	}

	summary := order.GetSummary()
	ctx.JSON(http.StatusOK, summary)
}

// ExpireOldOrders expires old pending orders
// @Summary Expire old orders
// @Description Mark old pending orders as expired
// @Tags Orders
// @Accept json
// @Produce json
// @Param request body ExpireOrdersRequest true "Expire request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /orders/expire [post]
func (c *OrderController) ExpireOldOrders(ctx *gin.Context) {
	var request struct {
		CutoffHours int `json:"cutoff_hours" binding:"required,min=1"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	expiredCount, err := c.orderService.ExpireOldOrders(ctx.Request.Context(), request.CutoffHours)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to expire orders",
			Details: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, SuccessResponse{
		Message: "Orders expired successfully",
		Data: map[string]interface{}{
			"expired_count": expiredCount,
			"cutoff_hours":  request.CutoffHours,
		},
	})
}

// HealthCheck performs health check
// @Summary Health check
// @Description Perform health check for the order service
// @Tags System
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /health [get]
func (c *OrderController) HealthCheck(ctx *gin.Context) {
	// Perform basic health checks
	// You can add more comprehensive checks here
	ctx.JSON(http.StatusOK, SuccessResponse{
		Message: "Order service is healthy",
		Data: map[string]interface{}{
			"timestamp": time.Now().UTC(),
			"service":   "order-service",
			"version":   "1.0.0",
		},
	})
}

// GetOrderAuditTrail retrieves the audit trail for an order
// @Summary Get order audit trail
// @Description Retrieve the complete audit trail for an order
// @Tags Orders
// @Produce json
// @Param order_id path string true "Order ID"
// @Success 200 {array} models.AuditEntry
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /orders/{order_id}/audit [get]
func (c *OrderController) GetOrderAuditTrail(ctx *gin.Context) {
	orderID := ctx.Param("order_id")
	if orderID == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Order ID is required",
		})
		return
	}

	order, err := c.orderService.GetOrder(ctx.Request.Context(), orderID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}
		
		ctx.JSON(statusCode, ErrorResponse{
			Error:   "Failed to get order",
			Details: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, order.AuditTrail)
}

// Middleware for request logging
func (c *OrderController) LoggingMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startTime := time.Now()
		
		// Process request
		ctx.Next()
		
		// Log request details
		duration := time.Since(startTime)
		statusCode := ctx.Writer.Status()
		
		// You can integrate with your logging system here
		// log.Printf("Request: %s %s - Status: %d - Duration: %v", 
		//     ctx.Request.Method, ctx.Request.URL.Path, statusCode, duration)
		
		// Add response headers
		ctx.Header("X-Response-Time", duration.String())
		ctx.Header("X-Service", "order-service")
	}
}

// CORS middleware
func (c *OrderController) CORSMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Header("Access-Control-Allow-Origin", "*")
		ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		ctx.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		
		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(204)
			return
		}
		
		ctx.Next()
	}
}

// Request validation middleware
func (c *OrderController) ValidationMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Add common validation logic here
		ctx.Next()
	}
} 