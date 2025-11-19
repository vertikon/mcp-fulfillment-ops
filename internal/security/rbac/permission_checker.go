package rbac

import (
	"sync"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// PermissionRequest represents the resource/action pair being requested.
type PermissionRequest struct {
	Resource string
	Action   string
	Context  PermissionContext
}

// PermissionContext propagates contextual attributes to advanced checks.
type PermissionContext struct {
	UserID     string
	Roles      []string
	Attributes map[string]string
}

// PermissionOverride allows fine-grained allow/deny rules scoped to specific roles.
type PermissionOverride struct {
	RoleID          string
	ResourcePattern string
	ActionPattern   string
	Effect          PolicyEffect
	Condition       PermissionCondition
	Description     string
}

// PermissionCondition defines dynamic checks (time, tenant, attribute-based, etc.).
type PermissionCondition interface {
	Evaluate(ctx PermissionContext) bool
	Describe() string
}

// PermissionConditionFunc makes it easy to plug custom conditions.
type PermissionConditionFunc struct {
	name string
	fn   func(ctx PermissionContext) bool
}

// Evaluate runs the custom function if present.
func (c PermissionConditionFunc) Evaluate(ctx PermissionContext) bool {
	if c.fn == nil {
		return true
	}
	return c.fn(ctx)
}

// Describe returns the condition label for observability.
func (c PermissionConditionFunc) Describe() string {
	return c.name
}

// PermissionChecker evaluates permissions combining static role permissions and overrides.
type PermissionChecker interface {
	HasPermission(role *Role, req PermissionRequest) bool
	RegisterOverride(override PermissionOverride)
	ListOverrides() []PermissionOverride
}

type defaultPermissionChecker struct {
	overrides []PermissionOverride
	mu        sync.RWMutex
	logger    *zap.Logger
}

// NewPermissionChecker returns the default PermissionChecker implementation.
func NewPermissionChecker() PermissionChecker {
	return &defaultPermissionChecker{
		overrides: make([]PermissionOverride, 0),
		logger:    logger.WithContext(nil),
	}
}

func (c *defaultPermissionChecker) HasPermission(role *Role, req PermissionRequest) bool {
	if role == nil {
		return false
	}

	ctx := req.Context
	ctx.Roles = append(ctx.Roles, role.ID)

	overrides := c.cloneOverrides()
	for _, override := range overrides {
		if override.RoleID != "" && override.RoleID != role.ID {
			continue
		}

		if !matchesPattern(override.ResourcePattern, req.Resource) ||
			!matchesPattern(override.ActionPattern, req.Action) {
			continue
		}

		if override.Condition != nil && !override.Condition.Evaluate(ctx) {
			continue
		}

		allowed := override.Effect == EffectAllow
		c.logger.Debug("permission override applied",
			zap.String("role_id", role.ID),
			zap.String("resource", req.Resource),
			zap.String("action", req.Action),
			zap.Bool("allowed", allowed),
			zap.String("override", override.Description),
		)
		return allowed
	}

	for _, perm := range role.Permissions {
		if matchesPattern(perm.Resource, req.Resource) &&
			matchesPattern(perm.Action, req.Action) {
			c.logger.Debug("permission granted by role",
				zap.String("role_id", role.ID),
				zap.String("resource", req.Resource),
				zap.String("action", req.Action),
			)
			return true
		}
	}

	c.logger.Debug("permission denied by role",
		zap.String("role_id", role.ID),
		zap.String("resource", req.Resource),
		zap.String("action", req.Action),
	)
	return false
}

func (c *defaultPermissionChecker) RegisterOverride(override PermissionOverride) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.overrides = append(c.overrides, override)
}

func (c *defaultPermissionChecker) ListOverrides() []PermissionOverride {
	return c.cloneOverrides()
}

func (c *defaultPermissionChecker) cloneOverrides() []PermissionOverride {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cloned := make([]PermissionOverride, len(c.overrides))
	copy(cloned, c.overrides)
	return cloned
}

// ConditionRequireRole enforces that the caller has at least one of the given roles.
func ConditionRequireRole(roles ...string) PermissionCondition {
	return PermissionConditionFunc{
		name: "require_role",
		fn: func(ctx PermissionContext) bool {
			for _, required := range roles {
				if hasRole(ctx.Roles, required) {
					return true
				}
			}
			return false
		},
	}
}

// ConditionAttributeEquals ensures the provided attribute equals one of the accepted values.
func ConditionAttributeEquals(key string, values ...string) PermissionCondition {
	return PermissionConditionFunc{
		name: "attribute_equals_" + key,
		fn: func(ctx PermissionContext) bool {
			if ctx.Attributes == nil {
				return false
			}
			value, ok := ctx.Attributes[key]
			if !ok {
				return false
			}
			for _, expected := range values {
				if value == expected {
					return true
				}
			}
			return false
		},
	}
}

func hasRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}
