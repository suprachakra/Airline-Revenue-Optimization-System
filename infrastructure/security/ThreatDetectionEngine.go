package security

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/iaros/common/logging"
)

// ThreatDetectionEngine provides comprehensive threat detection and mitigation
type ThreatDetectionEngine struct {
	config           *ThreatDetectionConfig
	ratelimiter     *RateLimiter
	ddosProtection  *DDoSProtection
	anomalyDetector *AnomalyDetector
	blacklistManager *BlacklistManager
	alertManager    *ThreatAlertManager
	logger          logging.Logger
	mu              sync.RWMutex
}

type ThreatDetectionConfig struct {
	Enabled              bool          `json:"enabled"`
	RateLimitingEnabled  bool          `json:"rate_limiting_enabled"`
	DDoSProtectionEnabled bool         `json:"ddos_protection_enabled"`
	AnomalyDetectionEnabled bool       `json:"anomaly_detection_enabled"`
	BlacklistEnabled     bool          `json:"blacklist_enabled"`
	AlertingEnabled      bool          `json:"alerting_enabled"`
	ScanInterval         time.Duration `json:"scan_interval"`
}

type RateLimiter struct {
	limits    map[string]*RateLimit
	buckets   map[string]*TokenBucket
	config    *RateLimitConfig
	logger    logging.Logger
	mu        sync.RWMutex
}

type RateLimit struct {
	Type        string        `json:"type"`        // ip, user, api_key
	Resource    string        `json:"resource"`    // endpoint pattern
	Limit       int           `json:"limit"`       // requests per window
	Window      time.Duration `json:"window"`      // time window
	BurstLimit  int           `json:"burst_limit"` // burst allowance
}

type TokenBucket struct {
	Tokens      float64   `json:"tokens"`
	MaxTokens   float64   `json:"max_tokens"`
	RefillRate  float64   `json:"refill_rate"`
	LastRefill  time.Time `json:"last_refill"`
}

type RateLimitConfig struct {
	DefaultLimits map[string]*RateLimit `json:"default_limits"`
	CustomLimits  map[string]*RateLimit `json:"custom_limits"`
}

type DDoSProtection struct {
	thresholds  map[string]*DDoSThreshold
	connections map[string]*ConnectionTracker
	logger      logging.Logger
	mu          sync.RWMutex
}

type DDoSThreshold struct {
	ConnectionsPerIP     int           `json:"connections_per_ip"`
	RequestsPerSecond    int           `json:"requests_per_second"`
	BandwidthPerIP       int64         `json:"bandwidth_per_ip"`
	TimeWindow           time.Duration `json:"time_window"`
	BlockDuration        time.Duration `json:"block_duration"`
}

type ConnectionTracker struct {
	IP               string    `json:"ip"`
	ConnectionCount  int       `json:"connection_count"`
	RequestCount     int       `json:"request_count"`
	Bandwidth        int64     `json:"bandwidth"`
	FirstSeen        time.Time `json:"first_seen"`
	LastSeen         time.Time `json:"last_seen"`
	Blocked          bool      `json:"blocked"`
	BlockedUntil     time.Time `json:"blocked_until"`
}

type AnomalyDetector struct {
	patterns      map[string]*Pattern
	baselines     map[string]*Baseline
	detectors     map[string]*Detector
	logger        logging.Logger
	mu            sync.RWMutex
}

