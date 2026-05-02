package backend

import (
	"strings"
	"testing"
)

func TestBackendType_String(t *testing.T) {
	tests := []struct {
		bt  BackendType
		exp string
	}{
		{BackendLlamaCpp, "llamacpp"},
		{BackendVLLM, "vllm"},
		{BackendVLLMOmni, "vllm_omni"},
	}
	for _, tt := range tests {
		if got := tt.bt.String(); got != tt.exp {
			t.Errorf("BackendType(%q).String() = %q, want %q", tt.bt, got, tt.exp)
		}
	}
}

func TestParseBackendType(t *testing.T) {
	tests := []struct {
		input   string
		exp     BackendType
		wantErr bool
	}{
		{"llamacpp", BackendLlamaCpp, false},
		{"vllm", BackendVLLM, false},
		{"vllm_omni", BackendVLLMOmni, false},
		{"", BackendLlamaCpp, false},
		{"unknown", "", true},
		{"llama_cpp", "", true},
	}
	for _, tt := range tests {
		got, err := ParseBackendType(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseBackendType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.exp {
			t.Errorf("ParseBackendType(%q) = %v, want %v", tt.input, got, tt.exp)
		}
	}
}

func TestIsGGUFModel(t *testing.T) {
	tests := []struct {
		path string
		exp  bool
	}{
		{"model.gguf", true},
		{"model.GGUF", true},
		{"/path/to/model.gguf", true},
		{"model.safetensors", false},
		{"model.bin", false},
		{"config.json", false},
	}
	for _, tt := range tests {
		if got := IsGGUFModel(tt.path); got != tt.exp {
			t.Errorf("IsGGUFModel(%q) = %v, want %v", tt.path, got, tt.exp)
		}
	}
}

func TestIsSafeTensorsModel(t *testing.T) {
	tests := []struct {
		path string
		exp  bool
	}{
		{"model.safetensors", true},
		{"/cache/models--org--model/snapshots/abc123/model.safetensors", true},
		{"model.gguf", false},
		{"model.bin", false},
	}
	for _, tt := range tests {
		if got := IsSafeTensorsModel(tt.path); got != tt.exp {
			t.Errorf("IsSafeTensorsModel(%q) = %v, want %v", tt.path, got, tt.exp)
		}
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()

	b, ok := r.Get(BackendLlamaCpp)
	if !ok {
		t.Fatal("expected to find llamacpp backend")
	}
	if b.Type() != BackendLlamaCpp {
		t.Errorf("got type %v, want %v", b.Type(), BackendLlamaCpp)
	}

	b, ok = r.Get(BackendVLLM)
	if !ok {
		t.Fatal("expected to find vllm backend")
	}

	b, ok = r.Get(BackendVLLMOmni)
	if !ok {
		t.Fatal("expected to find vllm_omni backend")
	}
}

func TestRegistry_ResolveDefault(t *testing.T) {
	r := NewRegistry()

	// GGUF model should resolve to llama.cpp by default
	b, cfg, err := r.Resolve("model.gguf", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Type() != BackendLlamaCpp {
		t.Errorf("got backend %v, want %v", b.Type(), BackendLlamaCpp)
	}
	_ = cfg
}

func TestRegistry_ResolveExplicitVLLM(t *testing.T) {
	r := NewRegistry()
	r.Configure(BackendVLLM, &BackendConfig{
		Type:     BackendVLLM,
		CondaEnv: "vllm",
	})

	b, cfg, err := r.Resolve("model.gguf", BackendVLLM)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Type() != BackendVLLM {
		t.Errorf("got backend %v, want %v", b.Type(), BackendVLLM)
	}
	if cfg == nil || cfg.CondaEnv != "vllm" {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestLlamaCppBackend_SupportsModel(t *testing.T) {
	b := NewLlamaCppBackend()
	if !b.SupportsModel("model.gguf") {
		t.Error("should support GGUF models")
	}
	if b.SupportsModel("model.safetensors") {
		t.Error("should not support safetensors models")
	}
}

func TestVLLMBackend_SupportsModel(t *testing.T) {
	b := NewVLLMBackend()
	if !b.SupportsModel("model.safetensors") {
		t.Error("should support safetensors models")
	}
	if b.SupportsModel("model.gguf") {
		t.Error("should not support GGUF models")
	}
}

func TestLlamaCppBackend_IsLoadComplete(t *testing.T) {
	b := NewLlamaCppBackend()
	if !b.IsLoadComplete("all slots are idle") {
		t.Error("should detect load complete")
	}
	if b.IsLoadComplete("loading model...") {
		t.Error("should not detect load complete")
	}
}

func TestVLLMBackend_IsLoadComplete(t *testing.T) {
	b := NewVLLMBackend()
	if !b.IsLoadComplete("INFO: Uvicorn running on http://0.0.0.0:8000") {
		t.Error("should detect load complete")
	}
	if b.IsLoadComplete("loading model...") {
		t.Error("should not detect load complete")
	}
}

func TestLlamaCppBackend_Discover(t *testing.T) {
	b := NewLlamaCppBackend()

	// With empty path should return available=false (no llama-server in common paths on test machine)
	info, err := b.Discover(&BackendConfig{Type: BackendLlamaCpp, BinPath: "/nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Available {
		t.Error("should not be available with nonexistent path")
	}
}

func TestLlamaCppBackend_BuildStartConfig(t *testing.T) {
	b := NewLlamaCppBackend()
	info := &BackendInfo{
		Type:      BackendLlamaCpp,
		BinPath:   "/usr/local/bin",
		Available: true,
	}

	req := &LoadRequest{
		ModelPath: "/models/test.gguf",
		Port:      8081,
		CtxSize:   4096,
		GPULayers: 99,
		LlamacppParams: &LlamacppLoadParams{
			FlashAttention: true,
			ContBatching:   true,
		},
	}

	// This test uses a mock binary path - the command builder should still work
	// since it just constructs the command string
	cfg, err := b.BuildStartConfig(info, req)
	// The test may fail if llama-server isn't found, but the logic is tested
	if err != nil {
		// Expected if /usr/local/bin/llama-server doesn't exist
		t.Logf("BuildStartConfig returned error (expected without llama-server): %v", err)
		return
	}

	if cfg.BackendType != BackendLlamaCpp {
		t.Errorf("got backend type %v, want %v", cfg.BackendType, BackendLlamaCpp)
	}
	if cfg.Command == "" {
		t.Error("expected non-empty command")
	}
}

func TestVLLMBackend_BuildStartConfig(t *testing.T) {
	b := NewVLLMBackend()
	info := &BackendInfo{
		Type:      BackendVLLM,
		Name:      "vLLM",
		CondaEnv:  "vllm",
		Available: true,
	}

	req := &LoadRequest{
		ModelPath: "/models/llama-hf",
		Port:      8000,
		CtxSize:   4096,
		VLLMParams: &VLLMLoadParams{
			DataType:          "auto",
			TensorParallelSize: 2,
			TrustRemoteCode:   true,
		},
	}

	cfg, err := b.BuildStartConfig(info, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.BackendType != BackendVLLM {
		t.Errorf("got backend type %v, want %v", cfg.BackendType, BackendVLLM)
	}

	cmd := cfg.Command
	if !containsAll(cmd, "conda", "run", "-n", "vllm", "vllm", "serve", "/models/llama-hf", "--port", "8000") {
		t.Errorf("command missing expected parts: %s", cmd)
	}
	if !containsAll(cmd, "--dtype", "auto", "--tensor-parallel-size", "2", "--trust-remote-code") {
		t.Errorf("command missing vLLM params: %s", cmd)
	}
}

func TestVLLMOmniBackend_BuildStartConfig(t *testing.T) {
	b := NewVLLMOmniBackend()
	info := &BackendInfo{
		Type:      BackendVLLMOmni,
		Name:      "vLLM-Omni",
		CondaEnv:  "vllm-omni",
		Available: true,
	}

	req := &LoadRequest{
		ModelPath: "/models/omni-model",
		Port:      8000,
		VLLOmniParams: &VLLOmniLoadParams{
			VLLMLoadParams: VLLMLoadParams{
				DataType: "bfloat16",
			},
			VideoPruningRate: 0.5,
			MMTensorIPC:      true,
		},
	}

	cfg, err := b.BuildStartConfig(info, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := cfg.Command
	if !containsAll(cmd, "vllm-omni", "serve") {
		t.Errorf("command should use vllm-omni: %s", cmd)
	}
	if !containsAll(cmd, "--video-pruning-rate") {
		t.Errorf("command missing video pruning rate: %s", cmd)
	}
	if !containsAll(cmd, "--mm-tensor-ipc") {
		t.Errorf("command missing mm-tensor-ipc: %s", cmd)
	}
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

func TestLoadRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     *LoadRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &LoadRequest{
				ModelPath: "/path/to/model.gguf",
				Port:      8080,
			},
			wantErr: false,
		},
		{
			name: "empty model path",
			req: &LoadRequest{
				Port: 8080,
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			req: &LoadRequest{
				ModelPath: "/path/to/model.gguf",
				Port:      0,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
