package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the JWT payload encoded in the access token.
type Claims struct {
	UserID string `json:"sub"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTManager handles signing and parsing of JWT tokens.
type JWTManager struct {
	secret      []byte
	expiry      time.Duration
	signingAlgo *jwt.SigningMethodHMAC
}

func NewJWTManager(secret string, expiryHours int) *JWTManager {
	return &JWTManager{
		secret:      []byte(secret),
		expiry:      time.Duration(expiryHours) * time.Hour,
		signingAlgo: jwt.SigningMethodHS256,
	}
}

// Generate creates a signed JWT for the given user.
func (m *JWTManager) Generate(u *User) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID: u.ID,
		Email:  u.Email,
		Role:   u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "sims-backend",
			Subject:   u.ID,
		},
	}
	token := jwt.NewWithClaims(m.signingAlgo, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// Parse validates a token string and returns its claims.
// Returns an error if the token is malformed, expired, or has an invalid signature.
func (m *JWTManager) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
