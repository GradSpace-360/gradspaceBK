package controller

import (
	"gradspaceBK/controller/admin"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func SetupRouter(app *fiber.App) {
	base_api := app.Group("/api/v1")
	base_api.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:5173",
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
	}))
	AuthRoutes(base_api.(*fiber.Group))
	RegisterRoutes(base_api.(*fiber.Group))
	OnboardRoutes(base_api.(*fiber.Group))
	admin.AdminUserManagementRoutes(base_api.(*fiber.Group))
	admin.RegisterAnalyticsRoutes(base_api.(*fiber.Group)) 
}
