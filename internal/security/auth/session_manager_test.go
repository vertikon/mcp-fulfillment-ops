package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSessionStore is a mock implementation of SessionStore
type MockSessionStore struct {
	mock.Mock
}

func (m *MockSessionStore) Create(ctx context.Context, session *Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Session), args.Error(1)
}

func (m *MockSessionStore) GetByUserID(ctx context.Context, userID string) ([]*Session, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Session), args.Error(1)
}

func (m *MockSessionStore) Update(ctx context.Context, session *Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionStore) Delete(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockSessionStore) DeleteByUserID(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func TestSessionManager_Create(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		token         string
		setupMocks    func(*MockSessionStore)
		expectedError error
	}{
		{
			name:   "successful creation",
			userID: "user123",
			token:  "test_token",
			setupMocks: func(store *MockSessionStore) {
				store.On("GetByUserID", mock.Anything, "user123").Return([]*Session{}, nil)
				store.On("Create", mock.Anything, mock.AnythingOfType("*auth.Session")).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:   "creation with max sessions limit",
			userID: "user123",
			token:  "test_token",
			setupMocks: func(store *MockSessionStore) {
				oldSessions := []*Session{
					{ID: "old1", CreatedAt: time.Now().Add(-2 * time.Hour)},
					{ID: "old2", CreatedAt: time.Now().Add(-1 * time.Hour)},
					{ID: "old3", CreatedAt: time.Now().Add(-30 * time.Minute)},
					{ID: "old4", CreatedAt: time.Now().Add(-15 * time.Minute)},
					{ID: "old5", CreatedAt: time.Now().Add(-5 * time.Minute)},
				}
				store.On("GetByUserID", mock.Anything, "user123").Return(oldSessions, nil)
				store.On("Delete", mock.Anything, "old1").Return(nil)
				store.On("Create", mock.Anything, mock.AnythingOfType("*auth.Session")).Return(nil)
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := new(MockSessionStore)
			tt.setupMocks(mockStore)

			config := SessionManagerConfig{
				SessionTTL:  24 * time.Hour,
				MaxSessions: 5,
			}

			manager := NewSessionManager(mockStore, config)

			session, err := manager.Create(context.Background(), tt.userID, tt.token, "127.0.0.1", "test-agent")

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Nil(t, session)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, session)
				assert.Equal(t, tt.userID, session.UserID)
				assert.Equal(t, tt.token, session.Token)
				assert.True(t, session.Active)
			}

			mockStore.AssertExpectations(t)
		})
	}
}

func TestSessionManager_Validate(t *testing.T) {
	tests := []struct {
		name          string
		sessionID     string
		setupMocks    func(*MockSessionStore)
		expectedError error
	}{
		{
			name:      "valid session",
			sessionID: "session123",
			setupMocks: func(store *MockSessionStore) {
				session := &Session{
					ID:        "session123",
					UserID:    "user123",
					Active:    true,
					ExpiresAt: time.Now().Add(1 * time.Hour),
				}
				store.On("Get", mock.Anything, "session123").Return(session, nil)
			},
			expectedError: nil,
		},
		{
			name:      "expired session",
			sessionID: "session123",
			setupMocks: func(store *MockSessionStore) {
				session := &Session{
					ID:        "session123",
					UserID:    "user123",
					Active:    true,
					ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
				}
				store.On("Get", mock.Anything, "session123").Return(session, nil)
				store.On("Update", mock.Anything, mock.MatchedBy(func(s *Session) bool {
					return !s.Active
				})).Return(nil)
			},
			expectedError: ErrSessionExpired,
		},
		{
			name:      "inactive session",
			sessionID: "session123",
			setupMocks: func(store *MockSessionStore) {
				session := &Session{
					ID:        "session123",
					UserID:    "user123",
					Active:    false,
					ExpiresAt: time.Now().Add(1 * time.Hour),
				}
				store.On("Get", mock.Anything, "session123").Return(session, nil)
			},
			expectedError: ErrSessionNotFound,
		},
		{
			name:      "session not found",
			sessionID: "nonexistent",
			setupMocks: func(store *MockSessionStore) {
				store.On("Get", mock.Anything, "nonexistent").Return(nil, ErrSessionNotFound)
			},
			expectedError: ErrSessionNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := new(MockSessionStore)
			tt.setupMocks(mockStore)

			config := SessionManagerConfig{
				SessionTTL:  24 * time.Hour,
				MaxSessions: 5,
			}

			manager := NewSessionManager(mockStore, config)

			session, err := manager.Validate(context.Background(), tt.sessionID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, session)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, session)
			}

			mockStore.AssertExpectations(t)
		})
	}
}

