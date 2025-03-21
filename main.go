package main

import (
	"fmt"
	"time"

	"gradspaceBK/config"
	"gradspaceBK/controller"
	"gradspaceBK/database"
	"gradspaceBK/ws"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	RunServer()
}

//need sql based migration
// func Migrate() {
// 	database.DBConnection()
// 	session := database.Session.Db
// 	fmt.Println("Connected to database")
// 	database.MigrateDB(session)
// }

func RunServer() {
	config.LoadConfig() // Load .env first
	database.DBConnection()

	// Start cleanup job
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		time.Sleep(1 * time.Minute)
		if err := database.CleanupOldNotifications(); err != nil {
			fmt.Printf("Initial notification cleanup failed: %v\n", err)
		}

		for range ticker.C {
			if err := database.CleanupOldNotifications(); err != nil {
				fmt.Printf("Notification cleanup error: %v\n", err)
			}
		}
	}()

	app := fiber.New()
	// Static files and logger
	app.Static("/api/v1/uploads", "./uploads")
	app.Use(logger.New())
	// **WebSocket setup (BEFORE other routes)**
	ws.SetupWebSocket(app)
	controller.SetupRouter(app)
	app.Listen(":8003")
}
