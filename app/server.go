package app

import (
	"os"
	"strings"
	"time"

	"github.com/gofiber/contrib/swagger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"github.com/aphrollo/pulse/handlers"
)

func New() *fiber.App {
	typesStr := os.Getenv("ALLOWED_AGENT_TYPES")
	if typesStr == "" {
		// default fallback
		handlers.AllowedAgentTypes = []string{"default"}
	} else {
		handlers.AllowedAgentTypes = strings.Split(typesStr, ",")
		for i := range handlers.AllowedAgentTypes {
			handlers.AllowedAgentTypes[i] = strings.TrimSpace(handlers.AllowedAgentTypes[i])
		}
	}

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

	client := app.Group("/agent")
	client.Post("register", handlers.AgentRegisterHandler)
	client.Post("update", handlers.AgentUpdateHandler)
	client.Post("heartbeat", handlers.AgentHeartbeatHandler)

	return app
}
