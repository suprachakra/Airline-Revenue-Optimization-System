package performance

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/iaros/common/logging"
	"github.com/iaros/common/metrics"
)

// Request Optimizer Configuration
type RequestOptimizerConfig struct {
	CompressionEnabled bool          `json:"compression_enabled"`
	CompressionLevel   int           `json:"compression_level"`
	MinCompressSize    int           `json:"min_compress_size"`
	KeepAliveEnabled   bool          `json:"keep_alive_enabled"`
	PipeliningEnabled  bool          `json:"pipelining_enabled"`
	MaxConnections     int           `json:"max_connections"`
}

type RequestOptimizer struct {
	config *RequestOptimizerConfig
	logger logging.Logger
	stats  *RequestStats
	mu     sync.RWMutex
}

type RequestStats struct {
	TotalRequests     int64         `json:"total_requests"`
	CompressedRequests int64        `json:"compressed_requests"`
	AverageSize       float64       `json:"average_size"`
	CompressionRatio  float64       `json:"compression_ratio"`
}

func NewRequestOptimizer(config *RequestOptimizerConfig) *RequestOptimizer {
	return &RequestOptimizer{
		config: config,
		logger: logging.GetLogger("request_optimizer"),
		stats:  &RequestStats{},
	}
}

func (ro *RequestOptimizer) OptimizeRequest(req *http.Request) *http.Request {
	ro.mu.Lock()
	defer ro.mu.Unlock()
	
	ro.stats.TotalRequests++
	
	// Add compression headers if enabled
	if ro.config.CompressionEnabled {
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	}
	
	// Enable keep-alive
	if ro.config.KeepAliveEnabled {
		req.Header.Set("Connection", "keep-alive")
	}
	
	ro.logger.Debug("Request optimized", "url", req.URL.String())
	return req
}

// Response Optimizer Configuration
type ResponseOptimizerConfig struct {
	CompressionEnabled  bool `json:"compression_enabled"`
	MinificationEnabled bool `json:"minification_enabled"`
	GzipEnabled         bool `json:"gzip_enabled"`
	BrotliEnabled       bool `json:"brotli_enabled"`
	CompressionLevel    int  `json:"compression_level"`
}

type ResponseOptimizer struct {
	config *ResponseOptimizerConfig
	logger logging.Logger
	stats  *ResponseStats
	mu     sync.RWMutex
}

type ResponseStats struct {
	TotalResponses    int64   `json:"total_responses"`
	CompressedResponses int64 `json:"compressed_responses"`
	AverageSize       float64 `json:"average_size"`
	CompressionSavings float64 `json:"compression_savings"`
}

func NewResponseOptimizer(config *ResponseOptimizerConfig) *ResponseOptimizer {
	return &ResponseOptimizer{
		config: config,
		logger: logging.GetLogger("response_optimizer"),
		stats:  &ResponseStats{},
	}
}

func (ro *ResponseOptimizer) OptimizeResponse(w http.ResponseWriter, data []byte) ([]byte, error) {
	ro.mu.Lock()
	defer ro.mu.Unlock()
	
	ro.stats.TotalResponses++
	originalSize := len(data)
	
	// Apply compression if enabled and beneficial
	if ro.config.CompressionEnabled && originalSize > 1024 {
		compressed, err := ro.compressData(data)
		if err == nil && len(compressed) < originalSize {
			w.Header().Set("Content-Encoding", "gzip")
			ro.stats.CompressedResponses++
			ro.stats.CompressionSavings += float64(originalSize-len(compressed)) / float64(originalSize)
			ro.logger.Debug("Response compressed", "original", originalSize, "compressed", len(compressed))
			return compressed, nil
		}
	}
	
	return data, nil
}

func (ro *ResponseOptimizer) compressData(data []byte) ([]byte, error) {
	var compressed []byte
	// Implementation would use gzip/brotli compression
	return compressed, nil
}

// Network Optimizer Configuration
type NetworkOptimizerConfig struct {
	TCPNoDelay                  bool          `json:"tcp_no_delay"`
	KeepAliveEnabled            bool          `json:"keep_alive_enabled"`
	KeepAliveInterval           time.Duration `json:"keep_alive_interval"`
	MaxIdleConnections          int           `json:"max_idle_connections"`
	MaxIdleConnectionsPerHost   int           `json:"max_idle_connections_per_host"`
	IdleConnTimeout             time.Duration `json:"idle_conn_timeout"`
	TLSHandshakeTimeout         time.Duration `json:"tls_handshake_timeout"`
	ExpectContinueTimeout       time.Duration `json:"expect_continue_timeout"`
}

