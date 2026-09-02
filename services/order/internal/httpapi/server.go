package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/savor/order/internal/catalog"
	"github.com/savor/order/internal/live"
	"github.com/savor/order/internal/store"
)

type Server struct {
	store   *store.Store
	catalog *catalog.Client
	live    *live.Bus
	log     *slog.Logger
}

func New(st *store.Store, cat *catalog.Client, bus *live.Bus, log *slog.Logger) http.Handler {
	s := &Server{store: st, catalog: cat, live: bus, log: log}
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	r.Get("/ready", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ready"}) })
	r.Handle("/metrics", promhttp.Handler())
	r.Get("/cart", s.getCart)
	r.Post("/cart/items", s.addItem)
	r.Patch("/cart/items/{id}", s.patchItem)
	r.Delete("/cart/items/{id}", s.deleteItem)
	r.Get("/orders", s.listOrders)
	r.Post("/orders", s.createOrder)
	r.Get("/orders/{id}", s.getOrder)
	r.Post("/orders/{id}/status", s.setStatus)
	return r
}

func (s *Server) getCart(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		writeErr(w, 401, "unauthorized", "missing user")
		return
	}
	c, err := s.store.GetOrCreateCart(r.Context(), uid)
	if err != nil {
		s.log.Error("cart", "err", err)
		writeErr(w, 500, "internal", "cart failed")
		return
	}
	writeJSON(w, 200, cartJSON(c))
}

func (s *Server) addItem(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		writeErr(w, 401, "unauthorized", "missing user")
		return
	}
	var req struct {
		DishID   string `json:"dishId"`
		Quantity int    `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DishID == "" {
		writeErr(w, 400, "invalid", "dishId required")
		return
	}
	if req.Quantity < 1 {
		req.Quantity = 1
	}
	dish, err := s.catalog.Dish(req.DishID)
	if err != nil || !dish.IsAvailable {
		writeErr(w, 400, "dish_unavailable", "dish not available")
		return
	}
	rest, err := s.catalog.Restaurant(dish.RestaurantID)
	if err != nil || !rest.IsOpen {
		writeErr(w, 400, "restaurant_closed", "restaurant is closed")
		return
	}
	dishID, _ := uuid.Parse(dish.ID)
	restID, _ := uuid.Parse(rest.ID)
	ownerID, _ := uuid.Parse(rest.OwnerID)
	c, err := s.store.ReplaceCartItem(r.Context(), uid, dishID, restID, ownerID, rest.Name, dish.Name, dish.PriceCents, req.Quantity)
	if err != nil {
		s.log.Error("add item", "err", err)
		writeErr(w, 500, "internal", "could not add item")
		return
	}
	writeJSON(w, 200, cartJSON(c))
}

func (s *Server) patchItem(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		writeErr(w, 401, "unauthorized", "missing user")
		return
	}
	itemID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, 400, "invalid", "bad id")
		return
	}
	var req struct {
		Quantity int `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "invalid", "quantity required")
		return
	}
	c, err := s.store.SetItemQty(r.Context(), uid, itemID, req.Quantity)
	if err != nil {
		writeErr(w, 500, "internal", "update failed")
		return
	}
	writeJSON(w, 200, cartJSON(c))
}

