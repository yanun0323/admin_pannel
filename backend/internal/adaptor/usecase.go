package adaptor

import (
	"context"

	"control_page/internal/model"
	"control_page/internal/model/enum"
)

// AuthUseCase defines the interface for authentication operations
type AuthUseCase interface {
	Register(ctx context.Context, username, email, password string) (*model.RegisterResult, error)
	ActivateAccount(ctx context.Context, userID int64, code string) error
	Login(ctx context.Context, username, password string) (*model.LoginResult, error)
	VerifyTOTP(ctx context.Context, userID int64, code string) (string, *model.UserWithRoles, error)
	ValidateToken(ctx context.Context, token string) (*model.UserWithRoles, error)
	HasPermission(ctx context.Context, userID int64, permission enum.Permission) (bool, error)
	ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string) error
	SetupTOTPRebind(ctx context.Context, userID int64, password string) (*model.TOTPSetup, error)
	ConfirmTOTPRebind(ctx context.Context, userID int64, code string) error
	CancelTOTPRebind(ctx context.Context, userID int64) error
}

// UserUseCase defines the interface for user management operations
type UserUseCase interface {
	GetUser(ctx context.Context, id int64) (*model.UserWithRoles, error)
	ListUsers(ctx context.Context) ([]model.UserWithRoles, error)
	UpdateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, id int64) error
	AssignRole(ctx context.Context, userID, roleID int64) error
	RemoveRole(ctx context.Context, userID, roleID int64) error
}

// RoleUseCase defines the interface for role management operations
type RoleUseCase interface {
	CreateRole(ctx context.Context, name, description string, permissions []enum.Permission) (*model.RoleWithPermissions, error)
	GetRole(ctx context.Context, id int64) (*model.RoleWithPermissions, error)
	ListRoles(ctx context.Context) ([]model.RoleWithPermissions, error)
	UpdateRole(ctx context.Context, id int64, name, description string) (*model.RoleWithPermissions, error)
	DeleteRole(ctx context.Context, id int64) error
	SetPermissions(ctx context.Context, roleID int64, permissions []enum.Permission) error
	GetPermissions(ctx context.Context, roleID int64) ([]enum.Permission, error)
	GetAllPermissions() []enum.Permission
}

// KlineUseCase defines the interface for kline operations
type KlineUseCase interface {
	GetAvailableSymbols(ctx context.Context) ([]string, error)
	GetAvailableIntervals(ctx context.Context) ([]string, error)
}

// APIKeyUseCase defines the interface for API key management operations
type APIKeyUseCase interface {
	Create(ctx context.Context, userID int64, req *model.CreateAPIKeyRequest) (*model.APIKeyResponse, error)
	GetByID(ctx context.Context, userID, id int64) (*model.APIKeyResponse, error)
	List(ctx context.Context, userID int64) ([]model.APIKeyResponse, error)
	Update(ctx context.Context, userID, id int64, req *model.UpdateAPIKeyRequest) (*model.APIKeyResponse, error)
	Delete(ctx context.Context, userID, id int64) error
	GetPlatforms() []model.Platform
}
