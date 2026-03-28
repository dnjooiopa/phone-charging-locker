package usecase

import (
	"context"
	"log/slog"

	"github.com/dnjooiopa/phone-charging-locker/internal/domain"
	"github.com/moonrhythm/validator"
)

type CreateLockerParams struct {
	Name string
}

func (p *CreateLockerParams) Valid() error {
	v := validator.New()
	v.Must(p.Name != "", "name is required")
	return v.Error()
}

type CreateLockerResult struct {
	Locker *domain.Locker
}

func (u *Usecase) CreateLocker(ctx context.Context, params *CreateLockerParams) (*CreateLockerResult, error) {
	if err := params.Valid(); err != nil {
		slog.Error("CreateLocker: invalid params", "error", err)
		return nil, err
	}

	id, err := u.lockerRepository.Create(ctx, params.Name)
	if err != nil {
		slog.Error("CreateLocker: failed to create locker", "error", err, "name", params.Name)
		return nil, err
	}

	locker, err := u.lockerRepository.FindByID(ctx, id)
	if err != nil {
		slog.Error("CreateLocker: failed to find locker", "error", err, "id", id)
		return nil, err
	}

	return &CreateLockerResult{Locker: locker}, nil
}

func (u *Usecase) ListLockers(ctx context.Context) ([]*domain.Locker, error) {
	return u.lockerRepository.FindAll(ctx)
}
