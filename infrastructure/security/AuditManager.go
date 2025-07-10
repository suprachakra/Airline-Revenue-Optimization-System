package security

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/iaros/common/logging"
)

// AuditManager handles immutable audit logging for IAROS
type AuditManager struct {
	config        *AuditConfig
	auditLogger   *AuditLogger
	siemForwarder *SIEMForwarder
	chainVerifier *ChainVerifier
	logger        logging.Logger
	mu            sync.RWMutex
}

type AuditConfig struct {
	Enabled           bool          `json:"enabled"`
	LogLevel          string        `json:"log_level"`
	RetentionPeriod   time.Duration `json:"retention_period"`
	EncryptionEnabled bool          `json:"encryption_enabled"`
	SIEMEnabled       bool          `json:"siem_enabled"`
	ChainValidation   bool          `json:"chain_validation"`
	BufferSize        int           `json:"buffer_size"`
}

type AuditLogger struct {
	config     *AuditConfig
	buffer     []*AuditEvent
	bufferSize int
	lastHash   string
	eventCount int64
	logger     logging.Logger
	mu         sync.RWMutex
}

type AuditEvent struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	EventType     string                 `json:"event_type"`
	UserID        string                 `json:"user_id"`
	SessionID     string                 `json:"session_id"`
	Resource      string                 `json:"resource"`
	Action        string                 `json:"action"`
	Result        string                 `json:"result"`
	IPAddress     string                 `json:"ip_address"`
	UserAgent     string                 `json:"user_agent"`
	Details       map[string]interface{} `json:"details"`
	Hash          string                 `json:"hash"`
	PreviousHash  string                 `json:"previous_hash"`
	Signature     string                 `json:"signature"`
}

type SIEMForwarder struct {
	config   *SIEMConfig
	client   *SIEMClient
	logger   logging.Logger
}

type SIEMConfig struct {
	Endpoint    string `json:"endpoint"`
	APIKey      string `json:"api_key"`
	BatchSize   int    `json:"batch_size"`
	MaxRetries  int    `json:"max_retries"`
	RetryDelay  time.Duration `json:"retry_delay"`
}

type SIEMClient struct {
	endpoint string
	apiKey   string
	logger   logging.Logger
}

type ChainVerifier struct {
	logger logging.Logger
}

func NewAuditManager(config *AuditConfig) *AuditManager {
	return &AuditManager{
		config:        config,
		auditLogger:   NewAuditLogger(config),
		siemForwarder: NewSIEMForwarder(&SIEMConfig{
			Endpoint:   "https://siem.iaros.com/api/events",
			APIKey:     "siem-api-key",
			BatchSize:  100,
			MaxRetries: 3,
			RetryDelay: 5 * time.Second,
		}),
		chainVerifier: NewChainVerifier(),
		logger:        logging.GetLogger("audit_manager"),
	}
}

func NewAuditLogger(config *AuditConfig) *AuditLogger {
	return &AuditLogger{
		config:     config,
		buffer:     make([]*AuditEvent, 0, config.BufferSize),
		bufferSize: config.BufferSize,
		lastHash:   "genesis",
		logger:     logging.GetLogger("audit_logger"),
	}
}

func NewSIEMForwarder(config *SIEMConfig) *SIEMForwarder {
	return &SIEMForwarder{
		config: config,
		client: &SIEMClient{
			endpoint: config.Endpoint,
			apiKey:   config.APIKey,
			logger:   logging.GetLogger("siem_client"),
		},
		logger: logging.GetLogger("siem_forwarder"),
	}
}

func NewChainVerifier() *ChainVerifier {
	return &ChainVerifier{
		logger: logging.GetLogger("chain_verifier"),
	}
}

// Core audit methods
func (am *AuditManager) LogAuthenticationEvent(userID, sessionID, result, ipAddress, userAgent string, details map[string]interface{}) {
	event := &AuditEvent{
		EventType: "authentication",
		UserID:    userID,
		SessionID: sessionID,
		Resource:  "auth",
		Action:    "login",
		Result:    result,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Details:   details,
	}
	am.logEvent(event)
}

func (am *AuditManager) LogAuthorizationEvent(userID, sessionID, resource, action, result, ipAddress string, details map[string]interface{}) {
	event := &AuditEvent{
		EventType: "authorization",
		UserID:    userID,
		SessionID: sessionID,
		Resource:  resource,
		Action:    action,
		Result:    result,
		IPAddress: ipAddress,
		Details:   details,
	}
	am.logEvent(event)
}

func (am *AuditManager) LogDataAccess(userID, sessionID, resource, action, result string, details map[string]interface{}) {
	event := &AuditEvent{
		EventType: "data_access",
		UserID:    userID,
		SessionID: sessionID,
		Resource:  resource,
		Action:    action,
		Result:    result,
		Details:   details,
	}
	am.logEvent(event)
}

