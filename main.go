package main

import (
	"fmt"
	"os"
	"time"

	"gradspaceBK/config"
	"gradspaceBK/controller"
	"gradspaceBK/database"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	if len(os.Args) < 2 {
		panic("No command provided")
	}
	config.LoadConfig()
	arg := os.Args[1]
	switch arg {
	case "migrate":
		Migrate()
	case "runserver":
		RunServer()
	default:
		panic("Invalid command")
	}
}

func Migrate() {
	database.DBConnection()
	session := database.Session.Db
	fmt.Println("Connected to database")
	database.MigrateDB(session)
}

func RunServer() {
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
	app.Static("/api/v1/uploads", "./uploads") // Makes 'uploads' folder accessible via URLs
	app.Use(logger.New())
	controller.SetupRouter(app)
	app.Listen(":8003")
}
