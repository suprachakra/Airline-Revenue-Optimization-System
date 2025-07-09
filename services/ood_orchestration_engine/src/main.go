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

	"iaros/ood_orchestration_engine/src/config"
	"iaros/ood_orchestration_engine/src/handlers"
	"iaros/ood_orchestration_engine/src/services"
	"iaros/ood_orchestration_engine/src/storage"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// main initializes and starts the OOD Orchestration Engine service
// This service coordinates the end-to-end customer journey across Offer, Order, and Distribution services
// ensuring IATA ONE Order compliance and NDC Level 4 certification
func main() {
	// Load configuration from environment and config files
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connections and storage layer
	storage, err := storage.NewStorage(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer storage.Close()

	// Initialize core OOD orchestration services
	journeyManager := services.NewJourneyManager(storage, cfg)
	complianceEngine := services.NewComplianceEngine(cfg.Compliance)
	analyticsEngine := services.NewAnalyticsEngine(storage, cfg.Analytics)
	
	// Initialize OOD orchestration engine with all dependencies
	oodEngine := services.NewOODOrchestrationEngine(
		journeyManager,
		complianceEngine, 
		analyticsEngine,
		cfg,
	)

	// Setup HTTP router with middleware
	router := setupRouter(oodEngine, cfg)

	// Create HTTP server with production-ready configuration
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine for graceful shutdown
	go func() {
		log.Printf("OOD Orchestration Engine starting on port %s", cfg.Server.Port)
		log.Printf("Environment: %s", cfg.Server.Environment)
		log.Printf("Health endpoint: http://localhost:%s/health", cfg.Server.Port)
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down OOD Orchestration Engine...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("OOD Orchestration Engine stopped")
}

// setupRouter configures the HTTP router with all endpoints and middleware
// Implements unified OOD APIs, NDC orchestration, and journey analytics
func setupRouter(oodEngine *services.OODOrchestrationEngine, cfg *config.Config) *gin.Engine {
	// Set gin mode based on environment
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware for logging, recovery, CORS, and security
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(securityMiddleware())

	// Initialize handlers with OOD orchestration engine
	journeyHandler := handlers.NewJourneyHandler(oodEngine)
	ndcHandler := handlers.NewNDCHandler(oodEngine)
	analyticsHandler := handlers.NewAnalyticsHandler(oodEngine)
	healthHandler := handlers.NewHealthHandler(oodEngine)

	// Health and metrics endpoints
	router.GET("/health", healthHandler.HealthCheck)
	router.GET("/health/ready", healthHandler.ReadinessCheck)
	router.GET("/health/live", healthHandler.LivenessCheck)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API version 1 routes
	v1 := router.Group("/v1")
	{
		// Unified Journey Orchestration APIs
		// Coordinates complete customer journey from shopping to fulfillment
		journey := v1.Group("/journey")
		{
			// Shopping journey endpoints
			shopping := journey.Group("/shopping")
			{
				shopping.POST("/search", journeyHandler.InitiateShoppingJourney)
				shopping.POST("/filter", journeyHandler.ApplyShoppingFilters)
				shopping.GET("/:journey_id/offers", journeyHandler.GetShoppingOffers)
				shopping.POST("/:journey_id/select", journeyHandler.SelectOffer)
			}

			// Booking journey endpoints  
			booking := journey.Group("/booking")
			{
				booking.POST("/create", journeyHandler.CreateBookingJourney)
				booking.PUT("/:journey_id/modify", journeyHandler.ModifyBookingJourney)
				booking.POST("/:journey_id/payment", journeyHandler.ProcessPaymentJourney)
				booking.GET("/:journey_id/status", journeyHandler.GetJourneyStatus)
				booking.DELETE("/:journey_id/cancel", journeyHandler.CancelBookingJourney)
			}

			// Journey management endpoints
			journey.GET("/:journey_id", journeyHandler.GetJourneyDetails)
			journey.POST("/:journey_id/events", journeyHandler.TrackJourneyEvent)
			journey.GET("/:journey_id/audit", journeyHandler.GetJourneyAuditTrail)
		}

		// NDC Orchestration APIs
		// Provides NDC Level 4 compliant message orchestration
		ndc := v1.Group("/ndc")
		{
			ndc.POST("/AirShopping", ndcHandler.OrchestateAirShopping)
			ndc.POST("/OfferPrice", ndcHandler.OrchestrateOfferPrice)
			ndc.POST("/OrderCreate", ndcHandler.OrchestrateOrderCreate)
			ndc.GET("/OrderRetrieve", ndcHandler.OrchestrateOrderRetrieve)
			ndc.POST("/OrderCancel", ndcHandler.OrchestrateOrderCancel)
			ndc.POST("/OrderChange", ndcHandler.OrchestrateOrderChange)
			ndc.POST("/SeatAvailability", ndcHandler.OrchestrateSeatAvailability)
		}

		// Journey Analytics APIs
		// Provides insights and optimization for customer journeys
		analytics := v1.Group("/analytics")
		{
			// Journey performance analytics
			journey := analytics.Group("/journey")
			{
				journey.GET("/funnel", analyticsHandler.GetJourneyFunnelAnalytics)
				journey.GET("/performance", analyticsHandler.GetJourneyPerformanceMetrics)
				journey.GET("/abandonment", analyticsHandler.GetAbandonmentAnalysis)
				journey.GET("/conversion", analyticsHandler.GetConversionAnalytics)
			}

			// Customer behavior analytics
			customer := analytics.Group("/customer")
			{
				customer.GET("/behavior", analyticsHandler.GetCustomerBehaviorInsights)
				customer.GET("/segments", analyticsHandler.GetCustomerSegmentAnalytics)
				customer.GET("/lifetime-value", analyticsHandler.GetCustomerLifetimeValue)
			}

			// A/B testing and experimentation
			experiments := analytics.Group("/experiments")
			{
				experiments.POST("", analyticsHandler.CreateExperiment)
				experiments.GET("", analyticsHandler.ListExperiments)
				experiments.GET("/:experiment_id", analyticsHandler.GetExperimentResults)
				experiments.PUT("/:experiment_id/status", analyticsHandler.UpdateExperimentStatus)
			}
		}

		// Compliance and audit endpoints
		compliance := v1.Group("/compliance")
		{
			compliance.GET("/validation/:journey_id", journeyHandler.ValidateJourneyCompliance)
			compliance.GET("/audit/report", analyticsHandler.GetComplianceReport)
			compliance.GET("/standards/iata", healthHandler.GetIATAComplianceStatus)
			compliance.GET("/standards/ndc", healthHandler.GetNDCComplianceStatus)
		}
	}

	return router
}

// corsMiddleware configures Cross-Origin Resource Sharing for multi-channel access
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// securityMiddleware adds security headers for enterprise-grade protection
func securityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers for enterprise compliance
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Request ID for distributed tracing
		c.Header("X-Request-ID", fmt.Sprintf("ood-%d", time.Now().UnixNano()))

		c.Next()
	}
} 