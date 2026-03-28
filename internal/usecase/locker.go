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

type DeleteLockerParams struct {
	LockerID int64
}

func (p *DeleteLockerParams) Valid() error {
	v := validator.New()
	v.Must(p.LockerID > 0, "locker_id is required")
	return v.Error()
}

func (u *Usecase) DeleteLocker(ctx context.Context, params *DeleteLockerParams) error {
	if err := params.Valid(); err != nil {
		slog.Error("DeleteLocker: invalid params", "error", err)
		return err
	}

	_, err := u.lockerRepository.FindByID(ctx, params.LockerID)
	if err != nil {
		slog.Error("DeleteLocker: failed to find locker", "error", err, "id", params.LockerID)
		return err
	}

	if err := u.sessionRepository.DeleteByLockerID(ctx, params.LockerID); err != nil {
		slog.Error("DeleteLocker: failed to delete sessions", "error", err, "locker_id", params.LockerID)
		return err
	}

	if err := u.lockerRepository.Delete(ctx, params.LockerID); err != nil {
		slog.Error("DeleteLocker: failed to delete locker", "error", err, "id", params.LockerID)
		return err
	}

	return nil
}
