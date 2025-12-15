package usecase

import (
	"context"
	"errors"

	"control_page/internal/adaptor"
	"control_page/internal/model"
)

var (
	ErrSwitcherNotFound = errors.New("switcher not found")
)

var _ adaptor.SwitcherUseCase = (*SwitcherUseCase)(nil)

type SwitcherUseCase struct {
	switcherRepo adaptor.SwitcherRepository
}

func NewSwitcherUseCase(switcherRepo adaptor.SwitcherRepository) *SwitcherUseCase {
	return &SwitcherUseCase{
		switcherRepo: switcherRepo,
	}
}

func (uc *SwitcherUseCase) List(ctx context.Context) ([]model.SwitcherResponse, error) {
	switchers, err := uc.switcherRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]model.SwitcherResponse, len(switchers))
	for i, s := range switchers {
		responses[i] = s.ToResponse()
	}

	return responses, nil
}

func (uc *SwitcherUseCase) GetByID(ctx context.Context, id string) (*model.SwitcherResponse, error) {
	switcher, err := uc.switcherRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if switcher == nil {
		return nil, ErrSwitcherNotFound
	}

	response := switcher.ToResponse()
	return &response, nil
}

func (uc *SwitcherUseCase) Create(ctx context.Context, req *model.UpdateSwitcherRequest) (*model.SwitcherResponse, error) {
	switcher := &model.Switcher{
		Pairs: req.Pairs,
	}

	if err := uc.switcherRepo.Create(ctx, switcher); err != nil {
		return nil, err
	}

	response := switcher.ToResponse()
	return &response, nil
}

func (uc *SwitcherUseCase) Update(ctx context.Context, id string, req *model.UpdateSwitcherRequest) (*model.SwitcherResponse, error) {
	switcher, err := uc.switcherRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if switcher == nil {
		return nil, ErrSwitcherNotFound
	}

	// Merge new pairs with existing ones
	for pair, config := range req.Pairs {
		switcher.Pairs[pair] = config
	}

	if err := uc.switcherRepo.Update(ctx, switcher); err != nil {
		return nil, err
	}

	response := switcher.ToResponse()
	return &response, nil
}

func (uc *SwitcherUseCase) UpdatePair(ctx context.Context, id string, pair string, enable bool) (*model.SwitcherResponse, error) {
	switcher, err := uc.switcherRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if switcher == nil {
		return nil, ErrSwitcherNotFound
	}

	if err := uc.switcherRepo.UpdatePair(ctx, id, pair, enable); err != nil {
		return nil, err
	}

	// Update local copy for response
	switcher.Pairs[pair] = model.SwitcherPair{Enable: enable}

	response := switcher.ToResponse()
	return &response, nil
}

func (uc *SwitcherUseCase) Delete(ctx context.Context, id string) error {
	switcher, err := uc.switcherRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if switcher == nil {
		return ErrSwitcherNotFound
	}

	return uc.switcherRepo.Delete(ctx, id)
}
