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
var ErrConflict = errors.New("conflict")

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Name         string
	Role         string
	CreatedAt    time.Time
}

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) CreateUser(ctx context.Context, u User) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, name, role)
		VALUES ($1, $2, $3, $4, $5)`,
		u.ID, u.Email, u.PasswordHash, u.Name, u.Role)
	if err != nil {
		if isUnique(err) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (s *Store) ByEmail(ctx context.Context, email string) (User, error) {
	return s.scanUser(s.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, name, role, created_at
		FROM users WHERE email = $1`, email))
}

func (s *Store) ByID(ctx context.Context, id uuid.UUID) (User, error) {
	return s.scanUser(s.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, name, role, created_at
		FROM users WHERE id = $1`, id))
}

func (s *Store) Count(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) SaveRefresh(ctx context.Context, id, userID uuid.UUID, hash string, exp time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)`, id, userID, hash, exp)
	return err
}

func (s *Store) RotateRefresh(ctx context.Context, oldHash, newHash string, newID, userID uuid.UUID, exp time.Time) (uuid.UUID, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	var uid uuid.UUID
	var expires time.Time
	var revoked bool
	err = tx.QueryRow(ctx, `
		SELECT user_id, expires_at, revoked FROM refresh_tokens WHERE token_hash = $1`, oldHash).
		Scan(&uid, &expires, &revoked)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}
	if revoked || time.Now().After(expires) || uid != userID && userID != uuid.Nil {
		return uuid.Nil, ErrNotFound
	}
	if _, err = tx.Exec(ctx, `UPDATE refresh_tokens SET revoked = true WHERE token_hash = $1`, oldHash); err != nil {
		return uuid.Nil, err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)`, newID, uid, newHash, exp); err != nil {
		return uuid.Nil, err
	}
	return uid, tx.Commit(ctx)
}

func (s *Store) RevokeRefresh(ctx context.Context, hash string) error {
	_, err := s.pool.Exec(ctx, `UPDATE refresh_tokens SET revoked = true WHERE token_hash = $1`, hash)
	return err
}

func (s *Store) scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func isUnique(err error) bool {
	return err != nil && (contains(err.Error(), "duplicate key") || contains(err.Error(), "unique"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})())
}
