package http

import (
	v1 "diary-service/internal/controller/http/v1"

	"github.com/gofiber/fiber/v3"
)


func NewRouter(app *fiber.App, ) {
	apiV1Group := app.Group("/api/v1") 
	{
		v1.NewRoutes(apiV1Group)
	}
}