package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"control_page/internal/adaptor"
	"control_page/internal/model"
)

var _ adaptor.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (username, password, email, is_active, totp_secret, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, user.Username, user.Password, user.Email, user.IsActive, user.TOTPSecret, now, now)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = id
	user.CreatedAt = now
	user.UpdatedAt = now

	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	query := `SELECT id, username, password, email, is_active, totp_secret, totp_enabled, pending_totp_secret, created_at, updated_at FROM users WHERE id = ?`
	if err := r.db.GetContext(ctx, &user, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	query := `SELECT id, username, password, email, is_active, totp_secret, totp_enabled, pending_totp_secret, created_at, updated_at FROM users WHERE username = ?`
	if err := r.db.GetContext(ctx, &user, query, username); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	query := `SELECT id, username, password, email, is_active, totp_secret, totp_enabled, pending_totp_secret, created_at, updated_at FROM users WHERE email = ?`
	if err := r.db.GetContext(ctx, &user, query, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users SET username = ?, email = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`
	user.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, query, user.Username, user.Email, user.IsActive, user.UpdatedAt, user.ID)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM users WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *UserRepository) List(ctx context.Context) ([]model.User, error) {
	var users []model.User
	query := `SELECT id, username, password, email, is_active, totp_secret, totp_enabled, pending_totp_secret, created_at, updated_at FROM users WHERE totp_enabled = 1 ORDER BY id`
	if err := r.db.SelectContext(ctx, &users, query); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id int64, hashedPassword string) error {
	query := `UPDATE users SET password = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, hashedPassword, time.Now(), id)
	return err
}

func (r *UserRepository) UpdateUsername(ctx context.Context, id int64, username string) error {
	query := `UPDATE users SET username = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, username, time.Now(), id)
	return err
}

func (r *UserRepository) UpdateRegistration(ctx context.Context, id int64, email, hashedPassword, totpSecret string) error {
	query := `UPDATE users SET email = ?, password = ?, totp_secret = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, email, hashedPassword, totpSecret, time.Now(), id)
	return err
}

func (r *UserRepository) SetTOTPSecret(ctx context.Context, id int64, secret string) error {
	query := `UPDATE users SET totp_secret = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, secret, time.Now(), id)
	return err
}

func (r *UserRepository) EnableTOTP(ctx context.Context, id int64) error {
	query := `UPDATE users SET totp_enabled = 1, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

func (r *UserRepository) Activate(ctx context.Context, id int64) error {
	query := `UPDATE users SET is_active = 1, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

func (r *UserRepository) SetPendingTOTPSecret(ctx context.Context, id int64, secret string) error {
	query := `UPDATE users SET pending_totp_secret = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, secret, time.Now(), id)
	return err
}

func (r *UserRepository) ConfirmTOTPRebind(ctx context.Context, id int64) error {
	query := `UPDATE users SET totp_secret = pending_totp_secret, pending_totp_secret = NULL, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

func (r *UserRepository) ClearPendingTOTPSecret(ctx context.Context, id int64) error {
	query := `UPDATE users SET pending_totp_secret = NULL, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}
