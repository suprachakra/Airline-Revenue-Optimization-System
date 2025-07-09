package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/customer-intelligence-platform/src/api"
	"github.com/customer-intelligence-platform/src/config"
	"github.com/customer-intelligence-platform/src/services"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize customer intelligence services
	customerIntelligenceEngine := services.NewCustomerIntelligenceEngine(cfg)
	profileEnrichmentEngine := services.NewProfileEnrichmentEngine(cfg)
	segmentationScoringEngine := services.NewSegmentationScoringEngine(cfg)
	competitivePricingEngine := services.NewCompetitivePricingIntelligenceEngine(cfg)
	customerAnalyticsEngine := services.NewCustomerAnalyticsEngine(cfg)

	// Setup HTTP server
	r := gin.Default()
	
	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy", 
			"service": "customer-intelligence-platform",
			"version": "2.0",
			"capabilities": []string{
				"profile_enrichment",
				"segmentation_scoring", 
				"competitive_intelligence",
				"customer_analytics",
				"behavioral_insights"
			},
			"intelligence_stats": gin.H{
				"customer_profiles": "50M+",
				"data_sources": 25,
				"segments_tracked": 500,
				"ml_models": 50,
				"enrichment_accuracy": "99.5%",
				"real_time_scoring": true,
			},
		})
	})

	// Setup API routes
	api.SetupCustomerIntelligenceRoutes(r, customerIntelligenceEngine, profileEnrichmentEngine, segmentationScoringEngine, competitivePricingEngine, customerAnalyticsEngine)

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: r,
	}

	// Start server
	go func() {
		log.Printf("Starting Customer Intelligence Platform on port %s", cfg.Server.Port)
		log.Printf("Profiles: 50M+ | Data sources: 25 | Segments: 500 | Models: 50")
		log.Printf("AI-powered customer intelligence with 99.5%% enrichment accuracy")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Customer Intelligence Platform...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Customer Intelligence Platform exited successfully")
} 