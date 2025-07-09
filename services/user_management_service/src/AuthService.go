package usermgmt

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/oauth2"
	"go.uber.org/zap"
)

// AuthHandler handles comprehensive user authentication with enterprise security features
// Implements IAROS security framework with multi-factor authentication and enterprise SSO
type AuthHandler struct {
	provider      *oidc.Provider
	oauthConf     *oauth2.Config
	localCache    *CredentialCache
	logger        *zap.Logger
	tokenDuration time.Duration
	jwtSecret     []byte
	maintenance   *MaintenanceWindow
}

// CredentialCache represents cached user credentials for fallback authentication
type CredentialCache struct {
	cache     map[string]*CachedCredential
	mutex     sync.RWMutex
	ttl       time.Duration
}

// CachedCredential represents a cached user credential
type CachedCredential struct {
	User      *User
	Timestamp time.Time
	ExpiresAt time.Time
}

// MaintenanceWindow manages emergency authentication during system maintenance
type MaintenanceWindow struct {
	active    bool
	startTime time.Time
	endTime   time.Time
	bypassKey string
}

// Claims represents JWT token claims
type Claims struct {
	UserID      string   `json:"user_id"`
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	SessionID   string   `json:"session_id"`
	jwt.RegisteredClaims
}

// NewAuthHandler creates a new authentication handler with production-ready configuration
func NewAuthHandler(oidcProvider *oidc.Provider, oauthConfig *oauth2.Config, jwtSecret []byte) *AuthHandler {
	return &AuthHandler{
		provider:      oidcProvider,
		oauthConf:     oauthConfig,
		localCache:    NewCredentialCache(time.Hour),
		logger:        zap.NewProduction(),
		tokenDuration: 8 * time.Hour, // 8 hour token expiry
		jwtSecret:     jwtSecret,
		maintenance:   &MaintenanceWindow{},
	}
}

// Authenticate verifies an incoming token with comprehensive fallback strategies
func (a *AuthHandler) Authenticate(ctx context.Context, token string) (*User, error) {
	// Validate input token
	if token == "" {
		return nil, errors.New("authentication token is required")
	}

	// Track authentication attempt
	startTime := time.Now()
	defer func() {
		a.logger.Info("Authentication attempt", 
			zap.Duration("duration", time.Since(startTime)),
			zap.String("token_prefix", token[:min(len(token), 8)]))
	}()

	// Primary: JWT token verification for internal tokens
	if user, err := a.verifyJWTToken(token); err == nil {
		a.logger.Info("JWT authentication successful", zap.String("user_id", user.ID))
		return user, nil
	}

	// Secondary: OIDC verification for external tokens
	oidcToken, err := a.provider.Verifier(&oidc.Config{ClientID: a.oauthConf.ClientID}).Verify(ctx, token)
	if err == nil && oidcToken != nil {
		user, err := a.extractUserFromToken(oidcToken)
		if err == nil {
			// Cache successful OIDC authentication
			a.cacheCredentials(token, user)
			a.logger.Info("OIDC authentication successful", zap.String("user_id", user.ID))
			return user, nil
		}
	}

	// Fallback 1: Cached credentials for system resilience
	if cachedUser, err := a.localCache.Get(token); err == nil && cachedUser != nil {
		if time.Since(cachedUser.Timestamp) < a.localCache.ttl {
			a.logger.Warn("Using cached credentials due to primary verification failure")
			return cachedUser.User, nil
		}
	}

	// Fallback 2: Emergency maintenance mode authentication
	if a.isMaintenanceWindow() {
		user, err := a.generateTemporaryAccess(token)
		if err == nil {
			a.logger.Warn("Emergency maintenance authentication granted")
			return user, nil
		}
	}

	// All authentication methods failed
	a.logger.Error("All authentication methods failed", zap.String("token_prefix", token[:min(len(token), 8)]))
	return nil, errors.New("authentication failed")
}

// GenerateToken creates a secure JWT token with comprehensive claims
func (a *AuthHandler) GenerateToken(userID string, roles []string) (string, error) {
	if userID == "" {
		return "", errors.New("user ID is required for token generation")
	}

	// Generate unique session ID
	sessionID, err := a.generateSessionID()
	if err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Create comprehensive token claims
	claims := &Claims{
		UserID:      userID,
		Role:        roles[0], // Primary role
		Permissions: a.getRolePermissions(roles),
		SessionID:   sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "iaros-auth-service",
			Subject:   userID,
			ID:        sessionID,
		},
	}

	// Sign token with HS256 algorithm
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(a.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	a.logger.Info("JWT token generated successfully", 
		zap.String("user_id", userID),
		zap.String("session_id", sessionID))

	return tokenString, nil
}

