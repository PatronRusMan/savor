package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")

type Cart struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	RestaurantID      *uuid.UUID
	RestaurantName    *string
	RestaurantOwnerID *uuid.UUID
	Items             []CartItem
}

type CartItem struct {
	ID         uuid.UUID
	DishID     uuid.UUID
	Name       string
	PriceCents int
	Quantity   int
}

type Order struct {
	ID                 uuid.UUID
	CustomerID         uuid.UUID
	CustomerName       string
	RestaurantID       uuid.UUID
	RestaurantName     string
	RestaurantOwnerID  uuid.UUID
	CourierID          *uuid.UUID
	CourierName        *string
	Status             string
	TotalCents         int
	Address            string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Items              []OrderItem
}

type OrderItem struct {
	ID         uuid.UUID
	DishID     uuid.UUID
	Name       string
	PriceCents int
	Quantity   int
}

type OutboxEvent struct {
	ID        uuid.UUID
	EventType string
	Payload   []byte
}

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) GetOrCreateCart(ctx context.Context, userID uuid.UUID) (Cart, error) {
	var c Cart
	err := s.pool.QueryRow(ctx, `
		INSERT INTO carts (id, user_id) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
		RETURNING id, user_id, restaurant_id, restaurant_name, restaurant_owner_id`,
		uuid.New(), userID).Scan(&c.ID, &c.UserID, &c.RestaurantID, &c.RestaurantName, &c.RestaurantOwnerID)
	if err != nil {
		return Cart{}, err
	}
	c.Items, err = s.cartItems(ctx, c.ID)
	return c, err
}

func (s *Store) ReplaceCartItem(ctx context.Context, userID, dishID, restaurantID, ownerID uuid.UUID, restaurantName, dishName string, price, qty int) (Cart, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Cart{}, err
	}
	defer tx.Rollback(ctx)

	var cart Cart
	err = tx.QueryRow(ctx, `
		INSERT INTO carts (id, user_id) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET updated_at = now()
		RETURNING id, user_id, restaurant_id, restaurant_name, restaurant_owner_id`,
		uuid.New(), userID).Scan(&cart.ID, &cart.UserID, &cart.RestaurantID, &cart.RestaurantName, &cart.RestaurantOwnerID)
	if err != nil {
		return Cart{}, err
	}
	if cart.RestaurantID != nil && *cart.RestaurantID != restaurantID {
		if _, err := tx.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1`, cart.ID); err != nil {
			return Cart{}, err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE carts SET restaurant_id=$2, restaurant_name=$3, restaurant_owner_id=$4, updated_at=now()
		WHERE id=$1`, cart.ID, restaurantID, restaurantName, ownerID); err != nil {
		return Cart{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO cart_items (id, cart_id, dish_id, name, price_cents, quantity)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (cart_id, dish_id) DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity, name = EXCLUDED.name, price_cents = EXCLUDED.price_cents`,
		uuid.New(), cart.ID, dishID, dishName, price, qty); err != nil {
		return Cart{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Cart{}, err
	}
	return s.GetOrCreateCart(ctx, userID)
}

func (s *Store) SetItemQty(ctx context.Context, userID, itemID uuid.UUID, qty int) (Cart, error) {
	cart, err := s.GetOrCreateCart(ctx, userID)
	if err != nil {
		return Cart{}, err
	}
	if qty <= 0 {
		_, err = s.pool.Exec(ctx, `DELETE FROM cart_items WHERE id=$1 AND cart_id=$2`, itemID, cart.ID)
	} else {
		_, err = s.pool.Exec(ctx, `UPDATE cart_items SET quantity=$3 WHERE id=$1 AND cart_id=$2`, itemID, cart.ID, qty)
	}
	if err != nil {
		return Cart{}, err
	}
	return s.GetOrCreateCart(ctx, userID)
}

func (s *Store) ClearCart(ctx context.Context, tx pgx.Tx, cartID uuid.UUID) error {
	if _, err := tx.Exec(ctx, `DELETE FROM cart_items WHERE cart_id=$1`, cartID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE carts SET restaurant_id=NULL, restaurant_name=NULL, restaurant_owner_id=NULL WHERE id=$1`, cartID)
	return err
}

