//go:build amdsmi
// +build amdsmi

// Package gpu provides GPU detection and monitoring abstractions.
// This file contains the AMD GPU provider that uses the AMD SMI Go SDK.
// To use this implementation, build with: go build -tags amdsmi
//
// Prerequisites:
//   - AMD ROCm installed at /opt/rocm
//   - libamdsmi.so available in /opt/rocm/lib or /opt/rocm/lib64
//   - CGO enabled (CGO_ENABLED=1)

package gpu

import (
	"context"
	"fmt"
	"time"
	"unsafe"

	goamdsmi "github.com/ROCm/amdsmi"
)

// amdProvider implements GPU detection for AMD GPUs using the AMD SMI SDK.
type amdProvider struct {
	logger Logger
}

// NewAMDProvider creates a new AMD GPU provider using the AMD SMI SDK.
func NewAMDProvider(logger Logger) Provider {
	return &amdProvider{logger: logger}
}

func (p *amdProvider) Name() string {
	return "amd"
}

func (p *amdProvider) Vendor() string {
	return "AMD"
}

// IsAvailable checks if the AMD SMI SDK is available.
func (p *amdProvider) IsAvailable() bool {
	// Try a quick init/shutdown cycle to verify the library works
	if !goamdsmi.GO_gpu_init() {
		return false
	}
	goamdsmi.GO_gpu_shutdown()
	return true
}

// Detect discovers all AMD GPUs using the AMD SMI SDK.
func (p *amdProvider) Detect(ctx context.Context) ([]Info, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Initialize AMD SMI
	if !goamdsmi.GO_gpu_init() {
		return nil, fmt.Errorf("failed to initialize AMD SMI library")
	}
	defer goamdsmi.GO_gpu_shutdown()

	// Get device count
	deviceCount := goamdsmi.GO_gpu_num_monitor_devices()
	if deviceCount == 0 {
		p.logger.Debugf("No AMD GPUs detected via AMD SMI SDK")
		return []Info{}, nil
	}

	p.logger.Infof("Detected %d AMD GPU device(s) via SDK", deviceCount)

	var gpus []Info
	for i := 0; i < int(deviceCount); i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		gpu := p.detectGPU(i)
		if gpu.Name != "" {
			gpus = append(gpus, gpu)
			p.logger.Debugf("AMD GPU[%d]: %s, Memory: %d MB, Temp: %.1f°C",
				gpu.Index, gpu.Name, gpu.TotalMemory/(1024*1024), gpu.Temperature)
		}
	}

	return gpus, nil
}

// detectGPU detects a single GPU using AMD SMI SDK.
func (p *amdProvider) detectGPU(index int) Info {
	gpu := Info{
		Index:  index,
		Vendor: "AMD",
	}

	// Get device name
	cName := goamdsmi.GO_gpu_dev_name_get(index)
	if cName != nil {
		gpu.Name = cStringToGo(cName)
	}

	// Fallback to device ID if name is empty
	if gpu.Name == "" {
		deviceID := goamdsmi.GO_gpu_dev_id_get(index)
		if deviceID != 0xFFFF {
			gpu.Name = fmt.Sprintf("AMD GPU 0x%04X", deviceID)
		}
	}

	// Get memory information
	memTotal := goamdsmi.GO_gpu_dev_gpu_memory_total_get(index)
	if memTotal != 0xFFFFFFFFFFFFFFFF {
		gpu.TotalMemory = int64(memTotal)
	}

	memUsage := goamdsmi.GO_gpu_dev_gpu_memory_usage_get(index)
	if memUsage != 0xFFFFFFFFFFFFFFFF {
		gpu.UsedMemory = int64(memUsage)
	}

	// Get temperature (with fallback to junction sensor)
	temp := goamdsmi.GO_gpu_dev_temp_metric_get(index, 0, 0) // sensor 0, metric 0
	if temp == 0xFFFFFFFFFFFFFFFF {
		// Try junction temperature (sensor 1)
		temp = goamdsmi.GO_gpu_dev_temp_metric_get(index, 1, 0)
	}
	if temp != 0xFFFFFFFFFFFFFFFF {
		gpu.Temperature = float64(temp)
	}

	// Get utilization
	busyPercent := goamdsmi.GO_gpu_dev_gpu_busy_percent_get(index)
	if busyPercent != 0xFFFFFFFF {
		gpu.Utilization = float64(busyPercent)
	}

	// Get power usage (in microwatts, convert to watts)
	powerUsage := goamdsmi.GO_gpu_dev_power_get(index)
	if powerUsage != 0xFFFFFFFFFFFFFFFF && powerUsage > 0 {
		gpu.PowerUsage = float64(powerUsage) / 1000000.0
	}

	// Get VBIOS version as driver info
	cVbios := goamdsmi.GO_gpu_dev_vbios_version_get(index)
	if cVbios != nil {
		gpu.DriverVersion = cStringToGo(cVbios)
	}

	return gpu
}

