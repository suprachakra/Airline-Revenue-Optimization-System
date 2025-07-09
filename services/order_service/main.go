package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	
	"iaros/order_service/src/controllers"
	"iaros/order_service/src/database"
	"iaros/order_service/src/models"
	"iaros/order_service/src/service"
)

// Config holds application configuration
type Config struct {
	ServerPort    string
	Environment   string
	DatabaseURL   string
	RedisURL      string
	LogLevel      string
	EnableSwagger bool
}

// MockServiceClients implements service interfaces for development
type MockOfferService struct{}
type MockPaymentService struct{}
type MockNotificationService struct{}

func (m *MockOfferService) ValidateOffer(ctx context.Context, offerID string) (*service.OfferValidationResult, error) {
	return &service.OfferValidationResult{
		Valid:     true,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Pricing: service.OfferPricing{
			BasePrice:  decimal.NewFromFloat(500.00),
			Taxes:      decimal.NewFromFloat(75.00),
			Fees:       decimal.NewFromFloat(25.00),
			TotalPrice: decimal.NewFromFloat(600.00),
			Currency:   "USD",
		},
		Availability: map[string]interface{}{
			"seats_available": 10,
			"class":          "ECONOMY",
		},
	}, nil
}

func (m *MockOfferService) GetOfferPricing(ctx context.Context, offerID string) (*service.OfferPricing, error) {
	return &service.OfferPricing{
		BasePrice:  decimal.NewFromFloat(500.00),
		Taxes:      decimal.NewFromFloat(75.00),
		Fees:       decimal.NewFromFloat(25.00),
		TotalPrice: decimal.NewFromFloat(600.00),
		Currency:   "USD",
	}, nil
}

func (m *MockPaymentService) ProcessPayment(ctx context.Context, request *service.PaymentRequest) (*service.PaymentResponse, error) {
	return &service.PaymentResponse{
		TransactionID: fmt.Sprintf("txn_%d", time.Now().Unix()),
		Status:        "SUCCESS",
		AuthCode:      "AUTH123456",
		Metadata: map[string]interface{}{
			"processor": "mock",
			"timestamp": time.Now().UTC(),
		},
	}, nil
}

func (m *MockPaymentService) RefundPayment(ctx context.Context, request *service.RefundRequest) (*service.RefundResponse, error) {
	return &service.RefundResponse{
		RefundID:    fmt.Sprintf("refund_%d", time.Now().Unix()),
		Status:      "SUCCESS",
		ProcessedAt: time.Now().UTC(),
	}, nil
}

func (m *MockNotificationService) SendOrderConfirmation(ctx context.Context, order *models.Order) error {
	log.Printf("Mock: Sending order confirmation for order %s", order.OrderID)
	return nil
}

func (m *MockNotificationService) SendOrderCancellation(ctx context.Context, order *models.Order) error {
	log.Printf("Mock: Sending order cancellation for order %s", order.OrderID)
	return nil
}

var logger *zap.Logger

func main() {
	// Initialize logger
	if err := initLogger(); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	// Load configuration
	config := loadConfig()
	
	// Initialize database
	if err := initDatabase(); err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer database.Close()

	// Initialize Redis
	redisClient := initRedis(config)
	defer redisClient.Close()

	// Initialize services
	orderService := initOrderService(redisClient)

	// Initialize HTTP server
	server := initHTTPServer(config, orderService)

	// Start server
	startServer(server, config)
}

func loadConfig() *Config {
	return &Config{
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		Environment:   getEnv("ENVIRONMENT", "development"),
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		EnableSwagger: getEnv("ENABLE_SWAGGER", "true") == "true",
	}
}

func initLogger() error {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	
	var err error
	logger, err = config.Build()
	if err != nil {
		return err
	}
	
	// Set global logger
	zap.ReplaceGlobals(logger)
	
	return nil
}

func initDatabase() error {
	// Connect to database
	_, err := database.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	// Run migrations
	if err := database.AutoMigrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %v", err)
	}

	logger.Info("Database initialized successfully")
	return nil
}

