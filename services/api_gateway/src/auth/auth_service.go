package auth

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"iaros/api_gateway/src/config"
)

// AuthService handles authentication and authorization
type AuthService struct {
	config         *config.AuthConfig
	jwtPublicKey   *rsa.PublicKey
	jwtPrivateKey  *rsa.PrivateKey
	cache          *redis.Client
	logger         *zap.Logger
	roles          *RBACManager
	sessions       *SessionManager
	apiKeys        *APIKeyManager
	mutex          sync.RWMutex
	ready          bool
}

// Claims represents JWT claims structure
type Claims struct {
	UserID       string   `json:"user_id"`
	Email        string   `json:"email"`
	Roles        []string `json:"roles"`
	Permissions  []string `json:"permissions"`
	Tier         string   `json:"tier"`
	Organization string   `json:"organization"`
	SessionID    string   `json:"session_id"`
	jwt.RegisteredClaims
}

// User represents an authenticated user
type User struct {
	ID           string                 `json:"id"`
	Email        string                 `json:"email"`
	Roles        []string               `json:"roles"`
	Permissions  []string               `json:"permissions"`
	Tier         string                 `json:"tier"`
	Organization string                 `json:"organization"`
	SessionID    string                 `json:"session_id"`
	Metadata     map[string]interface{} `json:"metadata"`
	LastActivity time.Time              `json:"last_activity"`
}

// APIKey represents an API key for service authentication
type APIKey struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Key         string            `json:"key"`
	Permissions []string          `json:"permissions"`
	RateLimit   int               `json:"rate_limit"`
	ExpiresAt   *time.Time        `json:"expires_at"`
	CreatedAt   time.Time         `json:"created_at"`
	LastUsed    *time.Time        `json:"last_used"`
	Metadata    map[string]string `json:"metadata"`
	Active      bool              `json:"active"`
}

// Session represents a user session
type Session struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	CreatedAt    time.Time              `json:"created_at"`
	LastActivity time.Time              `json:"last_activity"`
	ExpiresAt    time.Time              `json:"expires_at"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	Metadata     map[string]interface{} `json:"metadata"`
	Active       bool                   `json:"active"`
}

// AuthResult represents the result of authentication
type AuthResult struct {
	Authenticated bool                   `json:"authenticated"`
	User          *User                  `json:"user,omitempty"`
	APIKey        *APIKey                `json:"api_key,omitempty"`
	Method        string                 `json:"method"`
	Error         string                 `json:"error,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// NewAuthService creates a new authentication service
func NewAuthService(cfg *config.Config) (*AuthService, error) {
	logger, _ := zap.NewProduction()

	// Initialize Redis client for caching
	cache := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.AuthDB,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := cache.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Load JWT keys
	publicKey, privateKey, err := loadJWTKeys(cfg.Auth.JWTKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load JWT keys: %w", err)
	}

	// Initialize managers
	rbacManager, err := NewRBACManager(cfg.Auth.RBACConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize RBAC manager: %w", err)
	}

	sessionManager, err := NewSessionManager(cache, cfg.Auth.SessionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize session manager: %w", err)
	}

	apiKeyManager, err := NewAPIKeyManager(cache, cfg.Auth.APIKeyConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize API key manager: %w", err)
	}

	authService := &AuthService{
		config:        &cfg.Auth,
		jwtPublicKey:  publicKey,
		jwtPrivateKey: privateKey,
		cache:         cache,
		logger:        logger,
		roles:         rbacManager,
		sessions:      sessionManager,
		apiKeys:       apiKeyManager,
		ready:         true,
	}

	// Start background tasks
	go authService.startCleanupTasks()

	return authService, nil
}

