package adaptor

import (
	"context"

	"control_page/internal/model"
	"control_page/internal/model/enum"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id int64) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]model.User, error)
	UpdatePassword(ctx context.Context, id int64, hashedPassword string) error
	UpdateUsername(ctx context.Context, id int64, username string) error
	UpdateRegistration(ctx context.Context, id int64, email, hashedPassword, totpSecret string) error
	SetTOTPSecret(ctx context.Context, id int64, secret string) error
	EnableTOTP(ctx context.Context, id int64) error
	Activate(ctx context.Context, id int64) error
	SetPendingTOTPSecret(ctx context.Context, id int64, secret string) error
	ConfirmTOTPRebind(ctx context.Context, id int64) error
	ClearPendingTOTPSecret(ctx context.Context, id int64) error
}

// RoleRepository defines the interface for role data access
type RoleRepository interface {
	Create(ctx context.Context, role *model.Role) error
	GetByID(ctx context.Context, id int64) (*model.Role, error)
	GetByName(ctx context.Context, name string) (*model.Role, error)
	Update(ctx context.Context, role *model.Role) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]model.Role, error)
	GetRolesByUserID(ctx context.Context, userID int64) ([]model.Role, error)
	AddPermission(ctx context.Context, roleID int64, permission enum.Permission) error
	RemovePermission(ctx context.Context, roleID int64, permission enum.Permission) error
	GetPermissions(ctx context.Context, roleID int64) ([]enum.Permission, error)
	SetPermissions(ctx context.Context, roleID int64, permissions []enum.Permission) error
}

// UserRoleRepository defines the interface for user-role relationship data access
type UserRoleRepository interface {
	AssignRole(ctx context.Context, userID, roleID int64) error
	RemoveRole(ctx context.Context, userID, roleID int64) error
	GetUserPermissions(ctx context.Context, userID int64) ([]enum.Permission, error)
}

// APIKeyRepository defines the interface for API key data access
type APIKeyRepository interface {
	Create(ctx context.Context, apiKey *model.APIKey) error
	GetByID(ctx context.Context, id int64) (*model.APIKey, error)
	GetByUserID(ctx context.Context, userID int64) ([]model.APIKey, error)
	GetByUserIDAndPlatform(ctx context.Context, userID int64, platform model.Platform) ([]model.APIKey, error)
	Update(ctx context.Context, apiKey *model.APIKey) error
	Delete(ctx context.Context, id int64) error
	GetActiveByUserIDAndPlatform(ctx context.Context, userID int64, platform model.Platform, isTestnet bool) ([]model.APIKey, error)
}
