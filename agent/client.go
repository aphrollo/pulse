package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string       // Base URL of the Pulse API, e.g. "https://api.pulse.example.com"
	HTTPClient *http.Client // HTTP client to use for requests
}

func New(baseURL string, httpClient *http.Client, regReq WorkerRegisterRequest) (*Client, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	client := &Client{
		BaseURL:    baseURL,
		HTTPClient: httpClient,
	}

	// Register worker with Pulse
	ctx := context.Background()
	resp, err := client.RegisterWorker(ctx, regReq)
	if err != nil {
		return nil, fmt.Errorf("worker registration failed: %w", err)
	}

	fmt.Println("Registered worker:", resp.Message)
	return client, nil
}

func (c *Client) RegisterWorker(ctx context.Context, req WorkerRegisterRequest) (*ApiResponse, error) {
	// Marshal request JSON
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal register request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/worker/register", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read and decode response
	var apiResp ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check HTTP status code for errors
	if resp.StatusCode != http.StatusOK {
		return &apiResp, fmt.Errorf("server returned status %d: %s", resp.StatusCode, apiResp.Message)
	}

	return &apiResp, nil
}