func TestSessionManager_Refresh(t *testing.T) {
	mockStore := new(MockSessionStore)

	session := &Session{
		ID:        "session123",
		UserID:    "user123",
		Active:    true,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	mockStore.On("Get", mock.Anything, "session123").Return(session, nil)
	mockStore.On("Update", mock.Anything, mock.MatchedBy(func(s *Session) bool {
		return s.ID == "session123" && s.ExpiresAt.After(time.Now())
	})).Return(nil)

	config := SessionManagerConfig{
		SessionTTL:  24 * time.Hour,
		MaxSessions: 5,
	}

	manager := NewSessionManager(mockStore, config)

	err := manager.Refresh(context.Background(), "session123")

	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestSessionManager_Invalidate(t *testing.T) {
	mockStore := new(MockSessionStore)

	session := &Session{
		ID:     "session123",
		UserID: "user123",
		Active: true,
	}

	mockStore.On("Get", mock.Anything, "session123").Return(session, nil)
	mockStore.On("Update", mock.Anything, mock.MatchedBy(func(s *Session) bool {
		return s.ID == "session123" && !s.Active
	})).Return(nil)

	config := SessionManagerConfig{
		SessionTTL:  24 * time.Hour,
		MaxSessions: 5,
	}

	manager := NewSessionManager(mockStore, config)

	err := manager.Invalidate(context.Background(), "session123")

	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestSessionManager_InvalidateAll(t *testing.T) {
	mockStore := new(MockSessionStore)

	mockStore.On("DeleteByUserID", mock.Anything, "user123").Return(nil)

	config := SessionManagerConfig{
		SessionTTL:  24 * time.Hour,
		MaxSessions: 5,
	}

	manager := NewSessionManager(mockStore, config)

	err := manager.InvalidateAll(context.Background(), "user123")

	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestSessionManager_GetByUserID(t *testing.T) {
	mockStore := new(MockSessionStore)

	now := time.Now()
	sessions := []*Session{
		{ID: "session1", UserID: "user123", Active: true, ExpiresAt: now.Add(1 * time.Hour)},
		{ID: "session2", UserID: "user123", Active: false, ExpiresAt: now.Add(1 * time.Hour)},
		{ID: "session3", UserID: "user123", Active: true, ExpiresAt: now.Add(-1 * time.Hour)}, // Expired
		{ID: "session4", UserID: "user123", Active: true, ExpiresAt: now.Add(2 * time.Hour)},
	}

	mockStore.On("GetByUserID", mock.Anything, "user123").Return(sessions, nil)

	config := SessionManagerConfig{
		SessionTTL:  24 * time.Hour,
		MaxSessions: 5,
	}

	manager := NewSessionManager(mockStore, config)

	activeSessions, err := manager.GetByUserID(context.Background(), "user123")

	assert.NoError(t, err)
	// Should return session1 and session4 (both active and not expired)
	assert.Len(t, activeSessions, 2)
	// Verify both are in the result
	sessionIDs := make(map[string]bool)
	for _, s := range activeSessions {
		sessionIDs[s.ID] = true
	}
	assert.True(t, sessionIDs["session1"])
	assert.True(t, sessionIDs["session4"])
	assert.False(t, sessionIDs["session2"]) // Inactive
	assert.False(t, sessionIDs["session3"]) // Expired
}
