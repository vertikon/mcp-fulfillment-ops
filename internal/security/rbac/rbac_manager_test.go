package rbac

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRoleManager is a mock implementation of RoleManager
type MockRoleManager struct {
	mock.Mock
}

func (m *MockRoleManager) CreateRole(ctx context.Context, role *Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleManager) UpdateRole(ctx context.Context, role *Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleManager) DeleteRole(ctx context.Context, roleID string) error {
	args := m.Called(ctx, roleID)
	return args.Error(0)
}

func (m *MockRoleManager) GetRole(ctx context.Context, roleID string) (*Role, error) {
	args := m.Called(ctx, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Role), args.Error(1)
}

func (m *MockRoleManager) ListRoles(ctx context.Context) ([]*Role, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Role), args.Error(1)
}

func (m *MockRoleManager) Sync(ctx context.Context, roles []*Role) error {
	args := m.Called(ctx, roles)
	return args.Error(0)
}

// MockPermissionChecker is a mock implementation of PermissionChecker
type MockPermissionChecker struct {
	mock.Mock
}

func (m *MockPermissionChecker) HasPermission(role *Role, req PermissionRequest) bool {
	args := m.Called(role, req)
	return args.Bool(0)
}

func (m *MockPermissionChecker) RegisterOverride(override PermissionOverride) {
	m.Called(override)
}

func (m *MockPermissionChecker) ListOverrides() []PermissionOverride {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]PermissionOverride)
}

// MockPolicyEnforcer is a mock implementation of PolicyEnforcer
type MockPolicyEnforcer struct {
	mock.Mock
}

func (m *MockPolicyEnforcer) Register(policy *Policy) error {
	args := m.Called(policy)
	return args.Error(0)
}

func (m *MockPolicyEnforcer) Remove(policyID string) {
	m.Called(policyID)
}

func (m *MockPolicyEnforcer) Evaluate(ctx context.Context, request PolicyContext) (*PolicyDecision, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PolicyDecision), args.Error(1)
}

func (m *MockPolicyEnforcer) List() []*Policy {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]*Policy)
}

func (m *MockPolicyEnforcer) Clear() {
	m.Called()
}

func TestRBACManager_HasPermission(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		resource   string
		action     string
		setupMocks func(*MockRoleManager, *MockPermissionChecker, *MockPolicyEnforcer)
		expected   bool
	}{
		{
			name:     "permission granted",
			userID:   "user123",
			resource: "mcp",
			action:   "create",
			setupMocks: func(rm *MockRoleManager, pc *MockPermissionChecker, pe *MockPolicyEnforcer) {
				role := &Role{
					ID:   "admin",
					Name: "Admin",
					Permissions: []Permission{
						{Resource: "mcp", Action: "create"},
					},
				}
				rm.On("GetRole", mock.Anything, "admin").Return(role, nil)
				pc.On("HasPermission", role, mock.AnythingOfType("PermissionRequest")).Return(true)
				pe.On("Evaluate", mock.Anything, mock.AnythingOfType("PolicyContext")).Return(&PolicyDecision{Allowed: true}, nil)
			},
			expected: true,
		},
		{
			name:     "permission denied by role",
			userID:   "user123",
			resource: "mcp",
			action:   "delete",
			setupMocks: func(rm *MockRoleManager, pc *MockPermissionChecker, pe *MockPolicyEnforcer) {
				role := &Role{
					ID:   "user",
					Name: "User",
					Permissions: []Permission{
						{Resource: "mcp", Action: "read"},
					},
				}
				rm.On("GetRole", mock.Anything, "user").Return(role, nil)
				pc.On("HasPermission", role, mock.AnythingOfType("PermissionRequest")).Return(false)
			},
			expected: false,
		},
		{
			name:     "permission denied by policy",
			userID:   "user123",
			resource: "mcp",
			action:   "delete",
			setupMocks: func(rm *MockRoleManager, pc *MockPermissionChecker, pe *MockPolicyEnforcer) {
				role := &Role{
					ID:   "admin",
					Name: "Admin",
					Permissions: []Permission{
						{Resource: "mcp", Action: "delete"},
					},
				}
				rm.On("GetRole", mock.Anything, "admin").Return(role, nil)
				pc.On("HasPermission", role, mock.AnythingOfType("PermissionRequest")).Return(true)
				pe.On("Evaluate", mock.Anything, mock.AnythingOfType("PolicyContext")).Return(&PolicyDecision{Allowed: false, Reason: "policy_denied"}, nil)
			},
			expected: false,
		},
		{
			name:     "user has no roles",
			userID:   "user456",
			resource: "mcp",
			action:   "create",
			setupMocks: func(rm *MockRoleManager, pc *MockPermissionChecker, pe *MockPolicyEnforcer) {
				// No roles assigned
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoleManager := new(MockRoleManager)
			mockPermissionChecker := new(MockPermissionChecker)
			mockPolicyEnforcer := new(MockPolicyEnforcer)

			manager := NewRBACManager(mockRoleManager, mockPermissionChecker, mockPolicyEnforcer)

			// Assign role to user for tests that need it
			if tt.name != "user has no roles" {
				role := &Role{ID: "admin", Name: "Admin"}
				mockRoleManager.On("GetRole", mock.Anything, "admin").Return(role, nil).Maybe()
				_ = manager.AssignRole(context.Background(), tt.userID, "admin")
			}

			tt.setupMocks(mockRoleManager, mockPermissionChecker, mockPolicyEnforcer)

			result := manager.HasPermission(tt.userID, tt.resource, tt.action)
			assert.Equal(t, tt.expected, result)

			mockRoleManager.AssertExpectations(t)
			mockPermissionChecker.AssertExpectations(t)
			mockPolicyEnforcer.AssertExpectations(t)
		})
	}
}

