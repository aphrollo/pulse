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

func setupApp(t *testing.T) *fiber.App {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("Warning: .env file not found or failed to load")
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}

	app := fiber.New()
	app.Post("/worker/register", WorkerRegisterHandler)
	app.Post("/worker/update", WorkerUpdateHandler)
	app.Post("/worker/heartbeat", WorkerHeartbeatHandler)
	return app
}

func TestWorkerHandler(t *testing.T) {
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

	t.Run("WorkerRegisterInvalidUUID", TestWorkerRegisterHandler_InvalidUUID)
	t.Run("WorkerRegisterEmptyName", TestWorkerRegisterHandler_EmptyName)
	t.Run("WorkerRegisterMissingName", TestWorkerRegisterHandler_MissingName)
	t.Run("WorkerRegisterInvalidJSON", TestWorkerRegisterHandler_InvalidJSON)
	t.Run("WorkerRegisterInvalidType", TestWorkerRegisterHandler_InvalidType)

	t.Run("WorkerUpdate", TestWorkerUpdateHandler_InvalidUUID)
	t.Run("WorkerUpdate", TestWorkerUpdateHandler_InvalidJSON)
	t.Run("WorkerUpdate", TestWorkerUpdateHandler_InvalidStatus)
	t.Run("WorkerUpdate", TestWorkerUpdateHandler_MissingStatus)
	t.Run("WorkerUpdate", TestWorkerUpdateHandler_Success)

	t.Run("WorkerHeartbeat", TestWorkerHeartbeatHandler_InvalidJSON)
	t.Run("WorkerHeartbeat", TestWorkerHeartbeatHandler_InvalidUUID)
	t.Run("WorkerHeartbeat", TestWorkerHeartbeatHandler_MissingFields)
	t.Run("WorkerHeartbeat", TestWorkerHeartbeatHandler_InvalidStatus)
	t.Run("WorkerHeartbeat", TestWorkerHeartbeatHandler_EmptyStatus)
	t.Run("WorkerHeartbeat", TestWorkerHeartbeatHandler_Success)

	// Cleanup test data after assertion
	_, err = db.Pool.Exec(ctx, `DELETE FROM workers WHERE id = $1`, payload.ID)
	if err != nil {
		t.Logf("Failed to cleanup test data: %v", err)
	}
	t.Cleanup(db.Close)
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

func TestWorkerRegisterHandler_MissingName(t *testing.T) {
	app := setupApp(t)

	body := `{"id":"123e4567-e89b-12d3-a456-426614174000","type":"bot"}`
	req := httptest.NewRequest(http.MethodPost, "/worker/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request due to missing name, got %d", resp.StatusCode)
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

func TestWorkerRegisterHandler_InvalidType(t *testing.T) {
	app := setupApp(t)

	body := `{"id":"123e4567-e89b-12d3-a456-426614174000","name":"test-worker","type":"invalid-type"}`
	req := httptest.NewRequest(http.MethodPost, "/worker/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	// assuming your handler validates 'Type' and rejects invalid ones
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request due to invalid type, got %d", resp.StatusCode)
	}
}

func TestWorkerRegisterHandler_MissingType(t *testing.T) {
	app := setupApp(t)

	body := `{"id":"123e4567-e89b-12d3-a456-426614174000","name":"test-worker"}`
	req := httptest.NewRequest(http.MethodPost, "/worker/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request due to missing type, got %d", resp.StatusCode)
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

func TestWorkerUpdateHandler_InvalidStatus(t *testing.T) {
	app := setupApp(t)

	body := `{"id":"123e4567-e89b-12d3-a456-426614174000","status":"invalid_status","message":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/worker/update", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest && resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected 400 Bad Request or 500 Internal Server Error for invalid status, got %d", resp.StatusCode)
	}
}

func TestWorkerUpdateHandler_MissingStatus(t *testing.T) {
	app := setupApp(t)

	body := `{"id":"123e4567-e89b-12d3-a456-426614174000","message":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/worker/update", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	// Depends if your handler requires status or allows optional status updates
	if resp.StatusCode != fiber.StatusOK && resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 200 OK or 400 Bad Request for missing status, got %d", resp.StatusCode)
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

func TestWorkerHeartbeatHandler_InvalidJSON(t *testing.T) {
	app := setupApp(t)

	body := `{"id": "123e4567-e89b-12d3-a456-426614174000", "status":` // malformed JSON
	req := httptest.NewRequest(http.MethodPost, "/worker/heartbeat", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error on test request: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for malformed JSON, got %d", resp.StatusCode)
	}
}

func TestWorkerHeartbeatHandler_InvalidUUID(t *testing.T) {
	app := setupApp(t)

	body := `{"id": "invalid-uuid", "status": "healthy"}`
	req := httptest.NewRequest(http.MethodPost, "/worker/heartbeat", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error on test request: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for invalid UUID, got %d", resp.StatusCode)
	}
}

func TestWorkerHeartbeatHandler_MissingFields(t *testing.T) {
	app := setupApp(t)

	// Missing status field
	body := `{"id": "123e4567-e89b-12d3-a456-426614174000"}`
	req := httptest.NewRequest(http.MethodPost, "/worker/heartbeat", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error on test request: %v", err)
	}
	// If your handler doesn't explicitly check for missing fields, it might succeed. Adjust as needed.
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for missing fields, got %d", resp.StatusCode)
	}
}

func TestWorkerHeartbeatHandler_InvalidStatus(t *testing.T) {
	app := setupApp(t)

	body := `{"id": "123e4567-e89b-12d3-a456-426614174000", "status": "invalid_status"}`
	req := httptest.NewRequest(http.MethodPost, "/worker/heartbeat", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error on test request: %v", err)
	}
	// If your handler validates the status against enum and rejects invalid ones:
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for invalid status, got %d", resp.StatusCode)
	}
}

func TestWorkerHeartbeatHandler_EmptyStatus(t *testing.T) {
	app := setupApp(t)

	body := `{"id": "123e4567-e89b-12d3-a456-426614174000", "status": ""}`
	req := httptest.NewRequest(http.MethodPost, "/worker/heartbeat", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error on test request: %v", err)
	}

	// Adjust expected behavior depending on whether empty status is valid
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for empty status, got %d", resp.StatusCode)
	}
}
