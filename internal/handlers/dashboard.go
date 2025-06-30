package handlers

import (
	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"

	"github.com/aphrollo/pulse/internal/templates"
)

func DashboardHandler(c *fiber.Ctx) error {
	return adaptor.HTTPHandler(
		templ.Handler(templates.Dashboard()),
	)(c)
}
