package database

import (
	"fmt"
	"log"
	"time"

	"iaros/distribution_service/src/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	TimeZone string
}

// ConnectDatabase initializes database connection
func ConnectDatabase(config DatabaseConfig) error {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode, config.TimeZone,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db

	log.Println("Database connected successfully")
	return nil
}

// AutoMigrate runs database migrations
func AutoMigrate() error {
	if DB == nil {
		return fmt.Errorf("database connection not initialized")
	}

	// Run migrations for all models
	err := DB.AutoMigrate(
		&models.DistributionTransaction{},
		&models.ChannelConfiguration{},
		&models.ChannelLog{},
		&models.DistributionAuditEntry{},
		&models.NDCSession{},
		&models.GDSSession{},
		&models.DistributionMetric{},
		&models.ChannelCapability{},
		&models.TransformationRule{},
	)

	if err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// CreateIndexes creates additional database indexes for performance
func CreateIndexes() error {
	if DB == nil {
		return fmt.Errorf("database connection not initialized")
	}

	// Create composite indexes for better query performance
	indexes := []string{
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_distribution_transactions_status_created_at ON distribution_transactions(status, created_at)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_distribution_transactions_customer_status ON distribution_transactions(customer_id, status)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_distribution_transactions_source_channel_created_at ON distribution_transactions(source_channel, created_at)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_channel_logs_transaction_channel ON channel_logs(transaction_id, channel_id)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_channel_logs_channel_type_created_at ON channel_logs(channel_type, created_at)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_channel_logs_success_created_at ON channel_logs(success, created_at)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_channel_configurations_type_enabled ON channel_configurations(channel_type, enabled)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_channel_configurations_provider_enabled ON channel_configurations(provider, enabled)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ndc_sessions_customer_status ON ndc_sessions(customer_id, status)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ndc_sessions_expires_at ON ndc_sessions(expires_at)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_gds_sessions_provider_user_id ON gds_sessions(provider, user_id)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_gds_sessions_expires_at ON gds_sessions(expires_at)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_distribution_metrics_channel_timestamp ON distribution_metrics(channel_id, timestamp)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_distribution_metrics_metric_type_timestamp ON distribution_metrics(metric_type, timestamp)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_distribution_audit_entries_transaction_timestamp ON distribution_audit_entries(transaction_id, timestamp)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_distribution_audit_entries_user_timestamp ON distribution_audit_entries(user_id, timestamp)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_channel_capabilities_channel_type_enabled ON channel_capabilities(channel_id, channel_type, enabled)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_transformation_rules_source_target_enabled ON transformation_rules(source_channel, target_channel, enabled)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_transformation_rules_message_type_enabled ON transformation_rules(message_type, enabled)",
	}

	for _, index := range indexes {
		if err := DB.Exec(index).Error; err != nil {
			log.Printf("Warning: Failed to create index: %v", err)
		}
	}

	log.Println("Database indexes created successfully")
	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}

// CloseDB closes the database connection
func CloseDB() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	return sqlDB.Close()
}

// HealthCheck checks database connectivity
func HealthCheck() error {
	if DB == nil {
		return fmt.Errorf("database connection not initialized")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// GetConnectionStats returns database connection statistics
func GetConnectionStats() (map[string]interface{}, error) {
	if DB == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	stats := sqlDB.Stats()
	
	return map[string]interface{}{
		"open_connections":     stats.OpenConnections,
		"in_use":              stats.InUse,
		"idle":                stats.Idle,
		"wait_count":          stats.WaitCount,
		"wait_duration":       stats.WaitDuration,
		"max_idle_closed":     stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed": stats.MaxLifetimeClosed,
	}, nil
}

// Transaction executes a function within a database transaction
func Transaction(fn func(*gorm.DB) error) error {
	if DB == nil {
		return fmt.Errorf("database connection not initialized")
	}

	return DB.Transaction(fn)
} 