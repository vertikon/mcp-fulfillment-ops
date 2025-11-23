package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTokenManager is a mock implementation of TokenManager
type MockTokenManager struct {
	mock.Mock
}

func (m *MockTokenManager) Generate(ctx context.Context, userID, email string, roles []string) (string, error) {
	args := m.Called(ctx, userID, email, roles)
	return args.String(0), args.Error(1)
}

func (m *MockTokenManager) Validate(ctx context.Context, token string) (string, error) {
	args := m.Called(ctx, token)
	return args.String(0), args.Error(1)
}

func (m *MockTokenManager) Refresh(ctx context.Context, token string) (string, error) {
	args := m.Called(ctx, token)
	return args.String(0), args.Error(1)
}

func (m *MockTokenManager) Revoke(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

// MockSessionManager is a mock implementation of SessionManager
type MockSessionManager struct {
	mock.Mock
}

func (m *MockSessionManager) Create(ctx context.Context, userID, token, ipAddress, userAgent string) (*Session, error) {
	args := m.Called(ctx, userID, token, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Session), args.Error(1)
}

func (m *MockSessionManager) Get(ctx context.Context, sessionID string) (*Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Session), args.Error(1)
}

func (m *MockSessionManager) GetByUserID(ctx context.Context, userID string) ([]*Session, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Session), args.Error(1)
}

func (m *MockSessionManager) Validate(ctx context.Context, sessionID string) (*Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Session), args.Error(1)
}

func (m *MockSessionManager) Refresh(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockSessionManager) Invalidate(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockSessionManager) InvalidateAll(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// MockRBACManager is a mock implementation of RBACManager
type MockRBACManager struct {
	mock.Mock
}

func (m *MockRBACManager) HasPermission(userID string, resource string, action string) bool {
	args := m.Called(userID, resource, action)
	return args.Bool(0)
}

func (m *MockRBACManager) GetUserRoles(userID string) ([]string, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// MockUserStore is a mock implementation of UserStore
type MockUserStore struct {
	mock.Mock
}

func (m *MockUserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockUserStore) GetByID(ctx context.Context, userID string) (*User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockUserStore) Create(ctx context.Context, user *User, passwordHash string) error {
	args := m.Called(ctx, user, passwordHash)
	return args.Error(0)
}

func (m *MockUserStore) Update(ctx context.Context, user *User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func TestAuthManager_Authenticate(t *testing.T) {
	tests := []struct {
		name          string
		creds         Credentials
		setupMocks    func(*MockUserStore)
		expectedUser  *User
		expectedError error
	}{
		{
			name: "successful authentication",
			creds: Credentials{
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMocks: func(store *MockUserStore) {
				store.On("GetByEmail", mock.Anything, "test@example.com").Return(&User{
					ID:       "user123",
					Email:    "test@example.com",
					Username: "testuser",
					Roles:    []string{"user"},
					Active:   true,
				}, nil)
			},
			expectedUser: &User{
				ID:       "user123",
				Email:    "test@example.com",
				Username: "testuser",
				Roles:    []string{"user"},
				Active:   true,
			},
			expectedError: nil,
		},
		{
			name: "user not found",
			creds: Credentials{
				Email:    "notfound@example.com",
				Password: "password123",
			},
			setupMocks: func(store *MockUserStore) {
				store.On("GetByEmail", mock.Anything, "notfound@example.com").Return(nil, ErrUserNotFound)
			},
			expectedUser:  nil,
			expectedError: ErrInvalidCredentials,
		},
		{
			name: "inactive user",
			creds: Credentials{
				Email:    "inactive@example.com",
				Password: "password123",
			},
			setupMocks: func(store *MockUserStore) {
				store.On("GetByEmail", mock.Anything, "inactive@example.com").Return(&User{
					ID:       "user456",
					Email:    "inactive@example.com",
					Username: "inactive",
					Roles:    []string{"user"},
					Active:   false,
				}, nil)
			},
			expectedUser:  nil,
			expectedError: ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTokenManager := new(MockTokenManager)
			mockSessionManager := new(MockSessionManager)
			mockRBACManager := new(MockRBACManager)
			mockUserStore := new(MockUserStore)

			tt.setupMocks(mockUserStore)

			manager := NewAuthManager(
				mockTokenManager,
				mockSessionManager,
				mockRBACManager,
				mockUserStore,
			)

			user, err := manager.Authenticate(context.Background(), tt.creds)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.expectedUser.ID, user.ID)
				assert.Equal(t, tt.expectedUser.Email, user.Email)
			}

			mockUserStore.AssertExpectations(t)
		})
	}
}

func TestAuthManager_Register(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		username      string
		password      string
		setupMocks    func(*MockUserStore)
		expectedError error
	}{
		{
			name:     "successful registration",
			email:    "new@example.com",
			username: "newuser",
			password: "password123",
			setupMocks: func(store *MockUserStore) {
				store.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, ErrUserNotFound)
				store.On("Create", mock.Anything, mock.MatchedBy(func(user *User) bool {
					return user.Email == "new@example.com" && user.Username == "newuser"
				}), mock.AnythingOfType("string")).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:     "user already exists",
			email:    "existing@example.com",
			username: "existing",
			password: "password123",
			setupMocks: func(store *MockUserStore) {
				store.On("GetByEmail", mock.Anything, "existing@example.com").Return(&User{
					ID:    "user123",
					Email: "existing@example.com",
				}, nil)
			},
			expectedError: ErrUserAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTokenManager := new(MockTokenManager)
			mockSessionManager := new(MockSessionManager)
			mockRBACManager := new(MockRBACManager)
			mockUserStore := new(MockUserStore)

			tt.setupMocks(mockUserStore)

			manager := NewAuthManager(
				mockTokenManager,
				mockSessionManager,
				mockRBACManager,
				mockUserStore,
			)

			user, err := manager.Register(context.Background(), tt.email, tt.username, tt.password)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.email, user.Email)
				assert.Equal(t, tt.username, user.Username)
				assert.True(t, user.Active)
			}

			mockUserStore.AssertExpectations(t)
		})
	}
}

