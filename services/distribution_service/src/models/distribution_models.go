package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// DistributionTransaction represents a complete distribution transaction
type DistributionTransaction struct {
	ID              uint                    `gorm:"primaryKey" json:"id"`
	TransactionID   string                  `gorm:"uniqueIndex;size:36" json:"transaction_id"`
	RequestID       string                  `gorm:"index;size:36" json:"request_id"`
	SourceChannel   DistributionChannel     `gorm:"size:50" json:"source_channel"`
	TargetChannels  string                  `gorm:"type:text" json:"target_channels"` // JSON array
	RequestType     string                  `gorm:"size:50" json:"request_type"`
	Status          TransactionStatus       `gorm:"size:20" json:"status"`
	Priority        int                     `gorm:"default:5" json:"priority"`
	
	// Request/Response data
	RequestData     string                  `gorm:"type:text" json:"request_data"`   // JSON
	ResponseData    string                  `gorm:"type:text" json:"response_data"`  // JSON
	ErrorData       string                  `gorm:"type:text" json:"error_data"`     // JSON
	
	// Timing information
	RequestedAt     time.Time               `json:"requested_at"`
	ProcessedAt     *time.Time              `json:"processed_at,omitempty"`
	CompletedAt     *time.Time              `json:"completed_at,omitempty"`
	ProcessingTime  time.Duration           `json:"processing_time"`
	Timeout         time.Duration           `json:"timeout"`
	
	// Metadata
	CustomerID      string                  `gorm:"index;size:36" json:"customer_id"`
	SessionID       string                  `gorm:"index;size:100" json:"session_id"`
	UserAgent       string                  `gorm:"size:500" json:"user_agent"`
	IPAddress       string                  `gorm:"size:45" json:"ip_address"`
	Metadata        string                  `gorm:"type:text" json:"metadata"` // JSON
	
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
	
	// Relationships
	ChannelLogs     []ChannelLog            `gorm:"foreignKey:TransactionID;references:TransactionID" json:"channel_logs"`
	AuditEntries    []DistributionAuditEntry `gorm:"foreignKey:TransactionID;references:TransactionID" json:"audit_entries"`
}

type TransactionStatus string

const (
	TransactionStatusPending    TransactionStatus = "PENDING"
	TransactionStatusProcessing TransactionStatus = "PROCESSING"
	TransactionStatusCompleted  TransactionStatus = "COMPLETED"
	TransactionStatusFailed     TransactionStatus = "FAILED"
	TransactionStatusTimeout    TransactionStatus = "TIMEOUT"
	TransactionStatusCancelled  TransactionStatus = "CANCELLED"
)

// BeforeCreate sets the TransactionID
func (dt *DistributionTransaction) BeforeCreate(tx *gorm.DB) error {
	if dt.TransactionID == "" {
		dt.TransactionID = uuid.New().String()
	}
	return nil
}

// GetTargetChannels returns target channels as slice
func (dt *DistributionTransaction) GetTargetChannels() ([]DistributionChannel, error) {
	if dt.TargetChannels == "" {
		return []DistributionChannel{}, nil
	}
	var channels []DistributionChannel
	err := json.Unmarshal([]byte(dt.TargetChannels), &channels)
	return channels, err
}

// SetTargetChannels sets target channels from slice
func (dt *DistributionTransaction) SetTargetChannels(channels []DistributionChannel) error {
	data, err := json.Marshal(channels)
	if err != nil {
		return err
	}
	dt.TargetChannels = string(data)
	return nil
}

// GetMetadata returns metadata as map
func (dt *DistributionTransaction) GetMetadata() (map[string]interface{}, error) {
	if dt.Metadata == "" {
		return make(map[string]interface{}), nil
	}
	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(dt.Metadata), &metadata)
	return metadata, err
}

// SetMetadata sets metadata from map
func (dt *DistributionTransaction) SetMetadata(metadata map[string]interface{}) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	dt.Metadata = string(data)
	return nil
}

