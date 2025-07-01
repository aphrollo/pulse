package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

type Agent struct {
	ID     uuid.UUID
	Name   string
	Type   string
	Server string
	Client *http.Client
}

type registerPayload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type heartbeatPayload struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type updatePayload struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// New initializes a new Agent using env vars
func New(name, agentType string) *Agent {
	server := os.Getenv("PULSE_SERVER_URL")
	if server == "" {
		log.Fatal("PULSE_SERVER_URL not set")
	}

	return &Agent{
		ID:     uuid.New(),
		Name:   name,
		Type:   agentType,
		Server: server,
		Client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Register sends the registration request to Pulse
func (a *Agent) Register() error {
	payload := registerPayload{
		ID:   a.ID.String(),
		Name: a.Name,
		Type: a.Type,
	}
	return a.post("/agent/register", payload)
}

// Heartbeat sends a heartbeat signal to Pulse
func (a *Agent) Heartbeat(status string) error {
	payload := heartbeatPayload{
		ID:     a.ID.String(),
		Status: status,
	}
	return a.post("/agent/heartbeat", payload)
}

// Update sends a status update with optional message
func (a *Agent) Update(status, message string) error {
	payload := updatePayload{
		ID:      a.ID.String(),
		Status:  status,
		Message: message,
	}
	return a.post("/agent/update", payload)
}

func (a *Agent) post(path string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	url := fmt.Sprintf("%s%s", a.Server, path)
	resp, err := a.Client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("post error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	return nil
}
