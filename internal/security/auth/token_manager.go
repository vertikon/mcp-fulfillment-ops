package auth

import (
	"context"
	"crypto/rsa"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

var (
	ErrInvalidToken   = errors.New("invalid token")
	ErrExpiredToken   = errors.New("token expired")
	ErrTokenSignature = errors.New("invalid token signature")
)

// TokenClaims represents JWT claims
type TokenClaims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

// TokenManager handles JWT token operations
type TokenManager interface {
	// Generate creates a new JWT token
	Generate(ctx context.Context, userID, email string, roles []string) (string, error)

	// Validate validates a JWT token and returns user ID
	Validate(ctx context.Context, token string) (string, error)

	// Refresh generates a new token from an existing one
	Refresh(ctx context.Context, token string) (string, error)

	// Revoke invalidates a token
	Revoke(ctx context.Context, token string) error
}

// Manager implements TokenManager
type TokenManagerImpl struct {
	secretKey     []byte
	signingMethod jwt.SigningMethod
	tokenTTL      time.Duration
	refreshTTL    time.Duration
	revokedTokens map[string]time.Time // In-memory revocation list (use Redis in production)
	logger        *zap.Logger
	rsaPrivateKey *rsa.PrivateKey // For RS256 signing
}

// TokenManagerConfig holds configuration for TokenManager
type TokenManagerConfig struct {
	SecretKey     string
	SigningMethod string // "HS256" or "RS256"
	TokenTTL      time.Duration
	RefreshTTL    time.Duration
}

// NewTokenManager creates a new TokenManager
func NewTokenManager(config TokenManagerConfig) TokenManager {
	var signingMethod jwt.SigningMethod = jwt.SigningMethodHS256
	var rsaPrivateKey *rsa.PrivateKey
	if config.SigningMethod == "RS256" {
		signingMethod = jwt.SigningMethodRS256
		// For RS256, we'd need to load/generate RSA keys
		// For now, we'll use HS256 as default
	}

	return &TokenManagerImpl{
		secretKey:     []byte(config.SecretKey),
		signingMethod: signingMethod,
		tokenTTL:      config.TokenTTL,
		refreshTTL:    config.RefreshTTL,
		revokedTokens: make(map[string]time.Time),
		logger:        logger.WithContext(context.TODO()),
		rsaPrivateKey: rsaPrivateKey,
	}
}

// Generate creates a new JWT token
func (m *TokenManagerImpl) Generate(ctx context.Context, userID, email string, roles []string) (string, error) {
	now := time.Now()
	claims := TokenClaims{
		UserID: userID,
		Email:  email,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.tokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "mcp-fulfillment-ops",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(m.signingMethod, claims)
	tokenString, err := token.SignedString(m.secretKey)
	if err != nil {
		m.logger.Error("Failed to sign token", zap.Error(err))
		return "", err
	}

	m.logger.Debug("Token generated",
		zap.String("user_id", userID),
		zap.String("email", email),
	)

	return tokenString, nil
}

// Validate validates a JWT token and returns user ID
func (m *TokenManagerImpl) Validate(ctx context.Context, tokenString string) (string, error) {
	// Check if token is revoked
	if _, revoked := m.revokedTokens[tokenString]; revoked {
		m.logger.Warn("Token validation failed: token revoked",
			zap.String("token", tokenString[:20]+"..."),
		)
		return "", ErrInvalidToken
	}

	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenSignature
		}
		return m.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			m.logger.Warn("Token validation failed: token expired")
			return "", ErrExpiredToken
		}
		m.logger.Warn("Token validation failed",
			zap.Error(err),
		)
		return "", ErrInvalidToken
	}

	if !token.Valid {
		m.logger.Warn("Token validation failed: invalid token")
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok {
		m.logger.Warn("Token validation failed: invalid claims")
		return "", ErrInvalidToken
	}

	m.logger.Debug("Token validated successfully",
		zap.String("user_id", claims.UserID),
	)

	return claims.UserID, nil
}

// Refresh generates a new token from an existing one
func (m *TokenManagerImpl) Refresh(ctx context.Context, tokenString string) (string, error) {
	// Validate existing token
	_, err := m.Validate(ctx, tokenString)
	if err != nil {
		return "", err
	}

	// Parse to get claims
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return m.secretKey, nil
	})

	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok {
		return "", ErrInvalidToken
	}

	// Generate new token with same user info
	newToken, err := m.Generate(ctx, claims.UserID, claims.Email, claims.Roles)
	if err != nil {
		return "", err
	}

	// Revoke old token
	_ = m.Revoke(ctx, tokenString)

	return newToken, nil
}

// Revoke invalidates a token
func (m *TokenManagerImpl) Revoke(ctx context.Context, tokenString string) error {
	m.revokedTokens[tokenString] = time.Now()

	// Cleanup old revoked tokens (older than refresh TTL)
	for token, revokedAt := range m.revokedTokens {
		if time.Since(revokedAt) > m.refreshTTL {
			delete(m.revokedTokens, token)
		}
	}

	m.logger.Debug("Token revoked",
		zap.String("token", tokenString[:20]+"..."),
	)

	return nil
}
