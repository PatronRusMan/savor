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
	"github.com/savor/identity/internal/auth"
	"github.com/savor/identity/internal/store"
)

type Server struct {
	store  *store.Store
	secret string
	log    *slog.Logger
}

func New(st *store.Store, secret string, log *slog.Logger) http.Handler {
	s := &Server{store: st, secret: secret, log: log}
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/ready", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	r.Handle("/metrics", promhttp.Handler())
	r.Post("/auth/register", s.register)
	r.Post("/auth/login", s.login)
	r.Post("/auth/refresh", s.refresh)
	r.Post("/auth/logout", s.logout)
	r.Get("/auth/me", s.me)
	return r
}

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json", "invalid body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Name = strings.TrimSpace(req.Name)
	if req.Role == "" {
		req.Role = "customer"
	}
	if req.Email == "" || len(req.Password) < 8 || req.Name == "" || !auth.ValidRole(req.Role) {
		writeErr(w, http.StatusBadRequest, "invalid_input", "email, name, 8+ char password and valid role required")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "hash_failed", "could not hash password")
		return
	}
	u := store.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: hash,
		Name:         req.Name,
		Role:         req.Role,
	}
	if err := s.store.CreateUser(r.Context(), u); err != nil {
		if errors.Is(err, store.ErrConflict) {
			writeErr(w, http.StatusConflict, "email_taken", "email already registered")
			return
		}
		s.log.Error("create user", "err", err)
		writeErr(w, http.StatusInternalServerError, "internal", "could not create user")
		return
	}
	s.issue(r.Context(), w, u)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json", "invalid body")
		return
	}
	u, err := s.store.ByEmail(r.Context(), strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil || auth.CheckPassword(u.PasswordHash, req.Password) != nil {
		writeErr(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}
	s.issue(r.Context(), w, u)
}

func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		writeErr(w, http.StatusBadRequest, "invalid_json", "refreshToken required")
		return
	}
	raw, hash, exp := auth.SignRefresh()
	uid, err := s.store.RotateRefresh(r.Context(), auth.HashRefresh(req.RefreshToken), hash, uuid.New(), uuid.Nil, exp)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid_refresh", "refresh token invalid")
		return
	}
	u, err := s.store.ByID(r.Context(), uid)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid_refresh", "user missing")
		return
	}
	access, err := auth.SignAccess(s.secret, u.ID.String(), u.Email, u.Name, u.Role)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "token", "could not sign token")
		return
	}
	writeJSON(w, http.StatusOK, tokenResp(u, access, raw, exp))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.RefreshToken != "" {
		_ = s.store.RevokeRefresh(r.Context(), auth.HashRefresh(req.RefreshToken))
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	if id := r.Header.Get("X-User-Id"); id != "" {
		uid, err := uuid.Parse(id)
		if err == nil {
			u, err := s.store.ByID(r.Context(), uid)
			if err == nil {
				writeJSON(w, http.StatusOK, userJSON(u))
				return
			}
		}
	}
	header := r.Header.Get("Authorization")
	token := strings.TrimPrefix(header, "Bearer ")
	if token == "" || token == header {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "missing token")
		return
	}
	claims, err := auth.ParseAccess(s.secret, token)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "invalid token")
		return
	}
	uid, err := uuid.Parse(claims.Subject)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "invalid subject")
		return
	}
	u, err := s.store.ByID(r.Context(), uid)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}
	writeJSON(w, http.StatusOK, userJSON(u))
}

func (s *Server) issue(ctx context.Context, w http.ResponseWriter, u store.User) {
	access, err := auth.SignAccess(s.secret, u.ID.String(), u.Email, u.Name, u.Role)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "token", "could not sign token")
		return
	}
	raw, hash, exp := auth.SignRefresh()
	if err := s.store.SaveRefresh(ctx, uuid.New(), u.ID, hash, exp); err != nil {
		s.log.Error("save refresh", "err", err)
		writeErr(w, http.StatusInternalServerError, "token", "could not persist refresh token")
		return
	}
	writeJSON(w, http.StatusOK, tokenResp(u, access, raw, exp))
}

func tokenResp(u store.User, access, refresh string, exp time.Time) map[string]any {
	return map[string]any{
		"accessToken":      access,
		"refreshToken":     refresh,
		"expiresIn":        900,
		"user":             userJSON(u),
		"refreshExpiresAt": exp.UTC().Format(time.RFC3339),
	}
}

func userJSON(u store.User) map[string]any {
	return map[string]any{
		"id":    u.ID.String(),
		"email": u.Email,
		"name":  u.Name,
		"role":  u.Role,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}
