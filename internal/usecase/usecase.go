package usecase

import (
	"context"
	"time"

	"github.com/dnjooiopa/phone-charging-locker/internal/domain"
)

type Config struct {
	ChargingDuration time.Duration
	ChargingAmount   int64
}

type LockerRepository interface {
	FindAll(ctx context.Context) ([]*domain.Locker, error)
	FindByID(ctx context.Context, id int64) (*domain.Locker, error)
	UpdateStatus(ctx context.Context, id int64, status domain.LockerStatus) error
}

type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) (int64, error)
	FindByID(ctx context.Context, id int64) (*domain.Session, error)
	UpdateStatus(ctx context.Context, id int64, status domain.SessionStatus) error
	UpdateQRCodeData(ctx context.Context, id int64, qrCodeData string) error
	UpdatePaymentConfirmed(ctx context.Context, id int64, startedAt, expiredAt time.Time) error
	FindExpiredChargingSessions(ctx context.Context, now time.Time) ([]*domain.Session, error)
}

type Usecase struct {
	config            *Config
	lockerRepository  LockerRepository
	sessionRepository SessionRepository
}

func New(
	cfg *Config,
	lockerRepository LockerRepository,
	sessionRepository SessionRepository,
) *Usecase {
	return &Usecase{
		config:            cfg,
		lockerRepository:  lockerRepository,
		sessionRepository: sessionRepository,
	}
}
