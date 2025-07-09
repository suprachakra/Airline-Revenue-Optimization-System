package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"iaros/api_gateway/src/config"
)

// RateLimiter handles rate limiting for the API Gateway
type RateLimiter struct {
	config      *config.RateLimitConfig
	cache       *redis.Client
	logger      *zap.Logger
	limiters    map[string]*WindowLimiter
	mutex       sync.RWMutex
	ready       bool
}

// WindowLimiter represents a sliding window rate limiter
type WindowLimiter struct {
	Limit    int           `json:"limit"`
	Window   time.Duration `json:"window"`
	Requests []time.Time   `json:"requests"`
	mutex    sync.RWMutex
}

// RateLimitResult represents the result of rate limiting check
type RateLimitResult struct {
	Allowed      bool          `json:"allowed"`
	Remaining    int           `json:"remaining"`
	ResetTime    time.Time     `json:"reset_time"`
	RetryAfter   time.Duration `json:"retry_after"`
	RateLimitKey string        `json:"rate_limit_key"`
}

// RateLimitConfig represents configuration for different rate limits
type RateLimitRule struct {
	Path         string        `json:"path"`
	Method       string        `json:"method"`
	Limit        int           `json:"limit"`
	Window       time.Duration `json:"window"`
	PerIP        bool          `json:"per_ip"`
	PerUser      bool          `json:"per_user"`
	PerAPIKey    bool          `json:"per_api_key"`
	BurstAllowed int           `json:"burst_allowed"`
	Priority     int           `json:"priority"`
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(cfg *config.Config) (*RateLimiter, error) {
	logger, _ := zap.NewProduction()

	// Initialize Redis client
	cache := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.RateLimitDB,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := cache.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	rateLimiter := &RateLimiter{
		config:   &cfg.RateLimit,
		cache:    cache,
		logger:   logger,
		limiters: make(map[string]*WindowLimiter),
		ready:    true,
	}

	// Initialize default limiters
	rateLimiter.initializeDefaultLimiters()

	// Start cleanup goroutine
	go rateLimiter.startCleanup()

	return rateLimiter, nil
}

// initializeDefaultLimiters sets up default rate limiters
func (rl *RateLimiter) initializeDefaultLimiters() {
	// Global rate limiter
	rl.limiters["global"] = &WindowLimiter{
		Limit:    rl.config.Global.Limit,
		Window:   rl.config.Global.Window,
		Requests: make([]time.Time, 0),
	}

	// Per-IP rate limiter
	rl.limiters["per_ip"] = &WindowLimiter{
		Limit:    rl.config.PerIP.Limit,
		Window:   rl.config.PerIP.Window,
		Requests: make([]time.Time, 0),
	}

	// Per-User rate limiter
	rl.limiters["per_user"] = &WindowLimiter{
		Limit:    rl.config.PerUser.Limit,
		Window:   rl.config.PerUser.Window,
		Requests: make([]time.Time, 0),
	}

	// Per-API-Key rate limiter
	rl.limiters["per_api_key"] = &WindowLimiter{
		Limit:    rl.config.PerAPIKey.Limit,
		Window:   rl.config.PerAPIKey.Window,
		Requests: make([]time.Time, 0),
	}
}

// Allow checks if a request is allowed based on rate limits
func (rl *RateLimiter) Allow(r *http.Request) bool {
	// Get rate limit keys
	keys := rl.generateRateLimitKeys(r)

	// Check each rate limit
	for _, key := range keys {
		result := rl.checkRateLimit(key)
		if !result.Allowed {
			rl.logger.Info("Rate limit exceeded",
				zap.String("key", key),
				zap.Int("remaining", result.Remaining),
				zap.Time("reset_time", result.ResetTime),
			)
			return false
		}
	}

	return true
}

// RouteLimit creates a route-specific rate limiter middleware
func (rl *RateLimiter) RouteLimit(limit int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := fmt.Sprintf("route:%s:%s", r.Method, r.URL.Path)
			
			result := rl.checkRateLimitWithCustomLimit(key, limit, time.Minute)
			if !result.Allowed {
				rl.handleRateLimitExceeded(w, r, result)
				return
			}

			// Set rate limit headers
			rl.setRateLimitHeaders(w, result)
			
			next.ServeHTTP(w, r)
		})
	}
}

