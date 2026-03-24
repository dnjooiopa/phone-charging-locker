package schema_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/dnjooiopa/phone-charging-locker/schema"
)

func TestMigrate(t *testing.T) {
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "pcl-schema-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("PRAGMA foreign_keys=ON")
	require.NoError(t, err)

	require.NoError(t, db.Ping())

	err = schema.Migrate(ctx, db)
	require.NoError(t, err)
}
