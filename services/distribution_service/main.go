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

	"iaros/distribution_service/src/controllers"
	"iaros/distribution_service/src/database"
	"iaros/distribution_service/src/services"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gopkg.in/yaml.v3"
)

// Config represents application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Services ServicesConfig `yaml:"services"`
	Logging  LoggingConfig  `yaml:"logging"`
}

type ServerConfig struct {
	Port         int    `yaml:"port"`
	Host         string `yaml:"host"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
	IdleTimeout  int    `yaml:"idle_timeout"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
	TimeZone string `yaml:"timezone"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type ServicesConfig struct {
	OrderServiceURL string `yaml:"order_service_url"`
	OfferServiceURL string `yaml:"offer_service_url"`
	NDCConfig       NDCConfig `yaml:"ndc"`
	GDSConfig       GDSConfig `yaml:"gds"`
}

type NDCConfig struct {
	Version         string `yaml:"version"`
	DefaultAirline  string `yaml:"default_airline"`
	SessionTimeout  int    `yaml:"session_timeout"`
	MaxConcurrency  int    `yaml:"max_concurrency"`
}

type GDSConfig struct {
	Amadeus    GDSProviderConfig `yaml:"amadeus"`
	Sabre      GDSProviderConfig `yaml:"sabre"`
	Travelport GDSProviderConfig `yaml:"travelport"`
}

type GDSProviderConfig struct {
	Enabled     bool   `yaml:"enabled"`
	BaseURL     string `yaml:"base_url"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	PseudoCity  string `yaml:"pseudo_city"`
	OfficeID    string `yaml:"office_id"`
	Timeout     int    `yaml:"timeout"`
	RetryCount  int    `yaml:"retry_count"`
}

type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	OutputPath string `yaml:"output_path"`
}

func main() {
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	dbConfig := database.DatabaseConfig{
		Host:     config.Database.Host,
		Port:     config.Database.Port,
		User:     config.Database.User,
		Password: config.Database.Password,
		DBName:   config.Database.DBName,
		SSLMode:  config.Database.SSLMode,
		TimeZone: config.Database.TimeZone,
	}

	if err := database.ConnectDatabase(dbConfig); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run database migrations
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	// Create database indexes
	if err := database.CreateIndexes(); err != nil {
		log.Printf("Warning: Failed to create database indexes: %v", err)
	}

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port),
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
		redisClient = nil
	}

	// Initialize services
	db := database.GetDB()
	sessionManager := services.NewSessionManager(db, redisClient)
	
	ndcService := services.NewNDCService(
		db,
		sessionManager,
		config.Services.OrderServiceURL,
		config.Services.OfferServiceURL,
	)
	
	gdsService := services.NewGDSService(db, sessionManager)
	gdsService.InitializeGDSConfigurations()
	
	transformerService := services.NewTransformerService(db, sessionManager)

	// Initialize controllers
	distributionController := controllers.NewDistributionController(
		ndcService,
		gdsService,
		sessionManager,
		transformerService,
	)

	// Setup HTTP server
	server := setupServer(config, distributionController)

	// Start background services
	go startBackgroundServices(sessionManager)

	// Start server
	log.Printf("Starting Distribution Service on %s:%d", config.Server.Host, config.Server.Port)
	
	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down Distribution Service...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Close database connection
	if err := database.CloseDB(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	// Close Redis connection
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			log.Printf("Error closing Redis: %v", err)
		}
	}

	log.Println("Distribution Service stopped")
}

func loadConfig() (*Config, error) {
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "config.yaml"
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Override with environment variables
	if host := os.Getenv("DB_HOST"); host != "" {
		config.Database.Host = host
	}
	if port := os.Getenv("DB_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &config.Database.Port)
	}
	if user := os.Getenv("DB_USER"); user != "" {
		config.Database.User = user
	}
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		config.Database.Password = password
	}
	if dbname := os.Getenv("DB_NAME"); dbname != "" {
		config.Database.DBName = dbname
	}

	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		config.Redis.Host = redisHost
	}
	if redisPort := os.Getenv("REDIS_PORT"); redisPort != "" {
		fmt.Sscanf(redisPort, "%d", &config.Redis.Port)
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		config.Redis.Password = redisPassword
	}

	return &config, nil
}

func setupServer(config *Config, controller *controllers.DistributionController) *http.Server {
	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// API routes
	v1 := router.Group("/api/v1")
	{
		// NDC routes
		ndc := v1.Group("/ndc")
		{
			ndc.POST("/airshopping", controller.ProcessNDCAirShopping)
			ndc.POST("/offerprice", controller.ProcessNDCOfferPrice)
			ndc.POST("/ordercreate", controller.ProcessNDCOrderCreate)
			ndc.POST("/orderretrieve", controller.ProcessNDCOrderRetrieve)
			ndc.POST("/ordercancel", controller.ProcessNDCOrderCancel)
		}

		// GDS routes
		gds := v1.Group("/gds")
		{
			gds.POST("/request", controller.ProcessGDSRequest)
		}

		// Multi-channel distribution
		distribution := v1.Group("/distribution")
		{
			distribution.POST("/multichannel", controller.ProcessMultiChannelDistribution)
		}

		// Session management
		sessions := v1.Group("/sessions")
		{
			sessions.POST("/ndc", controller.CreateNDCSession)
			sessions.GET("/ndc/:session_id", controller.GetNDCSession)
			sessions.DELETE("/:session_id", controller.ExpireSession)
			sessions.GET("/stats", controller.GetSessionStats)
		}

		// Configuration management
		config := v1.Group("/config")
		{
			config.GET("/channels", controller.ListChannelConfigurations)
			config.GET("/channels/:channel_id", controller.GetChannelConfiguration)
			config.PUT("/channels/:channel_id", controller.UpdateChannelConfiguration)
		}

		// Health and monitoring
		v1.GET("/health", controller.HealthCheck)
		v1.GET("/metrics", controller.GetMetrics)
	}

	// Root health check
	router.GET("/health", controller.HealthCheck)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(config.Server.IdleTimeout) * time.Second,
	}

	return server
}

func startBackgroundServices(sessionManager *services.SessionManager) {
	// Session cleanup ticker
	cleanupTicker := time.NewTicker(30 * time.Minute)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-cleanupTicker.C:
			ctx := context.Background()
			if err := sessionManager.CleanupExpiredSessions(ctx); err != nil {
				log.Printf("Error cleaning up expired sessions: %v", err)
			}
		}
	}
} 