// checkRateLimit checks rate limit for a specific key
func (rl *RateLimiter) checkRateLimit(key string) RateLimitResult {
	ctx := context.Background()
	now := time.Now()

	// Use Redis for distributed rate limiting
	pipe := rl.cache.Pipeline()
	
	// Get current count
	countKey := fmt.Sprintf("rate_limit:%s:count", key)
	windowKey := fmt.Sprintf("rate_limit:%s:window", key)
	
	// Sliding window implementation using Redis
	// Remove expired entries
	pipe.ZRemRangeByScore(ctx, windowKey, "0", fmt.Sprintf("%d", now.Add(-time.Minute).Unix()))
	
	// Add current request
	pipe.ZAdd(ctx, windowKey, redis.Z{
		Score:  float64(now.Unix()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})
	
	// Get current count
	pipe.ZCard(ctx, windowKey)
	
	// Set expiry
	pipe.Expire(ctx, windowKey, time.Minute)
	
	// Execute pipeline
	results, err := pipe.Exec(ctx)
	if err != nil {
		rl.logger.Error("Redis pipeline error", zap.Error(err))
		return RateLimitResult{
			Allowed:      false,
			Remaining:    0,
			ResetTime:    now.Add(time.Minute),
			RetryAfter:   time.Minute,
			RateLimitKey: key,
		}
	}

	// Get current count from results
	currentCount := int(results[2].(*redis.IntCmd).Val())
	
	// Determine rate limit based on key type
	limit := rl.getLimitForKey(key)
	
	allowed := currentCount <= limit
	remaining := limit - currentCount
	if remaining < 0 {
		remaining = 0
	}

	return RateLimitResult{
		Allowed:      allowed,
		Remaining:    remaining,
		ResetTime:    now.Add(time.Minute),
		RetryAfter:   time.Minute,
		RateLimitKey: key,
	}
}

// checkRateLimitWithCustomLimit checks rate limit with custom parameters
func (rl *RateLimiter) checkRateLimitWithCustomLimit(key string, limit int, window time.Duration) RateLimitResult {
	ctx := context.Background()
	now := time.Now()

	windowKey := fmt.Sprintf("rate_limit:%s:window", key)
	
	pipe := rl.cache.Pipeline()
	
	// Remove expired entries
	pipe.ZRemRangeByScore(ctx, windowKey, "0", fmt.Sprintf("%d", now.Add(-window).Unix()))
	
	// Add current request
	pipe.ZAdd(ctx, windowKey, redis.Z{
		Score:  float64(now.Unix()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})
	
	// Get current count
	pipe.ZCard(ctx, windowKey)
	
	// Set expiry
	pipe.Expire(ctx, windowKey, window)
	
	// Execute pipeline
	results, err := pipe.Exec(ctx)
	if err != nil {
		rl.logger.Error("Redis pipeline error", zap.Error(err))
		return RateLimitResult{
			Allowed:      false,
			Remaining:    0,
			ResetTime:    now.Add(window),
			RetryAfter:   window,
			RateLimitKey: key,
		}
	}

	currentCount := int(results[2].(*redis.IntCmd).Val())
	
	allowed := currentCount <= limit
	remaining := limit - currentCount
	if remaining < 0 {
		remaining = 0
	}

	return RateLimitResult{
		Allowed:      allowed,
		Remaining:    remaining,
		ResetTime:    now.Add(window),
		RetryAfter:   window,
		RateLimitKey: key,
	}
}

// generateRateLimitKeys generates rate limit keys for a request
func (rl *RateLimiter) generateRateLimitKeys(r *http.Request) []string {
	keys := []string{}

	// Global rate limit
	keys = append(keys, "global")

	// Per-IP rate limit
	clientIP := rl.getClientIP(r)
	keys = append(keys, fmt.Sprintf("ip:%s", clientIP))

	// Per-User rate limit (if authenticated)
	if userID := rl.getUserID(r); userID != "" {
		keys = append(keys, fmt.Sprintf("user:%s", userID))
	}

	// Per-API-Key rate limit (if using API key)
	if apiKey := rl.getAPIKey(r); apiKey != "" {
		keys = append(keys, fmt.Sprintf("api_key:%s", apiKey))
	}

	// Path-specific rate limit
	path := rl.normalizePath(r.URL.Path)
	keys = append(keys, fmt.Sprintf("path:%s", path))

	// Method-specific rate limit
	keys = append(keys, fmt.Sprintf("method:%s", r.Method))

	return keys
}

// getLimitForKey returns the appropriate limit for a rate limit key
func (rl *RateLimiter) getLimitForKey(key string) int {
	switch {
	case strings.HasPrefix(key, "global"):
		return rl.config.Global.Limit
	case strings.HasPrefix(key, "ip:"):
		return rl.config.PerIP.Limit
	case strings.HasPrefix(key, "user:"):
		return rl.config.PerUser.Limit
	case strings.HasPrefix(key, "api_key:"):
		return rl.config.PerAPIKey.Limit
	case strings.HasPrefix(key, "path:"):
		return rl.config.PerPath.Limit
	case strings.HasPrefix(key, "method:"):
		return rl.config.PerMethod.Limit
	default:
		return rl.config.Global.Limit
	}
}

// Helper methods for extracting information from requests
func (rl *RateLimiter) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	parts := strings.Split(r.RemoteAddr, ":")
	return parts[0]
}

func (rl *RateLimiter) getUserID(r *http.Request) string {
	// Check for user ID in context (set by auth middleware)
	if userCtx := r.Context().Value("user"); userCtx != nil {
		if user, ok := userCtx.(map[string]interface{}); ok {
			if userID, ok := user["id"].(string); ok {
				return userID
			}
		}
	}

	// Check for user ID in JWT claims
	if claims := r.Context().Value("claims"); claims != nil {
		if claimsMap, ok := claims.(map[string]interface{}); ok {
			if userID, ok := claimsMap["user_id"].(string); ok {
				return userID
			}
		}
	}

	return ""
}

func (rl *RateLimiter) getAPIKey(r *http.Request) string {
	// Check X-API-Key header
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return apiKey
	}

	// Check query parameter
	if apiKey := r.URL.Query().Get("api_key"); apiKey != "" {
		return apiKey
	}

	return ""
}