// ChannelConfiguration represents configuration for distribution channels
type ChannelConfiguration struct {
	ID              uint                    `gorm:"primaryKey" json:"id"`
	ChannelID       string                  `gorm:"uniqueIndex;size:100" json:"channel_id"`
	ChannelType     DistributionChannel     `gorm:"size:50" json:"channel_type"`
	Provider        string                  `gorm:"size:100" json:"provider"`
	Name            string                  `gorm:"size:200" json:"name"`
	Description     string                  `gorm:"size:500" json:"description"`
	Enabled         bool                    `gorm:"default:true" json:"enabled"`
	Priority        int                     `gorm:"default:5" json:"priority"`
	
	// Configuration data
	Configuration   string                  `gorm:"type:text" json:"configuration"`   // JSON
	Authentication  string                  `gorm:"type:text" json:"authentication"`  // JSON
	RateLimits      string                  `gorm:"type:text" json:"rate_limits"`     // JSON
	Features        string                  `gorm:"type:text" json:"features"`        // JSON
	
	// Connection settings
	BaseURL         string                  `gorm:"size:500" json:"base_url"`
	Timeout         time.Duration           `json:"timeout"`
	RetryAttempts   int                     `gorm:"default:3" json:"retry_attempts"`
	RetryDelay      time.Duration           `json:"retry_delay"`
	
	// Health monitoring
	HealthCheckURL  string                  `gorm:"size:500" json:"health_check_url"`
	LastHealthCheck *time.Time              `json:"last_health_check,omitempty"`
	HealthStatus    ChannelHealthStatus     `gorm:"size:20;default:UNKNOWN" json:"health_status"`
	
	// Metrics
	TotalRequests   int64                   `gorm:"default:0" json:"total_requests"`
	SuccessfulRequests int64                `gorm:"default:0" json:"successful_requests"`
	FailedRequests  int64                   `gorm:"default:0" json:"failed_requests"`
	AvgResponseTime time.Duration           `json:"avg_response_time"`
	
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
	
	// Relationships
	ChannelLogs     []ChannelLog            `gorm:"foreignKey:ChannelID;references:ChannelID" json:"channel_logs,omitempty"`
}

type ChannelHealthStatus string

const (
	ChannelHealthHealthy   ChannelHealthStatus = "HEALTHY"
	ChannelHealthUnhealthy ChannelHealthStatus = "UNHEALTHY"
	ChannelHealthDegraded  ChannelHealthStatus = "DEGRADED"
	ChannelHealthUnknown   ChannelHealthStatus = "UNKNOWN"
)

// GetConfiguration returns configuration as map
func (cc *ChannelConfiguration) GetConfiguration() (map[string]interface{}, error) {
	if cc.Configuration == "" {
		return make(map[string]interface{}), nil
	}
	var config map[string]interface{}
	err := json.Unmarshal([]byte(cc.Configuration), &config)
	return config, err
}

// SetConfiguration sets configuration from map
func (cc *ChannelConfiguration) SetConfiguration(config map[string]interface{}) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	cc.Configuration = string(data)
	return nil
}

// ChannelLog represents logs for individual channel interactions
type ChannelLog struct {
	ID              uint                    `gorm:"primaryKey" json:"id"`
	LogID           string                  `gorm:"uniqueIndex;size:36" json:"log_id"`
	TransactionID   string                  `gorm:"index;size:36" json:"transaction_id"`
	ChannelID       string                  `gorm:"index;size:100" json:"channel_id"`
	ChannelType     DistributionChannel     `gorm:"size:50" json:"channel_type"`
	
	// Request/Response details
	RequestType     string                  `gorm:"size:50" json:"request_type"`
	RequestURL      string                  `gorm:"size:1000" json:"request_url"`
	RequestHeaders  string                  `gorm:"type:text" json:"request_headers"`  // JSON
	RequestBody     string                  `gorm:"type:text" json:"request_body"`     // JSON
	ResponseStatus  int                     `json:"response_status"`
	ResponseHeaders string                  `gorm:"type:text" json:"response_headers"` // JSON
	ResponseBody    string                  `gorm:"type:text" json:"response_body"`    // JSON
	ErrorMessage    string                  `gorm:"size:1000" json:"error_message"`
	
	// Timing
	StartedAt       time.Time               `json:"started_at"`
	CompletedAt     *time.Time              `json:"completed_at,omitempty"`
	Duration        time.Duration           `json:"duration"`
	
	// Retry information
	AttemptNumber   int                     `gorm:"default:1" json:"attempt_number"`
	IsRetry         bool                    `gorm:"default:false" json:"is_retry"`
	RetryReason     string                  `gorm:"size:500" json:"retry_reason"`
	
	// Success/Failure
	Success         bool                    `json:"success"`
	
	CreatedAt       time.Time               `json:"created_at"`
}

// BeforeCreate sets the LogID
func (cl *ChannelLog) BeforeCreate(tx *gorm.DB) error {
	if cl.LogID == "" {
		cl.LogID = uuid.New().String()
	}
	return nil
}

