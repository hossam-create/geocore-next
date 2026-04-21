package security

import (
	"context"
	"sync"
)

// IAMPolicy defines allowed actions for a service.
type IAMPolicy struct {
	Service string   `json:"service"`
	Actions []string `json:"actions"`
}

// IAMEnforcer evaluates service-level access control.
type IAMEnforcer struct {
	mu      sync.RWMutex
	policies map[string]*IAMPolicy
}

// NewIAMEnforcer creates an enforcer with no policies.
func NewIAMEnforcer() *IAMEnforcer {
	return &IAMEnforcer{
		policies: make(map[string]*IAMPolicy),
	}
}

// SetPolicy sets the IAM policy for a service.
func (e *IAMEnforcer) SetPolicy(service string, actions []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.policies[service] = &IAMPolicy{Service: service, Actions: actions}
}

// Enforce checks if the service in context is allowed to perform the action.
// Returns true if allowed, false if denied or no policy found.
func (e *IAMEnforcer) Enforce(ctx context.Context, action string) bool {
	svc, _ := ctx.Value(serviceCtxKey).(string)
	if svc == "" {
		return false
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	policy, ok := e.policies[svc]
	if !ok {
		return false // deny by default
	}

	return contains(policy.Actions, action)
}

// EnforceService checks if a specific service is allowed to perform an action.
func (e *IAMEnforcer) EnforceService(service, action string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	policy, ok := e.policies[service]
	if !ok {
		return false
	}
	return contains(policy.Actions, action)
}

// GetPolicy returns the policy for a service (nil if not found).
func (e *IAMEnforcer) GetPolicy(service string) *IAMPolicy {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.policies[service]
}

// AllPolicies returns all registered policies.
func (e *IAMEnforcer) AllPolicies() []IAMPolicy {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var result []IAMPolicy
	for _, p := range e.policies {
		result = append(result, *p)
	}
	return result
}

type contextKey string

const serviceCtxKey contextKey = "service"

// WithService returns a context with the service identity set.
func WithService(ctx context.Context, service string) context.Context {
	return context.WithValue(ctx, serviceCtxKey, service)
}

// GetServiceFromContext extracts the service identity from context.
func GetServiceFromContext(ctx context.Context) string {
	svc, _ := ctx.Value(serviceCtxKey).(string)
	return svc
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
