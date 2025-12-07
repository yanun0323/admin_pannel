package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"control_page/internal/adaptor"
	"control_page/internal/model"
	"control_page/internal/model/enum"
)

var _ adaptor.RoleRepository = (*RoleRepository)(nil)

type RoleRepository struct {
	db *sqlx.DB
}

func NewRoleRepository(db *sqlx.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) Create(ctx context.Context, role *model.Role) error {
	query := `
		INSERT INTO roles (name, description, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`
	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, role.Name, role.Description, now, now)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	role.ID = id
	role.CreatedAt = now
	role.UpdatedAt = now

	return nil
}

func (r *RoleRepository) GetByID(ctx context.Context, id int64) (*model.Role, error) {
	var role model.Role
	query := `SELECT id, name, description, created_at, updated_at FROM roles WHERE id = ?`
	if err := r.db.GetContext(ctx, &role, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) GetByName(ctx context.Context, name string) (*model.Role, error) {
	var role model.Role
	query := `SELECT id, name, description, created_at, updated_at FROM roles WHERE name = ?`
	if err := r.db.GetContext(ctx, &role, query, name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) Update(ctx context.Context, role *model.Role) error {
	query := `
		UPDATE roles SET name = ?, description = ?, updated_at = ?
		WHERE id = ?
	`
	role.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, query, role.Name, role.Description, role.UpdatedAt, role.ID)
	return err
}

func (r *RoleRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM roles WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *RoleRepository) List(ctx context.Context) ([]model.Role, error) {
	var roles []model.Role
	query := `SELECT id, name, description, created_at, updated_at FROM roles ORDER BY id`
	if err := r.db.SelectContext(ctx, &roles, query); err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *RoleRepository) GetRolesByUserID(ctx context.Context, userID int64) ([]model.Role, error) {
	var roles []model.Role
	query := `
		SELECT r.id, r.name, r.description, r.created_at, r.updated_at
		FROM roles r
		INNER JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = ?
		ORDER BY r.id
	`
	if err := r.db.SelectContext(ctx, &roles, query, userID); err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *RoleRepository) AddPermission(ctx context.Context, roleID int64, permission enum.Permission) error {
	query := `INSERT OR IGNORE INTO role_permissions (role_id, permission) VALUES (?, ?)`
	_, err := r.db.ExecContext(ctx, query, roleID, permission.String())
	return err
}

func (r *RoleRepository) RemovePermission(ctx context.Context, roleID int64, permission enum.Permission) error {
	query := `DELETE FROM role_permissions WHERE role_id = ? AND permission = ?`
	_, err := r.db.ExecContext(ctx, query, roleID, permission.String())
	return err
}

func (r *RoleRepository) GetPermissions(ctx context.Context, roleID int64) ([]enum.Permission, error) {
	var permissions []string
	query := `SELECT permission FROM role_permissions WHERE role_id = ?`
	if err := r.db.SelectContext(ctx, &permissions, query, roleID); err != nil {
		return nil, err
	}

	result := make([]enum.Permission, len(permissions))
	for i, p := range permissions {
		result[i] = enum.Permission(p)
	}
	return result, nil
}

func (r *RoleRepository) SetPermissions(ctx context.Context, roleID int64, permissions []enum.Permission) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing permissions
	deleteQuery := `DELETE FROM role_permissions WHERE role_id = ?`
	if _, err := tx.ExecContext(ctx, deleteQuery, roleID); err != nil {
		return err
	}

	// Insert new permissions
	insertQuery := `INSERT INTO role_permissions (role_id, permission) VALUES (?, ?)`
	for _, p := range permissions {
		if _, err := tx.ExecContext(ctx, insertQuery, roleID, p.String()); err != nil {
			return err
		}
	}

	return tx.Commit()
}
