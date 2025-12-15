package model

import (
	"time"

	"control_page/internal/model/enum"
)

type User struct {
	ID                string    `json:"id"` // MongoDB ObjectID as string
	Username          string    `json:"username"`
	Password          string    `json:"-"`
	Email             string    `json:"email"`
	IsActive          bool      `json:"is_active"`
	TOTPSecret        *string   `json:"-"`
	TOTPEnabled       bool      `json:"totp_enabled"`
	PendingTOTPSecret *string   `json:"-"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Role struct {
	ID          string    `json:"id"` // MongoDB ObjectID as string
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RolePermission struct {
	ID         string          `json:"id"`
	RoleID     string          `json:"role_id"`
	Permission enum.Permission `json:"permission"`
}

type UserRole struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
	RoleID string `json:"role_id"`
}

type UserWithRoles struct {
	User
	Roles       []Role            `json:"roles"`
	Permissions []enum.Permission `json:"permissions"`
}

type RoleWithPermissions struct {
	Role
	Permissions []enum.Permission `json:"permissions"`
}

// LoginResult represents the result of a login attempt
type LoginResult struct {
	RequiresTOTP      bool           `json:"requires_totp"`
	RequiresTOTPSetup bool           `json:"requires_totp_setup"`
	Token             string         `json:"token,omitempty"`
	User              *UserWithRoles `json:"user,omitempty"`
	TempUserID        string         `json:"temp_user_id,omitempty"`
	TOTPSetup         *TOTPSetup     `json:"totp_setup,omitempty"`
}

// TOTPSetup contains information needed to set up TOTP
type TOTPSetup struct {
	Secret string `json:"secret"`
	QRCode string `json:"qr_code"`
}

// RegisterResult represents the result of a registration
type RegisterResult struct {
	UserID    string    `json:"user_id"`
	TOTPSetup TOTPSetup `json:"totp_setup"`
}
