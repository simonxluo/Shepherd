// Package gpu provides GPU detection and monitoring abstractions.
// It implements the Provider Pattern to support multiple GPU vendors (NVIDIA, AMD, Intel)
// with clean separation of concerns and easy extensibility.
package gpu

import (
	"context"
	"time"
)

// Provider defines the interface for GPU information providers.
// Each GPU vendor (NVIDIA, AMD, Intel, etc.) implements this interface.
type Provider interface {
	// Name returns the provider's name (e.g., "nvidia", "amd", "intel")
	Name() string

	// Vendor returns the GPU vendor name (e.g., "NVIDIA", "AMD", "Intel")
	Vendor() string

	// IsAvailable checks if the provider can detect GPUs on this system.
	// This should be a lightweight check (e.g., command existence).
	IsAvailable() bool

	// Detect discovers all GPUs of this vendor on the system.
	// Returns a slice of GPUInfo with initial static information.
	Detect(ctx context.Context) ([]Info, error)

	// Update refreshes dynamic GPU metrics (temperature, utilization, memory usage).
	// The passed GPUInfo is updated in-place.
	Update(ctx context.Context, gpu *Info) error
}

// Detector manages multiple GPU providers and coordinates detection/monitoring.
// It acts as a facade over all GPU providers.
type Detector struct {
	providers []Provider
	logger    Logger
}

// Logger interface for GPU package logging.
// This avoids direct dependency on internal/logger.
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// noopLogger is a no-op implementation of Logger.
type noopLogger struct{}

func (n noopLogger) Debugf(format string, args ...interface{}) {}
func (n noopLogger) Infof(format string, args ...interface{})  {}
func (n noopLogger) Errorf(format string, args ...interface{}) {}

// Config contains configuration for the GPU detector.
type Config struct {
	// Timeout for detection operations
	DetectionTimeout time.Duration
	// Timeout for update operations
	UpdateTimeout time.Duration
	// Logger for logging (optional)
	Logger Logger
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		DetectionTimeout: 10 * time.Second,
		UpdateTimeout:    5 * time.Second,
	}
}

// NewDetector creates a new GPU detector with all registered providers.
func NewDetector(cfg *Config) *Detector {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	logger := cfg.Logger
	if logger == nil {
		logger = noopLogger{}
	}

	return &Detector{
		providers: registerProviders(logger),
		logger:    logger,
	}
}

// DetectAll discovers all GPUs from all available providers.
// It runs detection concurrently for all available providers.
func (d *Detector) DetectAll(ctx context.Context) ([]Info, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var allGPUs []Info

	for _, provider := range d.providers {
		if !provider.IsAvailable() {
			d.logger.Debugf("GPU provider %s is not available", provider.Name())
			continue
		}

		gpus, err := provider.Detect(ctx)
		if err != nil {
			d.logger.Errorf("Failed to detect %s GPUs: %v", provider.Name(), err)
			continue
		}

		if len(gpus) > 0 {
			d.logger.Infof("Detected %d %s GPU(s)", len(gpus), provider.Vendor())
			allGPUs = append(allGPUs, gpus...)
		}
	}

	return allGPUs, nil
}

// Update refreshes dynamic metrics for a specific GPU.
func (d *Detector) Update(ctx context.Context, gpu *Info) error {
	if ctx == nil {
		ctx = context.Background()
	}

	for _, provider := range d.providers {
		if provider.Vendor() == gpu.Vendor {
			if !provider.IsAvailable() {
				return nil
			}
			return provider.Update(ctx, gpu)
		}
	}

	d.logger.Debugf("No provider found for GPU vendor: %s", gpu.Vendor)
	return nil
}

// GetAvailableProviders returns a list of available provider names.
func (d *Detector) GetAvailableProviders() []string {
	var names []string
	for _, provider := range d.providers {
		if provider.IsAvailable() {
			names = append(names, provider.Name())
		}
	}
	return names
}

// registerProviders returns all registered GPU providers.
// This is the central registry for GPU providers.
func registerProviders(logger Logger) []Provider {
	return []Provider{
		NewNvidiaProvider(logger),
		NewAMDProvider(logger),
		NewIntelProvider(logger),
	}
}
