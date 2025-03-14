package controller

import (
	"gradspaceBK/controller/admin"
	"gradspaceBK/controller/user"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func SetupRouter(app *fiber.App) {
	base_api := app.Group("/api/v1")

	base_api.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			allowedOrigins := map[string]bool{
				"https://feature-user.gradspace-frontend.pages.dev": true,
				"http://localhost:5173":                             true,
				"https://www.gradspace.me":                          true,
				"https://gradspace.me":                              true,
			}
			return allowedOrigins[origin]
		},
		AllowMethods: strings.Join([]string{
			fiber.MethodGet,
			fiber.MethodPost,
			fiber.MethodPut,
			fiber.MethodDelete,
			fiber.MethodPatch,
			fiber.MethodOptions,
		}, ","),
		// AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		// AllowCredentials: true,
		// ExposeHeaders:    "Set-Cookie",

		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,Upgrade,Connection", // Add WebSocket headers
		AllowCredentials: true,
		ExposeHeaders:    "Set-Cookie,Upgrade,Connection", // Add WebSocket headers

		// belwo is already existing comment.
		// Critical for cookie-based auth
		// AllowPrivateNetwork: true, // For local network access removed as field not supported
	}))

	AuthRoutes(base_api.(*fiber.Group))
	RegisterRoutes(base_api.(*fiber.Group))
	OnboardRoutes(base_api.(*fiber.Group))
	user.RegisterMessageRoutes(base_api.(*fiber.Group))
	admin.AdminUserManagementRoutes(base_api.(*fiber.Group))
	admin.RegisterAnalyticsRoutes(base_api.(*fiber.Group))
	user.RegisterProfileRoutes(base_api.(*fiber.Group))
	user.PostRoutes(base_api.(*fiber.Group))
	user.NotificationRoutes(base_api.(*fiber.Group))
	user.JobRoutes(base_api.(*fiber.Group))
	user.CompanyRoutes(base_api.(*fiber.Group))
	user.EventRoutes(base_api.(*fiber.Group))
	user.ProjectRoutes(base_api.(*fiber.Group))
	user.ConnectRoutes(base_api.(*fiber.Group))
}
