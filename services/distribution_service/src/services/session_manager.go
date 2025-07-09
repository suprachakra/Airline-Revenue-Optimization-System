package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"iaros/distribution_service/src/models"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// SessionManager manages NDC and GDS sessions
type SessionManager struct {
	db          *gorm.DB
	redisClient *redis.Client
}

// NewSessionManager creates a new session manager
func NewSessionManager(db *gorm.DB, redisClient *redis.Client) *SessionManager {
	return &SessionManager{
		db:          db,
		redisClient: redisClient,
	}
}

// CreateNDCSession creates a new NDC session
func (sm *SessionManager) CreateNDCSession(ctx context.Context, customerID, airlineCode string) (*models.NDCSession, error) {
	session := &models.NDCSession{
		SessionID:      uuid.New().String(),
		CustomerID:     customerID,
		AirlineCode:    airlineCode,
		Status:         models.NDCSessionActive,
		NDCVersion:     "20.3",
		CreatedAt:      time.Now().UTC(),
		LastAccessedAt: time.Now().UTC(),
		ExpiresAt:      time.Now().Add(4 * time.Hour),
		TTL:            4 * time.Hour,
	}

	// Save to database
	if err := sm.db.Create(session).Error; err != nil {
		return nil, fmt.Errorf("failed to create NDC session: %w", err)
	}

	// Cache in Redis
	if err := sm.cacheNDCSession(ctx, session); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: Failed to cache NDC session: %v\n", err)
	}

	return session, nil
}

// GetNDCSession retrieves an NDC session
func (sm *SessionManager) GetNDCSession(ctx context.Context, sessionID string) (*models.NDCSession, error) {
	// Try Redis first
	session, err := sm.getNDCSessionFromCache(ctx, sessionID)
	if err == nil && session != nil {
		return session, nil
	}

	// Fall back to database
	session = &models.NDCSession{}
	if err := sm.db.Where("session_id = ?", sessionID).First(session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("NDC session not found")
		}
		return nil, fmt.Errorf("failed to get NDC session: %w", err)
	}

	// Check if session is expired
	if session.ExpiresAt.Before(time.Now()) {
		session.Status = models.NDCSessionExpired
		sm.db.Save(session)
		return nil, fmt.Errorf("NDC session expired")
	}

	// Update last accessed time
	session.LastAccessedAt = time.Now().UTC()
	sm.db.Save(session)

	// Cache it
	sm.cacheNDCSession(ctx, session)

	return session, nil
}

// UpdateNDCShoppingContext updates shopping context in NDC session
func (sm *SessionManager) UpdateNDCShoppingContext(ctx context.Context, sessionID string, shoppingResponse *models.AirShoppingRS) error {
	session, err := sm.GetNDCSession(ctx, sessionID)
	if err != nil {
		return err
	}

	contextData, err := json.Marshal(shoppingResponse)
	if err != nil {
		return fmt.Errorf("failed to marshal shopping context: %w", err)
	}

	session.ShoppingContext = string(contextData)
	session.LastAccessedAt = time.Now().UTC()

	if err := sm.db.Save(session).Error; err != nil {
		return fmt.Errorf("failed to update NDC session: %w", err)
	}

	// Update cache
	sm.cacheNDCSession(ctx, session)

	return nil
}

// CreateGDSSession creates a new GDS session
func (sm *SessionManager) CreateGDSSession(ctx context.Context, provider models.GDSProvider, userID, pseudoCity string) (*models.GDSSession, error) {
	session := &models.GDSSession{
		SessionID:      uuid.New().String(),
		Provider:       provider,
		PseudoCity:     pseudoCity,
		UserID:         userID,
		Status:         models.GDSSessionActive,
		LastActivity:   time.Now().UTC(),
		CreatedAt:      time.Now().UTC(),
		ExpiresAt:      time.Now().Add(2 * time.Hour),
	}

	// Save to database
	if err := sm.db.Create(session).Error; err != nil {
		return nil, fmt.Errorf("failed to create GDS session: %w", err)
	}

	// Cache in Redis
	if err := sm.cacheGDSSession(ctx, session); err != nil {
		fmt.Printf("Warning: Failed to cache GDS session: %v\n", err)
	}

	return session, nil
}

// GetOrCreateGDSSession gets existing or creates new GDS session
func (sm *SessionManager) GetOrCreateGDSSession(ctx context.Context, provider models.GDSProvider, userID, pseudoCity string) (*models.GDSSession, error) {
	// Try to find existing active session
	session := &models.GDSSession{}
	err := sm.db.Where("provider = ? AND user_id = ? AND pseudo_city = ? AND status = ? AND expires_at > ?",
		provider, userID, pseudoCity, models.GDSSessionActive, time.Now()).
		First(session).Error

	if err == nil {
		// Update last activity
		session.LastActivity = time.Now().UTC()
		sm.db.Save(session)
		sm.cacheGDSSession(ctx, session)
		return session, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to query GDS session: %w", err)
	}

	// Create new session
	return sm.CreateGDSSession(ctx, provider, userID, pseudoCity)
}

