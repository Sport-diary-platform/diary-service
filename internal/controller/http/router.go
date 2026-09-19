package http

import (
	v1 "diary-service/internal/controller/http/v1"

	"github.com/gofiber/fiber/v3"
)

func NewRouter(app *fiber.App, handler *v1.V1, auth fiber.Handler) {
	app.Get("/health/live", func(c fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	apiV1Group := app.Group("/api/v1", auth)
	v1.NewRoutes(apiV1Group, handler)
}
