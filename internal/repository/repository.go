package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/samandr77/test_balance_api/internal/domain"
)

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

type CreateRequest struct {
	UserID         string
	Amount         decimal.Decimal
	Currency       string
	Destination    string
	IdempotencyKey string
	PayloadHash    string
}

func (r *Repo) CreateWithdrawal(ctx context.Context, req CreateRequest) (*domain.Withdrawal, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var balance decimal.Decimal
	err = tx.QueryRow(ctx,
		`SELECT amount
		FROM balances
		WHERE user_id = $1
		FOR UPDATE`,
		req.UserID,
	).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrWithdrawalNotFound
		}
		return nil, fmt.Errorf("lock balance: %w", err)
	}

	if balance.LessThan(req.Amount) {
		return nil, domain.ErrInsufficientFunds
	}

	_, err = tx.Exec(ctx,
		`UPDATE balances
		SET amount = amount - $1,
		updated_at = NOW()
		WHERE user_id = $2`,
		req.Amount, req.UserID,
	)
	if err != nil {
		return nil, fmt.Errorf("debit balance: %w", err)
	}

	var w domain.Withdrawal
	err = tx.QueryRow(ctx,
		`INSERT INTO withdrawals (user_id, amount, currency, destination, idempotency_key, payload_hash)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING
		 	id,
		 	user_id,
		 	amount,
		 	currency,
		 	destination,
		 	status,
		 	idempotency_key,
		 	payload_hash,
		 	created_at`,
		req.UserID, req.Amount, req.Currency, req.Destination, req.IdempotencyKey, req.PayloadHash,
	).Scan(
		&w.ID,
		&w.UserID,
		&w.Amount,
		&w.Currency,
		&w.Destination,
		&w.Status,
		&w.IdempotencyKey,
		&w.PayloadHash,
		&w.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("unique violation: %w", err)
		}
		return nil, fmt.Errorf("insert withdrawal: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &w, nil
}

func (r *Repo) GetWithdrawalByID(ctx context.Context, id string) (*domain.Withdrawal, error) {
	var w domain.Withdrawal
	err := r.pool.QueryRow(ctx,
		`SELECT
			id,
			user_id,
			amount,
			currency,
			destination,
			status,
			idempotency_key,
			payload_hash,
			created_at
		FROM withdrawals
		WHERE id = $1`,
		id,
	).Scan(
		&w.ID,
		&w.UserID,
		&w.Amount,
		&w.Currency,
		&w.Destination,
		&w.Status,
		&w.IdempotencyKey,
		&w.PayloadHash,
		&w.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrWithdrawalNotFound
		}
		return nil, fmt.Errorf("get withdrawal: %w", err)
	}

	return &w, nil
}

func (r *Repo) GetWithdrawalByIdempotencyKey(ctx context.Context, userID, key string) (*domain.Withdrawal, error) {
	var w domain.Withdrawal
	err := r.pool.QueryRow(ctx,
		`SELECT
			id,
			user_id,
			amount,
			currency,
			destination,
			status,
			idempotency_key,
			payload_hash,
			created_at
		FROM withdrawals
		WHERE user_id = $1
			AND idempotency_key = $2`,
		userID, key,
	).Scan(
		&w.ID,
		&w.UserID,
		&w.Amount,
		&w.Currency,
		&w.Destination,
		&w.Status,
		&w.IdempotencyKey,
		&w.PayloadHash,
		&w.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrWithdrawalNotFound
		}
		return nil, fmt.Errorf("get withdrawal by idempotency key: %w", err)
	}

	return &w, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
