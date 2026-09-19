package middleware

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"diary-service/pkg/logger"

	jwks "github.com/MicahParks/keyfunc/v3"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	actorIDLocal = "authenticated_actor_id"
	roleLocal    = "authenticated_actor_role"
)

var ErrUnauthenticated = errors.New("unauthenticated")

type JWTConfig struct {
	JWKSURL   string
	Issuer    string
	Audience  string
	ClockSkew time.Duration
}

type JWT struct {
	keyfunc  jwt.Keyfunc
	issuer   string
	audience string
	leeway   time.Duration
	logger   logger.Interface
}

type claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func NewJWT(ctx context.Context, cfg JWTConfig, log logger.Interface) (*JWT, error) {
	if strings.TrimSpace(cfg.JWKSURL) == "" {
		return nil, errors.New("JWKS URL is required")
	}
	keys, err := jwks.NewDefaultOverrideCtx(ctx, []string{cfg.JWKSURL}, jwks.Override{
		HTTPTimeout: 10 * time.Second,
		RefreshErrorHandlerFunc: func(_ string) func(context.Context, error) {
			return func(_ context.Context, refreshErr error) {
				if log != nil {
					log.ErrorDetails("http.middleware", "JWT.JWKSRefresh", "refresh JWKS", refreshErr)
				}
			}
		},
	})
	if err != nil {
		if log != nil {
			log.ErrorDetails("http.middleware", "NewJWT", "initialize JWKS key function", err)
		}
		return nil, fmt.Errorf("initialize JWKS: %w", err)
	}
	return NewJWTWithKeyfunc(cfg, keys.Keyfunc, log)
}

func NewJWTWithKeyfunc(cfg JWTConfig, keyfunc jwt.Keyfunc, log logger.Interface) (*JWT, error) {
	if keyfunc == nil {
		return nil, errors.New("JWT key function is required")
	}
	if strings.TrimSpace(cfg.Issuer) == "" {
		return nil, errors.New("JWT issuer is required")
	}
	if strings.TrimSpace(cfg.Audience) == "" {
		return nil, errors.New("JWT audience is required")
	}
	return &JWT{keyfunc: keyfunc, issuer: cfg.Issuer, audience: cfg.Audience, leeway: cfg.ClockSkew, logger: log}, nil
}

func (m *JWT) Handler(c fiber.Ctx) error {
	header := strings.TrimSpace(c.Get(fiber.HeaderAuthorization))
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return m.unauthorized(c, "read bearer token", ErrUnauthenticated)
	}

	parsedClaims := &claims{}
	token, err := jwt.ParseWithClaims(
		parts[1],
		parsedClaims,
		m.keyfunc,
		jwt.WithIssuer(m.issuer),
		jwt.WithAudience(m.audience),
		jwt.WithLeeway(m.leeway),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{
			jwt.SigningMethodRS256.Alg(), jwt.SigningMethodRS384.Alg(), jwt.SigningMethodRS512.Alg(),
			jwt.SigningMethodPS256.Alg(), jwt.SigningMethodPS384.Alg(), jwt.SigningMethodPS512.Alg(),
			jwt.SigningMethodES256.Alg(), jwt.SigningMethodES384.Alg(), jwt.SigningMethodES512.Alg(),
			jwt.SigningMethodEdDSA.Alg(),
		}),
	)
	if err != nil || !token.Valid {
		if err == nil {
			err = ErrUnauthenticated
		}
		return m.unauthorized(c, "validate JWT", err)
	}

	actorID, err := uuid.Parse(parsedClaims.Subject)
	if err != nil || actorID == uuid.Nil {
		if err == nil {
			err = errors.New("subject is empty")
		}
		return m.unauthorized(c, "parse JWT subject", err)
	}
	if parsedClaims.Role != "user" && parsedClaims.Role != "admin" {
		return m.unauthorized(c, "validate JWT role", errors.New("unsupported role"))
	}

	c.Locals(actorIDLocal, actorID)
	c.Locals(roleLocal, parsedClaims.Role)
	return c.Next()
}

func (m *JWT) unauthorized(c fiber.Ctx, cause string, err error) error {
	if m.logger != nil {
		m.logger.ErrorDetails("http.middleware", "JWT.Handler", cause, err)
	}
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error": fiber.Map{"code": "unauthorized", "message": "authentication required"},
	})
}

func ActorID(c fiber.Ctx) (uuid.UUID, error) {
	actorID, ok := c.Locals(actorIDLocal).(uuid.UUID)
	if !ok || actorID == uuid.Nil {
		return uuid.Nil, ErrUnauthenticated
	}
	return actorID, nil
}

func Role(c fiber.Ctx) (string, error) {
	role, ok := c.Locals(roleLocal).(string)
	if !ok || role == "" {
		return "", ErrUnauthenticated
	}
	return role, nil
}

// SetActor is intended for trusted middleware composition and transport tests.
func SetActor(c fiber.Ctx, actorID uuid.UUID, role string) {
	c.Locals(actorIDLocal, actorID)
	c.Locals(roleLocal, role)
}
