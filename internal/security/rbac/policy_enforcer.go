package rbac

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// PolicyEnforcer validates contextual policies after RBAC grants coarse access.
type PolicyEnforcer interface {
	Register(policy *Policy) error
	Remove(policyID string)
	Evaluate(ctx context.Context, request PolicyContext) (*PolicyDecision, error)
	List() []*Policy
	Clear()
}

// Policy describes a set of rules with the same lifecycle/resolution priority.
type Policy struct {
	ID          string
	Description string
	Priority    int
	Rules       []PolicyRule
	Tags        []string
}

// PolicyRule is a single decision point inside a policy.
type PolicyRule struct {
	Resource    string
	Action      string
	Effect      PolicyEffect
	Description string
	Conditions  []PolicyCondition
}

// PolicyContext carries runtime metadata required to evaluate policies.
type PolicyContext struct {
	UserID     string
	Roles      []string
	Resource   string
	Action     string
	TenantID   string
	Attributes map[string]string
	Metadata   map[string]string
}

// PolicyDecision is produced by the enforcer.
type PolicyDecision struct {
	Allowed         bool
	PolicyID        string
	RuleDescription string
	Reason          string
}

// PolicyCondition evaluates constraints (roles, tenants, attributes, schedules, etc.).
type PolicyCondition interface {
	Evaluate(ctx PolicyContext) bool
	Describe() string
}

// PolicyConditionFunc is a helper to create inline conditions.
type PolicyConditionFunc struct {
	name string
	fn   func(ctx PolicyContext) bool
}

// Evaluate executes the encapsulated function.
func (c PolicyConditionFunc) Evaluate(ctx PolicyContext) bool {
	if c.fn == nil {
		return true
	}
	return c.fn(ctx)
}

// Describe returns the label attached to the condition.
func (c PolicyConditionFunc) Describe() string {
	return c.name
}

// PolicyEnforcerConfig controls runtime behavior.
type PolicyEnforcerConfig struct {
	// FailOpen allows the system to continue when no policy matches (useful during bootstrap).
	FailOpen bool
}

type policyManager struct {
	policies map[string]*Policy
	mu       sync.RWMutex
	logger   *zap.Logger
	failOpen bool
}

// NewPolicyEnforcer instantiates the default PolicyEnforcer.
func NewPolicyEnforcer(cfg PolicyEnforcerConfig) PolicyEnforcer {
	return &policyManager{
		policies: make(map[string]*Policy),
		logger:   logger.WithContext(nil),
		failOpen: cfg.FailOpen,
	}
}

func (m *policyManager) Register(policy *Policy) error {
	if err := validatePolicy(policy); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.policies[policy.ID] = clonePolicy(policy)

	m.logger.Info("policy registered",
		zap.String("policy_id", policy.ID),
		zap.Int("priority", policy.Priority),
		zap.Int("rules", len(policy.Rules)),
	)
	return nil
}

func (m *policyManager) Remove(policyID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.policies, policyID)
	m.logger.Info("policy removed", zap.String("policy_id", policyID))
}

func (m *policyManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.policies = make(map[string]*Policy)
}

func (m *policyManager) Evaluate(ctx context.Context, request PolicyContext) (*PolicyDecision, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	policies := m.snapshotPolicies()

	for _, policy := range policies {
		for _, rule := range policy.Rules {
			if !matchesPattern(rule.Resource, request.Resource) ||
				!matchesPattern(rule.Action, request.Action) {
				continue
			}

			if !conditionsSatisfied(rule.Conditions, request) {
				continue
			}

			decision := &PolicyDecision{
				Allowed:         rule.Effect == EffectAllow,
				PolicyID:        policy.ID,
				RuleDescription: rule.Description,
			}

			if decision.Allowed {
				decision.Reason = "allow_rule_matched"
			} else {
				decision.Reason = "deny_rule_matched"
			}
			return decision, nil
		}
	}

	if m.failOpen {
		return &PolicyDecision{Allowed: true, Reason: "no_policy_matched"}, nil
	}

	return &PolicyDecision{Allowed: false, Reason: "no_policy_matched"}, nil
}

