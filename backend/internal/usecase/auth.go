package usecase

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image/png"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"control_page/internal/adaptor"
	"control_page/internal/model"
	"control_page/internal/model/enum"
)

var _ adaptor.AuthUseCase = (*AuthUseCase)(nil)

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrUserAlreadyExists    = errors.New("user already exists")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUserInactive         = errors.New("user is inactive")
	ErrUserNotActivated     = errors.New("user account not activated")
	ErrInvalidToken         = errors.New("invalid token")
	ErrIncorrectPassword    = errors.New("incorrect current password")
	ErrPasswordSameAsOld    = errors.New("new password cannot be the same as current password")
	ErrInvalidTOTPCode      = errors.New("invalid TOTP code")
	ErrTOTPNotSetup         = errors.New("TOTP is not set up")
)

type AuthUseCase struct {
	userRepo     adaptor.UserRepository
	roleRepo     adaptor.RoleRepository
	userRoleRepo adaptor.UserRoleRepository
	jwtSecret    []byte
	jwtExpiry    time.Duration
	appName      string
}

func NewAuthUseCase(
	userRepo adaptor.UserRepository,
	roleRepo adaptor.RoleRepository,
	userRoleRepo adaptor.UserRoleRepository,
	jwtSecret string,
	jwtExpiry time.Duration,
) *AuthUseCase {
	return &AuthUseCase{
		userRepo:     userRepo,
		roleRepo:     roleRepo,
		userRoleRepo: userRoleRepo,
		jwtSecret:    []byte(jwtSecret),
		jwtExpiry:    jwtExpiry,
		appName:      "Nova",
	}
}

func (uc *AuthUseCase) Register(ctx context.Context, username, email, password string) (*model.RegisterResult, error) {
	// Check if user exists by username
	existingUser, err := uc.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	// If user exists and already activated (2FA enabled), reject registration
	if existingUser != nil && existingUser.TOTPEnabled {
		return nil, ErrUserAlreadyExists
	}

	// Check if email is used by another activated user
	emailUser, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if emailUser != nil && emailUser.TOTPEnabled {
		return nil, ErrUserAlreadyExists
	}
	// If email belongs to a different inactive user, that's also a conflict
	if emailUser != nil && existingUser != nil && emailUser.ID != existingUser.ID {
		return nil, ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Generate TOTP key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      uc.appName,
		AccountName: username,
	})
	if err != nil {
		return nil, err
	}

	totpSecret := key.Secret()
	var userID int64

	if existingUser != nil {
		// User exists but not activated - update their credentials and TOTP secret
		if err := uc.userRepo.UpdateRegistration(ctx, existingUser.ID, email, string(hashedPassword), totpSecret); err != nil {
			return nil, err
		}
		userID = existingUser.ID
	} else if emailUser != nil {
		// Email exists with inactive user - update their credentials and TOTP secret
		if err := uc.userRepo.UpdateRegistration(ctx, emailUser.ID, email, string(hashedPassword), totpSecret); err != nil {
			return nil, err
		}
		// Also update username since they're re-registering
		if err := uc.userRepo.UpdateUsername(ctx, emailUser.ID, username); err != nil {
			return nil, err
		}
		userID = emailUser.ID
	} else {
		// New user - create
		user := &model.User{
			Username:   username,
			Email:      email,
			Password:   string(hashedPassword),
			IsActive:   false,
			TOTPSecret: &totpSecret,
		}

		if err := uc.userRepo.Create(ctx, user); err != nil {
			return nil, err
		}
		userID = user.ID

		// Assign default user role only for new users
		userRole, err := uc.roleRepo.GetByName(ctx, "user")
		if err != nil {
			return nil, err
		}
		if userRole != nil {
			if err := uc.userRoleRepo.AssignRole(ctx, userID, userRole.ID); err != nil {
				return nil, err
			}
		}
	}

	// Generate QR code
	img, err := key.Image(200, 200)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	qrCode := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	return &model.RegisterResult{
		UserID: userID,
		TOTPSetup: model.TOTPSetup{
			Secret: key.Secret(),
			QRCode: qrCode,
		},
	}, nil
}

func (uc *AuthUseCase) ActivateAccount(ctx context.Context, userID int64, code string) error {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	if user.IsActive && user.TOTPEnabled {
		return nil // Already activated
	}

	if user.TOTPSecret == nil {
		return ErrTOTPNotSetup
	}

	// Validate TOTP code
	if !totp.Validate(code, *user.TOTPSecret) {
		return ErrInvalidTOTPCode
	}

	// Enable TOTP and activate user
	if err := uc.userRepo.EnableTOTP(ctx, userID); err != nil {
		return err
	}

	return uc.userRepo.Activate(ctx, userID)
}

