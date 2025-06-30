package main

import (
	"github.com/aphrollo/pulse/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/", handlers.DashboardHandler)

	app.Static("/static", "./static") // For any static HTMX/js/css

	if err := app.Listen(":3000"); err != nil {
		panic(err)
	}
}
