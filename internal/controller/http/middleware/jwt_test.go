package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestJWTHandler(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	actorID := uuid.New()
	config := JWTConfig{Issuer: "auth-service", Audience: "diary-service", ClockSkew: time.Second}
	middleware, err := NewJWTWithKeyfunc(config, func(_ *jwt.Token) (any, error) {
		return &privateKey.PublicKey, nil
	}, nil)
	require.NoError(t, err)

	app := fiber.New()
	app.Use(middleware.Handler)
	app.Get("/protected", func(c fiber.Ctx) error {
		actualActorID, actorErr := ActorID(c)
		require.NoError(t, actorErr)
		return c.JSON(fiber.Map{"actor_id": actualActorID})
	})

	t.Run("accepts valid token and stores subject", func(t *testing.T) {
		token := signedToken(t, privateKey, actorID, time.Now().Add(time.Minute), "auth-service", "diary-service")
		request := httptest.NewRequest("GET", "/protected", nil)
		request.Header.Set("Authorization", "Bearer "+token)

		response, requestErr := app.Test(request)
		require.NoError(t, requestErr)
		require.Equal(t, fiber.StatusOK, response.StatusCode)
	})

	t.Run("rejects expired token", func(t *testing.T) {
		token := signedToken(t, privateKey, actorID, time.Now().Add(-time.Minute), "auth-service", "diary-service")
		request := httptest.NewRequest("GET", "/protected", nil)
		request.Header.Set("Authorization", "Bearer "+token)

		response, requestErr := app.Test(request)
		require.NoError(t, requestErr)
		require.Equal(t, fiber.StatusUnauthorized, response.StatusCode)
	})

	t.Run("rejects missing bearer token", func(t *testing.T) {
		request := httptest.NewRequest("GET", "/protected", nil)

		response, requestErr := app.Test(request)
		require.NoError(t, requestErr)
		require.Equal(t, fiber.StatusUnauthorized, response.StatusCode)
	})
}

func signedToken(t *testing.T, privateKey *rsa.PrivateKey, actorID uuid.UUID, expiresAt time.Time, issuer, audience string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims{
		Role: "user",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: actorID.String(), Issuer: issuer, Audience: jwt.ClaimStrings{audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt), NotBefore: jwt.NewNumericDate(time.Now().Add(-time.Second)),
		},
	})
	signed, err := token.SignedString(privateKey)
	require.NoError(t, err)
	return signed
}