// DistributionAuditEntry represents audit trail for distribution operations
type DistributionAuditEntry struct {
	ID              uint                    `gorm:"primaryKey" json:"id"`
	AuditID         string                  `gorm:"uniqueIndex;size:36" json:"audit_id"`
	TransactionID   string                  `gorm:"index;size:36" json:"transaction_id"`
	
	// Event details
	EventType       string                  `gorm:"size:100" json:"event_type"`
	EventSource     string                  `gorm:"size:100" json:"event_source"`
	Description     string                  `gorm:"size:1000" json:"description"`
	
	// Context
	UserID          string                  `gorm:"size:36" json:"user_id"`
	SessionID       string                  `gorm:"size:100" json:"session_id"`
	Channel         DistributionChannel     `gorm:"size:50" json:"channel"`
	
	// Data
	OldValue        string                  `gorm:"type:text" json:"old_value"`  // JSON
	NewValue        string                  `gorm:"type:text" json:"new_value"`  // JSON
	Metadata        string                  `gorm:"type:text" json:"metadata"`   // JSON
	
	// Security
	IPAddress       string                  `gorm:"size:45" json:"ip_address"`
	UserAgent       string                  `gorm:"size:500" json:"user_agent"`
	
	Timestamp       time.Time               `json:"timestamp"`
}

// BeforeCreate sets the AuditID
func (dae *DistributionAuditEntry) BeforeCreate(tx *gorm.DB) error {
	if dae.AuditID == "" {
		dae.AuditID = uuid.New().String()
	}
	return nil
}

// NDCSession represents NDC session management
type NDCSession struct {
	ID              uint                    `gorm:"primaryKey" json:"id"`
	SessionID       string                  `gorm:"uniqueIndex;size:100" json:"session_id"`
	CustomerID      string                  `gorm:"index;size:36" json:"customer_id"`
	AirlineCode     string                  `gorm:"size:10" json:"airline_code"`
	
	// Session state
	Status          NDCSessionStatus        `gorm:"size:20" json:"status"`
	ShoppingContext string                  `gorm:"type:text" json:"shopping_context"` // JSON
	OfferContext    string                  `gorm:"type:text" json:"offer_context"`    // JSON
	OrderContext    string                  `gorm:"type:text" json:"order_context"`    // JSON
	
	// NDC specific data
	NDCVersion      string                  `gorm:"size:20" json:"ndc_version"`
	Capability      string                  `gorm:"type:text" json:"capability"`      // JSON
	SourceQualifier string                  `gorm:"size:100" json:"source_qualifier"`
	
	// Timing
	CreatedAt       time.Time               `json:"created_at"`
	LastAccessedAt  time.Time               `json:"last_accessed_at"`
	ExpiresAt       time.Time               `json:"expires_at"`
	TTL             time.Duration           `json:"ttl"`
	
	// Security
	IPAddress       string                  `gorm:"size:45" json:"ip_address"`
	UserAgent       string                  `gorm:"size:500" json:"user_agent"`
	
	UpdatedAt       time.Time               `json:"updated_at"`
}

type NDCSessionStatus string

const (
	NDCSessionActive   NDCSessionStatus = "ACTIVE"
	NDCSessionExpired  NDCSessionStatus = "EXPIRED"
	NDCSessionInvalid  NDCSessionStatus = "INVALID"
	NDCSessionTerminated NDCSessionStatus = "TERMINATED"
)

