package session_repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/dnjooiopa/phone-charging-locker/pkg/dbctx"
	"github.com/dnjooiopa/phone-charging-locker/internal/domain"
	"github.com/dnjooiopa/phone-charging-locker/internal/usecase"
)

type sqliteDB struct{}

func New() usecase.SessionRepository {
	return &sqliteDB{}
}

func (p *sqliteDB) Create(ctx context.Context, session *domain.Session) (int64, error) {
	result, err := dbctx.Exec(ctx, `
		INSERT INTO session (
			locker_id,
			status,
			amount
		) VALUES (
			?,
			?,
			?
		)
	`,
		session.LockerID,
		session.Status,
		session.Amount,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (p *sqliteDB) FindByID(ctx context.Context, id int64) (*domain.Session, error) {
	var session domain.Session
	err := dbctx.QueryRow(ctx, `
		SELECT
			id,
			locker_id,
			status,
			qr_code_data,
			payment_hash,
			amount,
			started_at,
			expired_at,
			created_at,
			updated_at
		FROM session
		WHERE id = ?
	`, id).Scan(
		&session.ID,
		&session.LockerID,
		&session.Status,
		&session.QRCodeData,
		&session.PaymentHash,
		&session.Amount,
		&session.StartedAt,
		&session.ExpiredAt,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, usecase.ErrSessionNotFound
		}
		return nil, err
	}

	return &session, nil
}

func (p *sqliteDB) UpdateStatus(ctx context.Context, id int64, status domain.SessionStatus) error {
	_, err := dbctx.Exec(ctx, `
		UPDATE session
		SET status = ?, updated_at = datetime('now')
		WHERE id = ?
	`, status, id)
	return err
}

func (p *sqliteDB) UpdateInvoiceData(ctx context.Context, id int64, qrCodeData, paymentHash string) error {
	_, err := dbctx.Exec(ctx, `
		UPDATE session
		SET qr_code_data = ?, payment_hash = ?, updated_at = datetime('now')
		WHERE id = ?
	`, qrCodeData, paymentHash, id)
	return err
}

func (p *sqliteDB) UpdatePaymentConfirmed(ctx context.Context, id int64, startedAt, expiredAt time.Time) error {
	_, err := dbctx.Exec(ctx, `
		UPDATE session
		SET status = 'charging', started_at = ?, expired_at = ?, updated_at = datetime('now')
		WHERE id = ?
	`, startedAt, expiredAt, id)
	return err
}

func (p *sqliteDB) FindByPaymentHash(ctx context.Context, paymentHash string) (*domain.Session, error) {
	var session domain.Session
	err := dbctx.QueryRow(ctx, `
		SELECT
			id,
			locker_id,
			status,
			qr_code_data,
			payment_hash,
			amount,
			started_at,
			expired_at,
			created_at,
			updated_at
		FROM session
		WHERE payment_hash = ?
	`, paymentHash).Scan(
		&session.ID,
		&session.LockerID,
		&session.Status,
		&session.QRCodeData,
		&session.PaymentHash,
		&session.Amount,
		&session.StartedAt,
		&session.ExpiredAt,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, usecase.ErrSessionNotFound
		}
		return nil, err
	}

	return &session, nil
}

func (p *sqliteDB) FindExpiredChargingSessions(ctx context.Context, now time.Time) ([]*domain.Session, error) {
	rows, err := dbctx.Query(ctx, `
		SELECT
			id,
			locker_id,
			status,
			qr_code_data,
			payment_hash,
			amount,
			started_at,
			expired_at,
			created_at,
			updated_at
		FROM session
		WHERE status = 'charging'
		AND expired_at < ?
	`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*domain.Session
	for rows.Next() {
		var session domain.Session
		err := rows.Scan(
			&session.ID,
			&session.LockerID,
			&session.Status,
			&session.QRCodeData,
			&session.PaymentHash,
			&session.Amount,
			&session.StartedAt,
			&session.ExpiredAt,
			&session.CreatedAt,
			&session.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, &session)
	}

	return sessions, rows.Err()
}

func (p *sqliteDB) DeleteByLockerID(ctx context.Context, lockerID int64) error {
	_, err := dbctx.Exec(ctx, `
		DELETE FROM session WHERE locker_id = ?
	`, lockerID)
	return err
}
