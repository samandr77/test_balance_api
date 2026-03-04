package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type Withdrawal struct {
	ID             string
	UserID         string
	Amount         decimal.Decimal
	Currency       string
	Destination    string
	Status         string
	IdempotencyKey string
	PayloadHash    string
	CreatedAt      time.Time
}

type Balance struct {
	UserID    string
	Amount    decimal.Decimal
	Currency  string
	UpdatedAt time.Time
}