type NetworkOptimizer struct {
	config    *NetworkOptimizerConfig
	transport *http.Transport
	logger    logging.Logger
	stats     *NetworkStats
	mu        sync.RWMutex
}

type NetworkStats struct {
	ActiveConnections    int           `json:"active_connections"`
	IdleConnections      int           `json:"idle_connections"`
	AverageLatency       time.Duration `json:"average_latency"`
	ConnectionsCreated   int64         `json:"connections_created"`
	ConnectionsReused    int64         `json:"connections_reused"`
	DNSLookupTime        time.Duration `json:"dns_lookup_time"`
	TCPConnectTime       time.Duration `json:"tcp_connect_time"`
	TLSHandshakeTime     time.Duration `json:"tls_handshake_time"`
}

func NewNetworkOptimizer(config *NetworkOptimizerConfig) *NetworkOptimizer {
	transport := &http.Transport{
		MaxIdleConns:          config.MaxIdleConnections,
		MaxIdleConnsPerHost:   config.MaxIdleConnectionsPerHost,
		IdleConnTimeout:       config.IdleConnTimeout,
		TLSHandshakeTimeout:   config.TLSHandshakeTimeout,
		ExpectContinueTimeout: config.ExpectContinueTimeout,
		DisableKeepAlives:     !config.KeepAliveEnabled,
	}
	
	return &NetworkOptimizer{
		config:    config,
		transport: transport,
		logger:    logging.GetLogger("network_optimizer"),
		stats:     &NetworkStats{},
	}
}

func (no *NetworkOptimizer) GetOptimizedTransport() *http.Transport {
	return no.transport
}

func (no *NetworkOptimizer) RecordConnectionMetrics(latency time.Duration, reused bool) {
	no.mu.Lock()
	defer no.mu.Unlock()
	
	if reused {
		no.stats.ConnectionsReused++
	} else {
		no.stats.ConnectionsCreated++
	}
	
	// Update average latency using exponential moving average
	if no.stats.AverageLatency == 0 {
		no.stats.AverageLatency = latency
	} else {
		alpha := 0.1 // Smoothing factor
		no.stats.AverageLatency = time.Duration(float64(no.stats.AverageLatency)*(1-alpha) + float64(latency)*alpha)
	}
}

// Database Optimizer Configuration
type DatabaseOptimizerConfig struct {
	ConnectionPoolSize    int           `json:"connection_pool_size"`
	MaxConnectionLifetime time.Duration `json:"max_connection_lifetime"`
	QueryTimeout          time.Duration `json:"query_timeout"`
	PreparedStatements    bool          `json:"prepared_statements"`
	ReadOnlyReplicas      []string      `json:"read_only_replicas"`
	BatchSize             int           `json:"batch_size"`
}

type DatabaseOptimizer struct {
	config         *DatabaseOptimizerConfig
	connectionPool *ConnectionPool
	queryCache     *QueryCache
	logger         logging.Logger
	stats          *DatabaseStats
	mu             sync.RWMutex
}

type DatabaseStats struct {
	ActiveConnections    int           `json:"active_connections"`
	IdleConnections      int           `json:"idle_connections"`
	QueriesExecuted      int64         `json:"queries_executed"`
	CacheHits            int64         `json:"cache_hits"`
	CacheMisses          int64         `json:"cache_misses"`
	AverageQueryTime     time.Duration `json:"average_query_time"`
	SlowQueries          int64         `json:"slow_queries"`
}

type ConnectionPool struct {
	connections []interface{} // Database connections
	available   chan interface{}
	inUse       map[interface{}]bool
	mu          sync.RWMutex
}

type QueryCache struct {
	cache map[string]*CachedQuery
	mu    sync.RWMutex
}

type CachedQuery struct {
	SQL       string        `json:"sql"`
	Result    interface{}   `json:"result"`
	ExpiresAt time.Time     `json:"expires_at"`
	HitCount  int           `json:"hit_count"`
}

func NewDatabaseOptimizer(config *DatabaseOptimizerConfig) *DatabaseOptimizer {
	return &DatabaseOptimizer{
		config:         config,
		connectionPool: NewConnectionPool(config.ConnectionPoolSize),
		queryCache:     NewQueryCache(),
		logger:         logging.GetLogger("database_optimizer"),
		stats:          &DatabaseStats{},
	}
}

