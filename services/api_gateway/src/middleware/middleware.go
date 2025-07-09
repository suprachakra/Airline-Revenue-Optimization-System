package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"go.uber.org/zap"
)

// RequestIDKey is the context key for request ID
const RequestIDKey = "request_id"

// PanicRecovery middleware recovers from panics and returns a proper HTTP error
func PanicRecovery() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log the panic
					logger, _ := zap.NewProduction()
					logger.Error("Panic recovered",
						zap.String("panic", fmt.Sprintf("%v", err)),
						zap.String("stack", string(debug.Stack())),
						zap.String("request_id", GetRequestID(r.Context())),
						zap.String("path", r.URL.Path),
						zap.String("method", r.Method),
					)

					// Return HTTP 500 error
					response := map[string]interface{}{
						"error":      "Internal server error",
						"message":    "An unexpected error occurred",
						"timestamp":  time.Now().UTC(),
						"request_id": GetRequestID(r.Context()),
					}

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(response)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders middleware adds security headers to responses
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add security headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			w.Header().Set("Content-Security-Policy", "default-src 'self'")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			next.ServeHTTP(w, r)
		})
	}
}

// RequestID middleware generates and adds a unique request ID
func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if request ID already exists
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}

			// Add request ID to context
			ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

			// Add request ID to response header
			w.Header().Set("X-Request-ID", requestID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequestLogging middleware logs incoming requests
func RequestLogging() func(http.Handler) http.Handler {
	logger, _ := zap.NewProduction()
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status code
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     200,
				bytesWritten:   0,
			}

			// Log request
			logger.Info("Request started",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("query", r.URL.RawQuery),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()),
				zap.String("request_id", GetRequestID(r.Context())),
				zap.String("host", r.Host),
			)

			// Process request
			next.ServeHTTP(wrapped, r)

			// Log response
			duration := time.Since(start)
			logger.Info("Request completed",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", wrapped.statusCode),
				zap.Int64("bytes", wrapped.bytesWritten),
				zap.Duration("duration", duration),
				zap.String("request_id", GetRequestID(r.Context())),
			)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code and bytes written
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

// Cache middleware provides HTTP caching
func Cache(ttl time.Duration, keyFunc string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// For GET requests only
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			// Generate cache key
			cacheKey := generateCacheKey(r, keyFunc)

			// Set cache headers
			w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(ttl.Seconds())))
			w.Header().Set("ETag", cacheKey)

			// Check if client has cached version
			if match := r.Header.Get("If-None-Match"); match == cacheKey {
				w.WriteHeader(http.StatusNotModified)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Transform middleware applies request/response transformations
func Transform(transform *Transform) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Apply request transformations
			if transform != nil {
				// Transform request headers
				for key, value := range transform.RequestHeaders {
					r.Header.Set(key, value)
				}

				// Transform response headers
				for key, value := range transform.ResponseHeaders {
					w.Header().Set(key, value)
				}

				// TODO: Implement request/response body transformation
				// This would require buffering and transforming the body
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Transform represents request/response transformation rules
type Transform struct {
	RequestHeaders  map[string]string `json:"request_headers"`
	ResponseHeaders map[string]string `json:"response_headers"`
	RequestBody     string            `json:"request_body"`
	ResponseBody    string            `json:"response_body"`
}

// Timeout middleware adds a timeout to requests
func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Channel to signal completion
			done := make(chan struct{})
			go func() {
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				// Request completed normally
				return
			case <-ctx.Done():
				// Request timed out
				response := map[string]interface{}{
					"error":      "Request timeout",
					"message":    fmt.Sprintf("Request took longer than %v", timeout),
					"timestamp":  time.Now().UTC(),
					"request_id": GetRequestID(r.Context()),
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusGatewayTimeout)
				json.NewEncoder(w).Encode(response)
			}
		})
	}
}

// Compress middleware adds gzip compression
func Compress() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if client supports gzip
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			// Set compression headers
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Vary", "Accept-Encoding")

			// TODO: Implement actual gzip compression
			// This would require wrapping the response writer with gzip writer
			next.ServeHTTP(w, r)
		})
	}
}

// ContentType middleware sets the content type for responses
func ContentType(contentType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", contentType)
			next.ServeHTTP(w, r)
		})
	}
}

// APIVersion middleware handles API versioning
func APIVersion(version string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("API-Version", version)
			
			// Add version to context
			ctx := context.WithValue(r.Context(), "api_version", version)
			
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// LoadBalancer middleware adds load balancing information
func LoadBalancer(strategy string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Load-Balancer", strategy)
			next.ServeHTTP(w, r)
		})
	}
}

// HealthCheck middleware provides health check endpoint
func HealthCheck(healthEndpoint string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == healthEndpoint {
				response := map[string]interface{}{
					"status":    "healthy",
					"timestamp": time.Now().UTC(),
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Metrics middleware records metrics
func Metrics(recordFunc func(string, string, int, time.Duration)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     200,
				bytesWritten:   0,
			}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			recordFunc(r.Method, r.URL.Path, wrapped.statusCode, duration)
		})
	}
}

// Utility functions

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateCacheKey generates a cache key for a request
func generateCacheKey(r *http.Request, keyFunc string) string {
	// Simple implementation - in production, use more sophisticated key generation
	return fmt.Sprintf("%s:%s:%s", r.Method, r.URL.Path, r.URL.RawQuery)
}

// ClientIP extracts the client IP from the request
func ClientIP(r *http.Request) string {
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

// IsWebSocket checks if the request is a WebSocket upgrade request
func IsWebSocket(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Connection")) == "upgrade" &&
		strings.ToLower(r.Header.Get("Upgrade")) == "websocket"
}

// IsAjax checks if the request is an AJAX request
func IsAjax(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("X-Requested-With")) == "xmlhttprequest"
}

// SetNoCacheHeaders sets headers to prevent caching
func SetNoCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

// SetCacheHeaders sets headers for caching
func SetCacheHeaders(w http.ResponseWriter, maxAge int) {
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
	w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
} 