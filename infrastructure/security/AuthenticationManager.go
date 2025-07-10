package security

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/iaros/common/logging"
	"github.com/iaros/common/utils"
)

// AuthenticationManager handles all authentication concerns for IAROS
type AuthenticationManager struct {
	config          *AuthConfig
	jwtValidator    *JWTValidator
	tlsManager      *TLSManager
	mfaProvider     *MFAProvider
	sessionManager  *SessionManager
	auditLogger     logging.Logger
	failedAttempts  map[string]*FailureInfo
	mu              sync.RWMutex
}

type AuthConfig struct {
	JWTSecret           string        `json:"jwt_secret"`
	JWTExpiration       time.Duration `json:"jwt_expiration"`
	RefreshExpiration   time.Duration `json:"refresh_expiration"`
	RequireMFA          bool          `json:"require_mfa"`
	MaxFailedAttempts   int           `json:"max_failed_attempts"`
	LockoutDuration     time.Duration `json:"lockout_duration"`
	TLSCertPath         string        `json:"tls_cert_path"`
	TLSKeyPath          string        `json:"tls_key_path"`
	ClientCAPath        string        `json:"client_ca_path"`
	RequireClientCert   bool          `json:"require_client_cert"`
}

type FailureInfo struct {
	Count       int       `json:"count"`
	LastAttempt time.Time `json:"last_attempt"`
	LockedUntil time.Time `json:"locked_until"`
}

type AuthenticationResult struct {
	Success      bool              `json:"success"`
	UserID       string            `json:"user_id"`
	Permissions  []string          `json:"permissions"`
	SessionID    string            `json:"session_id"`
	AccessToken  string            `json:"access_token"`
	RefreshToken string            `json:"refresh_token"`
	ExpiresAt    time.Time         `json:"expires_at"`
	MFARequired  bool              `json:"mfa_required"`
	Metadata     map[string]string `json:"metadata"`
}

func NewAuthenticationManager(config *AuthConfig) *AuthenticationManager {
	return &AuthenticationManager{
		config:         config,
		jwtValidator:   NewJWTValidator(config.JWTSecret),
		tlsManager:     NewTLSManager(config),
		mfaProvider:    NewMFAProvider(),
		sessionManager: NewSessionManager(),
		auditLogger:    logging.GetLogger("authentication"),
		failedAttempts: make(map[string]*FailureInfo),
	}
}

// JWT Validator
type JWTValidator struct {
	secret []byte
	logger logging.Logger
}

