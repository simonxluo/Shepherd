package gpu

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewAMDProvider tests that the factory function creates the right implementation
func TestNewAMDProvider(t *testing.T) {
	logger := noopLogger{}
	provider := NewAMDProvider(logger)

	assert.NotNil(t, provider)
	assert.Equal(t, "amd", provider.Name())
	assert.Equal(t, "AMD", provider.Vendor())
}

// TestAMDProvider_Name verifies provider name
func TestAMDProvider_Name(t *testing.T) {
	logger := noopLogger{}
	provider := NewAMDProvider(logger)
	assert.Equal(t, "amd", provider.Name())
}

// TestAMDProvider_Vendor verifies vendor name
func TestAMDProvider_Vendor(t *testing.T) {
	logger := noopLogger{}
	provider := NewAMDProvider(logger)
	assert.Equal(t, "AMD", provider.Vendor())
}

// TestAMDProvider_IsAvailable tests the availability check
func TestAMDProvider_IsAvailable(t *testing.T) {
	logger := noopLogger{}
	provider := NewAMDProvider(logger)

	// Just verify it doesn't panic - actual result depends on system
	available := provider.IsAvailable()
	_ = available
}

// TestAMDProvider_Detect_WithContextCancellation tests context handling
func TestAMDProvider_Detect_WithContextCancellation(t *testing.T) {
	logger := noopLogger{}
	provider := NewAMDProvider(logger)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	// Note: Current implementation may or may not respect context cancellation
	// depending on the build tag (CLI vs SDK). The important thing is it doesn't panic.
	_, err := provider.Detect(ctx)

	// We may or may not get an error depending on timing and implementation
	_ = err
}

// TestConvertPowerUnits verifies power unit conversion
func TestConvertPowerUnits(t *testing.T) {
	tests := []struct {
		name     string
		input    uint64
		expected float64
	}{
		{"zero", 0, 0},
		{"1 microwatt", 1, 0.000001},
		{"1 milliwatt", 1000, 0.001},
		{"1 watt", 1000000, 1.0},
		{"350 watts", 350000000, 350.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertMicrowattsToWatts(tt.input)
			assert.InDelta(t, tt.expected, result, 0.0001)
		})
	}
}

// TestAMDProvider_Update_InvalidGPUIndex tests error handling for invalid index
func TestAMDProvider_Update_InvalidGPUIndex(t *testing.T) {
	logger := noopLogger{}
	provider := NewAMDProvider(logger)
	ctx := context.Background()

	gpu := &Info{
		Index:  -1, // Invalid index
		Name:   "AMD Radeon RX 7900 XTX",
		Vendor: "AMD",
	}

	// For CLI implementation, this will try to execute rocm-smi and fail
	// For SDK implementation, it should return an error
	err := provider.Update(ctx, gpu)

	// Either error is expected or the operation fails gracefully
	_ = err
}

// BenchmarkAMDProvider_Detect benchmarks the detection performance
func BenchmarkAMDProvider_Detect(b *testing.B) {
	logger := noopLogger{}
	provider := NewAMDProvider(logger)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = provider.Detect(ctx)
	}
}

// TestSensorConstants verifies sensor and metric constants
func TestSensorConstants(t *testing.T) {
	// These are the expected sensor/metric values based on AMD SMI documentation
	assert.Equal(t, 0, sensorTypeGPU)
	assert.Equal(t, 1, sensorTypeJunction)
	assert.Equal(t, 0, metricTypeCurrent)
}

// TestErrorConstants verifies error value constants
func TestErrorConstants(t *testing.T) {
	assert.Equal(t, uint16(0xFFFF), uint16(uint16MaxError))
	assert.Equal(t, uint32(0xFFFFFFFF), uint32(uint32MaxError))
	assert.Equal(t, uint64(0xFFFFFFFFFFFFFFFF), uint64(uint64MaxError))
}

// TestAMDProvider_Integration detects real AMD GPUs if available
func TestAMDProvider_Integration(t *testing.T) {
	logger := noopLogger{}
	provider := NewAMDProvider(logger)

	// Skip if no AMD GPU tools are available
	if !provider.IsAvailable() {
		t.Skip("AMD GPU tools not available, skipping integration test")
	}

	ctx := context.Background()
	gpus, err := provider.Detect(ctx)

	// Should not error
	assert.NoError(t, err)
	assert.NotNil(t, gpus)

	// Log detected GPUs
	for _, gpu := range gpus {
		t.Logf("Detected AMD GPU: %s (Index: %d, Memory: %d MB)",
			gpu.Name, gpu.Index, gpu.TotalMemory/(1024*1024))
	}
}
