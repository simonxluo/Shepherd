package utils

import (
	"strings"
	"testing"
)

func TestWriteStringQuietly(t *testing.T) {
	var writer strings.Builder
	WriteStringQuietly(&writer, "test message")
	if writer.String() != "test message" {
		t.Errorf("Expected 'test message', got '%s'", writer.String())
	}
}

func TestKillQuietly(t *testing.T) {
	// Just test that the function exists and doesn't panic
	KillQuietly(nil)
}

func TestSignalQuietly(t *testing.T) {
	// Just test that the function exists
	SignalQuietly(nil, 0)
}