func (rl *RateLimiter) normalizePath(path string) string {
	// Remove query parameters
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}

	// Remove trailing slash
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

	return path
}

// setRateLimitHeaders sets standard rate limit headers
func (rl *RateLimiter) setRateLimitHeaders(w http.ResponseWriter, result RateLimitResult) {
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Remaining+1))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetTime.Unix(), 10))
	
	if !result.Allowed {
		w.Header().Set("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
	}
}

// handleRateLimitExceeded handles rate limit exceeded scenarios
func (rl *RateLimiter) handleRateLimitExceeded(w http.ResponseWriter, r *http.Request, result RateLimitResult) {
	rl.setRateLimitHeaders(w, result)
	
	response := map[string]interface{}{
		"error":        "Rate limit exceeded",
		"message":      "Too many requests",
		"rate_limit":   result.RateLimitKey,
		"retry_after":  int(result.RetryAfter.Seconds()),
		"reset_time":   result.ResetTime.Unix(),
		"timestamp":    time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	
	// Don't return error on JSON encoding failure
	json.NewEncoder(w).Encode(response)
}

// GetRetryAfter returns the retry after duration for a request
func (rl *RateLimiter) GetRetryAfter(r *http.Request) int {
	// Return a default retry after duration
	return int(rl.config.Global.Window.Seconds())
}

// Reset resets all rate limiters (admin function)
func (rl *RateLimiter) Reset() {
	ctx := context.Background()
	
	// Get all rate limit keys
	keys, err := rl.cache.Keys(ctx, "rate_limit:*").Result()
	if err != nil {
		rl.logger.Error("Failed to get rate limit keys", zap.Error(err))
		return
	}

	// Delete all rate limit keys
	if len(keys) > 0 {
		if err := rl.cache.Del(ctx, keys...).Err(); err != nil {
			rl.logger.Error("Failed to delete rate limit keys", zap.Error(err))
		}
	}

	rl.logger.Info("Rate limiters reset", zap.Int("keys_deleted", len(keys)))
}

// GetStatus returns the current status of rate limiters
func (rl *RateLimiter) GetStatus() map[string]interface{} {
	ctx := context.Background()
	
	// Get statistics from Redis
	stats := make(map[string]interface{})
	
	// Get number of active rate limit keys
	keys, err := rl.cache.Keys(ctx, "rate_limit:*").Result()
	if err != nil {
		rl.logger.Error("Failed to get rate limit keys", zap.Error(err))
		stats["active_keys"] = 0
	} else {
		stats["active_keys"] = len(keys)
	}

	// Get memory usage
	memoryUsage, err := rl.cache.MemoryUsage(ctx, "rate_limit:*").Result()
	if err != nil {
		stats["memory_usage"] = 0
	} else {
		stats["memory_usage"] = memoryUsage
	}

	return map[string]interface{}{
		"ready":        rl.IsReady(),
		"statistics":   stats,
		"config": map[string]interface{}{
			"global_limit":    rl.config.Global.Limit,
			"per_ip_limit":    rl.config.PerIP.Limit,
			"per_user_limit":  rl.config.PerUser.Limit,
			"per_api_key_limit": rl.config.PerAPIKey.Limit,
		},
		"timestamp": time.Now().UTC(),
	}
}

// IsReady returns whether the rate limiter is ready
func (rl *RateLimiter) IsReady() bool {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()
	
	// Check Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	if err := rl.cache.Ping(ctx).Err(); err != nil {
		return false
	}
	
	return rl.ready
}

