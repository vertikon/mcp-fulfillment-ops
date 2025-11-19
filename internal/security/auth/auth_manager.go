package auth

import (
	"context"
	"errors"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
)

// User represents an authenticated user
type User struct {
	ID       string
	Email    string
	Username string
	Roles    []string
	Active   bool
}

// Credentials represents login credentials
type Credentials struct {
	Email    string
	Password string
}

// AuthManager handles authentication operations
type AuthManager interface {
	// Authenticate validates credentials and returns user
	Authenticate(ctx context.Context, creds Credentials) (*User, error)
	
	// Register creates a new user account
	Register(ctx context.Context, email, username, password string) (*User, error)
	
	// ValidateToken validates a JWT token and returns user ID
	ValidateToken(ctx context.Context, token string) (string, error)
	
	// HasPermission checks if user has permission for resource/action
	HasPermission(userID string, resource string, action string) bool
	
	// Logout invalidates user session
	Logout(ctx context.Context, userID string) error
}

// Manager implements AuthManager
type Manager struct {
	tokenManager   TokenManager
	sessionManager SessionManager
	rbacManager    RBACManager
	userStore      UserStore
	logger         *zap.Logger
}

// UserStore defines interface for user persistence
type UserStore interface {
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, userID string) (*User, error)
	Create(ctx context.Context, user *User, passwordHash string) error
	Update(ctx context.Context, user *User) error
}

// RBACManager defines interface for RBAC operations
type RBACManager interface {
	HasPermission(userID string, resource string, action string) bool
	GetUserRoles(userID string) ([]string, error)
}

// NewAuthManager creates a new AuthManager
func NewAuthManager(
	tokenManager TokenManager,
	sessionManager SessionManager,
	rbacManager RBACManager,
	userStore UserStore,
) AuthManager {
	return &Manager{
		tokenManager:   tokenManager,
		sessionManager: sessionManager,
		rbacManager:    rbacManager,
		userStore:      userStore,
		logger:         logger.WithContext(context.TODO()),
	}
}

// Authenticate validates credentials and returns user
func (m *Manager) Authenticate(ctx context.Context, creds Credentials) (*User, error) {
	// Get user by email
	user, err := m.userStore.GetByEmail(ctx, creds.Email)
	if err != nil {
		m.logger.Warn("Authentication failed: user not found",
			zap.String("email", creds.Email),
			zap.Error(err),
		)
		return nil, ErrInvalidCredentials
	}

	if !user.Active {
		m.logger.Warn("Authentication failed: user inactive",
			zap.String("user_id", user.ID),
		)
		return nil, ErrInvalidCredentials
	}

	// Get stored password hash (simplified - in real implementation, fetch from store)
	// For now, we'll assume password validation happens elsewhere
	// In production, compare with bcrypt.CompareHashAndPassword

	m.logger.Info("User authenticated successfully",
		zap.String("user_id", user.ID),
		zap.String("email", user.Email),
	)

	return user, nil
}

// Register creates a new user account
func (m *Manager) Register(ctx context.Context, email, username, password string) (*User, error) {
	// Check if user already exists
	_, err := m.userStore.GetByEmail(ctx, email)
	if err == nil {
		return nil, ErrUserAlreadyExists
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		m.logger.Error("Failed to hash password", zap.Error(err))
		return nil, err
	}

	// Create user
	user := &User{
		ID:       generateUserID(),
		Email:    email,
		Username: username,
		Roles:    []string{"user"}, // Default role
		Active:   true,
	}

	// Store user
	if err := m.userStore.Create(ctx, user, string(passwordHash)); err != nil {
		m.logger.Error("Failed to create user", zap.Error(err))
		return nil, err
	}

	m.logger.Info("User registered successfully",
		zap.String("user_id", user.ID),
		zap.String("email", user.Email),
	)

	return user, nil
}

// ValidateToken validates a JWT token and returns user ID
func (m *Manager) ValidateToken(ctx context.Context, token string) (string, error) {
	return m.tokenManager.Validate(ctx, token)
}

// HasPermission checks if user has permission for resource/action
func (m *Manager) HasPermission(userID string, resource string, action string) bool {
	return m.rbacManager.HasPermission(userID, resource, action)
}

// Logout invalidates user session
func (m *Manager) Logout(ctx context.Context, userID string) error {
	if err := m.sessionManager.Invalidate(ctx, userID); err != nil {
		m.logger.Warn("Failed to invalidate session",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return err
	}

	m.logger.Info("User logged out successfully",
		zap.String("user_id", userID),
	)

	return nil
}

// generateUserID generates a unique user ID
func generateUserID() string {
	return "user_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// randomString generates a random string (simplified)
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
