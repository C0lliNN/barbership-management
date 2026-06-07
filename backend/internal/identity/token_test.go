package identity_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gcollin65/barbershop/internal/identity"
)

const testTokenSecret = "test-secret"

func TestSignAndParseTokenRoundTrip(t *testing.T) {
	user := identity.User{ID: "user-1", Email: "joao@test.com"}

	token, err := identity.SignToken(testTokenSecret, time.Hour, user, "shop-1", identity.RoleOwner)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := identity.ParseToken(testTokenSecret, token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.Subject)
	assert.Equal(t, "joao@test.com", claims.Email)
	assert.Equal(t, "shop-1", claims.ShopID)
	assert.Equal(t, string(identity.RoleOwner), claims.Role)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
}

func TestParseTokenWrongSecret(t *testing.T) {
	user := identity.User{ID: "user-1", Email: "joao@test.com"}
	token, err := identity.SignToken(testTokenSecret, time.Hour, user, "", "")
	require.NoError(t, err)

	_, err = identity.ParseToken("a-different-secret", token)
	assert.Error(t, err)
}

func TestParseTokenExpired(t *testing.T) {
	user := identity.User{ID: "user-1", Email: "joao@test.com"}
	token, err := identity.SignToken(testTokenSecret, -time.Hour, user, "", "")
	require.NoError(t, err)

	_, err = identity.ParseToken(testTokenSecret, token)
	assert.Error(t, err)
}

func TestParseTokenTampered(t *testing.T) {
	user := identity.User{ID: "user-1", Email: "joao@test.com"}
	token, err := identity.SignToken(testTokenSecret, time.Hour, user, "", "")
	require.NoError(t, err)

	tampered := token[:len(token)-1] + "x"
	_, err = identity.ParseToken(testTokenSecret, tampered)
	assert.Error(t, err)
}

func TestParseTokenWrongAlgorithm(t *testing.T) {
	claims := identity.Claims{
		Email: "joao@test.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	// Sign with the "none" algorithm — ParseToken must reject anything that
	// isn't HMAC, even if the rest of the claims look valid.
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	signed, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = identity.ParseToken(testTokenSecret, signed)
	assert.Error(t, err)
}
