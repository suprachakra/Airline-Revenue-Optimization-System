package config

import (
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v2"
)

// Config represents the main configuration structure
type Config struct {
	Environment       string                    `yaml:"environment"`
	Server            ServerConfig              `yaml:"server"`
	Gateway           GatewayConfig             `yaml:"gateway"`
	Services          ServicesConfig            `yaml:"services"`
	Auth              AuthConfig                `yaml:"auth"`
	RateLimit         RateLimitConfig           `yaml:"rate_limit"`
	CircuitBreaker    CircuitBreakerConfig      `yaml:"circuit_breaker"`
	ServiceRegistry   ServiceRegistryConfig     `yaml:"service_registry"`
	LoadBalancer      LoadBalancerConfig        `yaml:"load_balancer"`
	Redis             RedisConfig               `yaml:"redis"`
	Monitoring        MonitoringConfig          `yaml:"monitoring"`
	CORS              CORSConfig                `yaml:"cors"`
	Security          SecurityConfig            `yaml:"security"`
	Logging           LoggingConfig             `yaml:"logging"`
}

// ServerConfig represents HTTP server configuration
type ServerConfig struct {
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
	TLS          TLSConfig     `yaml:"tls"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// GatewayConfig represents API Gateway specific configuration
type GatewayConfig struct {
	Version   string    `yaml:"version"`
	StartTime time.Time `yaml:"-"`
}

// ServicesConfig represents backend services configuration
type ServicesConfig struct {
	Pricing         ServiceConfig `yaml:"pricing"`
	Forecasting     ServiceConfig `yaml:"forecasting"`
	Offer           ServiceConfig `yaml:"offer"`
	Order           ServiceConfig `yaml:"order"`
	Distribution    ServiceConfig `yaml:"distribution"`
	Ancillary       ServiceConfig `yaml:"ancillary"`
	UserManagement  ServiceConfig `yaml:"user_management"`
	NetworkPlanning ServiceConfig `yaml:"network_planning"`
	Procurement     ServiceConfig `yaml:"procurement"`
	Promotion       ServiceConfig `yaml:"promotion"`
}

// ServiceConfig represents individual service configuration
type ServiceConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Primary   string `yaml:"primary"`
	Secondary string `yaml:"secondary"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	JWTEnabled       bool          `yaml:"jwt_enabled"`
	JWTKeyPath       string        `yaml:"jwt_key_path"`
	JWTIssuer        string        `yaml:"jwt_issuer"`
	JWTAudience      string        `yaml:"jwt_audience"`
	JWTExpiry        time.Duration `yaml:"jwt_expiry"`
	APIKeyEnabled    bool          `yaml:"api_key_enabled"`
	SessionEnabled   bool          `yaml:"session_enabled"`
	RBACConfig       string        `yaml:"rbac_config"`
	APIKeyConfig     string        `yaml:"api_key_config"`
	SessionConfig    string        `yaml:"session_config"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	Global    RateLimitRule `yaml:"global"`
	PerIP     RateLimitRule `yaml:"per_ip"`
	PerUser   RateLimitRule `yaml:"per_user"`
	PerAPIKey RateLimitRule `yaml:"per_api_key"`
	PerPath   RateLimitRule `yaml:"per_path"`
	PerMethod RateLimitRule `yaml:"per_method"`
}

// RateLimitRule represents a rate limiting rule
type RateLimitRule struct {
	Limit  int           `yaml:"limit"`
	Window time.Duration `yaml:"window"`
}

// CircuitBreakerConfig represents circuit breaker configuration
type CircuitBreakerConfig struct {
	DefaultFailureThreshold int           `yaml:"default_failure_threshold"`
	DefaultSuccessThreshold int           `yaml:"default_success_threshold"`
	DefaultTimeout          time.Duration `yaml:"default_timeout"`
	DefaultMaxRequests      int           `yaml:"default_max_requests"`
}

// ServiceRegistryConfig represents service registry configuration
type ServiceRegistryConfig struct {
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	HealthCheckTimeout  time.Duration `yaml:"health_check_timeout"`
	CacheExpiry         time.Duration `yaml:"cache_expiry"`
}

// LoadBalancerConfig represents load balancer configuration
type LoadBalancerConfig struct {
	Strategy string `yaml:"strategy"` // round_robin, weighted, least_connections
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Address     string `yaml:"address"`
	Password    string `yaml:"password"`
	AuthDB      int    `yaml:"auth_db"`
	RateLimitDB int    `yaml:"rate_limit_db"`
	ServiceDB   int    `yaml:"service_db"`
	CacheDB     int    `yaml:"cache_db"`
}

// MonitoringConfig represents monitoring configuration
type MonitoringConfig struct {
	Enabled         bool          `yaml:"enabled"`
	MetricsPath     string        `yaml:"metrics_path"`
	ReportInterval  time.Duration `yaml:"report_interval"`
	PrometheusAddr  string        `yaml:"prometheus_addr"`
	JaegerAddr      string        `yaml:"jaeger_addr"`
}

// CORSConfig represents CORS configuration
type CORSConfig struct {
	AllowedOrigins []string      `yaml:"allowed_origins"`
	AllowedMethods []string      `yaml:"allowed_methods"`
	AllowedHeaders []string      `yaml:"allowed_headers"`
	MaxAge         time.Duration `yaml:"max_age"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	EnableSecurityHeaders bool              `yaml:"enable_security_headers"`
	CSPPolicy             string            `yaml:"csp_policy"`
	HSTSMaxAge            int               `yaml:"hsts_max_age"`
	CustomHeaders         map[string]string `yaml:"custom_headers"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// LoadConfig loads configuration from file or environment variables
func LoadConfig() (*Config, error) {
	// Default configuration
	cfg := &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Server: ServerConfig{
			Port:         getEnvInt("SERVER_PORT", 8080),
			ReadTimeout:  getEnvDuration("READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getEnvDuration("WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  getEnvDuration("IDLE_TIMEOUT", 60*time.Second),
			TLS: TLSConfig{
				Enabled:  getEnvBool("TLS_ENABLED", false),
				CertFile: getEnv("TLS_CERT_FILE", ""),
				KeyFile:  getEnv("TLS_KEY_FILE", ""),
			},
		},
		Gateway: GatewayConfig{
			Version:   getEnv("GATEWAY_VERSION", "1.0.0"),
			StartTime: time.Now(),
		},
		Services: ServicesConfig{
			Pricing: ServiceConfig{
				Host:      getEnv("PRICING_SERVICE_HOST", "pricing-service"),
				Port:      getEnvInt("PRICING_SERVICE_PORT", 8080),
				Primary:   getEnv("PRICING_SERVICE_PRIMARY", "http://pricing-service:8080"),
				Secondary: getEnv("PRICING_SERVICE_SECONDARY", "http://pricing-service-2:8080"),
			},
			Forecasting: ServiceConfig{
				Host:      getEnv("FORECASTING_SERVICE_HOST", "forecasting-service"),
				Port:      getEnvInt("FORECASTING_SERVICE_PORT", 8080),
				Primary:   getEnv("FORECASTING_SERVICE_PRIMARY", "http://forecasting-service:8080"),
				Secondary: getEnv("FORECASTING_SERVICE_SECONDARY", "http://forecasting-service-2:8080"),
			},
			Offer: ServiceConfig{
				Host:      getEnv("OFFER_SERVICE_HOST", "offer-service"),
				Port:      getEnvInt("OFFER_SERVICE_PORT", 8080),
				Primary:   getEnv("OFFER_SERVICE_PRIMARY", "http://offer-service:8080"),
				Secondary: getEnv("OFFER_SERVICE_SECONDARY", "http://offer-service-2:8080"),
			},
			Order: ServiceConfig{
				Host:      getEnv("ORDER_SERVICE_HOST", "order-service"),
				Port:      getEnvInt("ORDER_SERVICE_PORT", 8080),
				Primary:   getEnv("ORDER_SERVICE_PRIMARY", "http://order-service:8080"),
				Secondary: getEnv("ORDER_SERVICE_SECONDARY", "http://order-service-2:8080"),
			},
			Distribution: ServiceConfig{
				Host:      getEnv("DISTRIBUTION_SERVICE_HOST", "distribution-service"),
				Port:      getEnvInt("DISTRIBUTION_SERVICE_PORT", 8080),
				Primary:   getEnv("DISTRIBUTION_SERVICE_PRIMARY", "http://distribution-service:8080"),
				Secondary: getEnv("DISTRIBUTION_SERVICE_SECONDARY", "http://distribution-service-2:8080"),
			},
			Ancillary: ServiceConfig{
				Host:      getEnv("ANCILLARY_SERVICE_HOST", "ancillary-service"),
				Port:      getEnvInt("ANCILLARY_SERVICE_PORT", 8080),
				Primary:   getEnv("ANCILLARY_SERVICE_PRIMARY", "http://ancillary-service:8080"),
				Secondary: getEnv("ANCILLARY_SERVICE_SECONDARY", "http://ancillary-service-2:8080"),
			},
			UserManagement: ServiceConfig{
				Host:      getEnv("USER_SERVICE_HOST", "user-service"),
				Port:      getEnvInt("USER_SERVICE_PORT", 8080),
				Primary:   getEnv("USER_SERVICE_PRIMARY", "http://user-service:8080"),
				Secondary: getEnv("USER_SERVICE_SECONDARY", "http://user-service-2:8080"),
			},
			NetworkPlanning: ServiceConfig{
				Host:      getEnv("NETWORK_SERVICE_HOST", "network-service"),
				Port:      getEnvInt("NETWORK_SERVICE_PORT", 8080),
				Primary:   getEnv("NETWORK_SERVICE_PRIMARY", "http://network-service:8080"),
				Secondary: getEnv("NETWORK_SERVICE_SECONDARY", "http://network-service-2:8080"),
			},
			Procurement: ServiceConfig{
				Host:      getEnv("PROCUREMENT_SERVICE_HOST", "procurement-service"),
				Port:      getEnvInt("PROCUREMENT_SERVICE_PORT", 8080),
				Primary:   getEnv("PROCUREMENT_SERVICE_PRIMARY", "http://procurement-service:8080"),
				Secondary: getEnv("PROCUREMENT_SERVICE_SECONDARY", "http://procurement-service-2:8080"),
			},
			Promotion: ServiceConfig{
				Host:      getEnv("PROMOTION_SERVICE_HOST", "promotion-service"),
				Port:      getEnvInt("PROMOTION_SERVICE_PORT", 8080),
				Primary:   getEnv("PROMOTION_SERVICE_PRIMARY", "http://promotion-service:8080"),
				Secondary: getEnv("PROMOTION_SERVICE_SECONDARY", "http://promotion-service-2:8080"),
			},
		},
		Auth: AuthConfig{
			JWTEnabled:    getEnvBool("JWT_ENABLED", true),
			JWTKeyPath:    getEnv("JWT_KEY_PATH", "/etc/jwt"),
			JWTIssuer:     getEnv("JWT_ISSUER", "iaros-api-gateway"),
			JWTAudience:   getEnv("JWT_AUDIENCE", "iaros-services"),
			JWTExpiry:     getEnvDuration("JWT_EXPIRY", 24*time.Hour),
			APIKeyEnabled: getEnvBool("API_KEY_ENABLED", true),
			SessionEnabled: getEnvBool("SESSION_ENABLED", true),
			RBACConfig:    getEnv("RBAC_CONFIG", "rbac.yaml"),
			APIKeyConfig:  getEnv("API_KEY_CONFIG", "api_keys.yaml"),
			SessionConfig: getEnv("SESSION_CONFIG", "sessions.yaml"),
		},
		RateLimit: RateLimitConfig{
			Global: RateLimitRule{
				Limit:  getEnvInt("RATE_LIMIT_GLOBAL", 10000),
				Window: getEnvDuration("RATE_LIMIT_GLOBAL_WINDOW", time.Minute),
			},
			PerIP: RateLimitRule{
				Limit:  getEnvInt("RATE_LIMIT_PER_IP", 1000),
				Window: getEnvDuration("RATE_LIMIT_PER_IP_WINDOW", time.Minute),
			},
			PerUser: RateLimitRule{
				Limit:  getEnvInt("RATE_LIMIT_PER_USER", 5000),
				Window: getEnvDuration("RATE_LIMIT_PER_USER_WINDOW", time.Minute),
			},
			PerAPIKey: RateLimitRule{
				Limit:  getEnvInt("RATE_LIMIT_PER_API_KEY", 10000),
				Window: getEnvDuration("RATE_LIMIT_PER_API_KEY_WINDOW", time.Minute),
			},
			PerPath: RateLimitRule{
				Limit:  getEnvInt("RATE_LIMIT_PER_PATH", 2000),
				Window: getEnvDuration("RATE_LIMIT_PER_PATH_WINDOW", time.Minute),
			},
			PerMethod: RateLimitRule{
				Limit:  getEnvInt("RATE_LIMIT_PER_METHOD", 3000),
				Window: getEnvDuration("RATE_LIMIT_PER_METHOD_WINDOW", time.Minute),
			},
		},
		CircuitBreaker: CircuitBreakerConfig{
			DefaultFailureThreshold: getEnvInt("CIRCUIT_BREAKER_FAILURE_THRESHOLD", 5),
			DefaultSuccessThreshold: getEnvInt("CIRCUIT_BREAKER_SUCCESS_THRESHOLD", 3),
			DefaultTimeout:          getEnvDuration("CIRCUIT_BREAKER_TIMEOUT", 30*time.Second),
			DefaultMaxRequests:      getEnvInt("CIRCUIT_BREAKER_MAX_REQUESTS", 100),
		},
		ServiceRegistry: ServiceRegistryConfig{
			HealthCheckInterval: getEnvDuration("SERVICE_HEALTH_CHECK_INTERVAL", 30*time.Second),
			HealthCheckTimeout:  getEnvDuration("SERVICE_HEALTH_CHECK_TIMEOUT", 5*time.Second),
			CacheExpiry:         getEnvDuration("SERVICE_CACHE_EXPIRY", 5*time.Minute),
		},
		LoadBalancer: LoadBalancerConfig{
			Strategy: getEnv("LOAD_BALANCER_STRATEGY", "round_robin"),
		},
		Redis: RedisConfig{
			Address:     getEnv("REDIS_ADDRESS", "localhost:6379"),
			Password:    getEnv("REDIS_PASSWORD", ""),
			AuthDB:      getEnvInt("REDIS_AUTH_DB", 0),
			RateLimitDB: getEnvInt("REDIS_RATE_LIMIT_DB", 1),
			ServiceDB:   getEnvInt("REDIS_SERVICE_DB", 2),
			CacheDB:     getEnvInt("REDIS_CACHE_DB", 3),
		},
		Monitoring: MonitoringConfig{
			Enabled:        getEnvBool("MONITORING_ENABLED", true),
			MetricsPath:    getEnv("METRICS_PATH", "/metrics"),
			ReportInterval: getEnvDuration("MONITORING_REPORT_INTERVAL", 30*time.Second),
			PrometheusAddr: getEnv("PROMETHEUS_ADDR", "localhost:9090"),
			JaegerAddr:     getEnv("JAEGER_ADDR", "localhost:14268"),
		},
		CORS: CORSConfig{
			AllowedOrigins: []string{
				getEnv("CORS_ALLOWED_ORIGINS", "*"),
			},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Content-Type", "Authorization", "X-API-Key"},
			MaxAge:         getEnvDuration("CORS_MAX_AGE", 12*time.Hour),
		},
		Security: SecurityConfig{
			EnableSecurityHeaders: getEnvBool("SECURITY_HEADERS_ENABLED", true),
			CSPPolicy:             getEnv("CSP_POLICY", "default-src 'self'"),
			HSTSMaxAge:            getEnvInt("HSTS_MAX_AGE", 31536000),
			CustomHeaders:         make(map[string]string),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
			Output: getEnv("LOG_OUTPUT", "stdout"),
		},
	}

	// Try to load from config file if specified
	if configFile := getEnv("CONFIG_FILE", ""); configFile != "" {
		if err := loadConfigFile(cfg, configFile); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// LoadTestConfig loads test configuration
func LoadTestConfig() *Config {
	cfg, _ := LoadConfig()
	cfg.Environment = "test"
	cfg.Server.Port = 0 // Use random port for tests
	return cfg
}

// loadConfigFile loads configuration from YAML file
func loadConfigFile(cfg *Config, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, cfg)
}

// Helper functions for environment variables
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
} 