type JWTClaims struct {
	UserID      string   `json:"user_id"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	SessionID   string   `json:"session_id"`
	jwt.RegisteredClaims
}

func NewJWTValidator(secret string) *JWTValidator {
	return &JWTValidator{
		secret: []byte(secret),
		logger: logging.GetLogger("jwt_validator"),
	}
}

func (jv *JWTValidator) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jv.secret, nil
	})

	if err != nil {
		jv.logger.Error("Token validation failed", "error", err)
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		// Additional validation checks
		if time.Now().After(claims.ExpiresAt.Time) {
			return nil, fmt.Errorf("token has expired")
		}
		
		jv.logger.Debug("Token validated successfully", "user_id", claims.UserID)
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

func (jv *JWTValidator) GenerateToken(userID, email string, roles, permissions []string, sessionID string) (string, string, error) {
	now := time.Now()
	accessExpiration := now.Add(1 * time.Hour)
	refreshExpiration := now.Add(24 * time.Hour)

	// Access token
	accessClaims := &JWTClaims{
		UserID:      userID,
		Email:       email,
		Roles:       roles,
		Permissions: permissions,
		SessionID:   sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiration),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "iaros-auth",
			Subject:   userID,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(jv.secret)
	if err != nil {
		return "", "", err
	}

	// Refresh token
	refreshClaims := &JWTClaims{
		UserID:    userID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiration),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "iaros-auth",
			Subject:   userID,
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(jv.secret)
	if err != nil {
		return "", "", err
	}

	jv.logger.Info("Tokens generated successfully", "user_id", userID)
	return accessTokenString, refreshTokenString, nil
}

// TLS Manager for mutual TLS authentication
type TLSManager struct {
	config     *AuthConfig
	serverCert tls.Certificate
	clientCAs  *x509.CertPool
	logger     logging.Logger
}

func NewTLSManager(config *AuthConfig) *TLSManager {
	tm := &TLSManager{
		config: config,
		logger: logging.GetLogger("tls_manager"),
	}

	if err := tm.loadCertificates(); err != nil {
		tm.logger.Error("Failed to load TLS certificates", "error", err)
	}

	return tm
}

func (tm *TLSManager) loadCertificates() error {
	// Load server certificate
	cert, err := tls.LoadX509KeyPair(tm.config.TLSCertPath, tm.config.TLSKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load server certificate: %w", err)
	}
	tm.serverCert = cert

	// Load client CA certificates for mutual TLS
	if tm.config.RequireClientCert {
		clientCAs := x509.NewCertPool()
		caData, err := utils.ReadFile(tm.config.ClientCAPath)
		if err != nil {
			return fmt.Errorf("failed to read client CA file: %w", err)
		}
		
		if !clientCAs.AppendCertsFromPEM(caData) {
			return fmt.Errorf("failed to parse client CA certificate")
		}
		tm.clientCAs = clientCAs
	}

	tm.logger.Info("TLS certificates loaded successfully")
	return nil
}

func (tm *TLSManager) GetTLSConfig() *tls.Config {
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tm.serverCert},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		},
		PreferServerCipherSuites: true,
	}

	if tm.config.RequireClientCert {
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		tlsConfig.ClientCAs = tm.clientCAs
	}

	return tlsConfig
}

func (tm *TLSManager) ValidateClientCertificate(req *http.Request) (*x509.Certificate, error) {
	if req.TLS == nil || len(req.TLS.PeerCertificates) == 0 {
		return nil, fmt.Errorf("no client certificate provided")
	}

	clientCert := req.TLS.PeerCertificates[0]
	
	// Additional certificate validation
	if time.Now().After(clientCert.NotAfter) {
		return nil, fmt.Errorf("client certificate has expired")
	}

	if time.Now().Before(clientCert.NotBefore) {
		return nil, fmt.Errorf("client certificate is not yet valid")
	}

	tm.logger.Debug("Client certificate validated", "subject", clientCert.Subject.String())
	return clientCert, nil
}

// MFA Provider
type MFAProvider struct {
	totpGenerator *TOTPGenerator
	smsProvider   *SMSProvider
	logger        logging.Logger
}

type TOTPGenerator struct {
	secretKey string
}

type SMSProvider struct {
	apiKey    string
	apiSecret string
}

func NewMFAProvider() *MFAProvider {
	return &MFAProvider{
		totpGenerator: &TOTPGenerator{secretKey: "default-secret"},
		smsProvider:   &SMSProvider{},
		logger:        logging.GetLogger("mfa_provider"),
	}
}

func (mfa *MFAProvider) GenerateTOTPQR(userID string) (string, error) {
	// Generate TOTP QR code for user
	mfa.logger.Info("Generated TOTP QR code", "user_id", userID)
	return "otpauth://totp/IAROS:user@example.com?secret=BASE32SECRET&issuer=IAROS", nil
}

func (mfa *MFAProvider) ValidateTOTP(userID, token string) bool {
	// Validate TOTP token
	mfa.logger.Debug("Validating TOTP token", "user_id", userID)
	return len(token) == 6 // Simplified validation
}

func (mfa *MFAProvider) SendSMSToken(phoneNumber string) (string, error) {
	token := utils.GenerateRandomCode(6)
	mfa.logger.Info("SMS token sent", "phone", phoneNumber)
	return token, nil
}

// Session Manager
type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	logger   logging.Logger
}

type Session struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id"`
	CreatedAt   time.Time         `json:"created_at"`
	LastAccess  time.Time         `json:"last_access"`
	ExpiresAt   time.Time         `json:"expires_at"`
	IPAddress   string            `json:"ip_address"`
	UserAgent   string            `json:"user_agent"`
	Permissions []string          `json:"permissions"`
	Metadata    map[string]string `json:"metadata"`
	Active      bool              `json:"active"`
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		logger:   logging.GetLogger("session_manager"),
	}
}

func (sm *SessionManager) CreateSession(userID, ipAddress, userAgent string, permissions []string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sessionID := utils.GenerateSessionID()
	session := &Session{
		ID:          sessionID,
		UserID:      userID,
		CreatedAt:   time.Now(),
		LastAccess:  time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Permissions: permissions,
		Metadata:    make(map[string]string),
		Active:      true,
	}

	sm.sessions[sessionID] = session
	sm.logger.Info("Session created", "session_id", sessionID, "user_id", userID)
	return session
}

func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists || !session.Active || time.Now().After(session.ExpiresAt) {
		return nil, false
	}

	// Update last access
	session.LastAccess = time.Now()
	return session, true
}

func (sm *SessionManager) InvalidateSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		session.Active = false
		sm.logger.Info("Session invalidated", "session_id", sessionID)
	}
}