// Update refreshes dynamic GPU metrics using AMD SMI SDK.
func (p *amdProvider) Update(ctx context.Context, gpu *Info) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if gpu.Index < 0 {
		return fmt.Errorf("invalid GPU index: %d", gpu.Index)
	}

	// Initialize AMD SMI
	if !goamdsmi.GO_gpu_init() {
		return fmt.Errorf("failed to initialize AMD SMI library for update")
	}
	defer goamdsmi.GO_gpu_shutdown()

	// Create a timeout context
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	idx := gpu.Index

	// Update temperature (with fallback)
	temp := goamdsmi.GO_gpu_dev_temp_metric_get(idx, 0, 0)
	if temp == 0xFFFFFFFFFFFFFFFF {
		temp = goamdsmi.GO_gpu_dev_temp_metric_get(idx, 1, 0)
	}
	if temp != 0xFFFFFFFFFFFFFFFF {
		gpu.Temperature = float64(temp)
	}

	// Update utilization
	busyPercent := goamdsmi.GO_gpu_dev_gpu_busy_percent_get(idx)
	if busyPercent != 0xFFFFFFFF {
		gpu.Utilization = float64(busyPercent)
	}

	// Update memory usage
	memUsage := goamdsmi.GO_gpu_dev_gpu_memory_usage_get(idx)
	if memUsage != 0xFFFFFFFFFFFFFFFF {
		gpu.UsedMemory = int64(memUsage)
	}

	// Update power usage
	powerUsage := goamdsmi.GO_gpu_dev_power_get(idx)
	if powerUsage != 0xFFFFFFFFFFFFFFFF && powerUsage > 0 {
		gpu.PowerUsage = float64(powerUsage) / 1000000.0
	}

	return nil
}

// cStringToGo converts a C string to a Go string.
// This helper function handles the conversion from C char* to Go string.
func cStringToGo(cstr *goamdsmi.C.char) string {
	if cstr == nil {
		return ""
	}
	// Use a reasonable maximum length for GPU names
	const maxLen = 256
	buf := make([]byte, maxLen)
	for i := 0; i < maxLen; i++ {
		b := byte(*(*goamdsmi.C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(cstr)) + uintptr(i))))
		if b == 0 {
			return string(buf[:i])
		}
		buf[i] = b
	}
	return string(buf)
}

// Sensor constants (exported for testing)
const (
	sensorTypeGPU      = 0 // GPU temperature sensor
	sensorTypeJunction = 1 // Junction temperature sensor
	metricTypeCurrent  = 0 // Current value metric

	// Error values returned by AMD SMI on failure
	uint16MaxError = 0xFFFF
	uint32MaxError = 0xFFFFFFFF
	uint64MaxError = 0xFFFFFFFFFFFFFFFF
)

// convertMicrowattsToWatts converts power from microwatts to watts.
func convertMicrowattsToWatts(microwatts uint64) float64 {
	return float64(microwatts) / 1000000.0
}
