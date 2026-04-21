package authz

// Role represents the access level granted to an API key or user.
type Role string

const (
	RoleOwner   Role = "owner"    // full platform access including billing
	RoleDev     Role = "dev"      // read/write, no billing or tenant admin
	RoleReadOnly Role = "readonly" // metrics, logs, dashboard — no mutations
)

// Permission is a fine-grained capability flag.
type Permission string

const (
	PermViewDashboard  Permission = "dashboard.view"
	PermManageTenant   Permission = "tenant.manage"
	PermManageKeys     Permission = "keys.manage"
	PermViewBilling    Permission = "billing.view"
	PermManageBilling  Permission = "billing.manage"
	PermRunStress      Permission = "stress.run"
	PermRunResLab      Permission = "reslab.run"
	PermManageServices Permission = "services.manage"
	PermViewLogs       Permission = "logs.view"
	PermViewMetrics    Permission = "metrics.view"
	PermWarRoom        Permission = "warroom.access"
)

var rolePerms = map[Role][]Permission{
	RoleOwner: {
		PermViewDashboard, PermManageTenant, PermManageKeys,
		PermViewBilling, PermManageBilling,
		PermRunStress, PermRunResLab,
		PermManageServices, PermViewLogs, PermViewMetrics,
		PermWarRoom,
	},
	RoleDev: {
		PermViewDashboard,
		PermRunStress, PermRunResLab,
		PermManageServices, PermViewLogs, PermViewMetrics,
	},
	RoleReadOnly: {
		PermViewDashboard, PermViewLogs, PermViewMetrics,
	},
}

// HasPermission returns true if the given role includes the requested permission.
func HasPermission(role Role, perm Permission) bool {
	for _, p := range rolePerms[role] {
		if p == perm {
			return true
		}
	}
	return false
}