func (m *policyManager) List() []*Policy {
	return m.snapshotPolicies()
}

func (m *policyManager) snapshotPolicies() []*Policy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*Policy, 0, len(m.policies))
	for _, policy := range m.policies {
		result = append(result, clonePolicy(policy))
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Priority == result[j].Priority {
			return result[i].ID < result[j].ID
		}
		return result[i].Priority > result[j].Priority
	})
	return result
}

func conditionsSatisfied(conditions []PolicyCondition, ctx PolicyContext) bool {
	for _, condition := range conditions {
		if condition == nil {
			continue
		}
		if !condition.Evaluate(ctx) {
			return false
		}
	}
	return true
}

func validatePolicy(policy *Policy) error {
	if policy == nil || policy.ID == "" {
		return fmt.Errorf("policy is required")
	}
	if len(policy.Rules) == 0 {
		return fmt.Errorf("policy %s has no rules", policy.ID)
	}
	return nil
}

func clonePolicy(policy *Policy) *Policy {
	if policy == nil {
		return nil
	}

	cp := *policy
	if len(policy.Rules) > 0 {
		cp.Rules = make([]PolicyRule, len(policy.Rules))
		for i, rule := range policy.Rules {
			cp.Rules[i] = rule
			if len(rule.Conditions) > 0 {
				conds := make([]PolicyCondition, len(rule.Conditions))
				copy(conds, rule.Conditions)
				cp.Rules[i].Conditions = conds
			}
		}
	}
	if len(policy.Tags) > 0 {
		cp.Tags = make([]string, len(policy.Tags))
		copy(cp.Tags, policy.Tags)
	}
	return &cp
}

// PolicyConditionRole restricts evaluation to users that contain at least one of the roles.
func PolicyConditionRole(roles ...string) PolicyCondition {
	return PolicyConditionFunc{
		name: "require_roles",
		fn: func(ctx PolicyContext) bool {
			for _, role := range roles {
				if hasRole(ctx.Roles, role) {
					return true
				}
			}
			return false
		},
	}
}

// PolicyConditionTenant enforces tenant isolation.
func PolicyConditionTenant(tenantIDs ...string) PolicyCondition {
	allowed := make(map[string]struct{}, len(tenantIDs))
	for _, tenant := range tenantIDs {
		allowed[tenant] = struct{}{}
	}

	return PolicyConditionFunc{
		name: "tenant_scope",
		fn: func(ctx PolicyContext) bool {
			if ctx.TenantID == "" {
				return false
			}
			_, ok := allowed[ctx.TenantID]
			return ok
		},
	}
}

// PolicyConditionAttributeEquals matches arbitrary attributes to the required values.
func PolicyConditionAttributeEquals(key string, values ...string) PolicyCondition {
	return PolicyConditionFunc{
		name: "attribute_equals_" + key,
		fn: func(ctx PolicyContext) bool {
			if ctx.Attributes == nil {
				return false
			}
			value, ok := ctx.Attributes[key]
			if !ok {
				return false
			}
			for _, val := range values {
				if value == val {
					return true
				}
			}
			return false
		},
	}
}

// PolicyConditionTimeWindow restricts execution to a specific hour range (UTC by default).
func PolicyConditionTimeWindow(startHour, endHour int, location *time.Location) PolicyCondition {
	if location == nil {
		location = time.UTC
	}

	name := fmt.Sprintf("time_window_%02d-%02d_%s", startHour, endHour, location.String())
	return PolicyConditionFunc{
		name: name,
		fn: func(ctx PolicyContext) bool {
			now := time.Now().In(location)
			hour := now.Hour()
			if startHour <= endHour {
				return hour >= startHour && hour < endHour
			}
			return hour >= startHour || hour < endHour
		},
	}
}
