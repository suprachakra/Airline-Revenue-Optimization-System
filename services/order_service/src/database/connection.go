package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"iaros/order_service/src/models"
)

// Config holds database configuration
type Config struct {
	Host            string
	Port            string
	User            string
	Password        string
	DatabaseName    string
	SSLMode         string
	MaxConnections  int
	MaxIdleConnections int
	ConnMaxLifetime time.Duration
}

// Database wrapper
type Database struct {
	DB *gorm.DB
}

var db *Database

// GetConfig returns database configuration from environment variables
func GetConfig() *Config {
	return &Config{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnv("DB_PORT", "5432"),
		User:            getEnv("DB_USER", "postgres"),
		Password:        getEnv("DB_PASSWORD", "password"),
		DatabaseName:    getEnv("DB_NAME", "iaros_orders"),
		SSLMode:         getEnv("DB_SSL_MODE", "disable"),
		MaxConnections:  getEnvInt("DB_MAX_CONNECTIONS", 25),
		MaxIdleConnections: getEnvInt("DB_MAX_IDLE_CONNECTIONS", 5),
		ConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME", 300)) * time.Second,
	}
}

// Connect establishes database connection
func Connect() (*Database, error) {
	config := GetConfig()
	
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DatabaseName, config.SSLMode,
	)
	
	// Configure GORM logger
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)
	
	// Open database connection
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	
	// Get underlying sql.DB to configure connection pool
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %v", err)
	}
	
	// Configure connection pool
	sqlDB.SetMaxOpenConns(config.MaxConnections)
	sqlDB.SetMaxIdleConns(config.MaxIdleConnections)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	
	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}
	
	db = &Database{DB: gormDB}
	
	log.Printf("Successfully connected to database: %s:%s/%s", config.Host, config.Port, config.DatabaseName)
	
	return db, nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	if db == nil {
		log.Fatal("Database not initialized. Call Connect() first.")
	}
	return db.DB
}

// AutoMigrate runs database migrations
func AutoMigrate() error {
	gormDB := GetDB()
	
	// Enable UUID extension
	if err := gormDB.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		log.Printf("Warning: Could not create uuid-ossp extension: %v", err)
	}
	
	// Run migrations
	err := gormDB.AutoMigrate(
		&models.Order{},
		&models.OrderItem{},
		&models.ContactInfo{},
		&models.PassengerInfo{},
		&models.PaymentMethod{},
		&models.AuditEntry{},
	)
	
	if err != nil {
		return fmt.Errorf("failed to run migrations: %v", err)
	}
	
	log.Println("Database migrations completed successfully")
	
	// Create indexes for better performance
	if err := createIndexes(gormDB); err != nil {
		log.Printf("Warning: Could not create all indexes: %v", err)
	}
	
	return nil
}

// createIndexes creates additional database indexes
func createIndexes(db *gorm.DB) error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_orders_customer_status ON orders(customer_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_orders_route_departure ON orders(route, departure_date)",
		"CREATE INDEX IF NOT EXISTS idx_order_items_service_type ON order_items(service_type)",
		"CREATE INDEX IF NOT EXISTS idx_audit_entries_timestamp ON audit_entries(timestamp DESC)",
		"CREATE INDEX IF NOT EXISTS idx_passengers_name ON passenger_info(first_name, last_name)",
	}
	
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}
	
	return nil
}

// Close closes the database connection
func Close() error {
	if db == nil {
		return nil
	}
	
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	
	return sqlDB.Close()
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := time.ParseDuration(value + "s"); err == nil {
			return int(intValue.Seconds())
		}
	}
	return defaultValue
}

// HealthCheck performs database health check
func HealthCheck() error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %v", err)
	}
	
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %v", err)
	}
	
	return nil
}

// GetStats returns database connection statistics
func GetStats() map[string]interface{} {
	if db == nil {
		return map[string]interface{}{"error": "database not initialized"}
	}
	
	sqlDB, err := db.DB.DB()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	
	stats := sqlDB.Stats()
	
	return map[string]interface{}{
		"max_open_connections":     stats.MaxOpenConnections,
		"open_connections":         stats.OpenConnections,
		"in_use":                  stats.InUse,
		"idle":                    stats.Idle,
		"wait_count":              stats.WaitCount,
		"wait_duration":           stats.WaitDuration.String(),
		"max_idle_closed":         stats.MaxIdleClosed,
		"max_idle_time_closed":    stats.MaxIdleTimeClosed,
		"max_lifetime_closed":     stats.MaxLifetimeClosed,
	}
} 