// AuthRequired middleware for routes requiring authentication
func (as *AuthService) AuthRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := as.Authenticate(r)
		
		if !result.Authenticated {
			as.handleAuthenticationError(w, r, result)
			return
		}

		// Add user context to request
		ctx := context.WithValue(r.Context(), "auth_result", result)
		ctx = context.WithValue(ctx, "user", result.User)
		
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AdminRequired middleware for admin-only routes
func (as *AuthService) AdminRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := as.Authenticate(r)
		
		if !result.Authenticated {
			as.handleAuthenticationError(w, r, result)
			return
		}

		// Check for admin role
		if !as.hasRole(result.User, "admin") {
			as.handleAuthorizationError(w, r, "Admin access required")
			return
		}

		ctx := context.WithValue(r.Context(), "auth_result", result)
		ctx = context.WithValue(ctx, "user", result.User)
		
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RoleRequired middleware for role-based access
func (as *AuthService) RoleRequired(requiredRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			result := as.Authenticate(r)
			
			if !result.Authenticated {
				as.handleAuthenticationError(w, r, result)
				return
			}

			// Check if user has any of the required roles
			hasRole := false
			for _, role := range requiredRoles {
				if as.hasRole(result.User, role) {
					hasRole = true
					break
				}
			}

			if !hasRole {
				as.handleAuthorizationError(w, r, fmt.Sprintf("Required roles: %v", requiredRoles))
				return
			}

			ctx := context.WithValue(r.Context(), "auth_result", result)
			ctx = context.WithValue(ctx, "user", result.User)
			
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// PermissionRequired middleware for permission-based access
func (as *AuthService) PermissionRequired(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			result := as.Authenticate(r)
			
			if !result.Authenticated {
				as.handleAuthenticationError(w, r, result)
				return
			}

			if !as.hasPermission(result.User, permission) {
				as.handleAuthorizationError(w, r, fmt.Sprintf("Required permission: %s", permission))
				return
			}

			ctx := context.WithValue(r.Context(), "auth_result", result)
			ctx = context.WithValue(ctx, "user", result.User)
			
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Authenticate performs authentication using multiple strategies
func (as *AuthService) Authenticate(r *http.Request) AuthResult {
	// Try JWT authentication first
	if token := as.extractJWTToken(r); token != "" {
		if result := as.authenticateJWT(token, r); result.Authenticated {
			return result
		}
	}

	// Try API key authentication
	if apiKey := as.extractAPIKey(r); apiKey != "" {
		if result := as.authenticateAPIKey(apiKey, r); result.Authenticated {
			return result
		}
	}

	// Try session authentication
	if sessionID := as.extractSessionID(r); sessionID != "" {
		if result := as.authenticateSession(sessionID, r); result.Authenticated {
			return result
		}
	}

	// Authentication failed
	return AuthResult{
		Authenticated: false,
		Method:        "none",
		Error:         "No valid authentication provided",
	}
}

// authenticateJWT validates JWT tokens
func (as *AuthService) authenticateJWT(tokenString string, r *http.Request) AuthResult {
	// Parse and validate JWT
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return as.jwtPublicKey, nil
	})

	if err != nil {
		return AuthResult{
			Authenticated: false,
			Method:        "jwt",
			Error:         fmt.Sprintf("Invalid JWT: %v", err),
		}
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return AuthResult{
			Authenticated: false,
			Method:        "jwt",
			Error:         "Invalid JWT claims",
		}
	}

	// Check if token is revoked
	if as.isTokenRevoked(tokenString) {
		return AuthResult{
			Authenticated: false,
			Method:        "jwt",
			Error:         "Token has been revoked",
		}
	}

	// Create user from claims
	user := &User{
		ID:           claims.UserID,
		Email:        claims.Email,
		Roles:        claims.Roles,
		Permissions:  claims.Permissions,
		Tier:         claims.Tier,
		Organization: claims.Organization,
		SessionID:    claims.SessionID,
		LastActivity: time.Now(),
	}

	// Update session activity
	if claims.SessionID != "" {
		as.sessions.UpdateActivity(claims.SessionID, r.RemoteAddr, r.UserAgent())
	}

	return AuthResult{
		Authenticated: true,
		User:          user,
		Method:        "jwt",
	}
}

// authenticateAPIKey validates API keys
func (as *AuthService) authenticateAPIKey(keyString string, r *http.Request) AuthResult {
	apiKey, err := as.apiKeys.ValidateKey(keyString)
	if err != nil {
		return AuthResult{
			Authenticated: false,
			Method:        "api_key",
			Error:         fmt.Sprintf("Invalid API key: %v", err),
		}
	}

	// Update last used timestamp
	as.apiKeys.UpdateLastUsed(apiKey.ID, time.Now())

	// Create user-like object from API key
	user := &User{
		ID:          apiKey.ID,
		Email:       fmt.Sprintf("api-key-%s", apiKey.Name),
		Permissions: apiKey.Permissions,
		Tier:        "api",
		Metadata: map[string]interface{}{
			"api_key_name": apiKey.Name,
			"rate_limit":   apiKey.RateLimit,
		},
		LastActivity: time.Now(),
	}

	return AuthResult{
		Authenticated: true,
		User:          user,
		APIKey:        apiKey,
		Method:        "api_key",
	}
}

// authenticateSession validates session-based authentication
func (as *AuthService) authenticateSession(sessionID string, r *http.Request) AuthResult {
	session, err := as.sessions.GetSession(sessionID)
	if err != nil {
		return AuthResult{
			Authenticated: false,
			Method:        "session",
			Error:         fmt.Sprintf("Invalid session: %v", err),
		}
	}

	// Validate session
	if !session.Active || session.ExpiresAt.Before(time.Now()) {
		return AuthResult{
			Authenticated: false,
			Method:        "session",
			Error:         "Session expired or inactive",
		}
	}

	// Get user from session
	user, err := as.getUserFromSession(session)
	if err != nil {
		return AuthResult{
			Authenticated: false,
			Method:        "session",
			Error:         fmt.Sprintf("Failed to get user: %v", err),
		}
	}

	// Update session activity
	as.sessions.UpdateActivity(sessionID, r.RemoteAddr, r.UserAgent())

	return AuthResult{
		Authenticated: true,
		User:          user,
		Method:        "session",
	}
}

