package main

import (
	"fmt"
	"os"

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
	app := fiber.New()
	app.Use(logger.New())
	controller.SetupRouter(app)
	app.Listen(":8003")
}