func (s *Server) deleteItem(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		writeErr(w, 401, "unauthorized", "missing user")
		return
	}
	itemID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, 400, "invalid", "bad id")
		return
	}
	c, err := s.store.SetItemQty(r.Context(), uid, itemID, 0)
	if err != nil {
		writeErr(w, 500, "internal", "delete failed")
		return
	}
	writeJSON(w, 200, cartJSON(c))
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		writeErr(w, 401, "unauthorized", "missing user")
		return
	}
	var req struct {
		Address string `json:"address"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if strings.TrimSpace(req.Address) == "" {
		req.Address = "12 Rustaveli Ave, Tbilisi"
	}
	o, err := s.store.CreateOrderFromCart(r.Context(), uid, header(r, "X-User-Name", "Guest"), req.Address)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			writeErr(w, 400, "empty_cart", "cart is empty")
			return
		}
		s.log.Error("create order", "err", err)
		writeErr(w, 500, "internal", "could not create order")
		return
	}
	s.publishLive(o)
	writeJSON(w, 201, orderJSON(o))
}

func (s *Server) listOrders(w http.ResponseWriter, r *http.Request) {
	uid := r.Header.Get("X-User-Id")
	role := r.Header.Get("X-User-Role")
	if uid == "" {
		writeErr(w, 401, "unauthorized", "missing user")
		return
	}
	list, err := s.store.ListFor(r.Context(), role, uid)
	if err != nil {
		s.log.Error("list", "err", err)
		writeErr(w, 500, "internal", "list failed")
		return
	}
	out := make([]any, 0, len(list))
	for _, o := range list {
		out = append(out, orderJSON(o))
	}
	writeJSON(w, 200, out)
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, 400, "invalid", "bad id")
		return
	}
	o, err := s.store.Get(r.Context(), id)
	if err != nil {
		writeErr(w, 404, "not_found", "order not found")
		return
	}
	uid := r.Header.Get("X-User-Id")
	role := r.Header.Get("X-User-Role")
	if !canSee(o, uid, role) {
		writeErr(w, 403, "forbidden", "not your order")
		return
	}
	writeJSON(w, 200, orderJSON(o))
}

func (s *Server) setStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, 400, "invalid", "bad id")
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "invalid", "status required")
		return
	}
	o, err := s.store.Get(r.Context(), id)
	if err != nil {
		writeErr(w, 404, "not_found", "order not found")
		return
	}
	uid := r.Header.Get("X-User-Id")
	role := r.Header.Get("X-User-Role")
	from, event, ok := allowed(role, uid, o, req.Status)
	if !ok {
		writeErr(w, 403, "forbidden", "transition not allowed")
		return
	}
	updated, err := s.store.Transition(r.Context(), id, from, req.Status, nil, event, nil)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			writeErr(w, 409, "conflict", "invalid state")
			return
		}
		s.log.Error("transition", "err", err)
		writeErr(w, 500, "internal", "status failed")
		return
	}
	s.publishLive(updated)
	writeJSON(w, 200, orderJSON(updated))
}

func allowed(role, uid string, o store.Order, to string) (from []string, event string, ok bool) {
	switch role {
	case "customer":
		if o.CustomerID.String() != uid {
			return nil, "", false
		}
		if to == "cancelled" {
			return []string{"pending_payment", "paid"}, "order.cancelled", true
		}
	case "restaurant":
		if o.RestaurantOwnerID.String() != uid {
			return nil, "", false
		}
		switch to {
		case "accepted":
			return []string{"paid"}, "order.accepted", true
		case "cooking":
			return []string{"accepted"}, "order.cooking", true
		case "ready":
			return []string{"cooking"}, "order.ready", true
		case "cancelled":
			return []string{"paid", "accepted"}, "order.cancelled", true
		}
	}
	return nil, "", false
}

func canSee(o store.Order, uid, role string) bool {
	if o.CustomerID.String() == uid || o.RestaurantOwnerID.String() == uid {
		return true
	}
	if o.CourierID != nil && o.CourierID.String() == uid {
		return true
	}
	return false
}

func (s *Server) publishLive(o store.Order) {
	ids := []string{o.CustomerID.String(), o.RestaurantOwnerID.String()}
	if o.CourierID != nil {
		ids = append(ids, o.CourierID.String())
	}
	s.live.Publish(context.Background(), ids, map[string]any{
		"type":    "order.updated",
		"orderId": o.ID.String(),
		"status":  o.Status,
		"order":   orderJSON(o),
	})
}

func userID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(r.Header.Get("X-User-Id"))
}

func header(r *http.Request, k, def string) string {
	if v := r.Header.Get(k); v != "" {
		return v
	}
	return def
}

func cartJSON(c store.Cart) map[string]any {
	items := make([]map[string]any, 0, len(c.Items))
	total := 0
	for _, it := range c.Items {
		total += it.PriceCents * it.Quantity
		items = append(items, map[string]any{
			"id": it.ID.String(), "dishId": it.DishID.String(), "name": it.Name,
			"priceCents": it.PriceCents, "quantity": it.Quantity,
		})
	}
	m := map[string]any{"id": c.ID.String(), "items": items, "totalCents": total}
	if c.RestaurantID != nil {
		m["restaurantId"] = c.RestaurantID.String()
		m["restaurantName"] = deref(c.RestaurantName)
	}
	return m
}

func orderJSON(o store.Order) map[string]any {
	items := make([]map[string]any, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, map[string]any{
			"id": it.ID.String(), "dishId": it.DishID.String(), "name": it.Name,
			"priceCents": it.PriceCents, "quantity": it.Quantity,
		})
	}
	m := map[string]any{
		"id": o.ID.String(), "customerId": o.CustomerID.String(), "customerName": o.CustomerName,
		"restaurantId": o.RestaurantID.String(), "restaurantName": o.RestaurantName,
		"restaurantOwnerId": o.RestaurantOwnerID.String(), "status": o.Status,
		"totalCents": o.TotalCents, "address": o.Address, "items": items,
		"createdAt": o.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt": o.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if o.CourierID != nil {
		m["courierId"] = o.CourierID.String()
		m["courierName"] = deref(o.CourierName)
	}
	return m
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}
