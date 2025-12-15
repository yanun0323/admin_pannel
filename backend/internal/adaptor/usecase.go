package adaptor

import (
	"context"

	"control_page/internal/model"
	"control_page/internal/model/enum"
)

// AuthUseCase defines the interface for authentication operations
type AuthUseCase interface {
	Register(ctx context.Context, username, email, password string) (*model.RegisterResult, error)
	ActivateAccount(ctx context.Context, userID string, code string) error
	Login(ctx context.Context, username, password string) (*model.LoginResult, error)
	VerifyTOTP(ctx context.Context, userID string, code string) (string, *model.UserWithRoles, error)
	ValidateToken(ctx context.Context, token string) (*model.UserWithRoles, error)
	HasPermission(ctx context.Context, userID string, permission enum.Permission) (bool, error)
	ChangePassword(ctx context.Context, userID string, currentPassword, newPassword string) error
	SetupTOTPRebind(ctx context.Context, userID string, password string) (*model.TOTPSetup, error)
	ConfirmTOTPRebind(ctx context.Context, userID string, code string) error
	CancelTOTPRebind(ctx context.Context, userID string) error
}

// UserUseCase defines the interface for user management operations
type UserUseCase interface {
	GetUser(ctx context.Context, id string) (*model.UserWithRoles, error)
	ListUsers(ctx context.Context) ([]model.UserWithRoles, error)
	UpdateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, id string) error
	AssignRole(ctx context.Context, userID, roleID string) error
	RemoveRole(ctx context.Context, userID, roleID string) error
}

// RoleUseCase defines the interface for role management operations
type RoleUseCase interface {
	CreateRole(ctx context.Context, name, description string, permissions []enum.Permission) (*model.RoleWithPermissions, error)
	GetRole(ctx context.Context, id string) (*model.RoleWithPermissions, error)
	ListRoles(ctx context.Context) ([]model.RoleWithPermissions, error)
	UpdateRole(ctx context.Context, id string, name, description string) (*model.RoleWithPermissions, error)
	DeleteRole(ctx context.Context, id string) error
	SetPermissions(ctx context.Context, roleID string, permissions []enum.Permission) error
	GetPermissions(ctx context.Context, roleID string) ([]enum.Permission, error)
	GetAllPermissions() []enum.Permission
}

// KlineUseCase defines the interface for kline operations
type KlineUseCase interface {
	GetAvailableSymbols(ctx context.Context) ([]string, error)
	GetAvailableIntervals(ctx context.Context) ([]string, error)
}

// APIKeyUseCase defines the interface for API key management operations
type APIKeyUseCase interface {
	Create(ctx context.Context, req *model.CreateAPIKeyRequest) (*model.APIKeyResponse, error)
	GetByID(ctx context.Context, id string) (*model.APIKeyResponse, error)
	List(ctx context.Context) ([]model.APIKeyResponse, error)
	Update(ctx context.Context, id string, req *model.UpdateAPIKeyRequest) (*model.APIKeyResponse, error)
	Delete(ctx context.Context, id string) error
	GetPlatforms() []model.Platform
}

// SwitcherUseCase defines the interface for switcher management operations
type SwitcherUseCase interface {
	List(ctx context.Context) ([]model.SwitcherResponse, error)
	GetByID(ctx context.Context, id string) (*model.SwitcherResponse, error)
	Create(ctx context.Context, req *model.UpdateSwitcherRequest) (*model.SwitcherResponse, error)
	Update(ctx context.Context, id string, req *model.UpdateSwitcherRequest) (*model.SwitcherResponse, error)
	UpdatePair(ctx context.Context, id string, pair string, enable bool) (*model.SwitcherResponse, error)
	Delete(ctx context.Context, id string) error
}

// SettingUseCase defines the interface for setting management operations
type SettingUseCase interface {
	List(ctx context.Context) ([]model.SettingResponse, error)
	GetByID(ctx context.Context, id string) (*model.SettingResponse, error)
	GetByBaseQuote(ctx context.Context, base, quote string) (*model.SettingResponse, error)
	Create(ctx context.Context, req *model.CreateSettingRequest) (*model.SettingResponse, error)
	Update(ctx context.Context, id string, req *model.UpdateSettingRequest) (*model.SettingResponse, error)
	UpdateParameters(ctx context.Context, id string, strategy string, parameters map[string]interface{}) (*model.SettingResponse, error)
	Delete(ctx context.Context, id string) error
}
