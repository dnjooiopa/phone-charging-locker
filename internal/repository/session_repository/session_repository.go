package session_repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/acoshift/pgsql/pgctx"

	"github.com/dnjooiopa/phone-charging-locker/internal/domain"
	"github.com/dnjooiopa/phone-charging-locker/internal/usecase"
)

type pgDB struct{}

func NewPostgresDB() usecase.SessionRepository {
	return &pgDB{}
}

func (p *pgDB) Create(ctx context.Context, session *domain.Session) (int64, error) {
	var id int64
	err := pgctx.QueryRow(ctx, `
		INSERT INTO session (
			locker_id,
			status,
			amount
		) VALUES (
			$1,
			$2,
			$3
		) RETURNING id
	`,
		session.LockerID,
		session.Status,
		session.Amount,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (p *pgDB) FindByID(ctx context.Context, id int64) (*domain.Session, error) {
	var session domain.Session
	err := pgctx.QueryRow(ctx, `
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
		WHERE id = $1
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

func (p *pgDB) UpdateStatus(ctx context.Context, id int64, status domain.SessionStatus) error {
	_, err := pgctx.Exec(ctx, `
		UPDATE session
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`, status, id)
	return err
}

func (p *pgDB) UpdateInvoiceData(ctx context.Context, id int64, qrCodeData, paymentHash string) error {
	_, err := pgctx.Exec(ctx, `
		UPDATE session
		SET qr_code_data = $1, payment_hash = $2, updated_at = NOW()
		WHERE id = $3
	`, qrCodeData, paymentHash, id)
	return err
}

func (p *pgDB) UpdatePaymentConfirmed(ctx context.Context, id int64, startedAt, expiredAt time.Time) error {
	_, err := pgctx.Exec(ctx, `
		UPDATE session
		SET status = 'charging', started_at = $1, expired_at = $2, updated_at = NOW()
		WHERE id = $3
	`, startedAt, expiredAt, id)
	return err
}

func (p *pgDB) FindByPaymentHash(ctx context.Context, paymentHash string) (*domain.Session, error) {
	var session domain.Session
	err := pgctx.QueryRow(ctx, `
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
		WHERE payment_hash = $1
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

func (p *pgDB) FindExpiredChargingSessions(ctx context.Context, now time.Time) ([]*domain.Session, error) {
	rows, err := pgctx.Query(ctx, `
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
		AND expired_at < $1
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
