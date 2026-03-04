package repository_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/samandr77/test_balance_api/internal/domain"
	"github.com/samandr77/test_balance_api/internal/repository"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { container.Terminate(ctx) }) //nolint:errcheck

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	runMigrations(t, connStr)

	return pool
}

func runMigrations(t *testing.T, connStr string) {
	t.Helper()

	// "file://../../migrations" — путь относительно этого файла
	m, err := migrate.New("file://../../migrations", connStr)
	require.NoError(t, err)

	err = m.Up()
	require.NoError(t, err)
}

func TestCreateWithdrawal_Concurrent(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	const userID = "00000000-0000-0000-0000-000000000001"

	// Баланс 50 USDT — только один из N запросов должен пройти
	_, err := pool.Exec(ctx,
		"INSERT INTO balances (user_id, amount) VALUES ($1, $2)",
		userID, decimal.NewFromFloat(50),
	)
	require.NoError(t, err)

	repo := repository.New(pool)

	const N = 10
	var (
		wg           sync.WaitGroup
		mu           sync.Mutex
		successCount int
	)

	for i := range N {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			// Каждая горутина использует уникальный idempotency_key —
			// иначе они все будут считаться повторами одной операции
			_, createErr := repo.CreateWithdrawal(ctx, repository.CreateRequest{
				UserID:         userID,
				Amount:         decimal.NewFromFloat(50),
				Currency:       "USDT",
				Destination:    "0xABC",
				IdempotencyKey: fmt.Sprintf("concurrent-key-%d", i),
				PayloadHash:    fmt.Sprintf("hash-%d", i),
			})

			if createErr == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			} else {
				assert.ErrorIs(t, createErr, domain.ErrInsufficientFunds)
			}
		}(i)
	}

	wg.Wait()

	// Ровно один вывод должен пройти
	assert.Equal(t, 1, successCount)

	// Баланс не ушёл в минус — это гарантирует CHECK и SELECT FOR UPDATE
	var balance decimal.Decimal
	err = pool.QueryRow(ctx,
		"SELECT amount FROM balances WHERE user_id = $1",
		userID,
	).Scan(&balance)
	require.NoError(t, err)
	assert.True(t, balance.GreaterThanOrEqual(decimal.Zero))
}