// startCleanup starts the cleanup goroutine
func (rl *RateLimiter) startCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup removes expired rate limit entries
func (rl *RateLimiter) cleanup() {
	ctx := context.Background()
	now := time.Now()

	// Get all rate limit window keys
	keys, err := rl.cache.Keys(ctx, "rate_limit:*:window").Result()
	if err != nil {
		rl.logger.Error("Failed to get rate limit window keys", zap.Error(err))
		return
	}

	cleaned := 0
	for _, key := range keys {
		// Remove expired entries from sorted set
		expiredBefore := now.Add(-time.Hour).Unix()
		removed, err := rl.cache.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", expiredBefore)).Result()
		if err != nil {
			rl.logger.Error("Failed to remove expired entries", zap.String("key", key), zap.Error(err))
			continue
		}
		
		if removed > 0 {
			cleaned += int(removed)
		}

		// Remove empty sets
		count, err := rl.cache.ZCard(ctx, key).Result()
		if err != nil {
			continue
		}
		
		if count == 0 {
			rl.cache.Del(ctx, key)
		}
	}

	if cleaned > 0 {
		rl.logger.Info("Rate limiter cleanup completed", zap.Int("entries_cleaned", cleaned))
	}
}

// Advanced rate limiting features

// BurstAllowed checks if burst requests are allowed
func (rl *RateLimiter) BurstAllowed(r *http.Request, burstSize int) bool {
	key := fmt.Sprintf("burst:%s", rl.getClientIP(r))
	
	result := rl.checkRateLimitWithCustomLimit(key, burstSize, time.Minute)
	return result.Allowed
}

// PriorityRequest handles priority requests with different limits
func (rl *RateLimiter) PriorityRequest(r *http.Request, priority int) bool {
	// Higher priority gets higher limits
	multiplier := 1.0
	switch priority {
	case 1: // High priority
		multiplier = 2.0
	case 2: // Medium priority
		multiplier = 1.5
	case 3: // Low priority
		multiplier = 0.5
	}

	key := fmt.Sprintf("priority:%d:%s", priority, rl.getClientIP(r))
	limit := int(float64(rl.config.PerIP.Limit) * multiplier)
	
	result := rl.checkRateLimitWithCustomLimit(key, limit, rl.config.PerIP.Window)
	return result.Allowed
}

// Custom rate limiting for specific endpoints
func (rl *RateLimiter) CustomLimit(key string, limit int, window time.Duration) RateLimitResult {
	return rl.checkRateLimitWithCustomLimit(key, limit, window)
}

// GetCurrentUsage returns current usage for a key
func (rl *RateLimiter) GetCurrentUsage(key string) (int, error) {
	ctx := context.Background()
	windowKey := fmt.Sprintf("rate_limit:%s:window", key)
	
	count, err := rl.cache.ZCard(ctx, windowKey).Result()
	if err != nil {
		return 0, err
	}
	
	return int(count), nil
} 