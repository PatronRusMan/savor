package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Role string

const (
	RoleCustomer   Role = "customer"
	RoleRestaurant Role = "restaurant"
	RoleCourier    Role = "courier"
)

func ValidRole(r string) bool {
	return r == string(RoleCustomer) || r == string(RoleRestaurant) || r == string(RoleCourier)
}

type Claims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	return string(b), err
}

func CheckPassword(hash, pw string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw))
}

func SignAccess(secret, userID, email, name, role string) (string, error) {
	now := time.Now()
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		Email: email,
		Name:  name,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			ID:        uuid.NewString(),
		},
	})
	return t.SignedString([]byte(secret))
}

func SignRefresh() (raw string, hash string, exp time.Time) {
	raw = uuid.NewString() + uuid.NewString()
	sum := uuid.NewSHA1(uuid.NameSpaceOID, []byte(raw))
	return raw, sum.String(), time.Now().Add(7 * 24 * time.Hour)
}

func HashRefresh(raw string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(raw)).String()
}

func ParseAccess(secret, token string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	c, ok := t.Claims.(*Claims)
	if !ok || !t.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return c, nil
}
