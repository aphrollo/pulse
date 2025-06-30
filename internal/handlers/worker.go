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

// WorkerRegisterHandler registers a new worker
// @Summary Register a worker
// @Description Registers a worker by UUID, name, type, and optional metadata
// @Tags Worker
// @Accept json
// @Produce json
// @Param request body WorkerRegisterRequest true "Worker registration info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /worker/register [post]
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

// WorkerUpdateHandler updates an existing worker's status or metadata
// @Summary Update worker status
// @Description Updates worker state and optional info (partial updates allowed)
// @Tags Worker
// @Accept json
// @Produce json
// @Param request body WorkerUpdateRequest true "Worker update info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /worker/update [post]
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

// WorkerHeartbeatHandler receives a heartbeat ping from a worker
// @Summary Heartbeat signal
// @Description Receives regular heartbeat signal from workers
// @Tags Worker
// @Accept json
// @Produce json
// @Param request body WorkerHeartbeatRequest true "Worker heartbeat info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /worker/heartbeat [post]
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
