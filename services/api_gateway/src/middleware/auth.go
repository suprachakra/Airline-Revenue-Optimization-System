package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	TokenGracePeriod    = 30 * time.Second
	CacheFallbackTTL    = 5 * time.Minute
	RevocationCheckURL  = "https://auth.iaros.ai/v3/revocation-list"
	AutoRotationTimeout = 500 * time.Millisecond
)

type AuthHandler struct {
	verifier       *oidc.IDTokenVerifier
	cache          *redis.ClusterClient
	keyAutoRotator *KeyRotationService
	logger         *zap.Logger
}

// ValidateToken implements enhanced JWT validation with multiple fallback layers.
func (a *AuthHandler) ValidateToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := extractToken(r)
		if tokenString == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Primary Validation using current key set.
		token, err := a.verifier.Verify(r.Context(), tokenString)
		if err == nil {
			if isRevoked := checkRevocationList(tokenString); !isRevoked {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Fallback 1: Use cached valid token claims.
		if cachedClaims, err := a.cache.Get(r.Context(), hashToken(tokenString)).Result(); err == nil {
			a.logger.Info("Using cached credentials", zap.String("token_hash", hashToken(tokenString)))
			ctx := context.WithValue(r.Context(), "claims", cachedClaims)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Fallback 2: Validate using previous rotation keys.
		if a.keyAutoRotator.ValidateWithPreviousKeys(tokenString) {
			a.logger.Warn("Used previous rotation key during key update outage")
			next.ServeHTTP(w, r)
			return
		}

		// Final Fallback: Enable limited guest access.
		if enableGuestMode(r) {
			a.logger.Error("Full auth failure - enabling guest mode", zap.Error(err))
			ctx := context.WithValue(r.Context(), "role", "guest")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		respondWithSecurityError(w, r, http.StatusUnauthorized, "AUTH_FAILURE_2025")
	})
}

// RotateKeys periodically fetches new JWKS and updates the verifier.
func (a *AuthHandler) RotateKeys() {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), AutoRotationTimeout)
		defer cancel()
		newVerifier, err := a.keyAutoRotator.FetchLatestVerifier(ctx)
		if err != nil {
			a.logger.Error("Key rotation failed - retaining current keys", zap.Error(err),
				zap.Time("retry_at", time.Now().Add(15*time.Minute)))
			continue
		}
		a.verifier = newVerifier
		a.cache.Set(ctx, "current_jwks_version", newVerifier.Version(), CacheFallbackTTL)
		a.logger.Info("Successfully rotated keys", zap.String("new_version", newVerifier.Version()))
	}
}