func NewConnectionPool(size int) *ConnectionPool {
	return &ConnectionPool{
		connections: make([]interface{}, 0, size),
		available:   make(chan interface{}, size),
		inUse:       make(map[interface{}]bool),
	}
}

func NewQueryCache() *QueryCache {
	return &QueryCache{
		cache: make(map[string]*CachedQuery),
	}
}

func (do *DatabaseOptimizer) OptimizeQuery(sql string) (interface{}, bool) {
	// Check query cache first
	if result, found := do.queryCache.Get(sql); found {
		do.mu.Lock()
		do.stats.CacheHits++
		do.mu.Unlock()
		return result, true
	}
	
	do.mu.Lock()
	do.stats.CacheMisses++
	do.mu.Unlock()
	
	return nil, false
}

func (qc *QueryCache) Get(sql string) (interface{}, bool) {
	qc.mu.RLock()
	defer qc.mu.RUnlock()
	
	cached, exists := qc.cache[sql]
	if !exists {
		return nil, false
	}
	
	if time.Now().After(cached.ExpiresAt) {
		delete(qc.cache, sql)
		return nil, false
	}
	
	cached.HitCount++
	return cached.Result, true
}

func (qc *QueryCache) Set(sql string, result interface{}, ttl time.Duration) {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	
	qc.cache[sql] = &CachedQuery{
		SQL:       sql,
		Result:    result,
		ExpiresAt: time.Now().Add(ttl),
		HitCount:  0,
	}
}

// Latency Tracker
type LatencyTracker struct {
	measurements []LatencyMeasurement
	logger       logging.Logger
	mu           sync.RWMutex
}

type LatencyMeasurement struct {
	Timestamp   time.Time     `json:"timestamp"`
	Endpoint    string        `json:"endpoint"`
	Method      string        `json:"method"`
	Latency     time.Duration `json:"latency"`
	StatusCode  int           `json:"status_code"`
	Component   string        `json:"component"`
}

func NewLatencyTracker() *LatencyTracker {
	return &LatencyTracker{
		measurements: make([]LatencyMeasurement, 0),
		logger:       logging.GetLogger("latency_tracker"),
	}
}

func (lt *LatencyTracker) RecordLatency(endpoint, method, component string, latency time.Duration, statusCode int) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	
	measurement := LatencyMeasurement{
		Timestamp:  time.Now(),
		Endpoint:   endpoint,
		Method:     method,
		Latency:    latency,
		StatusCode: statusCode,
		Component:  component,
	}
	
	lt.measurements = append(lt.measurements, measurement)
	
	// Keep only last 10000 measurements
	if len(lt.measurements) > 10000 {
		lt.measurements = lt.measurements[1:]
	}
	
	if latency > 100*time.Millisecond {
		lt.logger.Warn("High latency detected", 
			"endpoint", endpoint,
			"latency", latency,
			"target", "100ms")
	}
}

func (lt *LatencyTracker) GetLatencyStats(since time.Time) *LatencyStats {
	lt.mu.RLock()
	defer lt.mu.RUnlock()
	
	var measurements []LatencyMeasurement
	for _, m := range lt.measurements {
		if m.Timestamp.After(since) {
			measurements = append(measurements, m)
		}
	}
	
	if len(measurements) == 0 {
		return &LatencyStats{}
	}
	
	return calculateLatencyStats(measurements)
}

func calculateLatencyStats(measurements []LatencyMeasurement) *LatencyStats {
	if len(measurements) == 0 {
		return &LatencyStats{}
	}
	
	var total time.Duration
	var max time.Duration
	min := measurements[0].Latency
	
	latencies := make([]time.Duration, len(measurements))
	for i, m := range measurements {
		latencies[i] = m.Latency
		total += m.Latency
		if m.Latency > max {
			max = m.Latency
		}
		if m.Latency < min {
			min = m.Latency
		}
	}
	
	// Sort for percentile calculations
	// sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	
	return &LatencyStats{
		Count:   len(measurements),
		Average: total / time.Duration(len(measurements)),
		Min:     min,
		Max:     max,
		P50:     calculatePercentile(latencies, 0.5),
		P95:     calculatePercentile(latencies, 0.95),
		P99:     calculatePercentile(latencies, 0.99),
	}
}