func (am *AuditManager) LogConfigurationChange(userID, sessionID, resource, action string, details map[string]interface{}) {
	event := &AuditEvent{
		EventType: "configuration",
		UserID:    userID,
		SessionID: sessionID,
		Resource:  resource,
		Action:    action,
		Result:    "success",
		Details:   details,
	}
	am.logEvent(event)
}

func (am *AuditManager) LogSecurityEvent(eventType, userID, sessionID, ipAddress string, details map[string]interface{}) {
	event := &AuditEvent{
		EventType: "security",
		UserID:    userID,
		SessionID: sessionID,
		Resource:  "security",
		Action:    eventType,
		Result:    "alert",
		IPAddress: ipAddress,
		Details:   details,
	}
	am.logEvent(event)
}

func (am *AuditManager) logEvent(event *AuditEvent) {
	if !am.config.Enabled {
		return
	}

	// Populate event metadata
	event.ID = am.generateEventID()
	event.Timestamp = time.Now()

	// Create immutable hash chain
	am.auditLogger.AddToChain(event)

	// Forward to SIEM if enabled
	if am.config.SIEMEnabled {
		go am.siemForwarder.ForwardEvent(event)
	}

	am.logger.Debug("Audit event logged", 
		"event_id", event.ID,
		"event_type", event.EventType,
		"user_id", event.UserID,
		"resource", event.Resource,
		"action", event.Action)
}

// Audit Logger implementation
func (al *AuditLogger) AddToChain(event *AuditEvent) {
	al.mu.Lock()
	defer al.mu.Unlock()

	// Set previous hash for chain integrity
	event.PreviousHash = al.lastHash

	// Calculate event hash
	event.Hash = al.calculateEventHash(event)

	// Sign the event (simplified)
	event.Signature = al.signEvent(event)

	// Add to buffer
	al.buffer = append(al.buffer, event)
	al.eventCount++

	// Update last hash
	al.lastHash = event.Hash

	// Flush buffer if full
	if len(al.buffer) >= al.bufferSize {
		al.flushBuffer()
	}

	al.logger.Debug("Event added to audit chain", 
		"event_id", event.ID,
		"hash", event.Hash[:16]+"...",
		"chain_length", al.eventCount)
}

func (al *AuditLogger) calculateEventHash(event *AuditEvent) string {
	// Create deterministic hash of event data
	hashData := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
		event.ID,
		event.Timestamp.Format(time.RFC3339Nano),
		event.EventType,
		event.UserID,
		event.Resource,
		event.Action,
		event.Result,
		event.PreviousHash,
	)

	// Include details in hash
	if detailsJSON, err := json.Marshal(event.Details); err == nil {
		hashData += "|" + string(detailsJSON)
	}

	hash := sha256.Sum256([]byte(hashData))
	return hex.EncodeToString(hash[:])
}

func (al *AuditLogger) signEvent(event *AuditEvent) string {
	// Simplified digital signature - in production use proper PKI
	signatureData := event.Hash + "|iaros-audit-key"
	hash := sha256.Sum256([]byte(signatureData))
	return hex.EncodeToString(hash[:16])
}

func (al *AuditLogger) flushBuffer() {
	if len(al.buffer) == 0 {
		return
	}

	// In production, write to immutable storage (blockchain, append-only log)
	al.logger.Info("Flushing audit buffer", "events", len(al.buffer))

	// Process each event for long-term storage
	for _, event := range al.buffer {
		al.persistEvent(event)
	}

	// Clear buffer
	al.buffer = al.buffer[:0]
}

func (al *AuditLogger) persistEvent(event *AuditEvent) {
	// In production, persist to immutable storage
	al.logger.Debug("Persisting audit event", "event_id", event.ID)
}

// SIEM Forwarder implementation
func (sf *SIEMForwarder) ForwardEvent(event *AuditEvent) {
	if err := sf.client.SendEvent(event); err != nil {
		sf.logger.Error("Failed to forward event to SIEM", 
			"event_id", event.ID,
			"error", err)
		
		// Implement retry logic
		sf.retryEvent(event)
	} else {
		sf.logger.Debug("Event forwarded to SIEM", "event_id", event.ID)
	}
}

func (sf *SIEMForwarder) retryEvent(event *AuditEvent) {
	for attempt := 1; attempt <= sf.config.MaxRetries; attempt++ {
		time.Sleep(sf.config.RetryDelay * time.Duration(attempt))
		
		if err := sf.client.SendEvent(event); err == nil {
			sf.logger.Info("Event forwarded to SIEM after retry", 
				"event_id", event.ID,
				"attempt", attempt)
			return
		}
	}
	
	sf.logger.Error("Failed to forward event to SIEM after retries", 
		"event_id", event.ID,
		"max_attempts", sf.config.MaxRetries)
}

func (sc *SIEMClient) SendEvent(event *AuditEvent) error {
	// Convert to SIEM format
	siemEvent := sc.convertToSIEMFormat(event)
	
	// In production, send HTTP request to SIEM endpoint
	sc.logger.Debug("Sending event to SIEM", 
		"endpoint", sc.endpoint,
		"event_id", event.ID)
	
	return nil // Placeholder
}

