package rbac

import (
	"context"
	"errors"
	"sync"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

var (
	ErrRoleNotFound       = errors.New("role not found")
	ErrPermissionDenied   = errors.New("permission denied")
	ErrUserAlreadyHasRole = errors.New("user already has role")
)

// Role represents a role with permissions
type Role struct {
	ID          string
	Name        string
	Description string
	Permissions []Permission
}

// Permission represents a permission
type Permission struct {
	Resource string
	Action   string
}

// RBACManager handles role-based access control
type RBACManager interface {
	// HasPermission checks if user has permission for resource/action
	HasPermission(userID string, resource string, action string) bool

	// AssignRole assigns a role to a user
	AssignRole(ctx context.Context, userID string, roleID string) error

	// RevokeRole revokes a role from a user
	RevokeRole(ctx context.Context, userID string, roleID string) error

	// GetUserRoles returns all roles for a user
	GetUserRoles(userID string) ([]string, error)

	// CreateRole creates a new role
	CreateRole(ctx context.Context, role *Role) error

	// GetRole returns a role by ID
	GetRole(ctx context.Context, roleID string) (*Role, error)

	// ListRoles returns all roles
	ListRoles(ctx context.Context) ([]*Role, error)
}

// Manager implements RBACManager
type Manager struct {
	roleManager       RoleManager
	permissionChecker PermissionChecker
	policyEnforcer    PolicyEnforcer
	userRoles         map[string][]string // userID -> roleIDs
	mu                sync.RWMutex
	logger            *zap.Logger
}

// NewRBACManager creates a new RBACManager
func NewRBACManager(roleManager RoleManager, permissionChecker PermissionChecker, policyEnforcer PolicyEnforcer) RBACManager {
	return &Manager{
		roleManager:       roleManager,
		permissionChecker: permissionChecker,
		policyEnforcer:    policyEnforcer,
		userRoles:         make(map[string][]string),
		logger:            logger.WithContext(nil),
	}
}

// HasPermission checks if user has permission for resource/action
func (m *Manager) HasPermission(userID string, resource string, action string) bool {
	m.mu.RLock()
	roleIDs, ok := m.userRoles[userID]
	m.mu.RUnlock()

	if !ok {
		m.logger.Debug("User has no roles",
			zap.String("user_id", userID),
		)
		return false
	}

	permissionGranted := false
	permContext := PermissionContext{
		UserID: userID,
		Roles:  append([]string(nil), roleIDs...),
	}

	for _, roleID := range roleIDs {
		role, err := m.roleManager.GetRole(context.Background(), roleID)
		if err != nil {
			continue
		}

		if m.permissionChecker != nil {
			request := PermissionRequest{
				Resource: resource,
				Action:   action,
				Context:  permContext,
			}
			if m.permissionChecker.HasPermission(role, request) {
				permissionGranted = true
				break
			}
			continue
		}

		for _, perm := range role.Permissions {
			if perm.Resource == resource && perm.Action == action {
				permissionGranted = true
				break
			}
		}

		if permissionGranted {
			break
		}
	}

	if !permissionGranted {
		m.logger.Debug("Permission denied by RBAC layer",
			zap.String("user_id", userID),
			zap.String("resource", resource),
			zap.String("action", action),
		)
		return false
	}

	if m.policyEnforcer != nil {
		decision, err := m.policyEnforcer.Evaluate(context.Background(), PolicyContext{
			UserID:   userID,
			Roles:    roleIDs,
			Resource: resource,
			Action:   action,
		})
		if err != nil {
			m.logger.Error("Policy evaluation failed",
				zap.String("user_id", userID),
				zap.String("resource", resource),
				zap.String("action", action),
				zap.Error(err),
			)
			return false
		}

		if decision != nil && !decision.Allowed {
			m.logger.Warn("Policy denied access",
				zap.String("user_id", userID),
				zap.String("resource", resource),
				zap.String("action", action),
				zap.String("policy_id", decision.PolicyID),
				zap.String("reason", decision.Reason),
			)
			return false
		}
	}

	m.logger.Debug("Permission granted",
		zap.String("user_id", userID),
		zap.String("resource", resource),
		zap.String("action", action),
	)
	return true
}

// AssignRole assigns a role to a user
func (m *Manager) AssignRole(ctx context.Context, userID string, roleID string) error {
	// Verify role exists
	_, err := m.roleManager.GetRole(ctx, roleID)
	if err != nil {
		m.logger.Warn("Failed to assign role: role not found",
			zap.String("user_id", userID),
			zap.String("role_id", roleID),
		)
		return ErrRoleNotFound
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if user already has role
	roleIDs := m.userRoles[userID]
	for _, rid := range roleIDs {
		if rid == roleID {
			return ErrUserAlreadyHasRole
		}
	}

	// Assign role
	m.userRoles[userID] = append(m.userRoles[userID], roleID)

	m.logger.Info("Role assigned",
		zap.String("user_id", userID),
		zap.String("role_id", roleID),
	)

	return nil
}

// RevokeRole revokes a role from a user
func (m *Manager) RevokeRole(ctx context.Context, userID string, roleID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	roleIDs, ok := m.userRoles[userID]
	if !ok {
		return nil // User has no roles
	}

	// Remove role
	newRoleIDs := make([]string, 0)
	for _, rid := range roleIDs {
		if rid != roleID {
			newRoleIDs = append(newRoleIDs, rid)
		}
	}

	m.userRoles[userID] = newRoleIDs

	m.logger.Info("Role revoked",
		zap.String("user_id", userID),
		zap.String("role_id", roleID),
	)

	return nil
}

// GetUserRoles returns all roles for a user
func (m *Manager) GetUserRoles(userID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	roleIDs, ok := m.userRoles[userID]
	if !ok {
		return []string{}, nil
	}

	return roleIDs, nil
}

// CreateRole creates a new role
func (m *Manager) CreateRole(ctx context.Context, role *Role) error {
	return m.roleManager.CreateRole(ctx, role)
}

// GetRole returns a role by ID
func (m *Manager) GetRole(ctx context.Context, roleID string) (*Role, error) {
	return m.roleManager.GetRole(ctx, roleID)
}

// ListRoles returns all roles
func (m *Manager) ListRoles(ctx context.Context) ([]*Role, error) {
	return m.roleManager.ListRoles(ctx)
}
