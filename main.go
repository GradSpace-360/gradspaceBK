package main

import (
	"fmt"
	"os"

	"gradspaceBK/controller"
	"gradspaceBK/database"

	"github.com/gofiber/fiber/v2"
)

func main() {
	if len(os.Args) < 2 {
		panic("No command provided")
	}
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
	session := database.Session.Db
	fmt.Println("Connected to database")
	database.MigrateDB(session)
}

func RunServer() {
	database.DBConnection()
	app := fiber.New()
	controller.SetupRouter(app)
	app.Listen(":8003")
}