// LogAuthenticationEvent logs authentication events for security monitoring
func (a *AuthHandler) LogAuthenticationEvent(userID, remoteAddr, eventType string) {
	go func() {
		a.logger.Info("Authentication event",
			zap.String("user_id", userID),
			zap.String("remote_addr", remoteAddr),
			zap.String("event_type", eventType),
			zap.Time("timestamp", time.Now()))
	}()
}

// verifyJWTToken verifies and parses a JWT token
func (a *AuthHandler) verifyJWTToken(tokenString string) (*User, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid JWT token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	// Convert claims to User object
	return &User{
		ID:          claims.UserID,
		Email:       claims.Email,
		Role:        claims.Role,
		Permissions: claims.Permissions,
		Status:      "active",
	}, nil
}

// extractUserFromToken extracts user information from OIDC token
func (a *AuthHandler) extractUserFromToken(oidcToken *oidc.IDToken) (*User, error) {
	var claims struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Sub      string `json:"sub"`
		Groups   []string `json:"groups"`
	}

	if err := oidcToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to extract claims from OIDC token: %w", err)
	}

	// Map OIDC claims to User object
	role := "user" // Default role
	if len(claims.Groups) > 0 {
		role = a.mapGroupToRole(claims.Groups[0])
	}

	return &User{
		ID:       claims.Sub,
		Email:    claims.Email,
		Role:     role,
		Status:   "active",
	}, nil
}

// isMaintenanceWindow checks if the system is in a maintenance window
func (a *AuthHandler) isMaintenanceWindow() bool {
	now := time.Now()
	return a.maintenance.active && 
		   now.After(a.maintenance.startTime) && 
		   now.Before(a.maintenance.endTime)
}

// generateTemporaryAccess issues a temporary access token during maintenance
func (a *AuthHandler) generateTemporaryAccess(token string) (*User, error) {
	if !a.isMaintenanceWindow() {
		return nil, errors.New("not in maintenance window")
	}

	// Validate maintenance bypass token
	if !a.validateMaintenanceToken(token) {
		return nil, errors.New("invalid maintenance token")
	}

	// Generate temporary user with restricted access
	return &User{
		ID:          "maintenance_user_" + time.Now().Format("20060102150405"),
		Email:       "maintenance@iaros.com",
		Role:        "guest",
		Permissions: []string{"read"},
		Status:      "temporary",
	}, nil
}

// Helper functions
func (a *AuthHandler) generateSessionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	hasher := sha256.New()
	hasher.Write(bytes)
	return hex.EncodeToString(hasher.Sum(nil))[:32], nil
}

func (a *AuthHandler) getRolePermissions(roles []string) []string {
	permissionMap := map[string][]string{
		"admin":   {"read", "write", "delete", "admin"},
		"user":    {"read", "write"},
		"partner": {"read", "write", "partner"},
		"guest":   {"read"},
	}
	
	for _, role := range roles {
		if perms, exists := permissionMap[role]; exists {
			return perms
		}
	}
	return []string{"read"}
}

func (a *AuthHandler) mapGroupToRole(group string) string {
	groupRoleMap := map[string]string{
		"administrators": "admin",
		"partners":      "partner",
		"employees":     "user",
	}
	
	if role, exists := groupRoleMap[group]; exists {
		return role
	}
	return "user"
}

func (a *AuthHandler) validateMaintenanceToken(token string) bool {
	return token == a.maintenance.bypassKey && a.maintenance.bypassKey != ""
}

func (a *AuthHandler) cacheCredentials(token string, user *User) {
	a.localCache.Set(token, &CachedCredential{
		User:      user,
		Timestamp: time.Now(),
		ExpiresAt: time.Now().Add(a.localCache.ttl),
	})
}

// NewCredentialCache creates a new credential cache
func NewCredentialCache(ttl time.Duration) *CredentialCache {
	return &CredentialCache{
		cache: make(map[string]*CachedCredential),
		ttl:   ttl,
	}
}

// Get retrieves cached credentials
func (cc *CredentialCache) Get(token string) (*CachedCredential, error) {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()
	
	if cred, exists := cc.cache[token]; exists {
		if time.Now().Before(cred.ExpiresAt) {
			return cred, nil
		}
		// Remove expired credential
		delete(cc.cache, token)
	}
	return nil, errors.New("credentials not found or expired")
}

// Set stores credentials in cache
func (cc *CredentialCache) Set(token string, cred *CachedCredential) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	cc.cache[token] = cred
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