func (s *Store) CreateOrderFromCart(ctx context.Context, userID uuid.UUID, customerName, address string) (Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx)

	cart, err := s.GetOrCreateCart(ctx, userID)
	if err != nil {
		return Order{}, err
	}
	if len(cart.Items) == 0 || cart.RestaurantID == nil || cart.RestaurantOwnerID == nil {
		return Order{}, ErrConflict
	}
	total := 0
	for _, it := range cart.Items {
		total += it.PriceCents * it.Quantity
	}
	o := Order{
		ID:                uuid.New(),
		CustomerID:        userID,
		CustomerName:      customerName,
		RestaurantID:      *cart.RestaurantID,
		RestaurantName:    deref(cart.RestaurantName),
		RestaurantOwnerID: *cart.RestaurantOwnerID,
		Status:            "pending_payment",
		TotalCents:        total,
		Address:           address,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Items:             make([]OrderItem, 0, len(cart.Items)),
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO orders (id, customer_id, customer_name, restaurant_id, restaurant_name, restaurant_owner_id, status, total_cents, address)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		o.ID, o.CustomerID, o.CustomerName, o.RestaurantID, o.RestaurantName, o.RestaurantOwnerID, o.Status, o.TotalCents, o.Address); err != nil {
		return Order{}, err
	}
	for _, it := range cart.Items {
		oi := OrderItem{ID: uuid.New(), DishID: it.DishID, Name: it.Name, PriceCents: it.PriceCents, Quantity: it.Quantity}
		if _, err := tx.Exec(ctx, `
			INSERT INTO order_items (id, order_id, dish_id, name, price_cents, quantity)
			VALUES ($1,$2,$3,$4,$5,$6)`, oi.ID, o.ID, oi.DishID, oi.Name, oi.PriceCents, oi.Quantity); err != nil {
			return Order{}, err
		}
		o.Items = append(o.Items, oi)
	}
	payload, _ := json.Marshal(map[string]any{
		"orderId":            o.ID.String(),
		"customerId":         o.CustomerID.String(),
		"customerName":       o.CustomerName,
		"restaurantId":       o.RestaurantID.String(),
		"restaurantName":     o.RestaurantName,
		"restaurantOwnerId":  o.RestaurantOwnerID.String(),
		"totalCents":         o.TotalCents,
		"address":            o.Address,
		"status":             o.Status,
	})
	if _, err := tx.Exec(ctx, `INSERT INTO outbox (id, event_type, payload) VALUES ($1,$2,$3)`, uuid.New(), "order.created", payload); err != nil {
		return Order{}, err
	}
	if err := s.ClearCart(ctx, tx, cart.ID); err != nil {
		return Order{}, err
	}
	return o, tx.Commit(ctx)
}

func (s *Store) Get(ctx context.Context, id uuid.UUID) (Order, error) {
	o, err := s.scanOrder(s.pool.QueryRow(ctx, `
		SELECT id, customer_id, customer_name, restaurant_id, restaurant_name, restaurant_owner_id, courier_id, courier_name, status, total_cents, address, created_at, updated_at
		FROM orders WHERE id=$1`, id))
	if err != nil {
		return Order{}, err
	}
	o.Items, err = s.orderItems(ctx, o.ID)
	return o, err
}

