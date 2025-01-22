package main

import (
	"fmt"
	"os"

	"gradspaceBK/controller"
	"gradspaceBK/db"

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
	session, err := db.DBConnection()
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected to database")
	err = db.MigrateDB(session)
	if err != nil {
		panic(err)
	}
}

func RunServer() {
	app := fiber.New()
	controller.SetupRouter(app)
	app.Listen(":8003")
}
