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
	Create(ctx context.Context, name string) (int64, error)
	FindAll(ctx context.Context) ([]*domain.Locker, error)
	FindByID(ctx context.Context, id int64) (*domain.Locker, error)
	UpdateStatus(ctx context.Context, id int64, status domain.LockerStatus) error
	Delete(ctx context.Context, id int64) error
}

type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) (int64, error)
	FindByID(ctx context.Context, id int64) (*domain.Session, error)
	UpdateStatus(ctx context.Context, id int64, status domain.SessionStatus) error
	UpdateInvoiceData(ctx context.Context, id int64, qrCodeData, paymentHash string) error
	UpdatePaymentConfirmed(ctx context.Context, id int64, startedAt, expiredAt time.Time) error
	FindByPaymentHash(ctx context.Context, paymentHash string) (*domain.Session, error)
	FindExpiredChargingSessions(ctx context.Context, now time.Time) ([]*domain.Session, error)
	DeleteByLockerID(ctx context.Context, lockerID int64) error
}

type CreateInvoiceParams struct {
	Description string
	AmountSat   int64
	ExternalID  string
}

type CreateInvoiceResult struct {
	PaymentHash string
	Serialized  string
}

type InvoiceRepository interface {
	CreateInvoice(ctx context.Context, params *CreateInvoiceParams) (*CreateInvoiceResult, error)
	RegisterWebhookEndpoint(ctx context.Context, webhookURL string) error
}

type Usecase struct {
	config            *Config
	lockerRepository  LockerRepository
	sessionRepository SessionRepository
	invoiceRepository InvoiceRepository
}

func New(
	cfg *Config,
	lockerRepository LockerRepository,
	sessionRepository SessionRepository,
	invoiceRepository InvoiceRepository,
) *Usecase {
	return &Usecase{
		config:            cfg,
		lockerRepository:  lockerRepository,
		sessionRepository: sessionRepository,
		invoiceRepository: invoiceRepository,
	}
}