func (s *Store) ListFor(ctx context.Context, role, userID string) ([]Order, error) {
	q := `SELECT id, customer_id, customer_name, restaurant_id, restaurant_name, restaurant_owner_id, courier_id, courier_name, status, total_cents, address, created_at, updated_at FROM orders `
	var arg any
	switch role {
	case "restaurant":
		q += `WHERE restaurant_owner_id=$1 ORDER BY created_at DESC LIMIT 100`
		arg = userID
	case "courier":
		q += `WHERE courier_id=$1 ORDER BY created_at DESC LIMIT 100`
		arg = userID
	default:
		q += `WHERE customer_id=$1 ORDER BY created_at DESC LIMIT 100`
		arg = userID
	}
	rows, err := s.pool.Query(ctx, q, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Order
	for rows.Next() {
		o, err := s.scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	for i := range out {
		out[i].Items, _ = s.orderItems(ctx, out[i].ID)
	}
	return out, rows.Err()
}

func (s *Store) Transition(ctx context.Context, id uuid.UUID, from []string, to string, extra func(pgx.Tx) error, eventType string, extraPayload map[string]any) (Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx)

	o, err := s.scanOrder(tx.QueryRow(ctx, `
		SELECT id, customer_id, customer_name, restaurant_id, restaurant_name, restaurant_owner_id, courier_id, courier_name, status, total_cents, address, created_at, updated_at
		FROM orders WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return Order{}, err
	}
	ok := false
	for _, f := range from {
		if o.Status == f {
			ok = true
			break
		}
	}
	if !ok {
		return Order{}, ErrConflict
	}
	if extra != nil {
		if err := extra(tx); err != nil {
			return Order{}, err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE orders SET status=$2, updated_at=now() WHERE id=$1`, id, to); err != nil {
		return Order{}, err
	}
	o.Status = to
	payload := map[string]any{
		"orderId":           o.ID.String(),
		"customerId":        o.CustomerID.String(),
		"customerName":      o.CustomerName,
		"restaurantId":      o.RestaurantID.String(),
		"restaurantName":    o.RestaurantName,
		"restaurantOwnerId": o.RestaurantOwnerID.String(),
		"status":            to,
		"address":           o.Address,
		"totalCents":        o.TotalCents,
	}
	if o.CourierID != nil {
		payload["courierId"] = o.CourierID.String()
	}
	for k, v := range extraPayload {
		payload[k] = v
	}
	raw, _ := json.Marshal(payload)
	if _, err := tx.Exec(ctx, `INSERT INTO outbox (id, event_type, payload) VALUES ($1,$2,$3)`, uuid.New(), eventType, raw); err != nil {
		return Order{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Order{}, err
	}
	return s.Get(ctx, id)
}

func (s *Store) AssignCourier(ctx context.Context, orderID, courierID uuid.UUID, courierName string) (Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		UPDATE orders SET courier_id=$2, courier_name=$3, status='assigned', updated_at=now()
		WHERE id=$1 AND status='ready'`, orderID, courierID, courierName)
	if err != nil {
		return Order{}, err
	}
	if tag.RowsAffected() == 0 {
		return Order{}, ErrConflict
	}
	o, err := s.scanOrder(tx.QueryRow(ctx, `
		SELECT id, customer_id, customer_name, restaurant_id, restaurant_name, restaurant_owner_id, courier_id, courier_name, status, total_cents, address, created_at, updated_at
		FROM orders WHERE id=$1`, orderID))
	if err != nil {
		return Order{}, err
	}
	payload, _ := json.Marshal(map[string]any{
		"orderId":           o.ID.String(),
		"customerId":        o.CustomerID.String(),
		"restaurantOwnerId": o.RestaurantOwnerID.String(),
		"courierId":         courierID.String(),
		"courierName":       courierName,
		"status":            "assigned",
		"restaurantName":    o.RestaurantName,
		"address":           o.Address,
	})
	if _, err := tx.Exec(ctx, `INSERT INTO outbox (id, event_type, payload) VALUES ($1,$2,$3)`, uuid.New(), "order.assigned", payload); err != nil {
		return Order{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Order{}, err
	}
	return s.Get(ctx, orderID)
}

func (s *Store) Unpublished(ctx context.Context, limit int) ([]OutboxEvent, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, event_type, payload FROM outbox WHERE published_at IS NULL ORDER BY created_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OutboxEvent
	for rows.Next() {
		var e OutboxEvent
		if err := rows.Scan(&e.ID, &e.EventType, &e.Payload); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) MarkPublished(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE outbox SET published_at=now() WHERE id=$1`, id)
	return err
}

func (s *Store) cartItems(ctx context.Context, cartID uuid.UUID) ([]CartItem, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, dish_id, name, price_cents, quantity FROM cart_items WHERE cart_id=$1`, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []CartItem
	for rows.Next() {
		var it CartItem
		if err := rows.Scan(&it.ID, &it.DishID, &it.Name, &it.PriceCents, &it.Quantity); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if items == nil {
		items = []CartItem{}
	}
	return items, rows.Err()
}

func (s *Store) orderItems(ctx context.Context, orderID uuid.UUID) ([]OrderItem, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, dish_id, name, price_cents, quantity FROM order_items WHERE order_id=$1`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []OrderItem
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(&it.ID, &it.DishID, &it.Name, &it.PriceCents, &it.Quantity); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if items == nil {
		items = []OrderItem{}
	}
	return items, rows.Err()
}

func (s *Store) scanOrder(row pgx.Row) (Order, error) {
	var o Order
	err := row.Scan(&o.ID, &o.CustomerID, &o.CustomerName, &o.RestaurantID, &o.RestaurantName, &o.RestaurantOwnerID, &o.CourierID, &o.CourierName, &o.Status, &o.TotalCents, &o.Address, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	return o, err
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
