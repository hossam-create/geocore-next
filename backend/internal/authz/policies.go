package authz

import "fmt"

// Allow evaluates whether a caller can access a resource.
//
// Rules:
//  1. Cross-tenant access is always denied (hard isolation).
//  2. The caller's role must include the requested permission.
func Allow(callerTenantID, resourceTenantID string, role Role, perm Permission) error {
	if callerTenantID != resourceTenantID {
		return fmt.Errorf("access denied: cross-tenant resource access is prohibited")
	}
	if !HasPermission(role, perm) {
		return fmt.Errorf("access denied: role %q does not include permission %q", role, perm)
	}
	return nil
}

// AllowFeature checks if a plan grants access to a named feature.
// Returns an error with an upgrade hint when the plan is insufficient.
func AllowFeature(planID, feature string) error {
	featurePlans := map[string][]string{
		"aiops":        {"pro", "enterprise"},
		"chaos":        {"pro", "enterprise"},
		"reslab":       {"pro", "enterprise"},
		"multi_region": {"pro", "enterprise"},
		"warroom":      {"pro", "enterprise"},
	}
	required, ok := featurePlans[feature]
	if !ok {
		return nil // feature has no plan restriction
	}
	for _, p := range required {
		if p == planID {
			return nil
		}
	}
	return fmt.Errorf("feature %q requires plan: %v — upgrade at https://geocore.app/pricing", feature, required)
}