// Token extraction methods
func (as *AuthService) extractJWTToken(r *http.Request) string {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// Check query parameter
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}

	// Check cookie
	if cookie, err := r.Cookie("jwt_token"); err == nil {
		return cookie.Value
	}

	return ""
}

func (as *AuthService) extractAPIKey(r *http.Request) string {
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

func (as *AuthService) extractSessionID(r *http.Request) string {
	// Check session cookie
	if cookie, err := r.Cookie("session_id"); err == nil {
		return cookie.Value
	}

	// Check X-Session-ID header
	if sessionID := r.Header.Get("X-Session-ID"); sessionID != "" {
		return sessionID
	}

	return ""
}

// Token management
func (as *AuthService) GenerateJWT(user *User, sessionID string) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:       user.ID,
		Email:        user.Email,
		Roles:        user.Roles,
		Permissions:  user.Permissions,
		Tier:         user.Tier,
		Organization: user.Organization,
		SessionID:    sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    as.config.JWTIssuer,
			Subject:   user.ID,
			Audience:  []string{as.config.JWTAudience},
			ExpiresAt: jwt.NewNumericDate(now.Add(as.config.JWTExpiry)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(as.jwtPrivateKey)
}

func (as *AuthService) RevokeToken(tokenString string) error {
	// Add token to revocation list
	ctx := context.Background()
	key := fmt.Sprintf("revoked_token:%s", as.hashToken(tokenString))
	
	// Store with expiry equal to token expiry
	expiry := as.config.JWTExpiry
	return as.cache.Set(ctx, key, "revoked", expiry).Err()
}

func (as *AuthService) isTokenRevoked(tokenString string) bool {
	ctx := context.Background()
	key := fmt.Sprintf("revoked_token:%s", as.hashToken(tokenString))
	
	result := as.cache.Exists(ctx, key)
	return result.Val() > 0
}

// Permission and role checking
func (as *AuthService) hasRole(user *User, role string) bool {
	if user == nil {
		return false
	}
	
	for _, userRole := range user.Roles {
		if userRole == role {
			return true
		}
	}
	
	return false
}

func (as *AuthService) hasPermission(user *User, permission string) bool {
	if user == nil {
		return false
	}

	// Check direct permissions
	for _, userPermission := range user.Permissions {
		if userPermission == permission {
			return true
		}
	}

	// Check role-based permissions
	for _, role := range user.Roles {
		if as.roles.RoleHasPermission(role, permission) {
			return true
		}
	}

	return false
}

// Error handlers
func (as *AuthService) handleAuthenticationError(w http.ResponseWriter, r *http.Request, result AuthResult) {
	response := map[string]interface{}{
		"error":       "Authentication required",
		"message":     result.Error,
		"timestamp":   time.Now().UTC(),
		"path":        r.URL.Path,
		"method":      r.Method,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(response)
}

func (as *AuthService) handleAuthorizationError(w http.ResponseWriter, r *http.Request, message string) {
	response := map[string]interface{}{
		"error":       "Insufficient permissions",
		"message":     message,
		"timestamp":   time.Now().UTC(),
		"path":        r.URL.Path,
		"method":      r.Method,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(response)
}

// Utility methods
func (as *AuthService) hashToken(token string) string {
	// Simple hash implementation - in production, use proper hashing
	return fmt.Sprintf("%x", []byte(token)[:32])
}

func (as *AuthService) getUserFromSession(session *Session) (*User, error) {
	// Get user from cache or database
	ctx := context.Background()
	key := fmt.Sprintf("user:%s", session.UserID)
	
	userJSON, err := as.cache.Get(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("user not found in cache: %w", err)
	}

	var user User
	if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return &user, nil
}

func (as *AuthService) startCleanupTasks() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		// Cleanup expired sessions
		as.sessions.CleanupExpired()
		
		// Cleanup expired API keys
		as.apiKeys.CleanupExpired()
		
		// Cleanup expired revoked tokens
		as.cleanupRevokedTokens()
	}
}

func (as *AuthService) cleanupRevokedTokens() {
	// Implementation would clean up expired revoked tokens from cache
	as.logger.Info("Cleaning up expired revoked tokens")
}

// Status methods
func (as *AuthService) IsReady() bool {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	return as.ready
}

func (as *AuthService) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"ready":           as.IsReady(),
		"jwt_enabled":     as.config.JWTEnabled,
		"api_key_enabled": as.config.APIKeyEnabled,
		"session_enabled": as.config.SessionEnabled,
		"cache_connected": as.cache.Ping(context.Background()).Err() == nil,
	}
}

// Helper function to load JWT keys
func loadJWTKeys(keyPath string) (*rsa.PublicKey, *rsa.PrivateKey, error) {
	// Implementation would load actual RSA keys from files
	// For now, return mock keys or generate them
	
	// In production, load from PEM files:
	// publicKeyBytes, err := ioutil.ReadFile(keyPath + "/public.pem")
	// privateKeyBytes, err := ioutil.ReadFile(keyPath + "/private.pem")
	
	return nil, nil, nil // Placeholder
} 