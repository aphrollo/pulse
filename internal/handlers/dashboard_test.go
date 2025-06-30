package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

func TestDashboardHandler(t *testing.T) {
	app := fiber.New()

	// Register route with your handler
	app.Get("/", DashboardHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))

	// Optional: read and check body is non-empty (or contains some expected content)
	// body, _ := io.ReadAll(resp.Body)
	// require.Contains(t, string(body), "<html") // or something specific in your template
}