func initRedis(config *Config) *redis.Client {
	opt, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		logger.Error("Failed to parse Redis URL, using default", zap.Error(err))
		opt = &redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		}
	}

	client := redis.NewClient(opt)
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Error("Failed to connect to Redis", zap.Error(err))
		return nil
	}

	logger.Info("Redis initialized successfully")
	return client
}

func initOrderService(redisClient *redis.Client) *service.OrderService {
	// Initialize mock service clients
	offerService := &MockOfferService{}
	paymentService := &MockPaymentService{}
	notificationService := &MockNotificationService{}

	// Create order service
	orderService := service.NewOrderService(redisClient, offerService, paymentService, notificationService)

	logger.Info("Order service initialized successfully")
	return orderService
}

func initHTTPServer(config *Config, orderService *service.OrderService) *http.Server {
	// Set Gin mode
	if config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin router
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(loggingMiddleware())

	// Initialize controller
	orderController := controllers.NewOrderController(orderService)

	// Setup routes
	setupRoutes(router, orderController)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + config.ServerPort,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("HTTP server initialized", zap.String("port", config.ServerPort))
	return server
}

func setupRoutes(router *gin.Engine, orderController *controllers.OrderController) {
	// Health check
	router.GET("/health", orderController.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Order routes
		orders := v1.Group("/orders")
		{
			orders.POST("", orderController.CreateOrder)
			orders.GET("/:order_id", orderController.GetOrder)
			orders.GET("/reference/:order_reference", orderController.GetOrderByReference)
			orders.GET("/search", orderController.SearchOrders)
			orders.GET("/customer/:customer_id", orderController.GetOrdersByCustomer)
			orders.GET("/metrics", orderController.GetOrderMetrics)
			orders.GET("/:order_id/summary", orderController.GetOrderSummary)
			orders.GET("/:order_id/audit", orderController.GetOrderAuditTrail)
			
			// Order operations
			orders.PUT("/:order_id/modify", orderController.ModifyOrder)
			orders.POST("/:order_id/confirm", orderController.ConfirmOrder)
			orders.POST("/:order_id/cancel", orderController.CancelOrder)
			orders.POST("/:order_id/refund", orderController.RefundOrder)
			
			// Batch operations
			orders.POST("/expire", orderController.ExpireOldOrders)
		}
	}

	// Admin routes (if needed)
	admin := router.Group("/admin")
	{
		admin.GET("/metrics", orderController.GetOrderMetrics)
		admin.GET("/health/detailed", func(c *gin.Context) {
			// Detailed health check
			dbHealth := "healthy"
			if err := database.HealthCheck(); err != nil {
				dbHealth = "unhealthy: " + err.Error()
			}

			dbStats := database.GetStats()

			c.JSON(200, gin.H{
				"service":        "order-service",
				"version":        "1.0.0",
				"timestamp":      time.Now().UTC(),
				"database":       dbHealth,
				"database_stats": dbStats,
				"uptime":         time.Since(startTime).String(),
			})
		})
	}

	logger.Info("Routes configured successfully")
}

var startTime = time.Now()

func startServer(server *http.Server, config *Config) {
	// Start server in a goroutine
	go func() {
		logger.Info("Starting HTTP server", 
			zap.String("port", config.ServerPort),
			zap.String("environment", config.Environment))
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Give outstanding requests a deadline for completion
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server shutdown complete")
}

// Middleware functions
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Process request
		c.Next()

		// Log request details
		duration := time.Since(startTime)
		statusCode := c.Writer.Status()

		logger.Info("HTTP Request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", statusCode),
			zap.Duration("duration", duration),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)

		// Add response headers
		c.Header("X-Response-Time", duration.String())
		c.Header("X-Service", "order-service")
	}
}

// Utility functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Background tasks (can be moved to a separate service)
func startBackgroundTasks(orderService *service.OrderService) {
	// Start order expiration task
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				expiredCount, err := orderService.ExpireOldOrders(ctx, 24) // Expire orders older than 24 hours
				if err != nil {
					logger.Error("Failed to expire old orders", zap.Error(err))
				} else {
					logger.Info("Expired old orders", zap.Int64("count", expiredCount))
				}
				cancel()
			}
		}
	}()
}

func init() {
	// Initialize any global settings here
	startTime = time.Now()
} 