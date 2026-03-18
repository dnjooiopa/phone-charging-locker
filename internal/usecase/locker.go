package usecase

import (
	"context"

	"github.com/dnjooiopa/phone-charging-locker/internal/domain"
)

func (u *Usecase) ListLockers(ctx context.Context) ([]*domain.Locker, error) {
	return u.lockerRepository.FindAll(ctx)
}
