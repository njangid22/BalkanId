package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims describes the JWT session payload stored in the session cookie.
type Claims struct {
	UserID string `json:"uid"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTManager encapsulates signing and validation helpers for session tokens.
type JWTManager struct {
	secret []byte
	ttl    time.Duration
}

func NewJWTManager(secret string, ttl time.Duration) *JWTManager {
	return &JWTManager{secret: []byte(secret), ttl: ttl}
}

// Sign produces a compact JWT representing the provided claims.
func (m *JWTManager) Sign(now time.Time, userID, email, name, role string) (string, *Claims, error) {
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Name:   name,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	return signed, claims, err
}

// Parse validates the token and returns the embedded claims.
func (m *JWTManager) Parse(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, errors.New("empty token")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
