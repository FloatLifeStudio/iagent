// Package push implements the CMDB API push client
package push

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"iagent/internal/payload"
)

// Result is the push response: result has three branches, created / unchanged / diff_created
type Result struct {
	Result          string `json:"result"`
	DeviceID        int    `json:"device_id"`
	PendingChangeID *int   `json:"pending_change_id"`
}

type Client struct {
	serverURL string
	token     string
	http      *http.Client
}

func NewClient(serverURL, token string) *Client {
	return &Client{
		serverURL: serverURL,
		token:     token,
		http:      &http.Client{Timeout: 30 * time.Second},
	}
}

// Push pushes device data. No retry on failure (next cycle full sync self-heals)
func (c *Client) Push(p *payload.Payload) (Result, error) {
	body, err := json.Marshal(p)
	if err != nil {
		return Result{}, fmt.Errorf("marshal payload: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, c.serverURL+"/api/v1/devices", bytes.NewReader(body))
	if err != nil {
		return Result{}, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("push: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("push: HTTP %d", resp.StatusCode)
	}
	var result Result
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Result{}, fmt.Errorf("decode result: %w", err)
	}
	return result, nil
}
