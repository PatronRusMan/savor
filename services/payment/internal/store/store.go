package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type Payment struct {
	ID             uuid.UUID
	OrderID        uuid.UUID
	AmountCents    int
	Status         string
	IdempotencyKey string
	FailureReason  *string
	CreatedAt      time.Time
}

type Store struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) Charge(ctx context.Context, orderID uuid.UUID, amount int, fail bool) (Payment, bool, error) {
	key := orderID.String()
	var existing Payment
	err := s.scan(s.pool.QueryRow(ctx, `
		SELECT id, order_id, amount_cents, status, idempotency_key, failure_reason, created_at
		FROM payments WHERE idempotency_key=$1`, key), &existing)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Payment{}, false, err
	}

	p := Payment{
		ID:             uuid.New(),
		OrderID:        orderID,
		AmountCents:    amount,
		Status:         "processing",
		IdempotencyKey: key,
		CreatedAt:      time.Now(),
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO payments (id, order_id, amount_cents, status, idempotency_key)
		VALUES ($1,$2,$3,$4,$5)`, p.ID, p.OrderID, p.AmountCents, p.Status, p.IdempotencyKey); err != nil {
		if isUnique(err) {
			return s.ByOrder(ctx, orderID)
		}
		return Payment{}, false, err
	}
	status := "succeeded"
	var reason *string
	if fail {
		status = "failed"
		msg := "card_declined"
		reason = &msg
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE payments SET status=$2, failure_reason=$3, updated_at=now() WHERE id=$1`, p.ID, status, reason); err != nil {
		return Payment{}, false, err
	}
	p.Status = status
	p.FailureReason = reason
	return p, true, nil
}

func (s *Store) ByOrder(ctx context.Context, orderID uuid.UUID) (Payment, bool, error) {
	var p Payment
	err := s.scan(s.pool.QueryRow(ctx, `
		SELECT id, order_id, amount_cents, status, idempotency_key, failure_reason, created_at
		FROM payments WHERE order_id=$1`, orderID), &p)
	if err != nil {
		return Payment{}, false, err
	}
	return p, false, nil
}

func (s *Store) Get(ctx context.Context, id string) (Payment, error) {
	var p Payment
	err := s.scan(s.pool.QueryRow(ctx, `
		SELECT id, order_id, amount_cents, status, idempotency_key, failure_reason, created_at
		FROM payments WHERE id::text=$1 OR order_id::text=$1`, id), &p)
	return p, err
}

func (s *Store) scan(row pgx.Row, p *Payment) error {
	err := row.Scan(&p.ID, &p.OrderID, &p.AmountCents, &p.Status, &p.IdempotencyKey, &p.FailureReason, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func isUnique(err error) bool {
	return err != nil && (contains(err.Error(), "duplicate key") || contains(err.Error(), "unique"))
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
