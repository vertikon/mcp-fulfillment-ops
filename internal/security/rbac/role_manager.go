package rbac

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

var (
	// ErrRoleAlreadyExists indicates an attempt to create a duplicated role.
	ErrRoleAlreadyExists = errors.New("role already exists")
	// ErrInvalidRole indicates a role definition missing mandatory data.
	ErrInvalidRole = errors.New("invalid role definition")
)

// RoleManager provides CRUD operations for roles independent of the RBAC manager cache.
type RoleManager interface {
	CreateRole(ctx context.Context, role *Role) error
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, roleID string) error
	GetRole(ctx context.Context, roleID string) (*Role, error)
	ListRoles(ctx context.Context) ([]*Role, error)
	// Sync replaces the current role catalog with the provided set, keeping the op idempotent.
	Sync(ctx context.Context, roles []*Role) error
}

// RoleStore defines the persistence boundary for roles.
type RoleStore interface {
	Save(ctx context.Context, role *Role) error
	Delete(ctx context.Context, roleID string) error
	Get(ctx context.Context, roleID string) (*Role, error)
	List(ctx context.Context) ([]*Role, error)
	Clear(ctx context.Context) error
}

// InMemoryRoleStore is a thread-safe store primarily intended for tests and bootstrap scenarios.
type InMemoryRoleStore struct {
	mu    sync.RWMutex
	roles map[string]*Role
}

// NewInMemoryRoleStore creates an empty in-memory role store.
func NewInMemoryRoleStore() RoleStore {
	return &InMemoryRoleStore{
		roles: make(map[string]*Role),
	}
}

// Save persists (create/update) a role definition.
func (s *InMemoryRoleStore) Save(ctx context.Context, role *Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := copyRole(role)
	s.roles[role.ID] = cp
	return nil
}

// Delete removes a role definition.
func (s *InMemoryRoleStore) Delete(ctx context.Context, roleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.roles, roleID)
	return nil
}

// Get returns a single role.
func (s *InMemoryRoleStore) Get(ctx context.Context, roleID string) (*Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	role, ok := s.roles[roleID]
	if !ok {
		return nil, ErrRoleNotFound
	}
	return copyRole(role), nil
}

// List returns all roles sorted by ID for deterministic output.
func (s *InMemoryRoleStore) List(ctx context.Context) ([]*Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Role, 0, len(s.roles))
	for _, role := range s.roles {
		result = append(result, copyRole(role))
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

// Clear removes all roles from the store.
func (s *InMemoryRoleStore) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.roles = make(map[string]*Role)
	return nil
}

type managedRoleStore struct {
	store  RoleStore
	logger *zap.Logger
}

// NewRoleManager wires a RoleManager using the provided store.
func NewRoleManager(store RoleStore) RoleManager {
	return &managedRoleStore{
		store:  store,
		logger: logger.WithContext(nil),
	}
}

func (m *managedRoleStore) CreateRole(ctx context.Context, role *Role) error {
	if err := validateRole(role); err != nil {
		return err
	}

	if _, err := m.store.Get(ctx, role.ID); err == nil {
		return ErrRoleAlreadyExists
	} else if !errors.Is(err, ErrRoleNotFound) {
		return err
	}

	if err := m.store.Save(ctx, role); err != nil {
		return err
	}

	m.logger.Info("role created",
		zap.String("role_id", role.ID),
		zap.String("name", role.Name),
		zap.Int("permissions", len(role.Permissions)),
	)
	return nil
}

func (m *managedRoleStore) UpdateRole(ctx context.Context, role *Role) error {
	if err := validateRole(role); err != nil {
		return err
	}

	if _, err := m.store.Get(ctx, role.ID); err != nil {
		return err
	}

	if err := m.store.Save(ctx, role); err != nil {
		return err
	}

	m.logger.Info("role updated",
		zap.String("role_id", role.ID),
		zap.Int("permissions", len(role.Permissions)),
	)
	return nil
}

func (m *managedRoleStore) DeleteRole(ctx context.Context, roleID string) error {
	if roleID == "" {
		return ErrInvalidRole
	}

	if err := m.store.Delete(ctx, roleID); err != nil {
		return err
	}

	m.logger.Info("role deleted", zap.String("role_id", roleID))
	return nil
}

func (m *managedRoleStore) GetRole(ctx context.Context, roleID string) (*Role, error) {
	if roleID == "" {
		return nil, ErrInvalidRole
	}
	return m.store.Get(ctx, roleID)
}

func (m *managedRoleStore) ListRoles(ctx context.Context) ([]*Role, error) {
	return m.store.List(ctx)
}

func (m *managedRoleStore) Sync(ctx context.Context, roles []*Role) error {
	if err := m.store.Clear(ctx); err != nil {
		return err
	}

	for _, role := range roles {
		if err := m.store.Save(ctx, role); err != nil {
			return fmt.Errorf("failed to sync role %s: %w", role.ID, err)
		}
	}

	m.logger.Info("roles synchronized", zap.Int("total", len(roles)))
	return nil
}

func validateRole(role *Role) error {
	if role == nil || role.ID == "" || role.Name == "" {
		return ErrInvalidRole
	}
	return nil
}

func copyRole(role *Role) *Role {
	if role == nil {
		return nil
	}

	cp := *role
	if len(role.Permissions) > 0 {
		cp.Permissions = make([]Permission, len(role.Permissions))
		copy(cp.Permissions, role.Permissions)
	}

	return &cp
}