func (uc *AuthUseCase) Login(ctx context.Context, username, password string) (*model.LoginResult, error) {
	user, err := uc.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Check if 2FA is enabled
	if !user.TOTPEnabled {
		// User needs to setup 2FA first
		// Generate TOTP setup data
		totpSetup, err := uc.generateTOTPSetup(ctx, user)
		if err != nil {
			return nil, err
		}

		return &model.LoginResult{
			RequiresTOTPSetup: true,
			TempUserID:        user.ID,
			TOTPSetup:         totpSetup,
		}, nil
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// 2FA is mandatory, always require TOTP verification
	return &model.LoginResult{
		RequiresTOTP: true,
		TempUserID:   user.ID,
	}, nil
}

func (uc *AuthUseCase) VerifyTOTP(ctx context.Context, userID int64, code string) (string, *model.UserWithRoles, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", nil, err
	}
	if user == nil {
		return "", nil, ErrUserNotFound
	}

	if !user.TOTPEnabled || user.TOTPSecret == nil {
		return "", nil, ErrTOTPNotSetup
	}

	// Validate TOTP code
	if !totp.Validate(code, *user.TOTPSecret) {
		return "", nil, ErrInvalidTOTPCode
	}

	result, err := uc.completeLogin(ctx, user)
	if err != nil {
		return "", nil, err
	}

	return result.Token, result.User, nil
}

func (uc *AuthUseCase) completeLogin(ctx context.Context, user *model.User) (*model.LoginResult, error) {
	// Get user roles and permissions
	roles, err := uc.roleRepo.GetRolesByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	permissions, err := uc.userRoleRepo.GetUserPermissions(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	// Generate JWT token
	token, err := uc.generateToken(user.ID, user.Username)
	if err != nil {
		return nil, err
	}

	userWithRoles := &model.UserWithRoles{
		User:        *user,
		Roles:       roles,
		Permissions: permissions,
	}

	return &model.LoginResult{
		RequiresTOTP: false,
		Token:        token,
		User:         userWithRoles,
	}, nil
}

func (uc *AuthUseCase) ValidateToken(ctx context.Context, tokenString string) (*model.UserWithRoles, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return uc.jwtSecret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		return nil, ErrInvalidToken
	}

	user, err := uc.userRepo.GetByID(ctx, int64(userID))
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	roles, err := uc.roleRepo.GetRolesByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	permissions, err := uc.userRoleRepo.GetUserPermissions(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &model.UserWithRoles{
		User:        *user,
		Roles:       roles,
		Permissions: permissions,
	}, nil
}

func (uc *AuthUseCase) HasPermission(ctx context.Context, userID int64, permission enum.Permission) (bool, error) {
	permissions, err := uc.userRoleRepo.GetUserPermissions(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, p := range permissions {
		if p == permission {
			return true, nil
		}
	}
	return false, nil
}

func (uc *AuthUseCase) ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string) error {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return ErrIncorrectPassword
	}

	// Check if new password is same as current
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(newPassword)); err == nil {
		return ErrPasswordSameAsOld
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return uc.userRepo.UpdatePassword(ctx, userID, string(hashedPassword))
}

func (uc *AuthUseCase) generateToken(userID int64, username string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(uc.jwtExpiry).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(uc.jwtSecret)
}

func (uc *AuthUseCase) generateTOTPSetup(ctx context.Context, user *model.User) (*model.TOTPSetup, error) {
	// Generate new TOTP key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      uc.appName,
		AccountName: user.Username,
	})
	if err != nil {
		return nil, err
	}

	secret := key.Secret()

	// Store the secret (overwrite any existing one since user hasn't activated yet)
	if err := uc.userRepo.SetTOTPSecret(ctx, user.ID, secret); err != nil {
		return nil, err
	}

	// Generate QR code
	img, err := key.Image(200, 200)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	qrCode := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	return &model.TOTPSetup{
		Secret: secret,
		QRCode: qrCode,
	}, nil
}

func (uc *AuthUseCase) SetupTOTPRebind(ctx context.Context, userID int64, password string) (*model.TOTPSetup, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// Verify password before allowing rebind
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrIncorrectPassword
	}

	// Generate new TOTP key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      uc.appName,
		AccountName: user.Username,
	})
	if err != nil {
		return nil, err
	}

	// Store the new secret temporarily (will be activated after verification)
	// We store it but keep TOTPEnabled true so user can still login with old code
	if err := uc.userRepo.SetPendingTOTPSecret(ctx, userID, key.Secret()); err != nil {
		return nil, err
	}

	// Generate QR code
	img, err := key.Image(200, 200)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	qrCode := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	return &model.TOTPSetup{
		Secret: key.Secret(),
		QRCode: qrCode,
	}, nil
}

func (uc *AuthUseCase) ConfirmTOTPRebind(ctx context.Context, userID int64, code string) error {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	if user.PendingTOTPSecret == nil {
		return ErrTOTPNotSetup
	}

	// Validate the code with the new pending secret
	if !totp.Validate(code, *user.PendingTOTPSecret) {
		return ErrInvalidTOTPCode
	}

	// Confirm the rebind: move pending secret to active secret
	return uc.userRepo.ConfirmTOTPRebind(ctx, userID)
}

func (uc *AuthUseCase) CancelTOTPRebind(ctx context.Context, userID int64) error {
	return uc.userRepo.ClearPendingTOTPSecret(ctx, userID)
}
