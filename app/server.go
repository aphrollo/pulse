package app

import (
	"time"

	"github.com/gofiber/contrib/swagger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"github.com/aphrollo/pulse/handlers"
)

func New() *fiber.App {
	app := fiber.New(fiber.Config{
		// Customize Fiber config here
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		// ErrorHandler, JSON encoder, etc can be added here
	})

	// Middlewares
	app.Use(logger.New())

	cfg := swagger.Config{
		BasePath: "/",
		FilePath: "./docs/swagger.json",
		Path:     "docs",
		Title:    "API Docs",
	}

	app.Use(swagger.New(cfg))

	// Static files
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

	// Routes
	app.Get("/", handlers.DashboardHandler)

	worker := app.Group("/worker")
	worker.Post("register", handlers.WorkerRegisterHandler)
	worker.Post("update", handlers.WorkerUpdateHandler)
	worker.Post("heartbeat", handlers.WorkerHeartbeatHandler)

	return app
}