// UpdateGDSSessionAuth updates GDS session authentication
func (sm *SessionManager) UpdateGDSSessionAuth(ctx context.Context, sessionID, authToken, refreshToken string, expiresAt time.Time) error {
	session := &models.GDSSession{}
	if err := sm.db.Where("session_id = ?", sessionID).First(session).Error; err != nil {
		return fmt.Errorf("failed to find GDS session: %w", err)
	}

	session.AuthToken = authToken
	session.RefreshToken = refreshToken
	session.TokenExpiresAt = expiresAt
	session.Status = models.GDSSessionAuthenticated
	session.LastActivity = time.Now().UTC()

	if err := sm.db.Save(session).Error; err != nil {
		return fmt.Errorf("failed to update GDS session auth: %w", err)
	}

	// Update cache
	sm.cacheGDSSession(ctx, session)

	return nil
}

// ExpireSession expires a session
func (sm *SessionManager) ExpireSession(ctx context.Context, sessionID string, sessionType string) error {
	if sessionType == "NDC" {
		session := &models.NDCSession{}
		if err := sm.db.Where("session_id = ?", sessionID).First(session).Error; err != nil {
			return fmt.Errorf("failed to find NDC session: %w", err)
		}

		session.Status = models.NDCSessionExpired
		if err := sm.db.Save(session).Error; err != nil {
			return fmt.Errorf("failed to expire NDC session: %w", err)
		}

		// Remove from cache
		sm.redisClient.Del(ctx, fmt.Sprintf("ndc_session:%s", sessionID))
	} else if sessionType == "GDS" {
		session := &models.GDSSession{}
		if err := sm.db.Where("session_id = ?", sessionID).First(session).Error; err != nil {
			return fmt.Errorf("failed to find GDS session: %w", err)
		}

		session.Status = models.GDSSessionExpired
		if err := sm.db.Save(session).Error; err != nil {
			return fmt.Errorf("failed to expire GDS session: %w", err)
		}

		// Remove from cache
		sm.redisClient.Del(ctx, fmt.Sprintf("gds_session:%s", sessionID))
	}

	return nil
}

// CleanupExpiredSessions removes expired sessions
func (sm *SessionManager) CleanupExpiredSessions(ctx context.Context) error {
	now := time.Now().UTC()

	// Cleanup NDC sessions
	if err := sm.db.Where("expires_at < ?", now).Delete(&models.NDCSession{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup expired NDC sessions: %w", err)
	}

	// Cleanup GDS sessions
	if err := sm.db.Where("expires_at < ?", now).Delete(&models.GDSSession{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup expired GDS sessions: %w", err)
	}

	return nil
}

// GetSessionStats returns session statistics
func (sm *SessionManager) GetSessionStats(ctx context.Context) (map[string]interface{}, error) {
	var ndcActive, ndcExpired, ndcTotal int64
	var gdsActive, gdsExpired, gdsTotal int64

	// NDC session stats
	sm.db.Model(&models.NDCSession{}).Where("status = ?", models.NDCSessionActive).Count(&ndcActive)
	sm.db.Model(&models.NDCSession{}).Where("status = ?", models.NDCSessionExpired).Count(&ndcExpired)
	sm.db.Model(&models.NDCSession{}).Count(&ndcTotal)

	// GDS session stats
	sm.db.Model(&models.GDSSession{}).Where("status = ?", models.GDSSessionActive).Count(&gdsActive)
	sm.db.Model(&models.GDSSession{}).Where("status = ?", models.GDSSessionExpired).Count(&gdsExpired)
	sm.db.Model(&models.GDSSession{}).Count(&gdsTotal)

	return map[string]interface{}{
		"ndc_sessions": map[string]interface{}{
			"active":  ndcActive,
			"expired": ndcExpired,
			"total":   ndcTotal,
		},
		"gds_sessions": map[string]interface{}{
			"active":  gdsActive,
			"expired": gdsExpired,
			"total":   gdsTotal,
		},
	}, nil
}

// Private methods for caching

func (sm *SessionManager) cacheNDCSession(ctx context.Context, session *models.NDCSession) error {
	if sm.redisClient == nil {
		return nil
	}

	key := fmt.Sprintf("ndc_session:%s", session.SessionID)
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return sm.redisClient.Set(ctx, key, data, session.TTL).Err()
}

func (sm *SessionManager) getNDCSessionFromCache(ctx context.Context, sessionID string) (*models.NDCSession, error) {
	if sm.redisClient == nil {
		return nil, fmt.Errorf("redis client not available")
	}

	key := fmt.Sprintf("ndc_session:%s", sessionID)
	data, err := sm.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var session models.NDCSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (sm *SessionManager) cacheGDSSession(ctx context.Context, session *models.GDSSession) error {
	if sm.redisClient == nil {
		return nil
	}

	key := fmt.Sprintf("gds_session:%s", session.SessionID)
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	ttl := time.Until(session.ExpiresAt)
	return sm.redisClient.Set(ctx, key, data, ttl).Err()
}

func (sm *SessionManager) getGDSSessionFromCache(ctx context.Context, sessionID string) (*models.GDSSession, error) {
	if sm.redisClient == nil {
		return nil, fmt.Errorf("redis client not available")
	}

	key := fmt.Sprintf("gds_session:%s", sessionID)
	data, err := sm.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var session models.GDSSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}

	return &session, nil
} 