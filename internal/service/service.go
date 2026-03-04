package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/samandr77/test_balance_api/internal/domain"
	"github.com/samandr77/test_balance_api/internal/repository"
)

type Repository interface {
	CreateWithdrawal(ctx context.Context, req repository.CreateRequest) (*domain.Withdrawal, error)
	GetWithdrawalByID(ctx context.Context, id string) (*domain.Withdrawal, error)
	GetWithdrawalByIdempotencyKey(ctx context.Context, userID, key string) (*domain.Withdrawal, error)
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

type CreateRequest struct {
	UserID         string
	Amount         decimal.Decimal
	Currency       string
	Destination    string
	IdempotencyKey string
}

func (s *Service) CreateWithdrawal(ctx context.Context, req CreateRequest) (*domain.Withdrawal, bool, error) {
	if err := s.validate(req); err != nil {
		return nil, false, err
	}

	payloadHash := computeHash(req.UserID, req.Amount.String(), req.Currency, req.Destination)

	w, err := s.repo.CreateWithdrawal(ctx, repository.CreateRequest{
		UserID:         req.UserID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Destination:    req.Destination,
		IdempotencyKey: req.IdempotencyKey,
		PayloadHash:    payloadHash,
	})
	if err == nil {
		slog.Info("withdrawal created",
			"withdrawal_id", w.ID,
			"user_id", w.UserID,
			"amount", w.Amount,
			"status", w.Status,
		)
		return w, false, nil
	}

	if !errors.Is(err, domain.ErrDuplicateIdempotencyKey) {
		if errors.Is(err, domain.ErrInsufficientFunds) {
			slog.Error("insufficient funds",
				"user_id", req.UserID,
				"requested", req.Amount,
			)
		}
		return nil, false, err
	}

	existing, fetchErr := s.repo.GetWithdrawalByIdempotencyKey(ctx, req.UserID, req.IdempotencyKey)
	if fetchErr != nil {
		return nil, false, fetchErr
	}

	if existing.PayloadHash != payloadHash {
		return nil, false, domain.ErrIdempotencyConflict
	}

	slog.Warn("idempotent repeat",
		"user_id", req.UserID,
		"idempotency_key", req.IdempotencyKey,
	)

	return existing, true, nil
}

func (s *Service) GetWithdrawal(ctx context.Context, id string) (*domain.Withdrawal, error) {
	return s.repo.GetWithdrawalByID(ctx, id)
}

func (s *Service) validate(req CreateRequest) error {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return domain.ErrInvalidAmount
	}
	if req.Currency != "USDT" {
		return domain.ErrUnsupportedCurrency
	}
	if req.Destination == "" {
		return domain.ErrEmptyDestination
	}
	if req.IdempotencyKey == "" {
		return domain.ErrMissingIdempotencyKey
	}
	return nil
}

func computeHash(parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return fmt.Sprintf("%x", h)
}