type LatencyStats struct {
	Count   int           `json:"count"`
	Average time.Duration `json:"average"`
	Min     time.Duration `json:"min"`
	Max     time.Duration `json:"max"`
	P50     time.Duration `json:"p50"`
	P95     time.Duration `json:"p95"`
	P99     time.Duration `json:"p99"`
}

func calculatePercentile(latencies []time.Duration, percentile float64) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	
	index := int(float64(len(latencies)) * percentile)
	if index >= len(latencies) {
		index = len(latencies) - 1
	}
	
	return latencies[index]
}

// Bottleneck Detector
type BottleneckDetector struct {
	logger       logging.Logger
	detectors    map[string]*ComponentDetector
	mu           sync.RWMutex
}

type ComponentDetector struct {
	Component     string                `json:"component"`
	Metrics       []PerformanceMetric   `json:"metrics"`
	Thresholds    map[string]float64    `json:"thresholds"`
	Bottlenecks   []DetectedBottleneck  `json:"bottlenecks"`
}

type PerformanceMetric struct {
	Name      string      `json:"name"`
	Value     float64     `json:"value"`
	Timestamp time.Time   `json:"timestamp"`
	Unit      string      `json:"unit"`
}

type DetectedBottleneck struct {
	Component   string    `json:"component"`
	Metric      string    `json:"metric"`
	Value       float64   `json:"value"`
	Threshold   float64   `json:"threshold"`
	Severity    string    `json:"severity"`
	DetectedAt  time.Time `json:"detected_at"`
	Suggestions []string  `json:"suggestions"`
}

func NewBottleneckDetector() *BottleneckDetector {
	return &BottleneckDetector{
		logger:    logging.GetLogger("bottleneck_detector"),
		detectors: make(map[string]*ComponentDetector),
	}
}

func (bd *BottleneckDetector) AddComponentDetector(component string, thresholds map[string]float64) {
	bd.mu.Lock()
	defer bd.mu.Unlock()
	
	bd.detectors[component] = &ComponentDetector{
		Component:  component,
		Metrics:    make([]PerformanceMetric, 0),
		Thresholds: thresholds,
		Bottlenecks: make([]DetectedBottleneck, 0),
	}
}

func (bd *BottleneckDetector) RecordMetric(component, metric string, value float64, unit string) {
	bd.mu.Lock()
	defer bd.mu.Unlock()
	
	detector, exists := bd.detectors[component]
	if !exists {
		return
	}
	
	perfMetric := PerformanceMetric{
		Name:      metric,
		Value:     value,
		Timestamp: time.Now(),
		Unit:      unit,
	}
	
	detector.Metrics = append(detector.Metrics, perfMetric)
	
	// Check for bottlenecks
	if threshold, hasThreshold := detector.Thresholds[metric]; hasThreshold {
		if value > threshold {
			bottleneck := DetectedBottleneck{
				Component:  component,
				Metric:     metric,
				Value:      value,
				Threshold:  threshold,
				Severity:   bd.calculateSeverity(value, threshold),
				DetectedAt: time.Now(),
				Suggestions: bd.generateSuggestions(component, metric, value),
			}
			
			detector.Bottlenecks = append(detector.Bottlenecks, bottleneck)
			
			bd.logger.Warn("Bottleneck detected",
				"component", component,
				"metric", metric,
				"value", value,
				"threshold", threshold)
		}
	}
}

func (bd *BottleneckDetector) calculateSeverity(value, threshold float64) string {
	ratio := value / threshold
	if ratio >= 2.0 {
		return "critical"
	} else if ratio >= 1.5 {
		return "high"
	} else if ratio >= 1.2 {
		return "medium"
	}
	return "low"
}

func (bd *BottleneckDetector) generateSuggestions(component, metric string, value float64) []string {
	suggestions := make([]string, 0)
	
	switch component {
	case "database":
		suggestions = append(suggestions, "Consider adding database indexes", "Optimize query performance", "Scale database connections")
	case "network":
		suggestions = append(suggestions, "Enable connection pooling", "Implement caching", "Use CDN for static content")
	case "cpu":
		suggestions = append(suggestions, "Scale horizontally", "Optimize algorithms", "Implement caching")
	case "memory":
		suggestions = append(suggestions, "Optimize memory usage", "Implement garbage collection tuning", "Scale vertically")
	}
	
	return suggestions
}

// Compression Manager
type CompressionManager struct {
	config *CompressionConfig
	logger logging.Logger
	stats  *CompressionStats
	mu     sync.RWMutex
}

