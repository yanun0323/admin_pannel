package model

import (
	"time"

	"control_page/internal/model/enum"
)

type User struct {
	ID                int64     `db:"id" json:"id"`
	Username          string    `db:"username" json:"username"`
	Password          string    `db:"password" json:"-"`
	Email             string    `db:"email" json:"email"`
	IsActive          bool      `db:"is_active" json:"is_active"`
	TOTPSecret        *string   `db:"totp_secret" json:"-"`
	TOTPEnabled       bool      `db:"totp_enabled" json:"totp_enabled"`
	PendingTOTPSecret *string   `db:"pending_totp_secret" json:"-"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
}

type Role struct {
	ID          int64     `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type RolePermission struct {
	ID         int64           `db:"id" json:"id"`
	RoleID     int64           `db:"role_id" json:"role_id"`
	Permission enum.Permission `db:"permission" json:"permission"`
}

type UserRole struct {
	ID     int64 `db:"id" json:"id"`
	UserID int64 `db:"user_id" json:"user_id"`
	RoleID int64 `db:"role_id" json:"role_id"`
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
	RequiresTOTP     bool           `json:"requires_totp"`
	RequiresTOTPSetup bool          `json:"requires_totp_setup"`
	Token            string         `json:"token,omitempty"`
	User             *UserWithRoles `json:"user,omitempty"`
	TempUserID       int64          `json:"temp_user_id,omitempty"`
	TOTPSetup        *TOTPSetup     `json:"totp_setup,omitempty"`
}

// TOTPSetup contains information needed to set up TOTP
type TOTPSetup struct {
	Secret string `json:"secret"`
	QRCode string `json:"qr_code"`
}

// RegisterResult represents the result of a registration
type RegisterResult struct {
	UserID    int64     `json:"user_id"`
	TOTPSetup TOTPSetup `json:"totp_setup"`
}
