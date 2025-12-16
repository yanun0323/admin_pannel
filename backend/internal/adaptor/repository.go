package adaptor

import (
	"context"

	"control_page/internal/model"
	"control_page/internal/model/enum"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]model.User, error)
	UpdatePassword(ctx context.Context, id string, hashedPassword string) error
	UpdateUsername(ctx context.Context, id string, username string) error
	UpdateRegistration(ctx context.Context, id string, hashedPassword, totpSecret string) error
	SetTOTPSecret(ctx context.Context, id string, secret string) error
	EnableTOTP(ctx context.Context, id string) error
	Activate(ctx context.Context, id string) error
	SetPendingTOTPSecret(ctx context.Context, id string, secret string) error
	ConfirmTOTPRebind(ctx context.Context, id string) error
	ClearPendingTOTPSecret(ctx context.Context, id string) error
}

// RoleRepository defines the interface for role data access
type RoleRepository interface {
	Create(ctx context.Context, role *model.Role) error
	GetByID(ctx context.Context, id string) (*model.Role, error)
	GetByName(ctx context.Context, name string) (*model.Role, error)
	Update(ctx context.Context, role *model.Role) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]model.Role, error)
	GetRolesByUserID(ctx context.Context, userID string) ([]model.Role, error)
	AddPermission(ctx context.Context, roleID string, permission enum.Permission) error
	RemovePermission(ctx context.Context, roleID string, permission enum.Permission) error
	GetPermissions(ctx context.Context, roleID string) ([]enum.Permission, error)
	SetPermissions(ctx context.Context, roleID string, permissions []enum.Permission) error
}

// UserRoleRepository defines the interface for user-role relationship data access
type UserRoleRepository interface {
	AssignRole(ctx context.Context, userID, roleID string) error
	RemoveRole(ctx context.Context, userID, roleID string) error
	GetUserPermissions(ctx context.Context, userID string) ([]enum.Permission, error)
}

// APIKeyRepository defines the interface for API key data access
type APIKeyRepository interface {
	Create(ctx context.Context, apiKey *model.APIKey) error
	GetByID(ctx context.Context, id string) (*model.APIKey, error)
	List(ctx context.Context) ([]model.APIKey, error)
	Update(ctx context.Context, apiKey *model.APIKey) error
	Delete(ctx context.Context, id string) error
	GetByPlatform(ctx context.Context, platform model.Platform) ([]model.APIKey, error)
	GetActiveByPlatform(ctx context.Context, platform model.Platform, isTestnet bool) ([]model.APIKey, error)
}

// SwitcherRepository defines the interface for switcher data access
type SwitcherRepository interface {
	GetAll(ctx context.Context) ([]model.Switcher, error)
	GetByID(ctx context.Context, id string) (*model.Switcher, error)
	Create(ctx context.Context, switcher *model.Switcher) error
	Update(ctx context.Context, switcher *model.Switcher) error
	UpdatePair(ctx context.Context, id string, pair string, enable bool) error
	Delete(ctx context.Context, id string) error
}

// SettingRepository defines the interface for setting data access
type SettingRepository interface {
	GetAll(ctx context.Context) ([]model.Setting, error)
	GetByID(ctx context.Context, id string) (*model.Setting, error)
	GetByBaseQuote(ctx context.Context, base, quote string) (*model.Setting, error)
	Create(ctx context.Context, setting *model.Setting) error
	Update(ctx context.Context, setting *model.Setting) error
	UpdateParameters(ctx context.Context, id string, strategy string, parameters map[string]interface{}) error
	Delete(ctx context.Context, id string) error
}
