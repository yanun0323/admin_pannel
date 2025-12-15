package enum

type Permission string

const (
	PermissionViewDashboard  Permission = "view:dashboard"
	PermissionViewKline      Permission = "view:kline"
	PermissionViewAPIKeys    Permission = "view:api_keys"
	PermissionViewSettings   Permission = "view:settings"
	PermissionManageUsers    Permission = "manage:users"
	PermissionManageRoles    Permission = "manage:roles"
	PermissionManageAPIKeys  Permission = "manage:api_keys"
	PermissionManageSettings Permission = "manage:settings"
)

func (p Permission) String() string {
	return string(p)
}

func AllPermissions() []Permission {
	return []Permission{
		PermissionViewDashboard,
		PermissionViewKline,
		PermissionViewAPIKeys,
		PermissionViewSettings,
		PermissionManageUsers,
		PermissionManageRoles,
		PermissionManageAPIKeys,
		PermissionManageSettings,
	}
}
