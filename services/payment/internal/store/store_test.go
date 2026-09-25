package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentIdempotency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://savor:savor@localhost:5432/payments_test?sslmode=disable")
	if err != nil {
		t.Skipf("cannot connect to test db: %v", err)
		return
	}
	defer pool.Close()

	s := New(pool)
	orderID := uuid.New()

	cleanup(t, pool)
	defer cleanup(t, pool)

	p1, fresh1, err := s.Charge(ctx, orderID, 5000, false)
	require.NoError(t, err)
	assert.True(t, fresh1, "first charge should be fresh")
	assert.Equal(t, "succeeded", p1.Status)
	assert.Equal(t, 5000, p1.AmountCents)
	assert.Equal(t, orderID, p1.OrderID)

	p2, fresh2, err := s.Charge(ctx, orderID, 5000, false)
	require.NoError(t, err)
	assert.False(t, fresh2, "second charge should return cached result")
	assert.Equal(t, p1.ID, p2.ID, "should return same payment")
	assert.Equal(t, "succeeded", p2.Status)
}

func TestPaymentFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://savor:savor@localhost:5432/payments_test?sslmode=disable")
	if err != nil {
		t.Skipf("cannot connect to test db: %v", err)
		return
	}
	defer pool.Close()

	s := New(pool)
	orderID := uuid.New()

	cleanup(t, pool)
	defer cleanup(t, pool)

	p, fresh, err := s.Charge(ctx, orderID, 3000, true)
	require.NoError(t, err)
	assert.True(t, fresh)
	assert.Equal(t, "failed", p.Status)
	assert.NotNil(t, p.FailureReason)
	assert.Equal(t, "card_declined", *p.FailureReason)
}

func TestByOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://savor:savor@localhost:5432/payments_test?sslmode=disable")
	if err != nil {
		t.Skipf("cannot connect to test db: %v", err)
		return
	}
	defer pool.Close()

	s := New(pool)
	orderID := uuid.New()

	cleanup(t, pool)
	defer cleanup(t, pool)

	_, _, err = s.Charge(ctx, orderID, 2000, false)
	require.NoError(t, err)

	p, _, err := s.ByOrder(ctx, orderID)
	require.NoError(t, err)
	assert.Equal(t, orderID, p.OrderID)
	assert.Equal(t, 2000, p.AmountCents)
}

func TestGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://savor:savor@localhost:5432/payments_test?sslmode=disable")
	if err != nil {
		t.Skipf("cannot connect to test db: %v", err)
		return
	}
	defer pool.Close()

	s := New(pool)
	orderID := uuid.New()

	cleanup(t, pool)
	defer cleanup(t, pool)

	p1, _, err := s.Charge(ctx, orderID, 4500, false)
	require.NoError(t, err)

	p2, err := s.Get(ctx, p1.ID.String())
	require.NoError(t, err)
	assert.Equal(t, p1.ID, p2.ID)
	assert.Equal(t, 4500, p2.AmountCents)

	p3, err := s.Get(ctx, orderID.String())
	require.NoError(t, err)
	assert.Equal(t, p1.ID, p3.ID)
}

func cleanup(t *testing.T, pool *pgxpool.Pool) {
	_, err := pool.Exec(context.Background(), "TRUNCATE payments")
	if err != nil {
		t.Logf("cleanup failed: %v", err)
	}
}