type CompressionConfig struct {
	GzipEnabled    bool `json:"gzip_enabled"`
	BrotliEnabled  bool `json:"brotli_enabled"`
	Level          int  `json:"level"`
	MinSize        int  `json:"min_size"`
}

type CompressionStats struct {
	TotalRequests      int64   `json:"total_requests"`
	CompressedRequests int64   `json:"compressed_requests"`
	BytesSaved         int64   `json:"bytes_saved"`
	CompressionRatio   float64 `json:"compression_ratio"`
}

func NewCompressionManager(config *CompressionConfig) *CompressionManager {
	return &CompressionManager{
		config: config,
		logger: logging.GetLogger("compression_manager"),
		stats:  &CompressionStats{},
	}
}

func (cm *CompressionManager) CompressResponse(w http.ResponseWriter, r *http.Request, data []byte) ([]byte, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.stats.TotalRequests++
	
	if len(data) < cm.config.MinSize {
		return data, nil
	}
	
	acceptEncoding := r.Header.Get("Accept-Encoding")
	
	if cm.config.GzipEnabled && contains(acceptEncoding, "gzip") {
		compressed, err := cm.compressGzip(data)
		if err == nil {
			w.Header().Set("Content-Encoding", "gzip")
			cm.recordCompression(len(data), len(compressed))
			return compressed, nil
		}
	}
	
	return data, nil
}

func (cm *CompressionManager) compressGzip(data []byte) ([]byte, error) {
	// Simplified gzip compression implementation
	return nil, nil // Placeholder
}

func (cm *CompressionManager) recordCompression(originalSize, compressedSize int) {
	cm.stats.CompressedRequests++
	cm.stats.BytesSaved += int64(originalSize - compressedSize)
	
	if cm.stats.CompressedRequests > 0 {
		cm.stats.CompressionRatio = float64(cm.stats.BytesSaved) / float64(cm.stats.CompressedRequests)
	}
}

func contains(s, substr string) bool {
	// Simple contains implementation
	return len(substr) <= len(s) && (len(substr) == 0 || s[0:len(substr)] == substr)
}

// Constructor functions
func NewLatencyOptimizer(config interface{}) *LatencyOptimizer {
	return &LatencyOptimizer{
		logger:  logging.GetLogger("latency_optimizer"),
		metrics: metrics.NewLatencyMetrics(),
	}
}

// Connection Keep-Alive Manager
type ConnectionKeepAliveManager struct {
	config *KeepAliveConfig
	logger logging.Logger
}

type KeepAliveConfig struct {
	Enabled   bool          `json:"enabled"`
	Interval  time.Duration `json:"interval"`
	MaxIdle   int           `json:"max_idle"`
	Timeout   time.Duration `json:"timeout"`
}

func NewConnectionKeepAliveManager(config *KeepAliveConfig) *ConnectionKeepAliveManager {
	return &ConnectionKeepAliveManager{
		config: config,
		logger: logging.GetLogger("keep_alive_manager"),
	}
}

// Request Pipelining Manager
type RequestPipeliningManager struct {
	config *PipeliningConfig
	logger logging.Logger
}

type PipeliningConfig struct {
	Enabled     bool `json:"enabled"`
	MaxRequests int  `json:"max_requests"`
	BatchSize   int  `json:"batch_size"`
}

func NewRequestPipeliningManager(config *PipeliningConfig) *RequestPipeliningManager {
	return &RequestPipeliningManager{
		config: config,
		logger: logging.GetLogger("pipelining_manager"),
	}
}

// Latency optimization loops
func (lo *LatencyOptimizer) latencyOptimizationLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			lo.performLatencyOptimization(ctx)
		}
	}
}

func (lo *LatencyOptimizer) bottleneckDetectionLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			lo.detectBottlenecks(ctx)
		}
	}
}

func (lo *LatencyOptimizer) performLatencyOptimization(ctx context.Context) {
	if lo.latencyTracker != nil {
		stats := lo.latencyTracker.GetLatencyStats(time.Now().Add(-5 * time.Minute))
		if stats.P95 > 100*time.Millisecond {
			lo.logger.Warn("P95 latency exceeds target", "p95", stats.P95, "target", "100ms")
		}
	}
}

func (lo *LatencyOptimizer) detectBottlenecks(ctx context.Context) {
	if lo.bottleneckDetector != nil {
		lo.logger.Debug("Running bottleneck detection")
		// Implementation would analyze current performance metrics
	}
} 