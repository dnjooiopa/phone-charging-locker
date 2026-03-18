package locker_repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/acoshift/pgsql/pgctx"

	"github.com/dnjooiopa/phone-charging-locker/internal/domain"
	"github.com/dnjooiopa/phone-charging-locker/internal/usecase"
)

type pgDB struct{}

func NewPostgresDB() usecase.LockerRepository {
	return &pgDB{}
}

func (p *pgDB) FindAll(ctx context.Context) ([]*domain.Locker, error) {
	rows, err := pgctx.Query(ctx, `
		SELECT
			id,
			name,
			status,
			created_at,
			updated_at
		FROM locker
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lockers []*domain.Locker
	for rows.Next() {
		var locker domain.Locker
		err := rows.Scan(
			&locker.ID,
			&locker.Name,
			&locker.Status,
			&locker.CreatedAt,
			&locker.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		lockers = append(lockers, &locker)
	}

	return lockers, rows.Err()
}

func (p *pgDB) FindByID(ctx context.Context, id int64) (*domain.Locker, error) {
	var locker domain.Locker
	err := pgctx.QueryRow(ctx, `
		SELECT
			id,
			name,
			status,
			created_at,
			updated_at
		FROM locker
		WHERE id = $1
	`, id).Scan(
		&locker.ID,
		&locker.Name,
		&locker.Status,
		&locker.CreatedAt,
		&locker.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, usecase.ErrLockerNotFound
		}
		return nil, err
	}

	return &locker, nil
}

func (p *pgDB) UpdateStatus(ctx context.Context, id int64, status domain.LockerStatus) error {
	_, err := pgctx.Exec(ctx, `
		UPDATE locker
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`, status, id)
	return err
}
