package usecase

import (
	"context"
	"errors"

	"control_page/internal/adaptor"
	"control_page/internal/model"
)

var (
	ErrSettingNotFound     = errors.New("setting not found")
	ErrSettingBaseEmpty    = errors.New("base is required")
	ErrSettingQuoteEmpty   = errors.New("quote is required")
	ErrSettingStrategyEmpty = errors.New("strategy is required")
)

var _ adaptor.SettingUseCase = (*SettingUseCase)(nil)

type SettingUseCase struct {
	settingRepo adaptor.SettingRepository
}

func NewSettingUseCase(settingRepo adaptor.SettingRepository) *SettingUseCase {
	return &SettingUseCase{
		settingRepo: settingRepo,
	}
}

func (uc *SettingUseCase) List(ctx context.Context) ([]model.SettingResponse, error) {
	settings, err := uc.settingRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]model.SettingResponse, len(settings))
	for i, s := range settings {
		responses[i] = s.ToResponse()
	}

	return responses, nil
}

func (uc *SettingUseCase) GetByID(ctx context.Context, id string) (*model.SettingResponse, error) {
	setting, err := uc.settingRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		return nil, ErrSettingNotFound
	}

	response := setting.ToResponse()
	return &response, nil
}

func (uc *SettingUseCase) GetByBaseQuote(ctx context.Context, base, quote string) (*model.SettingResponse, error) {
	setting, err := uc.settingRepo.GetByBaseQuote(ctx, base, quote)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		return nil, ErrSettingNotFound
	}

	response := setting.ToResponse()
	return &response, nil
}

func (uc *SettingUseCase) Create(ctx context.Context, req *model.CreateSettingRequest) (*model.SettingResponse, error) {
	if req.Base == "" {
		return nil, ErrSettingBaseEmpty
	}
	if req.Quote == "" {
		return nil, ErrSettingQuoteEmpty
	}
	if req.Strategy == "" {
		return nil, ErrSettingStrategyEmpty
	}

	setting := &model.Setting{
		Base:       req.Base,
		Quote:      req.Quote,
		Strategy:   req.Strategy,
		Parameters: req.Parameters,
	}

	if err := uc.settingRepo.Create(ctx, setting); err != nil {
		return nil, err
	}

	response := setting.ToResponse()
	return &response, nil
}

func (uc *SettingUseCase) Update(ctx context.Context, id string, req *model.UpdateSettingRequest) (*model.SettingResponse, error) {
	setting, err := uc.settingRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		return nil, ErrSettingNotFound
	}

	if req.Base != nil {
		if *req.Base == "" {
			return nil, ErrSettingBaseEmpty
		}
		setting.Base = *req.Base
	}
	if req.Quote != nil {
		if *req.Quote == "" {
			return nil, ErrSettingQuoteEmpty
		}
		setting.Quote = *req.Quote
	}
	if req.Strategy != nil {
		if *req.Strategy == "" {
			return nil, ErrSettingStrategyEmpty
		}
		setting.Strategy = *req.Strategy
	}
	if req.Parameters != nil {
		setting.Parameters = req.Parameters
	}

	if err := uc.settingRepo.Update(ctx, setting); err != nil {
		return nil, err
	}

	response := setting.ToResponse()
	return &response, nil
}

func (uc *SettingUseCase) UpdateParameters(ctx context.Context, id string, strategy string, parameters map[string]interface{}) (*model.SettingResponse, error) {
	setting, err := uc.settingRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		return nil, ErrSettingNotFound
	}

	if err := uc.settingRepo.UpdateParameters(ctx, id, strategy, parameters); err != nil {
		return nil, err
	}

	// Update local copy for response
	if setting.Parameters == nil {
		setting.Parameters = make(map[string]interface{})
	}
	setting.Parameters[strategy] = parameters

	response := setting.ToResponse()
	return &response, nil
}

func (uc *SettingUseCase) Delete(ctx context.Context, id string) error {
	setting, err := uc.settingRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if setting == nil {
		return ErrSettingNotFound
	}

	return uc.settingRepo.Delete(ctx, id)
}
