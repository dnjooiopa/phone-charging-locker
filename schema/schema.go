package schema

import (
	"context"
	"database/sql"
	"embed"
)

//go:embed *.sql
var fs embed.FS

func Migrate(ctx context.Context, db *sql.DB) error {
	list, err := fs.ReadDir(".")
	if err != nil {
		return err
	}

	for _, x := range list {
		migrateID := x.Name()

		b, err := fs.ReadFile(migrateID)
		if err != nil {
			return err
		}
		_, err = db.ExecContext(ctx, string(b))
		if err != nil {
			return err
		}
	}

	return nil
}
