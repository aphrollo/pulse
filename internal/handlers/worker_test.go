package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
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
	app.Post("/worker/update", WorkerUpdateHandler)
	app.Post("/worker/heartbeat", WorkerHeartbeatHandler)
	return app
}

func TestWorkerHandler_Success(t *testing.T) {
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
		t.Fatalf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	// Verify the record was inserted in DB
	ctx := context.Background()
	var id uuid.UUID
	var name, wtype string
	err = db.Pool.QueryRow(ctx,
		`SELECT id, name, type FROM workers WHERE id = $1`, payload.ID,
	).Scan(&id, &name, &wtype)
	if err != nil {
		t.Fatalf("Failed to query inserted worker: %v", err)
	}
	if id.String() != payload.ID || name != payload.Name || wtype != payload.Type {
		t.Errorf("DB record does not match payload")
	}

	TestWorkerUpdateHandler_Success(t)
	TestWorkerHeartbeatHandler_Success(t)

	// Cleanup test data after assertion
	_, err = db.Pool.Exec(ctx, `DELETE FROM workers WHERE id = $1`, payload.ID)
	if err != nil {
		t.Logf("Failed to cleanup test data: %v", err)
	}
}

// Worker Register Handler
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

// Worker Update Handler
func TestWorkerUpdateHandler_Success(t *testing.T) {
	app := setupApp(t) // uses real DB connection

	payload := WorkerUpdateRequest{
		ID:      "12344567-e89b-12d3-a456-426614174000",
		Status:  "healthy",
		Message: "all systems go",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/worker/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error on test request: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	// Verify insert into worker_updates table
	ctx := context.Background()
	var workerID uuid.UUID
	var status, message string

	err = db.Pool.QueryRow(ctx,
		`SELECT worker_id, status, message FROM worker_updates WHERE worker_id = $1 ORDER BY time DESC LIMIT 1`,
		payload.ID,
	).Scan(&workerID, &status, &message)
	if err != nil {
		t.Fatalf("Failed to query inserted worker update: %v", err)
	}

	if workerID.String() != payload.ID {
		t.Errorf("Expected worker_id %s, got %s", payload.ID, workerID.String())
	}
	if status != payload.Status {
		t.Errorf("Expected status %s, got %s", payload.Status, status)
	}
	if message != payload.Message {
		t.Errorf("Expected message %q, got %q", payload.Message, message)
	}

	// Cleanup test data
	_, err = db.Pool.Exec(ctx, `DELETE FROM worker_updates WHERE worker_id = $1 AND message = $2`, payload.ID, payload.Message)
	if err != nil {
		t.Logf("Cleanup failed: %v", err)
	}
}

func TestWorkerUpdateHandler_InvalidUUID(t *testing.T) {
	app := setupApp(t)

	body := `{"id":"not-a-uuid","status":"active","message":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/worker/update", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for invalid UUID, got %d", resp.StatusCode)
	}
}

func TestWorkerUpdateHandler_InvalidJSON(t *testing.T) {
	app := setupApp(t)

	body := `{"id":"123e4567-e89b-12d3-a456-426614174000", "status":"active",` // malformed JSON
	req := httptest.NewRequest(http.MethodPost, "/worker/update", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for invalid JSON, got %d", resp.StatusCode)
	}
}

// Worker Heartbeat Handler
func TestWorkerHeartbeatHandler_Success(t *testing.T) {
	app := setupApp(t) // uses real DB connection

	payload := WorkerHeartbeatRequest{
		ID:     "12344567-e89b-12d3-a456-426614174000",
		Status: "healthy",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/worker/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error on test request: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	// Verify insert into worker_heartbeats table
	ctx := context.Background()
	var workerID uuid.UUID
	var status string

	err = db.Pool.QueryRow(ctx,
		`SELECT worker_id, status FROM worker_heartbeats WHERE worker_id = $1 ORDER BY time DESC LIMIT 1`,
		payload.ID,
	).Scan(&workerID, &status)
	if err != nil {
		t.Fatalf("Failed to query inserted heartbeat: %v", err)
	}

	if workerID.String() != payload.ID {
		t.Errorf("Expected worker_id %s, got %s", payload.ID, workerID.String())
	}
	if status != payload.Status {
		t.Errorf("Expected status %s, got %s", payload.Status, status)
	}

	// Cleanup test data
	_, err = db.Pool.Exec(ctx, `DELETE FROM worker_heartbeats WHERE worker_id = $1 AND status = $2`, payload.ID, payload.Status)
	if err != nil {
		t.Logf("Cleanup failed: %v", err)
	}
}
