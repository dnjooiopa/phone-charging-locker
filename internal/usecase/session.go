package usecase

import (
	"context"
	"encoding/base64"
	"fmt"
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
		return nil, err
	}

	locker, err := u.lockerRepository.FindByID(ctx, p.LockerID)
	if err != nil {
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
		return nil, err
	}

	qrData := fmt.Sprintf("PCL-PAY-%d", sessionID)

	err = u.sessionRepository.UpdateQRCodeData(ctx, sessionID, qrData)
	if err != nil {
		return nil, err
	}

	err = u.lockerRepository.UpdateStatus(ctx, locker.ID, domain.LockerStatusInUse)
	if err != nil {
		return nil, err
	}

	png, err := qrcode.Encode(qrData, qrcode.Medium, 256)
	if err != nil {
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
		return nil, err
	}

	session, err := u.sessionRepository.FindByID(ctx, p.SessionID)
	if err != nil {
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

	err = u.sessionRepository.UpdatePaymentConfirmed(ctx, session.ID, now, expiredAt)
	if err != nil {
		return nil, err
	}

	session.Status = domain.SessionStatusCharging
	session.StartedAt = &now
	session.ExpiredAt = &expiredAt

	return session, nil
}

func (u *Usecase) CheckSession(ctx context.Context, id int64) (*domain.Session, error) {
	v := validator.New()
	v.Must(id > 0, "id is required")
	if v.Error() != nil {
		return nil, v.Error()
	}

	session, err := u.sessionRepository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if session.Status == domain.SessionStatusCharging && session.ExpiredAt != nil && time.Now().After(*session.ExpiredAt) {
		err = u.sessionRepository.UpdateStatus(ctx, session.ID, domain.SessionStatusCompleted)
		if err != nil {
			return nil, err
		}

		err = u.lockerRepository.UpdateStatus(ctx, session.LockerID, domain.LockerStatusAvailable)
		if err != nil {
			return nil, err
		}

		session.Status = domain.SessionStatusCompleted
	}

	return session, nil
}

func (u *Usecase) ExpireSessions(ctx context.Context) (int, error) {
	sessions, err := u.sessionRepository.FindExpiredChargingSessions(ctx, time.Now())
	if err != nil {
		return 0, err
	}

	for _, session := range sessions {
		err = u.sessionRepository.UpdateStatus(ctx, session.ID, domain.SessionStatusCompleted)
		if err != nil {
			return 0, err
		}

		err = u.lockerRepository.UpdateStatus(ctx, session.LockerID, domain.LockerStatusAvailable)
		if err != nil {
			return 0, err
		}
	}

	return len(sessions), nil
}
