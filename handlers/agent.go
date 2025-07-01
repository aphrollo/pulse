package handlers

import (
	"context"
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	db "github.com/aphrollo/pulse/storage"
)

// ApiResponse represents a generic API response
type ApiResponse struct {
	Message string `json:"message" example:"OK"`
}

// ApiErrorResponse represents a generic error API response
type ApiErrorResponse struct {
	Message string `json:"message" example:"ERROR_MESSAGE"`
}

var allowedAgentTypes = map[string]bool{
	"default": true,
}

var allowedAgentStatus = map[string]bool{
	"starting": true, "healthy": true, "working": true, "idle": true,
	"error": true, "unreachable": true, "crashed": true, "stopped": true, "disabled": true,
}

// AgentRegisterRequest Request to register a Agent
type AgentRegisterRequest struct {
	ID   string `json:"id"`   // UUID string
	Name string `json:"name"` // Required
	Type string `json:"type"`
}

// AgentRegisterHandler registers a new Agent
// @Summary Register a Agent
// @Description Registers a Agent by UUID, name, type, and optional metadata
// @Tags Agent
// @Accept json
// @Produce json
// @Param request body AgentRegisterRequest true "Agent registration info"
// @Success 200 {object} ApiResponse "Success response `{"message":"OK"}`"
// @Failure 400 {object} ApiErrorResponse "BAD_REQUEST - The query contains errors. In the event that a request was created using a form and contains user generated data, the user should be notified that the data must be corrected before the query is repeated. `{"message":"BAD_REQUEST"}`"
// @Failure 401 {object} ApiErrorResponse "UNAUTHORIZED - There was an unauthorized attempt to use functionality available only to authorized users. `{"message":"UNAUTHORIZED"}`"
// @Router /agent/register [post]
func AgentRegisterHandler(c *fiber.Ctx) error {
	var req AgentRegisterRequest
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
	if !allowedAgentTypes[req.Type] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid Agent type"})
	}

	ctx := context.Background()

	// If you want to reject duplicates:
	sql := `
		INSERT INTO agents (id, name, type)
		VALUES ($1, $2, $3)
	`
	_, err = db.Pool.Exec(ctx, sql, id, req.Name, req.Type)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Agent ID already exists"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to register Agent"})
	}

	return c.JSON(fiber.Map{"status": "Agent registered"})
}

// AgentUpdateRequest Request to update a Agent's metadata/settings
type AgentUpdateRequest struct {
	ID      string `json:"id"`             // Agent UUID string
	Status  string `json:"status"`         // Must be one of Agent_status enum
	Message string `json:"info,omitempty"` // Partial updates allowed
}

// AgentUpdateHandler updates an existing Agent's status or metadata
// @Summary Update Agent status
// @Description Updates Agent state and optional info (partial updates allowed)
// @Tags Agent
// @Accept json
// @Produce json
// @Param request body AgentUpdateRequest true "Agent update info"
// @Success 200 {object} ApiResponse "Success response `{"message":"OK"}`"
// @Failure 400 {object} ApiErrorResponse "BAD_REQUEST - The query contains errors. In the event that a request was created using a form and contains user generated data, the user should be notified that the data must be corrected before the query is repeated. `{"message":"BAD_REQUEST"}`"
// @Failure 401 {object} ApiErrorResponse "UNAUTHORIZED - There was an unauthorized attempt to use functionality available only to authorized users. `{"message":"UNAUTHORIZED"}`"
// @Router /agent/update [post]
func AgentUpdateHandler(c *fiber.Ctx) error {
	var req AgentUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid UUID"})
	}

	// Validate status is provided (if required)
	if req.Status == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status is required"})
	}

	ctx := context.Background()
	sql := `
		INSERT INTO agent_updates (Agent_id, status, message)
		VALUES ($1, $2, $3)
	`
	_, err = db.Pool.Exec(ctx, sql, id, req.Status, req.Message)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update Agent status"})
	}

	return c.JSON(fiber.Map{"status": "Agent status updated"})
}

// AgentHeartbeatRequest Request to send a Agent heartbeat/status
type AgentHeartbeatRequest struct {
	ID     string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Status string `json:"status" example:"healthy"`
}

// AgentHeartbeatHandler receives a heartbeat ping from a Agent
// @Summary Heartbeat signal
// @Description Receives regular heartbeat signal from Agents
// @Tags Agent
// @Accept json
// @Produce json
// @Param request body handlers.AgentHeartbeatRequest true "Agent heartbeat. Possible: `starting`, `healthy`, `working`, `idle`, `error`, `unreachable`, `crashed`, `stopped`, `disabled`"
// @Success 200 {object} ApiResponse "Success response `{"message":"OK"}`"
// @Failure 400 {object} ApiErrorResponse "BAD_REQUEST - The query contains errors. In the event that a request was created using a form and contains user generated data, the user should be notified that the data must be corrected before the query is repeated. `{"message":"BAD_REQUEST"}`"
// @Failure 401 {object} ApiErrorResponse "UNAUTHORIZED - There was an unauthorized attempt to use functionality available only to authorized users. `{"message":"UNAUTHORIZED"}`"
// @Router /agent/heartbeat [post]
func AgentHeartbeatHandler(c *fiber.Ctx) error {
	var req AgentHeartbeatRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid UUID"})
	}

	if req.Status == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status is required"})
	}
	if !allowedAgentStatus[req.Status] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid status value"})
	}

	ctx := context.Background()
	sql := `
		INSERT INTO agent_heartbeats (Agent_id, status)
		VALUES ($1, $2)
	`
	_, err = db.Pool.Exec(ctx, sql, id, req.Status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to insert heartbeat"})
	}

	return c.JSON(fiber.Map{"status": "heartbeat recorded"})
}
