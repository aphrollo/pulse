package handlers

import (
	"context"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	db "github.com/aphrollo/pulse/internal/storage"
)

// WorkerRegisterRequest Request to register a worker
type WorkerRegisterRequest struct {
	ID   string                 `json:"id"`   // UUID string
	Name string                 `json:"name"` // Required
	Type string                 `json:"type"`
	Info map[string]interface{} `json:"info"` // Optional additional info
}

// WorkerRegisterHandler registers a new worker or updates if exists
func WorkerRegisterHandler(c *fiber.Ctx) error {
	var req WorkerRegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid UUID"})
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	// Convert Info to JSON
	infoJSON, err := json.Marshal(req.Info)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to encode info"})
	}

	ctx := context.Background()
	// Upsert worker (insert or update)
	sql := `
		INSERT INTO workers (id, name, type, info)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			info = EXCLUDED.info,
			time = now()
	`
	_, err = db.Pool.Exec(ctx, sql, id, req.Name, req.Type, infoJSON)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to register worker"})
	}

	return c.JSON(fiber.Map{"status": "worker registered"})
}

// WorkerUpdateRequest Request to update a worker's metadata/settings
type WorkerUpdateRequest struct {
	ID      string `json:"id"`             // Worker UUID string
	Status  string `json:"status"`         // Must be one of worker_status enum
	Message string `json:"info,omitempty"` // Partial updates allowed
}

// WorkerUpdateHandler logs a status update with optional message
func WorkerUpdateHandler(c *fiber.Ctx) error {
	var req WorkerUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid UUID"})
	}

	// Validate status? (optional, you can check against enum list)

	ctx := context.Background()
	sql := `
		INSERT INTO worker_updates (worker_id, status, message)
		VALUES ($1, $2, $3)
	`
	_, err = db.Pool.Exec(ctx, sql, id, req.Status, req.Message)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update worker status"})
	}

	return c.JSON(fiber.Map{"status": "worker status updated"})
}

// WorkerHeartbeatRequest Request to send a worker heartbeat/status
type WorkerHeartbeatRequest struct {
	ID     string `json:"id"`     // Worker UUID string
	Status string `json:"status"` // Must be one of worker_status enum
}

// WorkerHeartbeatHandler logs a heartbeat status
func WorkerHeartbeatHandler(c *fiber.Ctx) error {
	var req WorkerHeartbeatRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid UUID"})
	}

	// Validate status? (optional)

	ctx := context.Background()
	sql := `
		INSERT INTO worker_heartbeats (worker_id, status)
		VALUES ($1, $2)
	`
	_, err = db.Pool.Exec(ctx, sql, id, req.Status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to insert heartbeat"})
	}

	return c.JSON(fiber.Map{"status": "heartbeat recorded"})
}