type Pattern struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`        // statistical, ml, rule_based
	Parameters  map[string]interface{} `json:"parameters"`
	Threshold   float64                `json:"threshold"`
	Enabled     bool                   `json:"enabled"`
}

type Baseline struct {
	Metric      string    `json:"metric"`
	Mean        float64   `json:"mean"`
	StdDev      float64   `json:"std_dev"`
	Min         float64   `json:"min"`
	Max         float64   `json:"max"`
	SampleCount int       `json:"sample_count"`
	LastUpdate  time.Time `json:"last_update"`
}

type Detector struct {
	Name           string                   `json:"name"`
	Type           string                   `json:"type"`
	Config         map[string]interface{}   `json:"config"`
	Anomalies      []*DetectedAnomaly       `json:"anomalies"`
	LastRun        time.Time                `json:"last_run"`
}

type DetectedAnomaly struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Metrics     map[string]interface{} `json:"metrics"`
	DetectedAt  time.Time              `json:"detected_at"`
	Source      string                 `json:"source"`
}

type BlacklistManager struct {
	ipBlacklist     map[string]*BlacklistEntry
	userBlacklist   map[string]*BlacklistEntry
	domainBlacklist map[string]*BlacklistEntry
	logger          logging.Logger
	mu              sync.RWMutex
}

type BlacklistEntry struct {
	Value       string            `json:"value"`
	Type        string            `json:"type"` // ip, user, domain
	Reason      string            `json:"reason"`
	Severity    string            `json:"severity"`
	AddedAt     time.Time         `json:"added_at"`
	ExpiresAt   *time.Time        `json:"expires_at"`
	Metadata    map[string]string `json:"metadata"`
	Active      bool              `json:"active"`
}

type ThreatAlertManager struct {
	alerts     []*ThreatAlert
	handlers   map[string]AlertHandler
	logger     logging.Logger
	mu         sync.RWMutex
}

type ThreatAlert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Source      string                 `json:"source"`
	Description string                 `json:"description"`
	Metrics     map[string]interface{} `json:"metrics"`
	Actions     []string               `json:"actions"`
	CreatedAt   time.Time              `json:"created_at"`
	Resolved    bool                   `json:"resolved"`
	ResolvedAt  *time.Time             `json:"resolved_at"`
}

type AlertHandler interface {
	HandleAlert(alert *ThreatAlert) error
}

func NewThreatDetectionEngine(config *ThreatDetectionConfig) *ThreatDetectionEngine {
	return &ThreatDetectionEngine{
		config:           config,
		rateimiter:      NewRateLimiter(),
		ddosProtection:  NewDDoSProtection(),
		anomalyDetector: NewAnomalyDetector(),
		blacklistManager: NewBlacklistManager(),
		alertManager:    NewThreatAlertManager(),
		logger:          logging.GetLogger("threat_detection"),
	}
}

func NewRateLimiter() *RateLimiter {
	defaultLimits := map[string]*RateLimit{
		"api_general": {
			Type:       "ip",
			Resource:   "/api/*",
			Limit:      1000,
			Window:     time.Hour,
			BurstLimit: 100,
		},
		"auth_login": {
			Type:       "ip",
			Resource:   "/auth/login",
			Limit:      10,
			Window:     time.Minute,
			BurstLimit: 5,
		},
		"pricing_api": {
			Type:       "user",
			Resource:   "/api/pricing/*",
			Limit:      500,
			Window:     time.Hour,
			BurstLimit: 50,
		},
	}

	return &RateLimiter{
		limits:  defaultLimits,
		buckets: make(map[string]*TokenBucket),
		config: &RateLimitConfig{
			DefaultLimits: defaultLimits,
		},
		logger: logging.GetLogger("rate_limiter"),
	}
}

func NewDDoSProtection() *DDoSProtection {
	thresholds := map[string]*DDoSThreshold{
		"default": {
			ConnectionsPerIP:  100,
			RequestsPerSecond: 50,
			BandwidthPerIP:    10 * 1024 * 1024, // 10MB
			TimeWindow:        time.Minute,
			BlockDuration:     15 * time.Minute,
		},
		"strict": {
			ConnectionsPerIP:  20,
			RequestsPerSecond: 10,
			BandwidthPerIP:    1 * 1024 * 1024, // 1MB
			TimeWindow:        time.Minute,
			BlockDuration:     30 * time.Minute,
		},
	}

	return &DDoSProtection{
		thresholds:  thresholds,
		connections: make(map[string]*ConnectionTracker),
		logger:      logging.GetLogger("ddos_protection"),
	}
}

func NewAnomalyDetector() *AnomalyDetector {
	patterns := map[string]*Pattern{
		"login_frequency": {
			Name:      "Unusual Login Frequency",
			Type:      "statistical",
			Threshold: 3.0, // 3 standard deviations
			Enabled:   true,
		},
		"api_usage": {
			Name:      "Abnormal API Usage",
			Type:      "statistical",
			Threshold: 2.5,
			Enabled:   true,
		},
		"geographic_anomaly": {
			Name:      "Geographic Anomaly",
			Type:      "rule_based",
			Threshold: 1.0,
			Enabled:   true,
		},
	}

	return &AnomalyDetector{
		patterns:  patterns,
		baselines: make(map[string]*Baseline),
		detectors: make(map[string]*Detector),
		logger:    logging.GetLogger("anomaly_detector"),
	}
}

func NewBlacklistManager() *BlacklistManager {
	return &BlacklistManager{
		ipBlacklist:     make(map[string]*BlacklistEntry),
		userBlacklist:   make(map[string]*BlacklistEntry),
		domainBlacklist: make(map[string]*BlacklistEntry),
		logger:          logging.GetLogger("blacklist_manager"),
	}
}

func NewThreatAlertManager() *ThreatAlertManager {
	return &ThreatAlertManager{
		alerts:   make([]*ThreatAlert, 0),
		handlers: make(map[string]AlertHandler),
		logger:   logging.GetLogger("threat_alert_manager"),
	}
}

// Main threat detection methods
func (tde *ThreatDetectionEngine) AnalyzeRequest(ctx context.Context, req *ThreatAnalysisRequest) (*ThreatAnalysisResult, error) {
	result := &ThreatAnalysisResult{
		Allowed:    true,
		Threats:    make([]string, 0),
		Actions:    make([]string, 0),
		Score:      0.0,
		Details:    make(map[string]interface{}),
	}

	// Rate limiting check
	if tde.config.RateLimitingEnabled {
		if rateLimited, err := tde.rateimiter.CheckLimit(req.ClientIP, req.Endpoint, req.UserID); err != nil {
			result.Threats = append(result.Threats, "rate_limit_check_failed")
		} else if rateLimited {
			result.Allowed = false
			result.Threats = append(result.Threats, "rate_limited")
			result.Actions = append(result.Actions, "block_request")
			result.Score += 5.0
		}
	}

	// DDoS protection check
	if tde.config.DDoSProtectionEnabled {
		if blocked, err := tde.ddosProtection.CheckIP(req.ClientIP); err != nil {
			result.Threats = append(result.Threats, "ddos_check_failed")
		} else if blocked {
			result.Allowed = false
			result.Threats = append(result.Threats, "ddos_detected")
			result.Actions = append(result.Actions, "block_ip")
			result.Score += 10.0
		}
	}

	// Blacklist check
	if tde.config.BlacklistEnabled {
		if blacklisted := tde.blacklistManager.IsBlacklisted(req.ClientIP, req.UserID, ""); blacklisted {
			result.Allowed = false
			result.Threats = append(result.Threats, "blacklisted")
			result.Actions = append(result.Actions, "block_request")
			result.Score += 8.0
		}
	}

	// Anomaly detection
	if tde.config.AnomalyDetectionEnabled {
		anomalies := tde.anomalyDetector.DetectAnomalies(req)
		for _, anomaly := range anomalies {
			result.Threats = append(result.Threats, "anomaly_"+anomaly.Type)
			result.Score += 3.0
			if anomaly.Severity == "high" {
				result.Score += 5.0
			}
		}
	}

	// Generate alerts for high-score threats
	if result.Score >= 8.0 {
		go tde.alertManager.CreateAlert(&ThreatAlert{
			Type:        "high_threat_score",
			Severity:    "high",
			Source:      req.ClientIP,
			Description: fmt.Sprintf("High threat score detected: %.1f", result.Score),
			Metrics: map[string]interface{}{
				"score":      result.Score,
				"threats":    result.Threats,
				"client_ip":  req.ClientIP,
				"endpoint":   req.Endpoint,
				"user_id":    req.UserID,
			},
			CreatedAt: time.Now(),
		})
	}

	tde.logger.Debug("Threat analysis completed", 
		"client_ip", req.ClientIP,
		"allowed", result.Allowed,
		"score", result.Score,
		"threats", len(result.Threats))

	return result, nil
}

// Rate Limiter implementation
func (rl *RateLimiter) CheckLimit(clientIP, endpoint, userID string) (bool, error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Find applicable rate limit
	var limit *RateLimit
	for _, l := range rl.limits {
		if rl.matchesEndpoint(endpoint, l.Resource) {
			limit = l
			break
		}
	}

	if limit == nil {
		return false, nil // No limit configured
	}

	// Create bucket key based on limit type
	var bucketKey string
	switch limit.Type {
	case "ip":
		bucketKey = fmt.Sprintf("ip_%s_%s", clientIP, limit.Resource)
	case "user":
		bucketKey = fmt.Sprintf("user_%s_%s", userID, limit.Resource)
	default:
		bucketKey = fmt.Sprintf("default_%s_%s", clientIP, limit.Resource)
	}

	// Get or create token bucket
	bucket, exists := rl.buckets[bucketKey]
	if !exists {
		bucket = &TokenBucket{
			Tokens:     float64(limit.BurstLimit),
			MaxTokens:  float64(limit.Limit),
			RefillRate: float64(limit.Limit) / limit.Window.Seconds(),
			LastRefill: time.Now(),
		}
		rl.buckets[bucketKey] = bucket
	}

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(bucket.LastRefill).Seconds()
	bucket.Tokens = min(bucket.MaxTokens, bucket.Tokens+bucket.RefillRate*elapsed)
	bucket.LastRefill = now

	// Check if request can be allowed
	if bucket.Tokens >= 1.0 {
		bucket.Tokens--
		return false, nil // Request allowed
	}

	rl.logger.Warn("Rate limit exceeded", 
		"bucket_key", bucketKey,
		"tokens", bucket.Tokens,
		"limit", limit.Limit)

	return true, nil // Rate limited
}

func (rl *RateLimiter) matchesEndpoint(endpoint, pattern string) bool {
	// Simplified pattern matching - in production use proper regex
	if pattern == "*" || pattern == endpoint {
		return true
	}
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(endpoint) >= len(prefix) && endpoint[:len(prefix)] == prefix
	}
	return false
}

// DDoS Protection implementation
func (ddos *DDoSProtection) CheckIP(clientIP string) (bool, error) {
	ddos.mu.Lock()
	defer ddos.mu.Unlock()

	tracker, exists := ddos.connections[clientIP]
	if !exists {
		tracker = &ConnectionTracker{
			IP:        clientIP,
			FirstSeen: time.Now(),
		}
		ddos.connections[clientIP] = tracker
	}

	tracker.LastSeen = time.Now()
	tracker.RequestCount++

	// Check if currently blocked
	if tracker.Blocked && time.Now().Before(tracker.BlockedUntil) {
		return true, nil
	}

	// Reset block status if expired
	if tracker.Blocked && time.Now().After(tracker.BlockedUntil) {
		tracker.Blocked = false
		tracker.RequestCount = 0
		tracker.ConnectionCount = 0
		tracker.Bandwidth = 0
	}

	// Get threshold
	threshold := ddos.thresholds["default"]

	// Check thresholds
	timeWindow := time.Now().Add(-threshold.TimeWindow)
	if tracker.FirstSeen.After(timeWindow) {
		requestsPerSecond := float64(tracker.RequestCount) / time.Since(tracker.FirstSeen).Seconds()
		
		if tracker.ConnectionCount > threshold.ConnectionsPerIP ||
		   int(requestsPerSecond) > threshold.RequestsPerSecond ||
		   tracker.Bandwidth > threshold.BandwidthPerIP {
			
			// Block the IP
			tracker.Blocked = true
			tracker.BlockedUntil = time.Now().Add(threshold.BlockDuration)
			
			ddos.logger.Warn("IP blocked for DDoS", 
				"ip", clientIP,
				"connections", tracker.ConnectionCount,
				"requests_per_second", requestsPerSecond,
				"bandwidth", tracker.Bandwidth)
			
			return true, nil
		}
	}

	return false, nil
}

// Anomaly Detector implementation
func (ad *AnomalyDetector) DetectAnomalies(req *ThreatAnalysisRequest) []*DetectedAnomaly {
	anomalies := make([]*DetectedAnomaly, 0)

	// Geographic anomaly detection
	if pattern, exists := ad.patterns["geographic_anomaly"]; exists && pattern.Enabled {
		if anomaly := ad.detectGeographicAnomaly(req); anomaly != nil {
			anomalies = append(anomalies, anomaly)
		}
	}

	// Time-based anomaly detection
	if anomaly := ad.detectTimeAnomaly(req); anomaly != nil {
		anomalies = append(anomalies, anomaly)
	}

	return anomalies
}

func (ad *AnomalyDetector) detectGeographicAnomaly(req *ThreatAnalysisRequest) *DetectedAnomaly {
	// Simplified geographic anomaly detection
	// In production, use GeoIP services and user location history
	
	clientIP := net.ParseIP(req.ClientIP)
	if clientIP == nil {
		return nil
	}

	// Check for known suspicious IP ranges
	suspiciousRanges := []string{
		"10.0.0.0/8",    // Private networks trying to access from outside
		"172.16.0.0/12", // Private networks
		"192.168.0.0/16", // Private networks
	}

	for _, cidr := range suspiciousRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil && network.Contains(clientIP) {
			return &DetectedAnomaly{
				ID:          ad.generateAnomalyID(),
				Type:        "geographic",
				Severity:    "medium",
				Description: "Request from suspicious IP range",
				Metrics: map[string]interface{}{
					"client_ip": req.ClientIP,
					"range":     cidr,
				},
				DetectedAt: time.Now(),
				Source:     "geographic_detector",
			}
		}
	}

	return nil
}

func (ad *AnomalyDetector) detectTimeAnomaly(req *ThreatAnalysisRequest) *DetectedAnomaly {
	// Check for requests outside business hours
	now := time.Now()
	hour := now.Hour()

	// Anomalous if outside 6 AM - 10 PM
	if hour < 6 || hour > 22 {
		return &DetectedAnomaly{
			ID:          ad.generateAnomalyID(),
			Type:        "temporal",
			Severity:    "low",
			Description: "Request outside normal business hours",
			Metrics: map[string]interface{}{
				"hour":      hour,
				"timestamp": now,
			},
			DetectedAt: time.Now(),
			Source:     "time_detector",
		}
	}

	return nil
}

func (ad *AnomalyDetector) generateAnomalyID() string {
	return fmt.Sprintf("anomaly_%d", time.Now().UnixNano())
}

// Blacklist Manager implementation
func (bm *BlacklistManager) IsBlacklisted(ip, userID, domain string) bool {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	// Check IP blacklist
	if ip != "" {
		if entry, exists := bm.ipBlacklist[ip]; exists && entry.Active {
			if entry.ExpiresAt == nil || time.Now().Before(*entry.ExpiresAt) {
				return true
			}
		}
	}

	// Check user blacklist
	if userID != "" {
		if entry, exists := bm.userBlacklist[userID]; exists && entry.Active {
			if entry.ExpiresAt == nil || time.Now().Before(*entry.ExpiresAt) {
				return true
			}
		}
	}

	// Check domain blacklist
	if domain != "" {
		if entry, exists := bm.domainBlacklist[domain]; exists && entry.Active {
			if entry.ExpiresAt == nil || time.Now().Before(*entry.ExpiresAt) {
				return true
			}
		}
	}

	return false
}

func (bm *BlacklistManager) AddToBlacklist(value, entryType, reason, severity string, duration *time.Duration) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	var expiresAt *time.Time
	if duration != nil {
		expiry := time.Now().Add(*duration)
		expiresAt = &expiry
	}

	entry := &BlacklistEntry{
		Value:     value,
		Type:      entryType,
		Reason:    reason,
		Severity:  severity,
		AddedAt:   time.Now(),
		ExpiresAt: expiresAt,
		Active:    true,
	}

	switch entryType {
	case "ip":
		bm.ipBlacklist[value] = entry
	case "user":
		bm.userBlacklist[value] = entry
	case "domain":
		bm.domainBlacklist[value] = entry
	}

	bm.logger.Info("Added to blacklist", 
		"type", entryType,
		"value", value,
		"reason", reason)
}

// Helper types and functions
type ThreatAnalysisRequest struct {
	ClientIP    string            `json:"client_ip"`
	UserID      string            `json:"user_id"`
	SessionID   string            `json:"session_id"`
	Endpoint    string            `json:"endpoint"`
	Method      string            `json:"method"`
	UserAgent   string            `json:"user_agent"`
	Headers     map[string]string `json:"headers"`
	Timestamp   time.Time         `json:"timestamp"`
}

type ThreatAnalysisResult struct {
	Allowed bool                   `json:"allowed"`
	Threats []string               `json:"threats"`
	Actions []string               `json:"actions"`
	Score   float64                `json:"score"`
	Details map[string]interface{} `json:"details"`
}

func (tam *ThreatAlertManager) CreateAlert(alert *ThreatAlert) {
	tam.mu.Lock()
	defer tam.mu.Unlock()

	alert.ID = tam.generateAlertID()
	tam.alerts = append(tam.alerts, alert)

	tam.logger.Warn("Threat alert created", 
		"id", alert.ID,
		"type", alert.Type,
		"severity", alert.Severity,
		"source", alert.Source)

	// Execute alert handlers
	for name, handler := range tam.handlers {
		go func(name string, handler AlertHandler) {
			if err := handler.HandleAlert(alert); err != nil {
				tam.logger.Error("Alert handler failed", 
					"handler", name,
					"alert_id", alert.ID,
					"error", err)
			}
		}(name, handler)
	}
}

func (tam *ThreatAlertManager) generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
} 