func TestRBACManager_AssignRole(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		roleID        string
		setupMocks    func(*MockRoleManager)
		expectedError error
	}{
		{
			name:   "successful assignment",
			userID: "user123",
			roleID: "admin",
			setupMocks: func(rm *MockRoleManager) {
				rm.On("GetRole", mock.Anything, "admin").Return(&Role{ID: "admin", Name: "Admin"}, nil)
			},
			expectedError: nil,
		},
		{
			name:   "role not found",
			userID: "user123",
			roleID: "nonexistent",
			setupMocks: func(rm *MockRoleManager) {
				rm.On("GetRole", mock.Anything, "nonexistent").Return(nil, ErrRoleNotFound)
			},
			expectedError: ErrRoleNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoleManager := new(MockRoleManager)
			mockPermissionChecker := new(MockPermissionChecker)
			mockPolicyEnforcer := new(MockPolicyEnforcer)

			tt.setupMocks(mockRoleManager)

			manager := NewRBACManager(mockRoleManager, mockPermissionChecker, mockPolicyEnforcer)

			err := manager.AssignRole(context.Background(), tt.userID, tt.roleID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
			}

			mockRoleManager.AssertExpectations(t)
		})
	}
}

func TestRBACManager_RevokeRole(t *testing.T) {
	mockRoleManager := new(MockRoleManager)
	mockPermissionChecker := new(MockPermissionChecker)
	mockPolicyEnforcer := new(MockPolicyEnforcer)

	manager := NewRBACManager(mockRoleManager, mockPermissionChecker, mockPolicyEnforcer)

	// Assign role first
	role := &Role{ID: "admin", Name: "Admin"}
	mockRoleManager.On("GetRole", mock.Anything, "admin").Return(role, nil)
	_ = manager.AssignRole(context.Background(), "user123", "admin")

	// Revoke role
	err := manager.RevokeRole(context.Background(), "user123", "admin")
	assert.NoError(t, err)

	// Verify role is revoked
	roles, _ := manager.GetUserRoles("user123")
	assert.NotContains(t, roles, "admin")
}

func TestRBACManager_GetUserRoles(t *testing.T) {
	mockRoleManager := new(MockRoleManager)
	mockPermissionChecker := new(MockPermissionChecker)
	mockPolicyEnforcer := new(MockPolicyEnforcer)

	manager := NewRBACManager(mockRoleManager, mockPermissionChecker, mockPolicyEnforcer)

	// Assign roles
	role1 := &Role{ID: "admin", Name: "Admin"}
	role2 := &Role{ID: "user", Name: "User"}
	mockRoleManager.On("GetRole", mock.Anything, "admin").Return(role1, nil)
	mockRoleManager.On("GetRole", mock.Anything, "user").Return(role2, nil)

	_ = manager.AssignRole(context.Background(), "user123", "admin")
	_ = manager.AssignRole(context.Background(), "user123", "user")

	roles, err := manager.GetUserRoles("user123")
	assert.NoError(t, err)
	assert.Contains(t, roles, "admin")
	assert.Contains(t, roles, "user")
}
