package auth

import (
	"context"
	"sync"
	"time"
)

// InMemorySessionStore is an in-memory implementation of SessionStore
type InMemorySessionStore struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewInMemorySessionStore creates a new in-memory session store
func NewInMemorySessionStore() SessionStore {
	return &InMemorySessionStore{
		sessions: make(map[string]*Session),
	}
}

// Create creates a new session
func (s *InMemorySessionStore) Create(ctx context.Context, session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

// Get retrieves a session by ID
func (s *InMemorySessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

// GetByUserID retrieves all sessions for a user
func (s *InMemorySessionStore) GetByUserID(ctx context.Context, userID string) ([]*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var sessions []*Session
	for _, session := range s.sessions {
		if session.UserID == userID {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

// Update updates a session
func (s *InMemorySessionStore) Update(ctx context.Context, session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[session.ID]; !ok {
		return ErrSessionNotFound
	}
	s.sessions[session.ID] = session
	return nil
}

// Delete deletes a session
func (s *InMemorySessionStore) Delete(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
	return nil
}

// DeleteByUserID deletes all sessions for a user
func (s *InMemorySessionStore) DeleteByUserID(ctx context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, session := range s.sessions {
		if session.UserID == userID {
			delete(s.sessions, id)
		}
	}
	return nil
}

// CleanupExpired removes expired sessions (helper method)
func (s *InMemorySessionStore) CleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for id, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, id)
		}
	}
}
