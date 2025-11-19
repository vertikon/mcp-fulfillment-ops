package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)

// Session represents a user session
type Session struct {
	ID        string
	UserID    string
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
	IPAddress string
	UserAgent string
	Active    bool
}

// SessionStore defines interface for session persistence
type SessionStore interface {
	Create(ctx context.Context, session *Session) error
	Get(ctx context.Context, sessionID string) (*Session, error)
	GetByUserID(ctx context.Context, userID string) ([]*Session, error)
	Update(ctx context.Context, session *Session) error
	Delete(ctx context.Context, sessionID string) error
	DeleteByUserID(ctx context.Context, userID string) error
}

// SessionManager handles session operations
type SessionManager interface {
	// Create creates a new session for a user
	Create(ctx context.Context, userID, token, ipAddress, userAgent string) (*Session, error)
	
	// Get retrieves a session by ID
	Get(ctx context.Context, sessionID string) (*Session, error)
	
	// GetByUserID retrieves all active sessions for a user
	GetByUserID(ctx context.Context, userID string) ([]*Session, error)
	
	// Validate checks if session is valid
	Validate(ctx context.Context, sessionID string) (*Session, error)
	
	// Refresh extends session expiration
	Refresh(ctx context.Context, sessionID string) error
	
	// Invalidate invalidates a session
	Invalidate(ctx context.Context, sessionID string) error
	
	// InvalidateAll invalidates all sessions for a user
	InvalidateAll(ctx context.Context, userID string) error
}

// Manager implements SessionManager
type SessionManagerImpl struct {
	store       SessionStore
	sessionTTL  time.Duration
	maxSessions int // Maximum concurrent sessions per user
	logger      *zap.Logger
}

// SessionManagerConfig holds configuration for SessionManager
type SessionManagerConfig struct {
	SessionTTL  time.Duration
	MaxSessions int
}

// NewSessionManager creates a new SessionManager
func NewSessionManager(store SessionStore, config SessionManagerConfig) SessionManager {
	return &SessionManagerImpl{
		store:       store,
		sessionTTL:  config.SessionTTL,
		maxSessions: config.MaxSessions,
		logger:      logger.WithContext(context.TODO()),
	}
}

// Create creates a new session for a user
func (m *SessionManagerImpl) Create(ctx context.Context, userID, token, ipAddress, userAgent string) (*Session, error) {
	// Check max sessions limit
	sessions, err := m.store.GetByUserID(ctx, userID)
	if err == nil && len(sessions) >= m.maxSessions {
		// Invalidate oldest session
		if len(sessions) > 0 {
			oldest := sessions[0]
			for _, s := range sessions {
				if s.CreatedAt.Before(oldest.CreatedAt) {
					oldest = s
				}
			}
			_ = m.store.Delete(ctx, oldest.ID)
		}
	}

	now := time.Now()
	session := &Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     token,
		CreatedAt: now,
		ExpiresAt: now.Add(m.sessionTTL),
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Active:    true,
	}

	if err := m.store.Create(ctx, session); err != nil {
		m.logger.Error("Failed to create session", zap.Error(err))
		return nil, err
	}

	m.logger.Info("Session created",
		zap.String("session_id", session.ID),
		zap.String("user_id", userID),
	)

	return session, nil
}

// Get retrieves a session by ID
func (m *SessionManagerImpl) Get(ctx context.Context, sessionID string) (*Session, error) {
	session, err := m.store.Get(ctx, sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

// GetByUserID retrieves all active sessions for a user
func (m *SessionManagerImpl) GetByUserID(ctx context.Context, userID string) ([]*Session, error) {
	sessions, err := m.store.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Filter active sessions
	activeSessions := make([]*Session, 0)
	for _, s := range sessions {
		if s.Active && time.Now().Before(s.ExpiresAt) {
			activeSessions = append(activeSessions, s)
		}
	}

	return activeSessions, nil
}

// Validate checks if session is valid
func (m *SessionManagerImpl) Validate(ctx context.Context, sessionID string) (*Session, error) {
	session, err := m.store.Get(ctx, sessionID)
	if err != nil {
		m.logger.Warn("Session validation failed: not found",
			zap.String("session_id", sessionID),
		)
		return nil, ErrSessionNotFound
	}

	if !session.Active {
		m.logger.Warn("Session validation failed: inactive",
			zap.String("session_id", sessionID),
		)
		return nil, ErrSessionNotFound
	}

	if time.Now().After(session.ExpiresAt) {
		m.logger.Warn("Session validation failed: expired",
			zap.String("session_id", sessionID),
		)
		// Mark as inactive
		session.Active = false
		_ = m.store.Update(ctx, session)
		return nil, ErrSessionExpired
	}

	return session, nil
}

// Refresh extends session expiration
func (m *SessionManagerImpl) Refresh(ctx context.Context, sessionID string) error {
	session, err := m.Validate(ctx, sessionID)
	if err != nil {
		return err
	}

	session.ExpiresAt = time.Now().Add(m.sessionTTL)
	if err := m.store.Update(ctx, session); err != nil {
		m.logger.Error("Failed to refresh session", zap.Error(err))
		return err
	}

	m.logger.Debug("Session refreshed",
		zap.String("session_id", sessionID),
	)

	return nil
}

// Invalidate invalidates a session
func (m *SessionManagerImpl) Invalidate(ctx context.Context, sessionID string) error {
	session, err := m.store.Get(ctx, sessionID)
	if err != nil {
		return ErrSessionNotFound
	}

	session.Active = false
	if err := m.store.Update(ctx, session); err != nil {
		m.logger.Error("Failed to invalidate session", zap.Error(err))
		return err
	}

	m.logger.Info("Session invalidated",
		zap.String("session_id", sessionID),
	)

	return nil
}

// InvalidateAll invalidates all sessions for a user
func (m *SessionManagerImpl) InvalidateAll(ctx context.Context, userID string) error {
	if err := m.store.DeleteByUserID(ctx, userID); err != nil {
		m.logger.Error("Failed to invalidate all sessions", zap.Error(err))
		return err
	}

	m.logger.Info("All sessions invalidated",
		zap.String("user_id", userID),
	)

	return nil
}
