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

var _ adaptor.APIKeyRepository = (*APIKeyRepository)(nil)

type APIKeyRepository struct {
	db *sqlx.DB
}

func NewAPIKeyRepository(db *sqlx.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

func (r *APIKeyRepository) Create(ctx context.Context, apiKey *model.APIKey) error {
	query := `
		INSERT INTO api_keys (user_id, name, platform, api_key, api_secret, is_testnet, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		apiKey.UserID,
		apiKey.Name,
		apiKey.Platform,
		apiKey.APIKey,
		apiKey.APISecret,
		apiKey.IsTestnet,
		apiKey.IsActive,
		now,
		now,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	apiKey.ID = id
	apiKey.CreatedAt = now
	apiKey.UpdatedAt = now

	return nil
}

func (r *APIKeyRepository) GetByID(ctx context.Context, id int64) (*model.APIKey, error) {
	var apiKey model.APIKey
	query := `SELECT id, user_id, name, platform, api_key, api_secret, is_testnet, is_active, created_at, updated_at FROM api_keys WHERE id = ?`
	if err := r.db.GetContext(ctx, &apiKey, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &apiKey, nil
}

func (r *APIKeyRepository) GetByUserID(ctx context.Context, userID int64) ([]model.APIKey, error) {
	var apiKeys []model.APIKey
	query := `SELECT id, user_id, name, platform, api_key, api_secret, is_testnet, is_active, created_at, updated_at FROM api_keys WHERE user_id = ? ORDER BY created_at DESC`
	if err := r.db.SelectContext(ctx, &apiKeys, query, userID); err != nil {
		return nil, err
	}
	return apiKeys, nil
}

func (r *APIKeyRepository) GetByUserIDAndPlatform(ctx context.Context, userID int64, platform model.Platform) ([]model.APIKey, error) {
	var apiKeys []model.APIKey
	query := `SELECT id, user_id, name, platform, api_key, api_secret, is_testnet, is_active, created_at, updated_at FROM api_keys WHERE user_id = ? AND platform = ? ORDER BY created_at DESC`
	if err := r.db.SelectContext(ctx, &apiKeys, query, userID, platform); err != nil {
		return nil, err
	}
	return apiKeys, nil
}

func (r *APIKeyRepository) Update(ctx context.Context, apiKey *model.APIKey) error {
	query := `
		UPDATE api_keys SET name = ?, api_key = ?, api_secret = ?, is_testnet = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`
	apiKey.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, query,
		apiKey.Name,
		apiKey.APIKey,
		apiKey.APISecret,
		apiKey.IsTestnet,
		apiKey.IsActive,
		apiKey.UpdatedAt,
		apiKey.ID,
	)
	return err
}

func (r *APIKeyRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM api_keys WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *APIKeyRepository) GetActiveByUserIDAndPlatform(ctx context.Context, userID int64, platform model.Platform, isTestnet bool) ([]model.APIKey, error) {
	var apiKeys []model.APIKey
	query := `SELECT id, user_id, name, platform, api_key, api_secret, is_testnet, is_active, created_at, updated_at FROM api_keys WHERE user_id = ? AND platform = ? AND is_testnet = ? AND is_active = 1 ORDER BY created_at DESC`
	if err := r.db.SelectContext(ctx, &apiKeys, query, userID, platform, isTestnet); err != nil {
		return nil, err
	}
	return apiKeys, nil
}
