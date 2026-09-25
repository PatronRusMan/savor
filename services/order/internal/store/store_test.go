package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrCreateCart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://savor:savor@localhost:5432/orders_test?sslmode=disable")
	if err != nil {
		t.Skipf("cannot connect to test db: %v", err)
		return
	}
	defer pool.Close()

	s := New(pool)
	userID := uuid.New()

	cleanup(t, pool)
	defer cleanup(t, pool)

	cart1, err := s.GetOrCreateCart(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, userID, cart1.UserID)
	assert.Nil(t, cart1.RestaurantID)
	assert.Empty(t, cart1.Items)

	cart2, err := s.GetOrCreateCart(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, cart1.ID, cart2.ID, "should return same cart")
}

func TestReplaceCartItem(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://savor:savor@localhost:5432/orders_test?sslmode=disable")
	if err != nil {
		t.Skipf("cannot connect to test db: %v", err)
		return
	}
	defer pool.Close()

	s := New(pool)
	userID := uuid.New()
	dishID := uuid.New()
	restaurantID := uuid.New()
	ownerID := uuid.New()

	cleanup(t, pool)
	defer cleanup(t, pool)

	cart, err := s.ReplaceCartItem(ctx, userID, dishID, restaurantID, ownerID, "Test Restaurant", "Khachapuri", 1200, 2)
	require.NoError(t, err)
	assert.Equal(t, restaurantID, *cart.RestaurantID)
	assert.Equal(t, "Test Restaurant", *cart.RestaurantName)
	assert.Len(t, cart.Items, 1)
	assert.Equal(t, "Khachapuri", cart.Items[0].Name)
	assert.Equal(t, 1200, cart.Items[0].PriceCents)
	assert.Equal(t, 2, cart.Items[0].Quantity)
}

func TestCartRestaurantSwitch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://savor:savor@localhost:5432/orders_test?sslmode=disable")
	if err != nil {
		t.Skipf("cannot connect to test db: %v", err)
		return
	}
	defer pool.Close()

	s := New(pool)
	userID := uuid.New()
	dish1 := uuid.New()
	rest1 := uuid.New()
	owner1 := uuid.New()
	dish2 := uuid.New()
	rest2 := uuid.New()
	owner2 := uuid.New()

	cleanup(t, pool)
	defer cleanup(t, pool)

	cart1, err := s.ReplaceCartItem(ctx, userID, dish1, rest1, owner1, "Restaurant 1", "Dish 1", 1000, 1)
	require.NoError(t, err)
	assert.Len(t, cart1.Items, 1)

	cart2, err := s.ReplaceCartItem(ctx, userID, dish2, rest2, owner2, "Restaurant 2", "Dish 2", 1500, 1)
	require.NoError(t, err)
	assert.Equal(t, rest2, *cart2.RestaurantID)
	assert.Len(t, cart2.Items, 1, "switching restaurant should clear old items")
	assert.Equal(t, "Dish 2", cart2.Items[0].Name)
}

func TestSetItemQty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://savor:savor@localhost:5432/orders_test?sslmode=disable")
	if err != nil {
		t.Skipf("cannot connect to test db: %v", err)
		return
	}
	defer pool.Close()

	s := New(pool)
	userID := uuid.New()
	dishID := uuid.New()
	restaurantID := uuid.New()
	ownerID := uuid.New()

	cleanup(t, pool)
	defer cleanup(t, pool)

	cart1, err := s.ReplaceCartItem(ctx, userID, dishID, restaurantID, ownerID, "Restaurant", "Dish", 1000, 2)
	require.NoError(t, err)
	itemID := cart1.Items[0].ID

	cart2, err := s.SetItemQty(ctx, userID, itemID, 5)
	require.NoError(t, err)
	assert.Equal(t, 5, cart2.Items[0].Quantity)

	cart3, err := s.SetItemQty(ctx, userID, itemID, 0)
	require.NoError(t, err)
	assert.Empty(t, cart3.Items, "setting qty to 0 should remove item")
}

func TestCreateOrderFromCart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://savor:savor@localhost:5432/orders_test?sslmode=disable")
	if err != nil {
		t.Skipf("cannot connect to test db: %v", err)
		return
	}
	defer pool.Close()

	s := New(pool)
	userID := uuid.New()
	dish1 := uuid.New()
	dish2 := uuid.New()
	restaurantID := uuid.New()
	ownerID := uuid.New()

	cleanup(t, pool)
	defer cleanup(t, pool)

	_, err = s.ReplaceCartItem(ctx, userID, dish1, restaurantID, ownerID, "Test Restaurant", "Dish 1", 1200, 2)
	require.NoError(t, err)
	_, err = s.ReplaceCartItem(ctx, userID, dish2, restaurantID, ownerID, "Test Restaurant", "Dish 2", 800, 1)
	require.NoError(t, err)

	order, err := s.CreateOrderFromCart(ctx, userID, "John Doe", "123 Main St")
	require.NoError(t, err)
	assert.Equal(t, "pending_payment", order.Status)
	assert.Equal(t, "John Doe", order.CustomerName)
	assert.Equal(t, "123 Main St", order.Address)
	assert.Equal(t, 3200, order.TotalCents)
	assert.Len(t, order.Items, 2)

	cart, err := s.GetOrCreateCart(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, cart.Items, "cart should be cleared after order")
	assert.Nil(t, cart.RestaurantID)
}

func TestCreateOrderFromEmptyCart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://savor:savor@localhost:5432/orders_test?sslmode=disable")
	if err != nil {
		t.Skipf("cannot connect to test db: %v", err)
		return
	}
	defer pool.Close()

	s := New(pool)
	userID := uuid.New()

	cleanup(t, pool)
	defer cleanup(t, pool)

	_, err = s.CreateOrderFromCart(ctx, userID, "Jane Doe", "456 Oak St")
	assert.ErrorIs(t, err, ErrConflict, "creating order from empty cart should fail")
}

func TestTransition(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://savor:savor@localhost:5432/orders_test?sslmode=disable")
	if err != nil {
		t.Skipf("cannot connect to test db: %v", err)
		return
	}
	defer pool.Close()

	s := New(pool)
	userID := uuid.New()
	dishID := uuid.New()
	restaurantID := uuid.New()
	ownerID := uuid.New()

	cleanup(t, pool)
	defer cleanup(t, pool)

	_, err = s.ReplaceCartItem(ctx, userID, dishID, restaurantID, ownerID, "Restaurant", "Dish", 1500, 1)
	require.NoError(t, err)

	order, err := s.CreateOrderFromCart(ctx, userID, "Customer", "Address")
	require.NoError(t, err)
	assert.Equal(t, "pending_payment", order.Status)

	order2, err := s.Transition(ctx, order.ID, []string{"pending_payment"}, "paid", nil, "order.paid", nil)
	require.NoError(t, err)
	assert.Equal(t, "paid", order2.Status)

	_, err = s.Transition(ctx, order.ID, []string{"pending_payment"}, "cancelled", nil, "order.cancelled", nil)
	assert.ErrorIs(t, err, ErrConflict, "transitioning from wrong status should fail")
}

func cleanup(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	_, err := pool.Exec(ctx, "TRUNCATE orders CASCADE")
	if err != nil {
		t.Logf("cleanup orders failed: %v", err)
	}
	_, err = pool.Exec(ctx, "TRUNCATE carts CASCADE")
	if err != nil {
		t.Logf("cleanup carts failed: %v", err)
	}
	_, err = pool.Exec(ctx, "TRUNCATE outbox")
	if err != nil {
		t.Logf("cleanup outbox failed: %v", err)
	}
}
