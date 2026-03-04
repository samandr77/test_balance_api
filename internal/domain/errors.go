package domain

import "errors"

var (
	ErrInsufficientFunds     = errors.New("insufficient funds")
	ErrInvalidAmount         = errors.New("amount must be greater than zero")
	ErrUnsupportedCurrency   = errors.New("unsupported currency")
	ErrEmptyDestination      = errors.New("destination is required")
	ErrMissingIdempotencyKey = errors.New("idempotency key is required")
	ErrIdempotencyConflict    = errors.New("idempotency key conflict: different payload")
	ErrWithdrawalNotFound     = errors.New("withdrawal not found")
	ErrDuplicateIdempotencyKey = errors.New("duplicate idempotency key")
	ErrBalanceNotFound        = errors.New("balance not initialized for user")
)
