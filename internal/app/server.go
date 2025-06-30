package app

import (
	"time"

	"github.com/aphrollo/pulse/internal/handlers"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
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
	worker.Get("register", handlers.WorkerRegisterHandler)
	worker.Get("update", handlers.WorkerUpdateHandler)
	worker.Get("heartbeat", handlers.WorkerHeartbeatHandler)

	return app
}
