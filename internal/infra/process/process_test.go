package process

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitCommandLineArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple command",
			input:    "ls -la",
			expected: []string{"ls", "-la"},
		},
		{
			name:     "Command with spaces",
			input:    "echo hello world",
			expected: []string{"echo", "hello", "world"},
		},
		{
			name:     "Double quoted strings",
			input:    `echo "hello world"`,
			expected: []string{"echo", "hello world"},
		},
		{
			name:     "Single quoted strings (Unix)",
			input:    `echo 'hello world'`,
			expected: []string{"echo", "hello world"},
		},
		{
			name:     "Mixed quotes",
			input:    `echo "hello" 'world'`,
			expected: []string{"echo", "hello", "world"},
		},
		{
			name:     "Escaped quotes in double quotes",
			input:    `echo "hello \"world\""`,
			expected: []string{"echo", `hello "world"`},
		},
		{
			name:     "Complex path",
			input:    `llama-server -m /path/to/model.gguf -c 4096`,
			expected: []string{"llama-server", "-m", "/path/to/model.gguf", "-c", "4096"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Whitespace only",
			input:    "   \t  ",
			expected: []string{},
		},
		{
			name:     "Trailing whitespace",
			input:    "ls -la   ",
			expected: []string{"ls", "-la"},
		},
		{
			name:     "Multiple spaces between args",
			input:    "ls    -la",
			expected: []string{"ls", "-la"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := splitCommandLineArgs(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsWindows(t *testing.T) {
	result := isWindows()
	// On non-Windows systems, this should return false
	// We can't test both cases in the same test run
	assert.False(t, result || isWindows())
}

func TestNewProcess(t *testing.T) {
	process := NewProcess("test-id", "test-name", "echo hello", "/bin")

	assert.Equal(t, "test-id", process.ID)
	assert.Equal(t, "test-name", process.Name)
	assert.Equal(t, "echo hello", process.Cmd)
	assert.Equal(t, "/bin", process.BinPath)
	assert.False(t, process.IsRunning())
}

func TestProcessStartStop(t *testing.T) {
	t.Run("Simple echo command", func(t *testing.T) {
		process := NewProcess("echo-test", "echo", "echo hello", "")

		err := process.Start()
		require.NoError(t, err)
		assert.True(t, process.IsRunning())
		assert.Greater(t, process.GetPID(), 0)

		// Wait a bit for output
		// process.Stop() will wait for the process to finish

		err = process.Stop()
		assert.NoError(t, err)
		assert.False(t, process.IsRunning())
	})

	t.Run("Sleep command", func(t *testing.T) {
		process := NewProcess("sleep-test", "sleep", "sleep 0.1", "")

		err := process.Start()
		require.NoError(t, err)
		assert.True(t, process.IsRunning())

		// Process should be running
		assert.True(t, process.IsRunning())

		// Stop the process
		err = process.Stop()
		assert.NoError(t, err)
	})

	t.Run("Invalid command", func(t *testing.T) {
		process := NewProcess("invalid-test", "invalid", "this-command-does-not-exist-12345", "")

		err := process.Start()
		assert.Error(t, err)
		assert.False(t, process.IsRunning())
	})
}

func TestProcessOutputHandler(t *testing.T) {
	t.Run("With output handler", func(t *testing.T) {
		process := NewProcess("output-test", "echo", "echo test output", "")

		received := make(chan string, 10)
		process.SetOutputHandler(func(line string) {
			received <- line
		})

		err := process.Start()
		require.NoError(t, err)

		// Wait for output
		select {
		case line := <-received:
			assert.Contains(t, line, "test output")
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for output")
		}

		process.Stop()
	})
}

func TestProcessCtxSize(t *testing.T) {
	process := NewProcess("ctx-test", "test", "echo", "")

	assert.Equal(t, 0, process.GetCtxSize())

	process.SetCtxSize(4096)
	assert.Equal(t, 4096, process.GetCtxSize())
}

func TestProcessPort(t *testing.T) {
	process := NewProcess("port-test", "test", "echo", "")

	assert.Equal(t, 0, process.GetPort())

	process.SetPort(8081)
	assert.Equal(t, 8081, process.GetPort())
}

func TestManagerBasicOperations(t *testing.T) {
	manager := NewManager()

	// Start a process
	process, err := manager.Start("echo-test", "echo", "echo hello", "")
	require.NoError(t, err)
	assert.NotNil(t, process)

	// Get the process
	retrieved, exists := manager.Get("echo-test")
	assert.True(t, exists)
	assert.Equal(t, "echo-test", retrieved.ID)

	// Check running status
	assert.True(t, manager.IsRunning("echo-test"))

	// Stop the process
	err = manager.Stop("echo-test")
	assert.NoError(t, err)

	running, _ := manager.ListAll()
	assert.Len(t, running, 0)
}

func TestManagerDoubleStart(t *testing.T) {
	manager := NewManager()

	// Start first process
	_, err := manager.Start("test-id", "test", "sleep 0.5", "")
	require.NoError(t, err)

	// Try to start again with same ID
	_, err = manager.Start("test-id", "test", "echo hello", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already loaded")

	// Cleanup
	manager.Stop("test-id")
}

func TestManagerStopNonExistent(t *testing.T) {
	manager := NewManager()

	err := manager.Stop("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestQuoteAndJoin(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "Simple args",
			args:     []string{"echo", "hello"},
			expected: "echo hello",
		},
		{
			name:     "Arg with spaces",
			args:     []string{"echo", "hello world"},
			expected: `echo "hello world"`,
		},
		{
			name:     "Multiple args with spaces",
			args:     []string{"llama-server", "-m", "/path/to/model.gguf", "-c", "4096"},
			expected: `llama-server -m /path/to/model.gguf -c 4096`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteAndJoin(tt.args)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNeedsQuoting(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"simple", false},
		{"has space", true},
		{"has\ttab", true},
		{"has\"quote", true},
		{"has'apostrophe", true},
		{"has\\backslash", true},
		{"underscore_test", false},
		{"dash-test", false},
		{"dot.test", false},
		{"/path/to/file", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := needsQuoting(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapeQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{`has"quote`, `has\"quote`},
		{`has\backslash`, `has\\backslash`},
		{`"both"`, `\"both\"`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeQuotes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// LoadRequest is now defined in manager.go
// (previously defined locally to avoid import cycle)

// TestBuildCommandFromRequest tests the new LoadRequest-based BuildCommand
func TestBuildCommandFromRequest(t *testing.T) {
	binDir := t.TempDir()
	fakeServer := filepath.Join(binDir, "llama-server")
	if err := os.WriteFile(fakeServer, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("failed to create fake llama-server: %v", err)
	}

	tests := []struct {
		name        string
		req         *LoadRequest
		contains    []string
		notContains []string
	}{
		{
			name: "Basic command with defaults",
			req: &LoadRequest{
				ModelPath: "/models/model.gguf",
				Port:      8081,
			},
			contains: []string{"llama-server", "-m /models/model.gguf", "--port 8081", "--host 0.0.0.0"},
		},
		{
			name: "Single GPU selection",
			req: &LoadRequest{
				ModelPath: "/models/model.gguf",
				Port:      8081,
				GPULayers: 35,
				Devices:   []string{"cuda:0"},
				MainGPU:   0,
			},
			contains: []string{"-sm none", "-dev cuda:0", "-mg 0", "-ngl 35"},
		},
		{
			name: "Multi-GPU selection",
			req: &LoadRequest{
				ModelPath: "/models/model.gguf",
				Port:      8081,
				GPULayers: 99,
				Devices:   []string{"cuda:0", "cuda:1"},
			},
			contains:    []string{"-dev cuda:0,cuda:1", "-ngl 99"},
			notContains: []string{"-sm none", "-mg"},
		},
		{
			name: "Vision model with mmproj",
			req: &LoadRequest{
				ModelPath:     "/models/vision.gguf",
				Port:          8081,
				MmprojPath:    "/models/mmproj.gguf",
				EnableVision:  true,
				NoWebUI:       true,
				EnableMetrics: true,
			},
			contains: []string{"--mmproj /models/mmproj.gguf", "--no-webui", "--metrics"},
		},
		{
			name: "Feature flags",
			req: &LoadRequest{
				ModelPath:     "/models/model.gguf",
				Port:          8081,
				NoWebUI:       true,
				EnableMetrics: true,
				SlotSavePath:  "/cache/slots",
				CacheRAM:      -1,
			},
			contains: []string{"--no-webui", "--metrics", "--slot-save-path /cache/slots", "--cache-ram -1"},
		},
		{
			name: "Performance flags",
			req: &LoadRequest{
				ModelPath:      "/models/model.gguf",
				Port:           8081,
				FlashAttention: true,
				NoMmap:         true,
				LockMemory:     true,
			},
			contains: []string{"-fa", "--no-mmap", "--mlock"},
		},
		{
			name: "Chat template",
			req: &LoadRequest{
				ModelPath:        "/models/model.gguf",
				Port:             8081,
				ChatTemplateFile: "/templates/chat.jinja",
			},
			contains: []string{"--chat-template-file /templates/chat.jinja"},
		},
		{
			name: "Batch and parallel flags",
			req: &LoadRequest{
				ModelPath:     "/models/model.gguf",
				Port:          8081,
				UBatchSize:    128,
				ParallelSlots: 4,
			},
			contains: []string{"--ubatch-size 128", "--parallel 4"},
		},
		{
			name: "KV cache configuration",
			req: &LoadRequest{
				ModelPath:      "/models/model.gguf",
				Port:           8081,
				KVCacheTypeK:   "q8_0",
				KVCacheTypeV:   "q8_0",
				KVCacheUnified: true,
				KVCacheSize:    8192,
			},
			contains: []string{"-ctk q8_0", "-ctv q8_0", "-kvu", "--cache-ram 8192"},
		},
		{
			name: "Runtime configuration",
			req: &LoadRequest{
				ModelPath: "/models/model.gguf",
				Port:      8081,
				Timeout:   36000,
				Alias:     "my-model",
			},
			contains: []string{"--timeout 36000", "--alias my-model"},
		},
		{
			name: "Custom command",
			req: &LoadRequest{
				ModelPath: "/models/model.gguf",
				Port:      8081,
				CustomCmd: "--log-disable --debug",
			},
			contains: []string{"--log-disable --debug"},
		},
		{
			name: "Extra params",
			req: &LoadRequest{
				ModelPath:   "/models/model.gguf",
				Port:        8081,
				ExtraParams: "--special-arg value",
			},
			contains: []string{"--special-arg value"},
		},
		{
			name: "Path with spaces",
			req: &LoadRequest{
				ModelPath:        "/models/my model.gguf",
				Port:             8081,
				ChatTemplateFile: "/templates/my template.jinja",
			},
			contains: []string{"\"/models/my model.gguf\"", "\"/templates/my template.jinja\""},
		},
		{
			name: "Full featured command",
			req: &LoadRequest{
				ModelPath:        "/models/model.gguf",
				Port:             8081,
				CtxSize:          8192,
				BatchSize:        1024,
				GPULayers:        35,
				Devices:          []string{"cuda:0"},
				MainGPU:          0,
				FlashAttention:   true,
				NoMmap:           true,
				NoWebUI:          true,
				EnableMetrics:    true,
				SlotSavePath:     "/cache",
				CacheRAM:         -1,
				ChatTemplateFile: "/template.jinja",
				UBatchSize:       128,
				ParallelSlots:    4,
				KVCacheTypeK:     "f16",
				KVCacheUnified:   true,
				Timeout:          36000,
				Alias:            "my-model",
			},
			contains: []string{
				"llama-server",
				"-m /models/model.gguf",
				"--port 8081",
				"--host 0.0.0.0",
				"-c 8192",
				"-b 1024",
				"-ngl 35",
				"-sm none",
				"-dev cuda:0",
				"-mg 0",
				"-fa",
				"--no-mmap",
				"--no-webui",
				"--metrics",
				"--slot-save-path /cache",
				"--cache-ram -1",
				"--chat-template-file /template.jinja",
				"--ubatch-size 128",
				"--parallel 4",
				"-ctk f16",
				"-kvu",
				"--timeout 36000",
				"--alias my-model",
			},
		},
		{
			name: "Additional sampling parameters",
			req: &LoadRequest{
				ModelPath:        "/models/model.gguf",
				Port:             8081,
				Reranking:        true,
				MinP:             0.05,
				PresencePenalty:  0.1,
				FrequencyPenalty: 0.2,
			},
			contains: []string{
				"--reranking",
				"--min-p 0.05",
				"--presence-penalty 0.10",
				"--frequency-penalty 0.20",
			},
			notContains: []string{"--logits-all", "--dio"},
		},
		{
			name: "Template and processing flags",
			req: &LoadRequest{
				ModelPath:    "/models/model.gguf",
				Port:         8081,
				DisableJinja: true,
				ChatTemplate: "chatml",
				ContextShift: true,
			},
			contains: []string{
				"--no-jinja",
				"--chat-template chatml",
				"--context-shift",
			},
			notContains: []string{"--dio"},
		},
		{
			name: "All new fields combined",
			req: &LoadRequest{
				ModelPath: "/models/model.gguf",
				Port:      8081,
				CtxSize:   4096,
				Reranking:        true,
				MinP:             0.05,
				PresencePenalty:  0.1,
				FrequencyPenalty: 0.2,
				DisableJinja: true,
				ChatTemplate: "chatml",
				ContextShift: true,
			},
			contains: []string{
				"llama-server",
				"-m /models/model.gguf",
				"--port 8081",
				"-c 4096",
				"--reranking",
				"--min-p 0.05",
				"--presence-penalty 0.10",
				"--frequency-penalty 0.20",
				"--no-jinja",
				"--chat-template chatml",
				"--context-shift",
			},
			notContains: []string{"--dio"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := BuildCommandFromRequest(tt.req, binDir)
			require.NoError(t, err)

			for _, expected := range tt.contains {
				assert.Contains(t, cmd, expected, "command should contain %s", expected)
			}

			for _, notExpected := range tt.notContains {
				assert.NotContains(t, cmd, notExpected, "command should NOT contain %s", notExpected)
			}
		})
	}

	t.Run("Nil request", func(t *testing.T) {
		_, err := BuildCommandFromRequest(nil, binDir)
		assert.Error(t, err)
	})

	t.Run("Empty model path", func(t *testing.T) {
		_, err := BuildCommandFromRequest(&LoadRequest{Port: 8081}, binDir)
		assert.Error(t, err)
	})

	t.Run("Empty binary path", func(t *testing.T) {
		_, err := BuildCommandFromRequest(&LoadRequest{ModelPath: "/model.gguf", Port: 8081}, "")
		assert.Error(t, err)
	})

	t.Run("Invalid port", func(t *testing.T) {
		_, err := BuildCommandFromRequest(&LoadRequest{ModelPath: "/model.gguf", Port: 0}, binDir)
		assert.Error(t, err)
	})
}