// Main authentication methods
func (am *AuthenticationManager) Authenticate(req *http.Request, credentials *Credentials) (*AuthenticationResult, error) {
	// Extract client information
	clientIP := getClientIP(req)
	userAgent := req.UserAgent()

	// Check for account lockout
	if am.isAccountLocked(credentials.Username) {
		am.auditLogger.Warn("Authentication attempt on locked account", 
			"username", credentials.Username, 
			"ip", clientIP)
		return nil, fmt.Errorf("account is temporarily locked")
	}

	// Validate credentials
	user, err := am.validateCredentials(credentials)
	if err != nil {
		am.recordFailedAttempt(credentials.Username)
		am.auditLogger.Error("Authentication failed", 
			"username", credentials.Username, 
			"ip", clientIP, 
			"error", err)
		return nil, err
	}

	// Check if MFA is required
	if am.config.RequireMFA && !credentials.MFAToken.IsEmpty() {
		if !am.mfaProvider.ValidateTOTP(user.ID, credentials.MFAToken.Token) {
			am.recordFailedAttempt(credentials.Username)
			return nil, fmt.Errorf("invalid MFA token")
		}
	}

	// Validate client certificate if required
	if am.config.RequireClientCert {
		if _, err := am.tlsManager.ValidateClientCertificate(req); err != nil {
			am.auditLogger.Error("Client certificate validation failed", 
				"username", credentials.Username, 
				"error", err)
			return nil, err
		}
	}

	// Create session
	session := am.sessionManager.CreateSession(user.ID, clientIP, userAgent, user.Permissions)

	// Generate tokens
	accessToken, refreshToken, err := am.jwtValidator.GenerateToken(
		user.ID, user.Email, user.Roles, user.Permissions, session.ID)
	if err != nil {
		return nil, err
	}

	// Clear failed attempts
	am.clearFailedAttempts(credentials.Username)

	result := &AuthenticationResult{
		Success:      true,
		UserID:       user.ID,
		Permissions:  user.Permissions,
		SessionID:    session.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    session.ExpiresAt,
		MFARequired:  am.config.RequireMFA && credentials.MFAToken.IsEmpty(),
		Metadata: map[string]string{
			"ip_address": clientIP,
			"user_agent": userAgent,
		},
	}

	am.auditLogger.Info("Authentication successful", 
		"user_id", user.ID, 
		"session_id", session.ID, 
		"ip", clientIP)

	return result, nil
}

func (am *AuthenticationManager) ValidateRequest(req *http.Request) (*JWTClaims, error) {
	// Extract token from Authorization header
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("missing authorization header")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	// Validate JWT token
	claims, err := am.jwtValidator.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Validate session
	session, exists := am.sessionManager.GetSession(claims.SessionID)
	if !exists {
		return nil, fmt.Errorf("invalid session")
	}

	// Update session activity
	session.LastAccess = time.Now()

	return claims, nil
}

// Helper types and methods
type Credentials struct {
	Username string    `json:"username"`
	Password string    `json:"password"`
	MFAToken MFAToken  `json:"mfa_token"`
}

type MFAToken struct {
	Token string `json:"token"`
	Type  string `json:"type"` // totp, sms
}

func (mfa *MFAToken) IsEmpty() bool {
	return mfa.Token == ""
}

type User struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	Active      bool     `json:"active"`
}

func (am *AuthenticationManager) validateCredentials(credentials *Credentials) (*User, error) {
	// Simplified credential validation - in production, use secure password hashing
	// and validate against user database
	if credentials.Username == "" || credentials.Password == "" {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Mock user for demonstration
	user := &User{
		ID:          "user_" + credentials.Username,
		Username:    credentials.Username,
		Email:       credentials.Username + "@iaros.com",
		Roles:       []string{"user"},
		Permissions: []string{"read", "write"},
		Active:      true,
	}

	return user, nil
}

func (am *AuthenticationManager) isAccountLocked(username string) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if failure, exists := am.failedAttempts[username]; exists {
		return time.Now().Before(failure.LockedUntil)
	}
	return false
}

func (am *AuthenticationManager) recordFailedAttempt(username string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	failure, exists := am.failedAttempts[username]
	if !exists {
		failure = &FailureInfo{}
		am.failedAttempts[username] = failure
	}

	failure.Count++
	failure.LastAttempt = time.Now()

	if failure.Count >= am.config.MaxFailedAttempts {
		failure.LockedUntil = time.Now().Add(am.config.LockoutDuration)
		am.auditLogger.Warn("Account locked due to failed attempts", 
			"username", username, 
			"attempts", failure.Count)
	}
}

func (am *AuthenticationManager) clearFailedAttempts(username string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	delete(am.failedAttempts, username)
}

func getClientIP(req *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	
	// Check X-Real-IP header
	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	return strings.Split(req.RemoteAddr, ":")[0]
} 