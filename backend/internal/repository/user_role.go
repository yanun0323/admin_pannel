package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"control_page/internal/adaptor"
	"control_page/internal/model/enum"
)

var _ adaptor.UserRoleRepository = (*UserRoleRepository)(nil)

type UserRoleRepository struct {
	db *sqlx.DB
}

func NewUserRoleRepository(db *sqlx.DB) *UserRoleRepository {
	return &UserRoleRepository{db: db}
}

func (r *UserRoleRepository) AssignRole(ctx context.Context, userID, roleID int64) error {
	query := `INSERT OR IGNORE INTO user_roles (user_id, role_id) VALUES (?, ?)`
	_, err := r.db.ExecContext(ctx, query, userID, roleID)
	return err
}

func (r *UserRoleRepository) RemoveRole(ctx context.Context, userID, roleID int64) error {
	query := `DELETE FROM user_roles WHERE user_id = ? AND role_id = ?`
	_, err := r.db.ExecContext(ctx, query, userID, roleID)
	return err
}

func (r *UserRoleRepository) GetUserPermissions(ctx context.Context, userID int64) ([]enum.Permission, error) {
	var permissions []string
	query := `
		SELECT DISTINCT rp.permission
		FROM role_permissions rp
		INNER JOIN user_roles ur ON rp.role_id = ur.role_id
		WHERE ur.user_id = ?
	`
	if err := r.db.SelectContext(ctx, &permissions, query, userID); err != nil {
		return nil, err
	}

	result := make([]enum.Permission, len(permissions))
	for i, p := range permissions {
		result[i] = enum.Permission(p)
	}
	return result, nil
}