func TestAuthManager_ValidateToken(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		setupMocks     func(*MockTokenManager)
		expectedUserID string
		expectedError  error
	}{
		{
			name:  "valid token",
			token: "valid_token",
			setupMocks: func(tm *MockTokenManager) {
				tm.On("Validate", mock.Anything, "valid_token").Return("user123", nil)
			},
			expectedUserID: "user123",
			expectedError:  nil,
		},
		{
			name:  "invalid token",
			token: "invalid_token",
			setupMocks: func(tm *MockTokenManager) {
				tm.On("Validate", mock.Anything, "invalid_token").Return("", ErrInvalidToken)
			},
			expectedUserID: "",
			expectedError:  ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTokenManager := new(MockTokenManager)
			mockSessionManager := new(MockSessionManager)
			mockRBACManager := new(MockRBACManager)
			mockUserStore := new(MockUserStore)

			tt.setupMocks(mockTokenManager)

			manager := NewAuthManager(
				mockTokenManager,
				mockSessionManager,
				mockRBACManager,
				mockUserStore,
			)

			userID, err := manager.ValidateToken(context.Background(), tt.token)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUserID, userID)
			}

			mockTokenManager.AssertExpectations(t)
		})
	}
}

func TestAuthManager_HasPermission(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		resource   string
		action     string
		setupMocks func(*MockRBACManager)
		expected   bool
	}{
		{
			name:     "has permission",
			userID:   "user123",
			resource: "mcp",
			action:   "create",
			setupMocks: func(rbac *MockRBACManager) {
				rbac.On("HasPermission", "user123", "mcp", "create").Return(true)
			},
			expected: true,
		},
		{
			name:     "no permission",
			userID:   "user123",
			resource: "mcp",
			action:   "delete",
			setupMocks: func(rbac *MockRBACManager) {
				rbac.On("HasPermission", "user123", "mcp", "delete").Return(false)
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTokenManager := new(MockTokenManager)
			mockSessionManager := new(MockSessionManager)
			mockRBACManager := new(MockRBACManager)
			mockUserStore := new(MockUserStore)

			tt.setupMocks(mockRBACManager)

			manager := NewAuthManager(
				mockTokenManager,
				mockSessionManager,
				mockRBACManager,
				mockUserStore,
			)

			result := manager.HasPermission(tt.userID, tt.resource, tt.action)
			assert.Equal(t, tt.expected, result)

			mockRBACManager.AssertExpectations(t)
		})
	}
}

func TestAuthManager_Logout(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		setupMocks    func(*MockSessionManager)
		expectedError error
	}{
		{
			name:   "successful logout",
			userID: "user123",
			setupMocks: func(sm *MockSessionManager) {
				sm.On("Invalidate", mock.Anything, "user123").Return(nil)
			},
			expectedError: nil,
		},
		{
			name:   "logout error",
			userID: "user123",
			setupMocks: func(sm *MockSessionManager) {
				sm.On("Invalidate", mock.Anything, "user123").Return(ErrSessionNotFound)
			},
			expectedError: ErrSessionNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTokenManager := new(MockTokenManager)
			mockSessionManager := new(MockSessionManager)
			mockRBACManager := new(MockRBACManager)
			mockUserStore := new(MockUserStore)

			tt.setupMocks(mockSessionManager)

			manager := NewAuthManager(
				mockTokenManager,
				mockSessionManager,
				mockRBACManager,
				mockUserStore,
			)

			err := manager.Logout(context.Background(), tt.userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockSessionManager.AssertExpectations(t)
		})
	}
}
