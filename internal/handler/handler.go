package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/samandr77/test_balance_api/internal/domain"
	"github.com/samandr77/test_balance_api/internal/service"
)

type WithdrawalService interface {
	CreateWithdrawal(ctx context.Context, req service.CreateRequest) (*domain.Withdrawal, bool, error)
	GetWithdrawal(ctx context.Context, id string) (*domain.Withdrawal, error)
}

type Handler struct {
	svc WithdrawalService
}

func New(svc WithdrawalService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("POST /v1/withdrawals", authMiddleware(http.HandlerFunc(h.createWithdrawal)))
	mux.Handle("GET /v1/withdrawals/{id}", authMiddleware(http.HandlerFunc(h.getWithdrawal)))
}

type createRequest struct {
	UserID         string          `json:"user_id"`
	Amount         decimal.Decimal `json:"amount"`
	Currency       string          `json:"currency"`
	Destination    string          `json:"destination"`
	IdempotencyKey string          `json:"idempotency_key"`
}

type withdrawalResponse struct {
	ID             string          `json:"id"`
	UserID         string          `json:"user_id"`
	Amount         decimal.Decimal `json:"amount"`
	Currency       string          `json:"currency"`
	Destination    string          `json:"destination"`
	Status         string          `json:"status"`
	IdempotencyKey string          `json:"idempotency_key"`
	CreatedAt      string          `json:"created_at"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) createWithdrawal(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{"invalid request body"})
		return
	}

	if _, err := uuid.Parse(req.UserID); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{"invalid user_id: must be UUID"})
		return
	}

	withdrawal, isIdempotent, err := h.svc.CreateWithdrawal(r.Context(), service.CreateRequest{
		UserID:         req.UserID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Destination:    req.Destination,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	status := http.StatusCreated
	if isIdempotent {
		status = http.StatusOK
	}

	writeJSON(w, status, toResponse(withdrawal))
}

func (h *Handler) getWithdrawal(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if _, err := uuid.Parse(id); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{"invalid id: must be UUID"})
		return
	}

	withdrawal, err := h.svc.GetWithdrawal(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toResponse(withdrawal))
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidAmount),
		errors.Is(err, domain.ErrUnsupportedCurrency),
		errors.Is(err, domain.ErrEmptyDestination),
		errors.Is(err, domain.ErrMissingIdempotencyKey):
		writeJSON(w, http.StatusBadRequest, errorResponse{err.Error()})
	case errors.Is(err, domain.ErrInsufficientFunds):
		writeJSON(w, http.StatusConflict, errorResponse{err.Error()})
	case errors.Is(err, domain.ErrBalanceNotFound):
		writeJSON(w, http.StatusUnprocessableEntity, errorResponse{err.Error()})
	case errors.Is(err, domain.ErrIdempotencyConflict):
		writeJSON(w, http.StatusUnprocessableEntity, errorResponse{err.Error()})
	case errors.Is(err, domain.ErrWithdrawalNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{err.Error()})
	default:
		slog.Error("unexpected error", "err", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{"internal server error"})
	}
}

func toResponse(w *domain.Withdrawal) withdrawalResponse {
	return withdrawalResponse{
		ID:             w.ID,
		UserID:         w.UserID,
		Amount:         w.Amount,
		Currency:       w.Currency,
		Destination:    w.Destination,
		Status:         w.Status,
		IdempotencyKey: w.IdempotencyKey,
		CreatedAt:      w.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to write response", "err", err)
	}
}
