//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestMigrationsCreateDiarySchema(t *testing.T) {
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:18-alpine",
		postgres.WithDatabase("diary"),
		postgres.WithUsername("diary"),
		postgres.WithPassword("diary"),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })

	connectionString, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	migrations, err := migrate.New("file://../../migrations", connectionString)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = migrations.Close() })
	require.NoError(t, migrations.Up())

	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	connection, err := pgx.Connect(queryCtx, connectionString)
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close(ctx) })

	var count int
	err = connection.QueryRow(queryCtx, `SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('workouts','workout_results','outbox_events')`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 3, count)
}
