// Package client provides an HTTP client for communicating with a running Shepherd server.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client that communicates with the Shepherd API server.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new Client targeting the given host and port.
func NewClient(host string, port int) *Client {
	return &Client{
		BaseURL: fmt.Sprintf("http://%s:%d", host, port),
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// apiResponse is the envelope format returned by the Shepherd API.
type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *apiError       `json:"error,omitempty"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Get performs a GET request to the given path and returns the data payload.
func (c *Client) Get(path string) (json.RawMessage, error) {
	url := c.BaseURL + path
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, c.connectionError(err)
	}
	defer resp.Body.Close()

	return c.parseResponse(resp)
}

// Post performs a POST request with an optional JSON body.
func (c *Client) Post(path string, body interface{}) (json.RawMessage, error) {
	url := c.BaseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	resp, err := c.HTTPClient.Post(url, "application/json", bodyReader)
	if err != nil {
		return nil, c.connectionError(err)
	}
	defer resp.Body.Close()

	return c.parseResponse(resp)
}

// Ping checks if the server is reachable by calling GET /api/info.
func (c *Client) Ping() error {
	_, err := c.Get("/api/info")
	return err
}

// parseResponse reads the HTTP response and extracts the data field.
func (c *Client) parseResponse(resp *http.Response) (json.RawMessage, error) {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var envelope apiResponse
	if err := json.Unmarshal(bodyBytes, &envelope); err != nil {
		// If JSON parsing fails, return the raw body as an error
		return nil, fmt.Errorf("unexpected response (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	if !envelope.Success {
		if envelope.Error != nil {
			msg := envelope.Error.Message
			if envelope.Error.Details != "" {
				msg += ": " + envelope.Error.Details
			}
			return nil, fmt.Errorf("API error [%s]: %s", envelope.Error.Code, msg)
		}
		return nil, fmt.Errorf("request failed with HTTP %d", resp.StatusCode)
	}

	return envelope.Data, nil
}

// connectionError wraps a connection error with a user-friendly message.
func (c *Client) connectionError(err error) error {
	return fmt.Errorf("cannot connect to Shepherd server at %s\n"+
		"Please make sure the server is running (shepherd serve)\n"+
		"Error: %w", c.BaseURL, err)
}
