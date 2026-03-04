package service_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/samandr77/test_balance_api/internal/domain"
	"github.com/samandr77/test_balance_api/internal/repository"
	"github.com/samandr77/test_balance_api/internal/service"
)

// mockRepo реализует интерфейс service.Repository без реальной БД.
// Каждый метод делегирует вызов в поле-функцию — это позволяет
// задавать поведение прямо в каждом тесте.
type mockRepo struct {
	createFn              func(ctx context.Context, req repository.CreateRequest) (*domain.Withdrawal, error)
	getByIDFn             func(ctx context.Context, id string) (*domain.Withdrawal, error)
	getByIdempotencyKeyFn func(ctx context.Context, userID, key string) (*domain.Withdrawal, error)
}

func (m *mockRepo) CreateWithdrawal(ctx context.Context, req repository.CreateRequest) (*domain.Withdrawal, error) {
	return m.createFn(ctx, req)
}

func (m *mockRepo) GetWithdrawalByID(ctx context.Context, id string) (*domain.Withdrawal, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockRepo) GetWithdrawalByIdempotencyKey(ctx context.Context, userID, key string) (*domain.Withdrawal, error) {
	return m.getByIdempotencyKeyFn(ctx, userID, key)
}

// payloadHash воспроизводит алгоритм из service.computeHash,
// чтобы тест мог подставить правильный hash в mock-ответ.
func payloadHash(userID, amount, currency, destination string) string {
	h := sha256.Sum256([]byte(strings.Join([]string{userID, amount, currency, destination}, "|")))
	return fmt.Sprintf("%x", h)
}

func TestCreateWithdrawal_Success(t *testing.T) {
	expected := &domain.Withdrawal{
		ID:             "test-id",
		UserID:         "00000000-0000-0000-0000-000000000001",
		Amount:         decimal.NewFromFloat(50),
		Currency:       "USDT",
		Destination:    "0xABC",
		Status:         "pending",
		IdempotencyKey: "key-1",
		CreatedAt:      time.Now(),
	}

	repo := &mockRepo{
		createFn: func(_ context.Context, _ repository.CreateRequest) (*domain.Withdrawal, error) {
			return expected, nil
		},
	}

	svc := service.New(repo)
	w, isIdempotent, err := svc.CreateWithdrawal(context.Background(), service.CreateRequest{
		UserID:         expected.UserID,
		Amount:         expected.Amount,
		Currency:       expected.Currency,
		Destination:    expected.Destination,
		IdempotencyKey: expected.IdempotencyKey,
	})

	require.NoError(t, err)
	assert.False(t, isIdempotent)
	assert.Equal(t, expected.ID, w.ID)
	assert.Equal(t, "pending", w.Status)
}

func TestCreateWithdrawal_InsufficientFunds(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, _ repository.CreateRequest) (*domain.Withdrawal, error) {
			return nil, domain.ErrInsufficientFunds
		},
	}

	svc := service.New(repo)
	_, _, err := svc.CreateWithdrawal(context.Background(), service.CreateRequest{
		UserID:         "00000000-0000-0000-0000-000000000001",
		Amount:         decimal.NewFromFloat(999),
		Currency:       "USDT",
		Destination:    "0xABC",
		IdempotencyKey: "key-1",
	})

	assert.ErrorIs(t, err, domain.ErrInsufficientFunds)
}

func TestCreateWithdrawal_IdempotentRepeat(t *testing.T) {
	req := service.CreateRequest{
		UserID:         "00000000-0000-0000-0000-000000000001",
		Amount:         decimal.NewFromFloat(50),
		Currency:       "USDT",
		Destination:    "0xABC",
		IdempotencyKey: "key-1",
	}

	// Вычисляем hash так же как сервис — чтобы mock вернул совпадающий
	hash := payloadHash(req.UserID, req.Amount.String(), req.Currency, req.Destination)

	existing := &domain.Withdrawal{
		ID:             "existing-id",
		UserID:         req.UserID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Destination:    req.Destination,
		Status:         "pending",
		IdempotencyKey: req.IdempotencyKey,
		PayloadHash:    hash,
		CreatedAt:      time.Now(),
	}

	repo := &mockRepo{
		createFn: func(_ context.Context, _ repository.CreateRequest) (*domain.Withdrawal, error) {
			return nil, domain.ErrDuplicateIdempotencyKey
		},
		getByIdempotencyKeyFn: func(_ context.Context, _, _ string) (*domain.Withdrawal, error) {
			return existing, nil
		},
	}

	svc := service.New(repo)
	w, isIdempotent, err := svc.CreateWithdrawal(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, isIdempotent)
	assert.Equal(t, existing.ID, w.ID)
}
