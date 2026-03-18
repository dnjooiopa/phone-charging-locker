package schema_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"contrib.go.opencensus.io/integrations/ocsql"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/dnjooiopa/phone-charging-locker/schema"
)

func TestMigrate(t *testing.T) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	defer pgContainer.Terminate(ctx)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	driver, _ := ocsql.Register("postgres")
	db, err := sql.Open(driver, connStr)
	require.NoError(t, err)
	defer db.Close()

	require.NoError(t, db.Ping())

	err = schema.Migrate(ctx, db)
	require.NoError(t, err)
}
