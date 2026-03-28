package usecase

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	"github.com/moonrhythm/validator"
	qrcode "github.com/skip2/go-qrcode"

	"github.com/dnjooiopa/phone-charging-locker/internal/domain"
)

type SelectLockerParams struct {
	LockerID int64
}

func (p *SelectLockerParams) Valid() error {
	v := validator.New()
	v.Must(p.LockerID > 0, "locker_id is required")
	return v.Error()
}

type SelectLockerResult struct {
	SessionID  int64  `json:"session_id"`
	QRCodeData string `json:"qr_code_data"`
	QRCodePNG  string `json:"qr_code_png"`
}

func (u *Usecase) SelectLocker(ctx context.Context, p *SelectLockerParams) (*SelectLockerResult, error) {
	if err := p.Valid(); err != nil {
		slog.Error("SelectLocker: invalid params", "error", err)
		return nil, err
	}

	locker, err := u.lockerRepository.FindByID(ctx, p.LockerID)
	if err != nil {
		slog.Error("SelectLocker: failed to find locker", "error", err, "lockerID", p.LockerID)
		return nil, err
	}

	if locker.Status != domain.LockerStatusAvailable {
		return nil, ErrLockerNotAvailable
	}

	session := &domain.Session{
		LockerID: locker.ID,
		Status:   domain.SessionStatusPendingPayment,
		Amount:   u.config.ChargingAmount,
	}

	sessionID, err := u.sessionRepository.Create(ctx, session)
	if err != nil {
		slog.Error("SelectLocker: failed to create session", "error", err, "lockerID", p.LockerID)
		return nil, err
	}

	invoice, err := u.invoiceRepository.CreateInvoice(ctx, &CreateInvoiceParams{
		Description: fmt.Sprintf("Phone Charging Locker - Session %d", sessionID),
		AmountSat:   u.config.ChargingAmount,
		ExternalID:  fmt.Sprintf("%d", sessionID),
	})
	if err != nil {
		slog.Error("SelectLocker: failed to create invoice", "error", err, "sessionID", sessionID)
		return nil, ErrInvoiceCreationFailed
	}

	qrData := invoice.Serialized

	err = u.sessionRepository.UpdateInvoiceData(ctx, sessionID, qrData, invoice.PaymentHash)
	if err != nil {
		slog.Error("SelectLocker: failed to update invoice data", "error", err, "sessionID", sessionID)
		return nil, err
	}

	err = u.lockerRepository.UpdateStatus(ctx, locker.ID, domain.LockerStatusInUse)
	if err != nil {
		slog.Error("SelectLocker: failed to update locker status", "error", err, "lockerID", locker.ID)
		return nil, err
	}

	png, err := qrcode.Encode(qrData, qrcode.Low, 256)
	if err != nil {
		slog.Error("SelectLocker: failed to encode qr code", "error", err, "sessionID", sessionID)
		return nil, err
	}

	return &SelectLockerResult{
		SessionID:  sessionID,
		QRCodeData: qrData,
		QRCodePNG:  base64.StdEncoding.EncodeToString(png),
	}, nil
}

type ConfirmPaymentParams struct {
	SessionID int64
}

func (p *ConfirmPaymentParams) Valid() error {
	v := validator.New()
	v.Must(p.SessionID > 0, "session_id is required")
	return v.Error()
}

func (u *Usecase) ConfirmPayment(ctx context.Context, p *ConfirmPaymentParams) (*domain.Session, error) {
	if err := p.Valid(); err != nil {
		slog.Error("ConfirmPayment: invalid params", "error", err)
		return nil, err
	}

	session, err := u.sessionRepository.FindByID(ctx, p.SessionID)
	if err != nil {
		slog.Error("ConfirmPayment: failed to find session", "error", err, "sessionID", p.SessionID)
		return nil, err
	}

	switch session.Status {
	case domain.SessionStatusPendingPayment:
		// ok
	case domain.SessionStatusCharging:
		return nil, ErrSessionAlreadyPaid
	default:
		return nil, ErrInvalidSessionState
	}

	now := time.Now()
	expiredAt := now.Add(u.config.ChargingDuration)

	session, err = u.sessionRepository.UpdatePaymentConfirmed(ctx, session.ID, domain.SessionStatusCharging, now, expiredAt)
	if err != nil {
		slog.Error("ConfirmPayment: failed to update payment confirmed", "error", err, "sessionID", session.ID)
		return nil, err
	}

	return session, nil
}

