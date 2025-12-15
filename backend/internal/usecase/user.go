package usecase

import (
	"context"

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