// GDSSession represents GDS session management
type GDSSession struct {
	ID              uint                    `gorm:"primaryKey" json:"id"`
	SessionID       string                  `gorm:"uniqueIndex;size:100" json:"session_id"`
	Provider        GDSProvider             `gorm:"size:20" json:"provider"`
	PseudoCity      string                  `gorm:"size:20" json:"pseudo_city"`
	UserID          string                  `gorm:"size:50" json:"user_id"`
	
	// Session credentials
	AuthToken       string                  `gorm:"size:1000" json:"auth_token"`
	RefreshToken    string                  `gorm:"size:1000" json:"refresh_token"`
	TokenExpiresAt  time.Time               `json:"token_expires_at"`
	
	// Session state
	Status          GDSSessionStatus        `gorm:"size:20" json:"status"`
	LastActivity    time.Time               `json:"last_activity"`
	MessageSequence int                     `gorm:"default:0" json:"message_sequence"`
	
	// Context data
	SearchContext   string                  `gorm:"type:text" json:"search_context"`   // JSON
	BookingContext  string                  `gorm:"type:text" json:"booking_context"`  // JSON
	
	CreatedAt       time.Time               `json:"created_at"`
	ExpiresAt       time.Time               `json:"expires_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

type GDSSessionStatus string

const (
	GDSSessionActive      GDSSessionStatus = "ACTIVE"
	GDSSessionExpired     GDSSessionStatus = "EXPIRED"
	GDSSessionTerminated  GDSSessionStatus = "TERMINATED"
	GDSSessionAuthenticated GDSSessionStatus = "AUTHENTICATED"
)

// DistributionMetric represents performance metrics for distribution channels
type DistributionMetric struct {
	ID              uint                    `gorm:"primaryKey" json:"id"`
	MetricID        string                  `gorm:"uniqueIndex;size:36" json:"metric_id"`
	ChannelID       string                  `gorm:"index;size:100" json:"channel_id"`
	ChannelType     DistributionChannel     `gorm:"size:50" json:"channel_type"`
	
	// Metric data
	MetricType      string                  `gorm:"size:50" json:"metric_type"`
	MetricName      string                  `gorm:"size:100" json:"metric_name"`
	Value           decimal.Decimal         `gorm:"type:decimal(20,6)" json:"value"`
	Unit            string                  `gorm:"size:20" json:"unit"`
	
	// Aggregation
	AggregationType string                  `gorm:"size:20" json:"aggregation_type"` // COUNT, SUM, AVG, MIN, MAX
	TimeWindow      string                  `gorm:"size:20" json:"time_window"`      // MINUTE, HOUR, DAY, WEEK, MONTH
	
	// Context
	Tags            string                  `gorm:"type:text" json:"tags"`           // JSON
	Dimensions      string                  `gorm:"type:text" json:"dimensions"`     // JSON
	
	Timestamp       time.Time               `json:"timestamp"`
	CreatedAt       time.Time               `json:"created_at"`
}

// BeforeCreate sets the MetricID
func (dm *DistributionMetric) BeforeCreate(tx *gorm.DB) error {
	if dm.MetricID == "" {
		dm.MetricID = uuid.New().String()
	}
	return nil
}

// ChannelCapability represents capabilities supported by each channel
type ChannelCapability struct {
	ID              uint                    `gorm:"primaryKey" json:"id"`
	ChannelID       string                  `gorm:"index;size:100" json:"channel_id"`
	ChannelType     DistributionChannel     `gorm:"size:50" json:"channel_type"`
	
	// Capabilities
	CapabilityType  string                  `gorm:"size:50" json:"capability_type"`
	CapabilityName  string                  `gorm:"size:100" json:"capability_name"`
	Supported       bool                    `gorm:"default:false" json:"supported"`
	Version         string                  `gorm:"size:20" json:"version"`
	
	// Configuration
	Configuration   string                  `gorm:"type:text" json:"configuration"`   // JSON
	Limitations     string                  `gorm:"type:text" json:"limitations"`     // JSON
	
	// Status
	Enabled         bool                    `gorm:"default:true" json:"enabled"`
	LastTested      *time.Time              `json:"last_tested,omitempty"`
	TestStatus      string                  `gorm:"size:20" json:"test_status"`
	
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

// TransformationRule represents data transformation rules between channels
type TransformationRule struct {
	ID              uint                    `gorm:"primaryKey" json:"id"`
	RuleID          string                  `gorm:"uniqueIndex;size:36" json:"rule_id"`
	Name            string                  `gorm:"size:200" json:"name"`
	Description     string                  `gorm:"size:1000" json:"description"`
	
	// Source and target
	SourceChannel   DistributionChannel     `gorm:"size:50" json:"source_channel"`
	TargetChannel   DistributionChannel     `gorm:"size:50" json:"target_channel"`
	MessageType     string                  `gorm:"size:50" json:"message_type"`
	
	// Rule definition
	RuleDefinition  string                  `gorm:"type:text" json:"rule_definition"`  // JSON
	FieldMappings   string                  `gorm:"type:text" json:"field_mappings"`   // JSON
	Conditions      string                  `gorm:"type:text" json:"conditions"`      // JSON
	
	// Execution
	Priority        int                     `gorm:"default:5" json:"priority"`
	Enabled         bool                    `gorm:"default:true" json:"enabled"`
	ExecutionCount  int64                   `gorm:"default:0" json:"execution_count"`
	SuccessCount    int64                   `gorm:"default:0" json:"success_count"`
	ErrorCount      int64                   `gorm:"default:0" json:"error_count"`
	
	// Timing
	LastExecuted    *time.Time              `json:"last_executed,omitempty"`
	AvgExecutionTime time.Duration          `json:"avg_execution_time"`
	
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
	CreatedBy       string                  `gorm:"size:36" json:"created_by"`
	UpdatedBy       string                  `gorm:"size:36" json:"updated_by"`
}

// BeforeCreate sets the RuleID
func (tr *TransformationRule) BeforeCreate(tx *gorm.DB) error {
	if tr.RuleID == "" {
		tr.RuleID = uuid.New().String()
	}
	return nil
}

// Table names
func (DistributionTransaction) TableName() string  { return "distribution_transactions" }
func (ChannelConfiguration) TableName() string     { return "channel_configurations" }
func (ChannelLog) TableName() string               { return "channel_logs" }
func (DistributionAuditEntry) TableName() string   { return "distribution_audit_entries" }
func (NDCSession) TableName() string               { return "ndc_sessions" }
func (GDSSession) TableName() string               { return "gds_sessions" }
func (DistributionMetric) TableName() string       { return "distribution_metrics" }
func (ChannelCapability) TableName() string        { return "channel_capabilities" }
func (TransformationRule) TableName() string       { return "transformation_rules" } 