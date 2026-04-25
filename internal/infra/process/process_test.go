package process

import (
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

