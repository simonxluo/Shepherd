package client

// ServerInfo represents the response from GET /api/info.
type ServerInfo struct {
	Version   string    `json:"version"`
	BuildTime string    `json:"buildTime"`
	GitCommit string    `json:"gitCommit"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Role      string    `json:"role"`
	Ports     PortsInfo `json:"ports"`
}

// PortsInfo contains server port configuration.
type PortsInfo struct {
	Web       int `json:"web"`
	Anthropic int `json:"anthropic"`
	Ollama    int `json:"ollama"`
	LMStudio  int `json:"lmstudio"`
}

// ConfigResponse represents the response from GET /api/config.
type ConfigResponse struct {
	Role    string       `json:"role"`
	Server  ServerConfig `json:"server"`
	Storage StorageInfo  `json:"storage"`
	Models  ModelsConfig `json:"models"`
	Node    NodeConfig   `json:"node"`
	Llamacpp LlamacppConfig `json:"llamacpp"`
}

type ServerConfig struct {
	Host          string `json:"host"`
	WebPort       int    `json:"web_port"`
	AnthropicPort int    `json:"anthropic_port"`
}

type StorageInfo struct {
	Type   string      `json:"type"`
	SQLite interface{} `json:"sqlite"`
}

type ModelsConfig struct {
	Paths    []string `json:"paths"`
	AutoScan bool     `json:"auto_scan"`
}

type NodeConfig struct {
	Role string `json:"role"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type LlamacppConfig struct {
	Paths []LlamacppPath `json:"paths"`
}

type LlamacppPath struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ModelItem represents a model in the list response.
type ModelItem struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	DisplayName string                 `json:"displayName"`
	Alias       string                 `json:"alias"`
	Path        string                 `json:"path"`
	Size        int64                  `json:"size"`
	TotalSize   int64                  `json:"totalSize,omitempty"`
	Favourite   bool                   `json:"favourite"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	Status      string                 `json:"status"`
	IsLoaded    bool                   `json:"isLoaded"`
	Port        int                    `json:"port,omitempty"`
	BackendType string                 `json:"backendType,omitempty"`
	ScannedAt   string                 `json:"scannedAt,omitempty"`
}

// LoadedModelItem represents a loaded model.
type LoadedModelItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Alias       string `json:"alias,omitempty"`
	State       string `json:"state"`
	ProcessID   string `json:"processId"`
	Port        int    `json:"port"`
	CtxSize     int    `json:"ctxSize"`
	BackendType string `json:"backendType,omitempty"`
	LoadedAt    string `json:"loadedAt,omitempty"`
}

// ProcessInfo represents a running process.
type ProcessInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	PID     int    `json:"pid"`
	Port    int    `json:"port"`
	CtxSize int    `json:"ctx_size"`
	Running bool   `json:"running"`
	Loading bool   `json:"loading"`
}

// GPUInfo represents a detected GPU.
type GPUInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	TotalMemory string `json:"totalMemory"`
	FreeMemory  string `json:"freeMemory"`
	Available   bool   `json:"available"`
}

// BackendInfo represents an inference backend.
type BackendInfo struct {
	Type        string `json:"type"`
	Path        string `json:"path,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Available   bool   `json:"available"`
	CondaEnv    string `json:"condaEnv,omitempty"`
	Enabled     bool   `json:"enabled,omitempty"`
}

// ResourcesResponse represents the response from GET /api/system/resources.
type ResourcesResponse struct {
	CPU     CPUResources     `json:"cpu"`
	Memory  MemoryResources  `json:"memory"`
	Disk    DiskResources    `json:"disk"`
	GPU     []GPUResource    `json:"gpu,omitempty"`
	Load    []float64        `json:"loadAverage,omitempty"`
	Uptime  int64            `json:"uptime"`
	Kernel  string           `json:"kernelVersion,omitempty"`
	ROCm    string           `json:"rocmVersion,omitempty"`
}

type CPUResources struct {
	Used    int64   `json:"used"`    // millicores
	Total   int64   `json:"total"`   // millicores
	Percent float64 `json:"percent"`
}

type MemoryResources struct {
	Used    int64   `json:"used"`    // bytes
	Total   int64   `json:"total"`   // bytes
	Percent float64 `json:"percent"`
}

type DiskResources struct {
	Used    int64   `json:"used"`    // bytes
	Total   int64   `json:"total"`   // bytes
	Percent float64 `json:"percent"`
}

type GPUResource struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	Vendor      string `json:"vendor"`
	MemoryUsed  int64  `json:"memoryUsed"`
	MemoryTotal int64  `json:"memoryTotal"`
}

// ModelLoadConfig represents a named load configuration.
type ModelLoadConfig struct {
	Name    string                 `json:"name"`
	Config  map[string]interface{} `json:"config"`
}
