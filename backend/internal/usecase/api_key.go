package usecase

import (
	"context"
	"errors"

	"control_page/internal/adaptor"
	"control_page/internal/model"
)

var (
	ErrAPIKeyNotFound    = errors.New("api key not found")
	ErrAPIKeyUnauthorized = errors.New("unauthorized to access this api key")
	ErrInvalidPlatform   = errors.New("invalid platform")
	ErrAPIKeyNameEmpty   = errors.New("api key name is required")
	ErrAPIKeyEmpty       = errors.New("api key is required")
	ErrAPISecretEmpty    = errors.New("api secret is required")
)

var _ adaptor.APIKeyUseCase = (*APIKeyUseCase)(nil)

type APIKeyUseCase struct {
	apiKeyRepo adaptor.APIKeyRepository
}

func NewAPIKeyUseCase(apiKeyRepo adaptor.APIKeyRepository) *APIKeyUseCase {
	return &APIKeyUseCase{
		apiKeyRepo: apiKeyRepo,
	}
}

func (uc *APIKeyUseCase) Create(ctx context.Context, userID int64, req *model.CreateAPIKeyRequest) (*model.APIKeyResponse, error) {
	if req.Name == "" {
		return nil, ErrAPIKeyNameEmpty
	}
	if req.APIKey == "" {
		return nil, ErrAPIKeyEmpty
	}
	if req.APISecret == "" {
		return nil, ErrAPISecretEmpty
	}
	if !req.Platform.IsValid() {
		return nil, ErrInvalidPlatform
	}

	apiKey := &model.APIKey{
		UserID:    userID,
		Name:      req.Name,
		Platform:  req.Platform,
		APIKey:    req.APIKey,
		APISecret: req.APISecret,
		IsTestnet: req.IsTestnet,
		IsActive:  true,
	}

	if err := uc.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, err
	}

	response := apiKey.ToResponse()
	return &response, nil
}

func (uc *APIKeyUseCase) GetByID(ctx context.Context, userID, id int64) (*model.APIKeyResponse, error) {
	apiKey, err := uc.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if apiKey == nil {
		return nil, ErrAPIKeyNotFound
	}
	if apiKey.UserID != userID {
		return nil, ErrAPIKeyUnauthorized
	}

	response := apiKey.ToResponse()
	return &response, nil
}

func (uc *APIKeyUseCase) List(ctx context.Context, userID int64) ([]model.APIKeyResponse, error) {
	apiKeys, err := uc.apiKeyRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]model.APIKeyResponse, len(apiKeys))
	for i, key := range apiKeys {
		responses[i] = key.ToResponse()
	}

	return responses, nil
}

func (uc *APIKeyUseCase) Update(ctx context.Context, userID, id int64, req *model.UpdateAPIKeyRequest) (*model.APIKeyResponse, error) {
	apiKey, err := uc.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if apiKey == nil {
		return nil, ErrAPIKeyNotFound
	}
	if apiKey.UserID != userID {
		return nil, ErrAPIKeyUnauthorized
	}

	if req.Name != nil {
		if *req.Name == "" {
			return nil, ErrAPIKeyNameEmpty
		}
		apiKey.Name = *req.Name
	}
	if req.APIKey != nil {
		if *req.APIKey == "" {
			return nil, ErrAPIKeyEmpty
		}
		apiKey.APIKey = *req.APIKey
	}
	if req.APISecret != nil {
		if *req.APISecret == "" {
			return nil, ErrAPISecretEmpty
		}
		apiKey.APISecret = *req.APISecret
	}
	if req.IsTestnet != nil {
		apiKey.IsTestnet = *req.IsTestnet
	}
	if req.IsActive != nil {
		apiKey.IsActive = *req.IsActive
	}

	if err := uc.apiKeyRepo.Update(ctx, apiKey); err != nil {
		return nil, err
	}

	response := apiKey.ToResponse()
	return &response, nil
}

func (uc *APIKeyUseCase) Delete(ctx context.Context, userID, id int64) error {
	apiKey, err := uc.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if apiKey == nil {
		return ErrAPIKeyNotFound
	}
	if apiKey.UserID != userID {
		return ErrAPIKeyUnauthorized
	}

	return uc.apiKeyRepo.Delete(ctx, id)
}

func (uc *APIKeyUseCase) GetPlatforms() []model.Platform {
	return model.AllPlatforms()
}
