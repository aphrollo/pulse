package handlers

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	db "github.com/aphrollo/pulse/internal/storage"
)

// ApiResponse represents a generic API response
type ApiResponse struct {
	Message string `json:"message" example:"OK"`
}

// ApiErrorResponse represents a generic error API response
type ApiErrorResponse struct {
	Message string `json:"message" example:"ERROR_MESSAGE"`
}

// WorkerRegisterRequest Request to register a worker
type WorkerRegisterRequest struct {
	ID   string `json:"id"`   // UUID string
	Name string `json:"name"` // Required
	Type string `json:"type"`
}

// WorkerRegisterHandler registers a new worker
// @Summary Register a worker
// @Description Registers a worker by UUID, name, type, and optional metadata
// @Tags Worker
// @Accept json
// @Produce json
// @Param request body WorkerRegisterRequest true "Worker registration info"
// @Success 200 {object} ApiResponse "Success response `{"message":"OK"}`"
// @Failure 400 {object} ApiErrorResponse "BAD_REQUEST - The query contains errors. In the event that a request was created using a form and contains user generated data, the user should be notified that the data must be corrected before the query is repeated. `{"message":"BAD_REQUEST"}`"
// @Failure 401 {object} ApiErrorResponse "UNAUTHORIZED - There was an unauthorized attempt to use functionality available only to authorized users. `{"message":"UNAUTHORIZED"}`"
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

	ctx := context.Background()
	// Upsert worker (insert or update)
	sql := `
		INSERT INTO workers (id, name, type)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			time = now()
	`
	_, err = db.Pool.Exec(ctx, sql, id, req.Name, req.Type)
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
// @Success 200 {object} ApiResponse "Success response `{"message":"OK"}`"
// @Failure 400 {object} ApiErrorResponse "BAD_REQUEST - The query contains errors. In the event that a request was created using a form and contains user generated data, the user should be notified that the data must be corrected before the query is repeated. `{"message":"BAD_REQUEST"}`"
// @Failure 401 {object} ApiErrorResponse "UNAUTHORIZED - There was an unauthorized attempt to use functionality available only to authorized users. `{"message":"UNAUTHORIZED"}`"
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
	ID     string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Status string `json:"status" example:"healthy"`
}

// WorkerHeartbeatHandler receives a heartbeat ping from a worker
// @Summary Heartbeat signal
// @Description Receives regular heartbeat signal from workers
// @Tags Worker
// @Accept json
// @Produce json
// @Param request body handlers.WorkerHeartbeatRequest true "Worker heartbeat. Possible: `starting`, `healthy`, `working`, `idle`, `error`, `unreachable`, `crashed`, `stopped`, `disabled`"
// @Success 200 {object} ApiResponse "Success response `{"message":"OK"}`"
// @Failure 400 {object} ApiErrorResponse "BAD_REQUEST - The query contains errors. In the event that a request was created using a form and contains user generated data, the user should be notified that the data must be corrected before the query is repeated. `{"message":"BAD_REQUEST"}`"
// @Failure 401 {object} ApiErrorResponse "UNAUTHORIZED - There was an unauthorized attempt to use functionality available only to authorized users. `{"message":"UNAUTHORIZED"}`"
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
