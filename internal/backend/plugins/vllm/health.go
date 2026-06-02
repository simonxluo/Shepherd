package vllm

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/simonxluo/Shepherd/internal/backend"
)

// checkHealth probes http://localhost:<port>/health and returns the result.
func checkHealth(port int) (*backend.HealthResult, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/health", port))
	if err != nil {
		return &backend.HealthResult{Healthy: false}, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return &backend.HealthResult{Healthy: false, Body: string(body)}, nil
	}
	return &backend.HealthResult{Healthy: true, Body: string(body)}, nil
}
