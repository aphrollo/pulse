package main

import (
	"time"

	"github.com/aphrollo/pulse/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	// Static resources; images, js, css
	app.Static("/", "./static", fiber.Static{
		Compress:      true,
		ByteRange:     true,
		Browse:        false,
		Download:      false,
		CacheDuration: 5 * time.Minute,
		MaxAge:        86400,
		ModifyResponse: func(c *fiber.Ctx) error {
			c.Set("Cache-Control", "public, max-age=86400")
			return nil
		},
	})

	app.Get("/", handlers.DashboardHandler)

	// Favicon
	//app.Use(favicon.New(favicon.Config{
	//	File: "./static/images/favicon.ico",
	//	URL:  "/favicon.ico",
	//}))

	if err := app.Listen(":3000"); err != nil {
		panic(err)
	}
}
