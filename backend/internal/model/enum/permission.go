package enum

type Permission string

const (
	PermissionViewDashboard Permission = "view:dashboard"
	PermissionViewKline     Permission = "view:kline"
	PermissionManageUsers   Permission = "manage:users"
	PermissionManageRoles   Permission = "manage:roles"
	PermissionManageAPIKeys Permission = "manage:api_keys"
)

func (p Permission) String() string {
	return string(p)
}

func AllPermissions() []Permission {
	return []Permission{
		PermissionViewDashboard,
		PermissionViewKline,
		PermissionManageUsers,
		PermissionManageRoles,
		PermissionManageAPIKeys,
	}
}
