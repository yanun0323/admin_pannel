package usecase

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"

	"github.com/pquerna/otp/totp"

	"control_page/internal/adaptor"
	"control_page/internal/model"
)

var _ adaptor.UserUseCase = (*UserUseCase)(nil)

type UserUseCase struct {
	userRepo     adaptor.UserRepository
	roleRepo     adaptor.RoleRepository
	userRoleRepo adaptor.UserRoleRepository
}

func NewUserUseCase(
	userRepo adaptor.UserRepository,
	roleRepo adaptor.RoleRepository,
	userRoleRepo adaptor.UserRoleRepository,
) *UserUseCase {
	return &UserUseCase{
		userRepo:     userRepo,
		roleRepo:     roleRepo,
		userRoleRepo: userRoleRepo,
	}
}

func (uc *UserUseCase) GetUser(ctx context.Context, id string) (*model.UserWithRoles, error) {
	user, err := uc.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
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

func (uc *UserUseCase) ListUsers(ctx context.Context) ([]model.UserWithRoles, error) {
	users, err := uc.userRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]model.UserWithRoles, len(users))
	for i, user := range users {
		roles, err := uc.roleRepo.GetRolesByUserID(ctx, user.ID)
		if err != nil {
			return nil, err
		}

		permissions, err := uc.userRoleRepo.GetUserPermissions(ctx, user.ID)
		if err != nil {
			return nil, err
		}

		result[i] = model.UserWithRoles{
			User:        user,
			Roles:       roles,
			Permissions: permissions,
		}
	}

	return result, nil
}

func (uc *UserUseCase) CreateUser(ctx context.Context, user *model.User, roleIDs []string) (*model.UserWithRoles, error) {
	if user.Username == "" || user.Password == "" {
		return nil, fmt.Errorf("username and password are required")
	}

	// Ensure username unique
	existing, err := uc.userRepo.GetByUsername(ctx, user.Username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("user already exists")
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Assign roles if provided
	for _, roleID := range roleIDs {
		if err := uc.userRoleRepo.AssignRole(ctx, user.ID, roleID); err != nil {
			return nil, err
		}
	}

	roles, err := uc.roleRepo.GetRolesByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	perms, err := uc.userRoleRepo.GetUserPermissions(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &model.UserWithRoles{
		User:        *user,
		Roles:       roles,
		Permissions: perms,
	}, nil
}

func (uc *UserUseCase) UpdateUser(ctx context.Context, user *model.User) error {
	return uc.userRepo.Update(ctx, user)
}

func (uc *UserUseCase) DeleteUser(ctx context.Context, id string) error {
	return uc.userRepo.Delete(ctx, id)
}

func (uc *UserUseCase) AssignRole(ctx context.Context, userID, roleID string) error {
	return uc.userRoleRepo.AssignRole(ctx, userID, roleID)
}

func (uc *UserUseCase) RemoveRole(ctx context.Context, userID, roleID string) error {
	return uc.userRoleRepo.RemoveRole(ctx, userID, roleID)
}

func (uc *UserUseCase) ResetUserTOTP(ctx context.Context, userID string) (*model.TOTPSetup, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Nova",
		AccountName: user.Username,
	})
	if err != nil {
		return nil, err
	}

	secret := key.Secret()
	if err := uc.userRepo.SetTOTPSecret(ctx, userID, secret); err != nil {
		return nil, err
	}
	if err := uc.userRepo.EnableTOTP(ctx, userID); err != nil {
		return nil, err
	}

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
