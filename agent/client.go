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
	ID        uuid.UUID
	Name      string
	Type      string
	Info      map[string]interface{}
	Server    string
	heartbeat time.Duration
	Client    *http.Client
	stopChan  chan struct{}
}

// New initializes a new Agent using env vars
func New(name, agentType string) *Agent {
	server := os.Getenv("PULSE_SERVER_URL")
	if server == "" {
		log.Fatal("PULSE_SERVER_URL not set")
	}
	interval := 60 * time.Second // default
	if v := os.Getenv("PULSE_HEARTBEAT_INTERVAL"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			interval = parsed
		}
	}

	return &Agent{
		ID:        uuid.New(),
		Name:      name,
		Type:      agentType,
		Server:    server,
		heartbeat: interval,
		Client:    &http.Client{Timeout: 5 * time.Second},
		stopChan:  make(chan struct{}),
	}
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

type registerPayload struct {
	ID   string                 `json:"id"`
	Name string                 `json:"name"`
	Type string                 `json:"type"`
	Info map[string]interface{} `json:"info,omitempty"`
}

// Register sends the registration request to Pulse
func (a *Agent) Register() error {
	payload := registerPayload{
		ID:   a.ID.String(),
		Name: a.Name,
		Type: a.Type,
		Info: a.Info,
	}
	return a.post("/agent/register", payload)
}

type heartbeatPayload struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// Heartbeat sends a heartbeat signal to Pulse
func (a *Agent) Heartbeat(status string) error {
	payload := heartbeatPayload{
		ID:     a.ID.String(),
		Status: status,
	}
	return a.post("/agent/heartbeat", payload)
}

func (a *Agent) StartHeartbeatLoop() {
	ticker := time.NewTicker(a.heartbeat)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := a.Heartbeat("healthy")
				if err != nil {
					log.Printf("heartbeat error: %v", err)
				} else {
					log.Printf("heartbeat sent for agent %s", a.ID)
				}
			case <-a.stopChan:
				log.Println("heartbeat loop stopped")
				return
			}
		}
	}()
}

func (a *Agent) StopHeartbeatLoop() {
	close(a.stopChan)
}

type updatePayload struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
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
