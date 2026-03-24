package locker_repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dnjooiopa/phone-charging-locker/pkg/dbctx"
	"github.com/dnjooiopa/phone-charging-locker/internal/domain"
	"github.com/dnjooiopa/phone-charging-locker/internal/usecase"
)

type sqliteDB struct{}

func New() usecase.LockerRepository {
	return &sqliteDB{}
}

func (p *sqliteDB) FindAll(ctx context.Context) ([]*domain.Locker, error) {
	rows, err := dbctx.Query(ctx, `
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

func (p *sqliteDB) FindByID(ctx context.Context, id int64) (*domain.Locker, error) {
	var locker domain.Locker
	err := dbctx.QueryRow(ctx, `
		SELECT
			id,
			name,
			status,
			created_at,
			updated_at
		FROM locker
		WHERE id = ?
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

func (p *sqliteDB) UpdateStatus(ctx context.Context, id int64, status domain.LockerStatus) error {
	_, err := dbctx.Exec(ctx, `
		UPDATE locker
		SET status = ?, updated_at = datetime('now')
		WHERE id = ?
	`, status, id)
	return err
}
