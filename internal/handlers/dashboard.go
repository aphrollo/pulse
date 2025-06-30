package handlers

import (
	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"

	"github.com/aphrollo/pulse/internal/templates"
)

// DashboardHandler renders the main dashboard UI
// @Summary Dashboard view
// @Description Main Pulse dashboard displaying workers and their statuses
// @Tags Dashboard
// @Produce html
// @Success 200 {string} string "HTML content"
// @Router / [get]
func DashboardHandler(c *fiber.Ctx) error {
	return adaptor.HTTPHandler(
		templ.Handler(templates.Dashboard()),
	)(c)
}
