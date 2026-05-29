package model

import "time"

// RecordRequest records a completed request's statistics for a model.
// It is safe to call concurrently.
func (m *Manager) RecordRequest(modelID string, promptTokens, completionTokens, latencyMs int64, success bool) {
	m.mu.RLock()
	status, exists := m.statuses[modelID]
	m.mu.RUnlock()
	if !exists {
		return
	}

	now := time.Now()
	status.tokenMu.Lock()
	defer status.tokenMu.Unlock()

	status.RequestCount++
	status.TotalPromptTokens += promptTokens
	status.TotalCompletionTokens += completionTokens
	status.TotalLatencyMs += latencyMs
	status.LastRequestAt = now
	if status.FirstRequestAt.IsZero() {
		status.FirstRequestAt = now
	}
	if !success {
		status.ErrorCount++
	}
}

// ModelStatistics represents aggregated statistics for a loaded model.
type ModelStatistics struct {
	ModelID               string  `json:"modelId"`
	ModelName             string  `json:"modelName"`
	InstanceID            string  `json:"instanceId"`
	State                 string  `json:"state"`
	BackendType           string  `json:"backendType"`
	Port                  int     `json:"port"`
	LoadedAt              int64   `json:"loadedAt"` // unix timestamp
	UptimeSeconds         int64   `json:"uptimeSeconds"`
	RequestCount          int64   `json:"requestCount"`
	ErrorCount            int64   `json:"errorCount"`
	TotalPromptTokens     int64   `json:"totalPromptTokens"`
	TotalCompletionTokens int64   `json:"totalCompletionTokens"`
	AvgLatencyMs          float64 `json:"avgLatencyMs"`
	InflightCount         int32   `json:"inflightCount"`
	LastRequestAt         int64   `json:"lastRequestAt,omitempty"` // unix timestamp, 0 if never
}

// GetModelStatistics returns statistics for all loaded models.
func (m *Manager) GetModelStatistics() []ModelStatistics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	stats := make([]ModelStatistics, 0, len(m.statuses))

	for id, status := range m.statuses {
		if status.State == StateUnloaded {
			continue
		}

		status.tokenMu.Lock()
		var avgLatency float64
		if status.RequestCount > 0 {
			avgLatency = float64(status.TotalLatencyMs) / float64(status.RequestCount)
		}
		var lastReqAt int64
		if !status.LastRequestAt.IsZero() {
			lastReqAt = status.LastRequestAt.Unix()
		}
		s := ModelStatistics{
			ModelID:               id,
			ModelName:             status.Name,
			InstanceID:            status.InstanceID,
			State:                 status.State.String(),
			BackendType:           status.BackendType,
			Port:                  status.Port,
			LoadedAt:              status.LoadedAt.Unix(),
			UptimeSeconds:         int64(now.Sub(status.LoadedAt).Seconds()),
			RequestCount:          status.RequestCount,
			ErrorCount:            status.ErrorCount,
			TotalPromptTokens:     status.TotalPromptTokens,
			TotalCompletionTokens: status.TotalCompletionTokens,
			AvgLatencyMs:          avgLatency,
			InflightCount:         status.InflightCount.Load(),
			LastRequestAt:         lastReqAt,
		}
		status.tokenMu.Unlock()

		stats = append(stats, s)
	}
	return stats
}
