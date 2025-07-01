package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	db "github.com/aphrollo/pulse/storage"
)

func loadEnvFromRoot() {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get working dir: %v", err)
	}

	for {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			err = godotenv.Load(envPath)
			if err != nil {
				log.Printf("Failed to load .env from %s: %v", envPath, err)
			}
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			log.Println("Warning: .env file not found in any parent directory")
			return
		}
		dir = parent
	}
}

func setupApp(t *testing.T) *fiber.App {
	loadEnvFromRoot()
	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}
	app := fiber.New()
	app.Post("/agent/register", AgentRegisterHandler)
	app.Post("/agent/update", AgentUpdateHandler)
	app.Post("/agent/heartbeat", AgentHeartbeatHandler)
	return app
}

func TestAgentHandler(t *testing.T) {
	app := setupApp(t)

	payload := AgentRegisterRequest{
		ID:   "12344567-e89b-12d3-a456-426614174000",
		Name: "test-Agent",
		Type: "default",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/agent/register", bytes.NewReader(body))
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
		`SELECT id, name, type FROM Agents WHERE id = $1`, payload.ID,
	).Scan(&id, &name, &wtype)
	if err != nil {
		t.Fatalf("Failed to query inserted Agent: %v", err)
	}
	if id.String() != payload.ID || name != payload.Name || wtype != payload.Type {
		t.Errorf("DB record does not match payload")
	}

	t.Run("AgentRegisterInvalidUUID", TestAgentRegisterHandler_InvalidUUID)
	t.Run("AgentRegisterEmptyName", TestAgentRegisterHandler_EmptyName)
	t.Run("AgentRegisterMissingName", TestAgentRegisterHandler_MissingName)
	t.Run("AgentRegisterInvalidJSON", TestAgentRegisterHandler_InvalidJSON)
	t.Run("AgentRegisterInvalidType", TestAgentRegisterHandler_InvalidType)

	t.Run("AgentUpdate", TestAgentUpdateHandler_InvalidUUID)
	t.Run("AgentUpdate", TestAgentUpdateHandler_InvalidJSON)
	t.Run("AgentUpdate", TestAgentUpdateHandler_InvalidStatus)
	t.Run("AgentUpdate", TestAgentUpdateHandler_MissingStatus)
	t.Run("AgentUpdate", TestAgentUpdateHandler_Success)

	t.Run("AgentHeartbeat", TestAgentHeartbeatHandler_InvalidJSON)
	t.Run("AgentHeartbeat", TestAgentHeartbeatHandler_InvalidUUID)
	t.Run("AgentHeartbeat", TestAgentHeartbeatHandler_MissingFields)
	t.Run("AgentHeartbeat", TestAgentHeartbeatHandler_InvalidStatus)
	t.Run("AgentHeartbeat", TestAgentHeartbeatHandler_EmptyStatus)
	t.Run("AgentHeartbeat", TestAgentHeartbeatHandler_Success)

	// Cleanup test data after assertion
	_, err = db.Pool.Exec(ctx, `DELETE FROM Agents WHERE id = $1`, payload.ID)
	if err != nil {
		t.Logf("Failed to cleanup test data: %v", err)
	}
	t.Cleanup(db.Close)
}

// Agent Register Handler
func TestAgentRegisterHandler_InvalidUUID(t *testing.T) {
	app := setupApp(t)

	body := `{"id":"not-a-uuid","name":"test","type":"bot"}`
	req := httptest.NewRequest(http.MethodPost, "/agent/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestAgentRegisterHandler_EmptyName(t *testing.T) {
	app := setupApp(t)

	body := `{"id":"123e4567-e89b-12d3-a456-426614174000","name":"","type":"bot"}`
	req := httptest.NewRequest(http.MethodPost, "/agent/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request due to empty name, got %d", resp.StatusCode)
	}
}

func TestAgentRegisterHandler_MissingName(t *testing.T) {
	app := setupApp(t)

	body := `{"id":"123e4567-e89b-12d3-a456-426614174000","type":"bot"}`
	req := httptest.NewRequest(http.MethodPost, "/agent/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request due to missing name, got %d", resp.StatusCode)
	}
}

func TestAgentRegisterHandler_InvalidJSON(t *testing.T) {
	app := setupApp(t)

	body := `{"id": "123e4567-e89b-12d3-a456-426614174000", "name": "test-Agent",` // malformed JSON
	req := httptest.NewRequest(http.MethodPost, "/agent/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request due to invalid JSON, got %d", resp.StatusCode)
	}
}

func TestAgentRegisterHandler_InvalidType(t *testing.T) {
	app := setupApp(t)

	body := `{"id":"123e4567-e89b-12d3-a456-426614174000","name":"test-Agent","type":"invalid-type"}`
	req := httptest.NewRequest(http.MethodPost, "/agent/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	// assuming your handler validates 'Type' and rejects invalid ones
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request due to invalid type, got %d", resp.StatusCode)
	}
}

func TestAgentRegisterHandler_MissingType(t *testing.T) {
	app := setupApp(t)

	body := `{"id":"123e4567-e89b-12d3-a456-426614174000","name":"test-Agent"}`
	req := httptest.NewRequest(http.MethodPost, "/agent/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request due to missing type, got %d", resp.StatusCode)
	}
}

