package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/joho/godotenv"

	db "github.com/aphrollo/pulse/internal/storage"
)

// helper to setup app and DB once per test
func setupApp(t *testing.T) *fiber.App {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("Warning: .env file not found or failed to load")
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}
	t.Cleanup(db.Close)

	app := fiber.New()
	app.Post("/worker/register", WorkerRegisterHandler)
	return app
}

func TestWorkerRegisterHandler_Success(t *testing.T) {
	app := setupApp(t)

	payload := WorkerRegisterRequest{
		ID:   "12344567-e89b-12d3-a456-426614174000",
		Name: "test-worker",
		Type: "bot",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/worker/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error on test request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}
}

func TestWorkerRegisterHandler_InvalidUUID(t *testing.T) {
	app := setupApp(t)

	body := `{"id":"not-a-uuid","name":"test","type":"bot"}`
	req := httptest.NewRequest(http.MethodPost, "/worker/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestWorkerRegisterHandler_EmptyName(t *testing.T) {
	app := setupApp(t)

	body := `{"id":"123e4567-e89b-12d3-a456-426614174000","name":"","type":"bot"}`
	req := httptest.NewRequest(http.MethodPost, "/worker/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request due to empty name, got %d", resp.StatusCode)
	}
}

func TestWorkerRegisterHandler_InvalidJSON(t *testing.T) {
	app := setupApp(t)

	body := `{"id": "123e4567-e89b-12d3-a456-426614174000", "name": "test-worker",` // malformed JSON
	req := httptest.NewRequest(http.MethodPost, "/worker/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request due to invalid JSON, got %d", resp.StatusCode)
	}
}

// To test DB failure, temporarily replace db.Pool with a failing mock.
// Here’s an example of a minimal mock you can extend:

type mockPool struct{}

func (m *mockPool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	var tag pgconn.CommandTag // zero value
	return tag, fmt.Errorf("mock error")
}

func (m *mockPool) Close() {
	// nothing to do, mock close
}

func TestWorkerRegisterHandler_DBFailure(t *testing.T) {
	app := fiber.New()

	origPool := db.Pool
	db.Pool = &mockPool{}
	defer func() { db.Pool = origPool }()

	app.Post("/worker/register", WorkerRegisterHandler)

	payload := WorkerRegisterRequest{
		ID:   "123e4567-e89b-12d3-a456-426614174000",
		Name: "test-worker",
		Type: "bot",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/worker/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected 500 Internal Server Error on DB failure, got %d", resp.StatusCode)
	}
}