type WebhookPayload struct {
	Type        string  `json:"type"`
	Timestamp   int64   `json:"timestamp"`
	AmountSat   int64   `json:"amountSat"`
	PaymentHash string  `json:"paymentHash"`
	ExternalID  *string `json:"externalId"`
	PayerNote   *string `json:"payerNote"`
	PayerKey    *string `json:"payerKey"`
}

func (u *Usecase) HandleWebhook(ctx context.Context, payload *WebhookPayload) (*domain.Session, error) {
	if payload.Type != "payment_received" {
		slog.Error("HandleWebhook: unsupported webhook type", "type", payload.Type)
		return nil, ErrUnsupportedWebhookType
	}

	session, err := u.sessionRepository.FindByPaymentHash(ctx, payload.PaymentHash)
	if err != nil {
		slog.Error("HandleWebhook: failed to find session by payment hash", "error", err, "paymentHash", payload.PaymentHash)
		return nil, err
	}

	switch session.Status {
	case domain.SessionStatusPendingPayment:
		// ok
	case domain.SessionStatusCharging:
		return session, nil
	default:
		slog.Error("HandleWebhook: invalid session status", "status", session.Status)
		return nil, ErrInvalidSessionState
	}

	now := time.Now()
	expiredAt := now.Add(u.config.ChargingDuration)

	session, err = u.sessionRepository.UpdatePaymentConfirmed(ctx, session.ID, domain.SessionStatusCharging, now, expiredAt)
	if err != nil {
		slog.Error("HandleWebhook: failed to update payment confirmed", "error", err, "sessionID", session.ID)
		return nil, err
	}

	return session, nil
}

func (u *Usecase) CheckSession(ctx context.Context, id int64) (*domain.Session, error) {
	v := validator.New()
	v.Must(id > 0, "id is required")
	if v.Error() != nil {
		slog.Error("CheckSession: invalid params", "error", v.Error())
		return nil, v.Error()
	}

	session, err := u.sessionRepository.FindByID(ctx, id)
	if err != nil {
		slog.Error("CheckSession: failed to find session", "error", err, "id", id)
		return nil, err
	}

	if session.Status == domain.SessionStatusCharging && session.ExpiredAt != nil && time.Now().After(*session.ExpiredAt) {
		err = u.sessionRepository.UpdateStatus(ctx, session.ID, domain.SessionStatusCompleted)
		if err != nil {
			slog.Error("CheckSession: failed to update session status", "error", err, "sessionID", session.ID)
			return nil, err
		}

		err = u.lockerRepository.UpdateStatus(ctx, session.LockerID, domain.LockerStatusAvailable)
		if err != nil {
			slog.Error("CheckSession: failed to update locker status", "error", err, "lockerID", session.LockerID)
			return nil, err
		}

		session.Status = domain.SessionStatusCompleted
	}

	return session, nil
}

func (u *Usecase) ExpireSessions(ctx context.Context) (int, error) {
	sessions, err := u.sessionRepository.FindExpiredChargingSessions(ctx, time.Now())
	if err != nil {
		slog.Error("ExpireSessions: failed to find expired sessions", "error", err)
		return 0, err
	}

	for _, session := range sessions {
		err = u.sessionRepository.UpdateStatus(ctx, session.ID, domain.SessionStatusCompleted)
		if err != nil {
			slog.Error("ExpireSessions: failed to update session status", "error", err, "sessionID", session.ID)
			return 0, err
		}

		err = u.lockerRepository.UpdateStatus(ctx, session.LockerID, domain.LockerStatusAvailable)
		if err != nil {
			slog.Error("ExpireSessions: failed to update locker status", "error", err, "lockerID", session.LockerID)
			return 0, err
		}
	}

	return len(sessions), nil
}