// Agent Update Handler
func TestAgentUpdateHandler_Success(t *testing.T) {
	app := setupApp(t) // uses real DB connection

	payload := AgentUpdateRequest{
		ID:      "12344567-e89b-12d3-a456-426614174000",
		Status:  "healthy",
		Message: "all systems go",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/agent/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error on test request: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	// Verify insert into Agent_updates table
	ctx := context.Background()
	var AgentID uuid.UUID
	var status, message string

	err = db.Pool.QueryRow(ctx,
		`SELECT Agent_id, status, message FROM Agent_updates WHERE Agent_id = $1 ORDER BY time DESC LIMIT 1`,
		payload.ID,
	).Scan(&AgentID, &status, &message)
	if err != nil {
		t.Fatalf("Failed to query inserted Agent update: %v", err)
	}

	if AgentID.String() != payload.ID {
		t.Errorf("Expected Agent_id %s, got %s", payload.ID, AgentID.String())
	}
	if status != payload.Status {
		t.Errorf("Expected status %s, got %s", payload.Status, status)
	}
	if message != payload.Message {
		t.Errorf("Expected message %q, got %q", payload.Message, message)
	}

	// Cleanup test data
	_, err = db.Pool.Exec(ctx, `DELETE FROM Agent_updates WHERE Agent_id = $1 AND message = $2`, payload.ID, payload.Message)
	if err != nil {
		t.Logf("Cleanup failed: %v", err)
	}
}

func TestAgentUpdateHandler_InvalidUUID(t *testing.T) {
	app := setupApp(t)

	body := `{"id":"not-a-uuid","status":"active","message":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/agent/update", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for invalid UUID, got %d", resp.StatusCode)
	}
}

func TestAgentUpdateHandler_InvalidJSON(t *testing.T) {
	app := setupApp(t)

	body := `{"id":"123e4567-e89b-12d3-a456-426614174000", "status":"active",` // malformed JSON
	req := httptest.NewRequest(http.MethodPost, "/agent/update", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for invalid JSON, got %d", resp.StatusCode)
	}
}

func TestAgentUpdateHandler_InvalidStatus(t *testing.T) {
	app := setupApp(t)

	body := `{"id":"123e4567-e89b-12d3-a456-426614174000","status":"invalid_status","message":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/agent/update", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest && resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected 400 Bad Request or 500 Internal Server Error for invalid status, got %d", resp.StatusCode)
	}
}

func TestAgentUpdateHandler_MissingStatus(t *testing.T) {
	app := setupApp(t)

	body := `{"id":"123e4567-e89b-12d3-a456-426614174000","message":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/agent/update", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	// Depends on if your handler requires status or allows optional status updates
	if resp.StatusCode != fiber.StatusOK && resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 200 OK or 400 Bad Request for missing status, got %d", resp.StatusCode)
	}
}

// Agent Heartbeat Handler
func TestAgentHeartbeatHandler_Success(t *testing.T) {
	app := setupApp(t) // uses real DB connection

	payload := AgentHeartbeatRequest{
		ID:     "12344567-e89b-12d3-a456-426614174000",
		Status: "healthy",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/agent/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error on test request: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	// Verify insert into Agent_heartbeats table
	ctx := context.Background()
	var AgentID uuid.UUID
	var status string

	err = db.Pool.QueryRow(ctx,
		`SELECT Agent_id, status FROM Agent_heartbeats WHERE Agent_id = $1 ORDER BY time DESC LIMIT 1`,
		payload.ID,
	).Scan(&AgentID, &status)
	if err != nil {
		t.Fatalf("Failed to query inserted heartbeat: %v", err)
	}

	if AgentID.String() != payload.ID {
		t.Errorf("Expected Agent_id %s, got %s", payload.ID, AgentID.String())
	}
	if status != payload.Status {
		t.Errorf("Expected status %s, got %s", payload.Status, status)
	}

	// Cleanup test data
	_, err = db.Pool.Exec(ctx, `DELETE FROM Agent_heartbeats WHERE Agent_id = $1 AND status = $2`, payload.ID, payload.Status)
	if err != nil {
		t.Logf("Cleanup failed: %v", err)
	}
}

func TestAgentHeartbeatHandler_InvalidJSON(t *testing.T) {
	app := setupApp(t)

	body := `{"id": "123e4567-e89b-12d3-a456-426614174000", "status":` // malformed JSON
	req := httptest.NewRequest(http.MethodPost, "/agent/heartbeat", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error on test request: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for malformed JSON, got %d", resp.StatusCode)
	}
}

func TestAgentHeartbeatHandler_InvalidUUID(t *testing.T) {
	app := setupApp(t)

	body := `{"id": "invalid-uuid", "status": "healthy"}`
	req := httptest.NewRequest(http.MethodPost, "/agent/heartbeat", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error on test request: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for invalid UUID, got %d", resp.StatusCode)
	}
}

func TestAgentHeartbeatHandler_MissingFields(t *testing.T) {
	app := setupApp(t)

	// Missing status field
	body := `{"id": "123e4567-e89b-12d3-a456-426614174000"}`
	req := httptest.NewRequest(http.MethodPost, "/agent/heartbeat", bytes.NewBufferString(body))
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

func TestAgentHeartbeatHandler_InvalidStatus(t *testing.T) {
	app := setupApp(t)

	body := `{"id": "123e4567-e89b-12d3-a456-426614174000", "status": "invalid_status"}`
	req := httptest.NewRequest(http.MethodPost, "/agent/heartbeat", bytes.NewBufferString(body))
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

func TestAgentHeartbeatHandler_EmptyStatus(t *testing.T) {
	app := setupApp(t)

	body := `{"id": "123e4567-e89b-12d3-a456-426614174000", "status": ""}`
	req := httptest.NewRequest(http.MethodPost, "/agent/heartbeat", bytes.NewBufferString(body))
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
