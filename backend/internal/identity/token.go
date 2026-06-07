package identity

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the JWT payload embedded in each access token.
type Claims struct {
	Email  string `json:"email"`
	ShopID string `json:"shop_id,omitempty"`
	Role   string `json:"role,omitempty"`
	jwt.RegisteredClaims
}

// SignToken creates a signed JWT for the given user and primary shop membership.
// Exported so tests outside this package can mint tokens for AuthRequired/handler tests.
func SignToken(secret string, expiry time.Duration, user User, shopID string, role Role) (string, error) {
	now := time.Now()
	claims := Claims{
		Email:  user.Email,
		ShopID: shopID,
		Role:   string(role),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// ParseToken validates a JWT string and returns its claims.
// Returns an error if the token is invalid, expired, or uses an unexpected algorithm.
func ParseToken(secret, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}
