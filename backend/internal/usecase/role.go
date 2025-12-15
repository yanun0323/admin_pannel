package usecase

import (
	"context"
	"errors"

	"control_page/internal/adaptor"
	"control_page/internal/model"
	"control_page/internal/model/enum"
)

var _ adaptor.RoleUseCase = (*RoleUseCase)(nil)

var (
	ErrRoleNotFound     = errors.New("role not found")
	ErrRoleAlreadyExists = errors.New("role already exists")
)

type RoleUseCase struct {
	roleRepo adaptor.RoleRepository
}

func NewRoleUseCase(roleRepo adaptor.RoleRepository) *RoleUseCase {
	return &RoleUseCase{roleRepo: roleRepo}
}

func (uc *RoleUseCase) CreateRole(ctx context.Context, name, description string, permissions []enum.Permission) (*model.RoleWithPermissions, error) {
	// Check if role already exists
	existing, err := uc.roleRepo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrRoleAlreadyExists
	}

	role := &model.Role{
		Name:        name,
		Description: description,
	}

	if err := uc.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}

	// Set permissions
	if len(permissions) > 0 {
		if err := uc.roleRepo.SetPermissions(ctx, role.ID, permissions); err != nil {
			return nil, err
		}
	}

	return &model.RoleWithPermissions{
		Role:        *role,
		Permissions: permissions,
	}, nil
}

func (uc *RoleUseCase) GetRole(ctx context.Context, id string) (*model.RoleWithPermissions, error) {
	role, err := uc.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, ErrRoleNotFound
	}

	permissions, err := uc.roleRepo.GetPermissions(ctx, id)
	if err != nil {
		return nil, err
	}

	return &model.RoleWithPermissions{
		Role:        *role,
		Permissions: permissions,
	}, nil
}

func (uc *RoleUseCase) ListRoles(ctx context.Context) ([]model.RoleWithPermissions, error) {
	roles, err := uc.roleRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]model.RoleWithPermissions, len(roles))
	for i, role := range roles {
		permissions, err := uc.roleRepo.GetPermissions(ctx, role.ID)
		if err != nil {
			return nil, err
		}
		result[i] = model.RoleWithPermissions{
			Role:        role,
			Permissions: permissions,
		}
	}

	return result, nil
}

func (uc *RoleUseCase) UpdateRole(ctx context.Context, id string, name, description string) (*model.RoleWithPermissions, error) {
	role, err := uc.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, ErrRoleNotFound
	}

	// Check if another role with the same name exists
	if name != role.Name {
		existing, err := uc.roleRepo.GetByName(ctx, name)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, ErrRoleAlreadyExists
		}
	}

	role.Name = name
	role.Description = description

	if err := uc.roleRepo.Update(ctx, role); err != nil {
		return nil, err
	}

	permissions, err := uc.roleRepo.GetPermissions(ctx, id)
	if err != nil {
		return nil, err
	}

	return &model.RoleWithPermissions{
		Role:        *role,
		Permissions: permissions,
	}, nil
}

func (uc *RoleUseCase) DeleteRole(ctx context.Context, id string) error {
	role, err := uc.roleRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if role == nil {
		return ErrRoleNotFound
	}

	return uc.roleRepo.Delete(ctx, id)
}

func (uc *RoleUseCase) SetPermissions(ctx context.Context, roleID string, permissions []enum.Permission) error {
	role, err := uc.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return ErrRoleNotFound
	}

	return uc.roleRepo.SetPermissions(ctx, roleID, permissions)
}

func (uc *RoleUseCase) GetPermissions(ctx context.Context, roleID string) ([]enum.Permission, error) {
	return uc.roleRepo.GetPermissions(ctx, roleID)
}

func (uc *RoleUseCase) GetAllPermissions() []enum.Permission {
	return enum.AllPermissions()
}
