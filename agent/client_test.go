package agent

import (
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Helper function to create agent with test server URL
func newTestAgent(tsURL string) *Agent {
	return &Agent{
		ID:     uuid.New(),
		Name:   "test-agent",
		Type:   "default",
		Server: tsURL,
		Client: &http.Client{},
	}
}

// Test successful Register request
func TestAgent_Register_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/agent/register" {
			t.Fatalf("expected /agent/register, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("expected content-type application/json, got %s", ct)
		}

		var payload registerPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode JSON payload: %v", err)
		}
		if payload.ID == "" || payload.Name == "" || payload.Type == "" {
			http.Error(w, "missing fields", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	agent := newTestAgent(ts.URL)

	if err := agent.Register(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// Test Register when server returns non-200 status code
func TestAgent_Register_NonOKStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer ts.Close()

	agent := newTestAgent(ts.URL)

	err := agent.Register()
	if err == nil || !strings.Contains(err.Error(), "server returned status 400") {
		t.Errorf("expected status error, got %v", err)
	}
}

// Test Register with invalid server URL (connection refused)
func TestAgent_Register_InvalidServer(t *testing.T) {
	agent := newTestAgent("http://invalid:1234")

	err := agent.Register()
	if err == nil || !strings.Contains(err.Error(), "post error") {
		t.Errorf("expected post error, got %v", err)
	}
}

// Test Register with invalid JSON marshal (simulate by embedding invalid type)
func TestAgent_Register_MarshalError(t *testing.T) {
	agent := newTestAgent("http://example.com")

	// Trick: override post method to call with invalid payload type
	agent.ID = uuid.Nil // normal ID still OK
	agent.Name = "test"
	agent.Type = "default"

	// Override post to send a channel, which json.Marshal can't handle
	err := agent.post("/agent/register", make(chan int))
	if err == nil || !strings.Contains(err.Error(), "marshal error") {
		t.Errorf("expected marshal error, got %v", err)
	}
}

// Test Register with missing PULSE_SERVER_URL env variable triggers fatal
func TestNew_FatalOnMissingServerURL(t *testing.T) {
	// Unset env var
	os.Unsetenv("PULSE_SERVER_URL")

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected fatal log panic due to missing PULSE_SERVER_URL, but did not panic")
		}
	}()

	_ = New("test-agent", "default")
}

// Test Register with server rejecting due to missing fields (simulated server)
func TestAgent_Register_ServerRejectsMissingFields(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload registerPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if payload.ID == "" || payload.Name == "" || payload.Type == "" {
			http.Error(w, "missing fields", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	agent := newTestAgent(ts.URL)
	agent.Name = "" // trigger missing name error

	err := agent.Register()
	if err == nil || !strings.Contains(err.Error(), "server returned status 400") {
		t.Errorf("expected 400 status error, got %v", err)
	}
}