func (sc *SIEMClient) convertToSIEMFormat(event *AuditEvent) map[string]interface{} {
	return map[string]interface{}{
		"timestamp":    event.Timestamp.Format(time.RFC3339),
		"event_type":   event.EventType,
		"user_id":      event.UserID,
		"session_id":   event.SessionID,
		"source_ip":    event.IPAddress,
		"resource":     event.Resource,
		"action":       event.Action,
		"result":       event.Result,
		"user_agent":   event.UserAgent,
		"details":      event.Details,
		"audit_hash":   event.Hash,
		"signature":    event.Signature,
		"source":       "iaros-audit",
	}
}

// Chain Verifier implementation
func (cv *ChainVerifier) VerifyChain(events []*AuditEvent) (bool, error) {
	if len(events) == 0 {
		return true, nil
	}

	for i, event := range events {
		// Verify event hash
		expectedHash := cv.calculateEventHash(event)
		if event.Hash != expectedHash {
			cv.logger.Error("Hash verification failed", 
				"event_id", event.ID,
				"expected", expectedHash,
				"actual", event.Hash)
			return false, fmt.Errorf("hash verification failed for event %s", event.ID)
		}

		// Verify chain linkage
		if i > 0 {
			previousEvent := events[i-1]
			if event.PreviousHash != previousEvent.Hash {
				cv.logger.Error("Chain verification failed", 
					"event_id", event.ID,
					"expected_previous", previousEvent.Hash,
					"actual_previous", event.PreviousHash)
				return false, fmt.Errorf("chain verification failed for event %s", event.ID)
			}
		}

		// Verify signature
		if !cv.verifySignature(event) {
			cv.logger.Error("Signature verification failed", "event_id", event.ID)
			return false, fmt.Errorf("signature verification failed for event %s", event.ID)
		}
	}

	cv.logger.Info("Chain verification successful", "events", len(events))
	return true, nil
}

func (cv *ChainVerifier) calculateEventHash(event *AuditEvent) string {
	// Same logic as AuditLogger.calculateEventHash
	hashData := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
		event.ID,
		event.Timestamp.Format(time.RFC3339Nano),
		event.EventType,
		event.UserID,
		event.Resource,
		event.Action,
		event.Result,
		event.PreviousHash,
	)

	if detailsJSON, err := json.Marshal(event.Details); err == nil {
		hashData += "|" + string(detailsJSON)
	}

	hash := sha256.Sum256([]byte(hashData))
	return hex.EncodeToString(hash[:])
}

func (cv *ChainVerifier) verifySignature(event *AuditEvent) bool {
	// Simplified signature verification
	expectedSignature := cv.calculateSignature(event.Hash)
	return event.Signature == expectedSignature
}

func (cv *ChainVerifier) calculateSignature(hash string) string {
	signatureData := hash + "|iaros-audit-key"
	signatureHash := sha256.Sum256([]byte(signatureData))
	return hex.EncodeToString(signatureHash[:16])
}

// Query and reporting methods
func (am *AuditManager) QueryEvents(query *AuditQuery) ([]*AuditEvent, error) {
	// Implementation would query audit storage
	am.logger.Info("Querying audit events", 
		"start_time", query.StartTime,
		"end_time", query.EndTime,
		"event_type", query.EventType,
		"user_id", query.UserID)
	
	return []*AuditEvent{}, nil // Placeholder
}

func (am *AuditManager) GenerateComplianceReport(startTime, endTime time.Time) (*ComplianceReport, error) {
	report := &ComplianceReport{
		StartTime:     startTime,
		EndTime:       endTime,
		GeneratedAt:   time.Now(),
		TotalEvents:   0,
		EventsByType:  make(map[string]int),
		FailedEvents:  make([]*AuditEvent, 0),
		Compliance:    true,
	}

	am.logger.Info("Generated compliance report", 
		"start_time", startTime,
		"end_time", endTime,
		"total_events", report.TotalEvents)

	return report, nil
}

// Utility functions
func (am *AuditManager) generateEventID() string {
	timestamp := time.Now().UnixNano()
	hash := sha256.Sum256([]byte(fmt.Sprintf("audit_%d", timestamp)))
	return hex.EncodeToString(hash[:8])
}

// Helper types
type AuditQuery struct {
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	EventType   string    `json:"event_type"`
	UserID      string    `json:"user_id"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	IPAddress   string    `json:"ip_address"`
	Limit       int       `json:"limit"`
	Offset      int       `json:"offset"`
}

type ComplianceReport struct {
	StartTime    time.Time            `json:"start_time"`
	EndTime      time.Time            `json:"end_time"`
	GeneratedAt  time.Time            `json:"generated_at"`
	TotalEvents  int                  `json:"total_events"`
	EventsByType map[string]int       `json:"events_by_type"`
	FailedEvents []*AuditEvent        `json:"failed_events"`
	Compliance   bool                 `json:"compliance"`
	Issues       []string             `json:"issues"`
} 