//go:build !amdsmi
// +build !amdsmi

package gpu

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// amdProvider implements GPU detection for AMD GPUs using ROCm tools.
type amdProvider struct {
	logger Logger
}

// NewAMDProvider creates a new AMD GPU provider.
func NewAMDProvider(logger Logger) Provider {
	return &amdProvider{logger: logger}
}

func (p *amdProvider) Name() string {
	return "amd"
}

func (p *amdProvider) Vendor() string {
	return "AMD"
}

func (p *amdProvider) IsAvailable() bool {
	_, err := exec.LookPath("rocm-smi")
	return err == nil
}

func (p *amdProvider) Detect(ctx context.Context) ([]Info, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	gpus, err := p.detectViaRocminfo(ctx)
	if err == nil && len(gpus) > 0 {
		return gpus, nil
	}

	return p.detectViaRocmSMI(ctx)
}

func (p *amdProvider) Update(ctx context.Context, gpu *Info) error {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rocm-smi",
		"--showmeminfo", "vram", "--showtemp", "--showuse",
		"-d", fmt.Sprintf("%d", gpu.Index))
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("rocm-smi update failed: %w", err)
	}

	return p.parseUpdateOutput(string(output), gpu)
}

func (p *amdProvider) detectViaRocminfo(ctx context.Context) ([]Info, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if _, err := exec.LookPath("rocminfo"); err != nil {
		return nil, fmt.Errorf("rocminfo not found")
	}

	cmd := exec.CommandContext(ctx, "rocminfo")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("rocminfo failed: %w", err)
	}

	return p.parseRocminfoOutput(string(output))
}

func (p *amdProvider) detectViaRocmSMI(ctx context.Context) ([]Info, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rocm-smi", "--showproductname")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("rocm-smi failed: %w", err)
	}

	return p.parseRocmSMIOutput(string(output))
}

func (p *amdProvider) parseRocminfoOutput(output string) ([]Info, error) {
	var gpus []Info
	lines := strings.Split(output, "\n")
	var currentGPU *Info
	var deviceType string
	gpuIndex := 0

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		if strings.HasPrefix(line, "******") && i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			if strings.HasPrefix(nextLine, "Agent") {
				if currentGPU != nil && deviceType == "GPU" {
					currentGPU.Index = gpuIndex
					p.enrichGPUInfo(currentGPU)
					gpus = append(gpus, *currentGPU)
					p.logger.Debugf("Detected AMD GPU[%d]: %s, Memory: %d MB", currentGPU.Index, currentGPU.Name, currentGPU.TotalMemory/1024/1024)
					gpuIndex++
				}
				currentGPU = &Info{Vendor: "AMD"}
				deviceType = ""
			}
			continue
		}

		if currentGPU == nil {
			continue
		}

		if strings.HasPrefix(line, "Marketing Name:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				currentGPU.Name = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Device Type:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				deviceType = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Name:") && currentGPU.Name == "" {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[1])
				if !strings.HasPrefix(name, "amdgcn") {
					currentGPU.Name = name
				}
			}
		} else if strings.HasPrefix(line, "Pool 1:") {
			for j := i + 1; j < len(lines) && j < i+10; j++ {
				memLine := strings.TrimSpace(lines[j])
				if strings.HasPrefix(memLine, "Size:") {
					parts := strings.SplitN(memLine, ":", 2)
					if len(parts) == 2 {
						sizeStr := strings.TrimSpace(parts[1])
						if mbIdx := strings.Index(sizeStr, "MB"); mbIdx != -1 {
							numStr := strings.TrimSpace(sizeStr[:mbIdx])
							if sizeMB, err := strconv.ParseInt(numStr, 10, 64); err == nil {
								currentGPU.TotalMemory = sizeMB * 1024 * 1024
							}
						} else if kbIdx := strings.Index(sizeStr, "KB"); kbIdx != -1 {
							numStr := strings.TrimSpace(sizeStr[:kbIdx])
							if sizeKB, err := strconv.ParseInt(numStr, 10, 64); err == nil {
								currentGPU.TotalMemory = sizeKB * 1024
							}
						}
					}
					break
				}
			}
		}
	}

	if currentGPU != nil && deviceType == "GPU" {
		currentGPU.Index = gpuIndex
		p.enrichGPUInfo(currentGPU)
		gpus = append(gpus, *currentGPU)
		p.logger.Debugf("Detected AMD GPU[%d]: %s, Memory: %d MB", currentGPU.Index, currentGPU.Name, currentGPU.TotalMemory/1024/1024)
	}

	return gpus, nil
}

// enrichGPUInfo enriches GPU info with additional data from rocm-smi
func (p *amdProvider) enrichGPUInfo(gpu *Info) {
	if drv := p.detectDriverVersion(); drv != "" {
		gpu.DriverVersion = drv
	}

	if gpu.TotalMemory == 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "rocm-smi", "--showmeminfo", "vram", "-d", fmt.Sprintf("%d", gpu.Index))
		if output, err := cmd.Output(); err == nil {
			p.parseMemoryOutput(string(output), gpu)
		}
	}
}

// detectDriverVersion detects AMD GPU driver version
func (p *amdProvider) detectDriverVersion() string {
	cmd := exec.Command("modinfo", "amdgpu")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "version:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}

	if data, err := os.ReadFile("/sys/module/amdgpu/version"); err == nil {
		return strings.TrimSpace(string(data))
	}

	return ""
}

// parseMemoryOutput parses rocm-smi memory output
func (p *amdProvider) parseMemoryOutput(output string, gpu *Info) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "VRAM Total Memory (B):") {
			parts := strings.Split(line, "VRAM Total Memory (B):")
			if len(parts) == 2 {
				valStr := strings.TrimSpace(parts[1])
				if val, err := strconv.ParseInt(valStr, 10, 64); err == nil {
					gpu.TotalMemory = val
				}
			}
		}
		if strings.Contains(line, "VRAM Total Used Memory (B):") {
			parts := strings.Split(line, "VRAM Total Used Memory (B):")
			if len(parts) == 2 {
				valStr := strings.TrimSpace(parts[1])
				if val, err := strconv.ParseInt(valStr, 10, 64); err == nil {
					gpu.UsedMemory = val
				}
			}
		}
	}
}

func (p *amdProvider) parseRocmSMIOutput(output string) ([]Info, error) {
	var gpus []Info
	lines := strings.Split(output, "\n")
	index := 0

	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "card series") {
			name := p.extractGPUName(line)
			if name != "" {
				gpus = append(gpus, Info{
					Index:       index,
					Name:        name,
					Vendor:      "AMD",
					TotalMemory: 0,
					UsedMemory:  0,
					Temperature: 0,
					Utilization: 0,
				})
				p.logger.Debugf("Detected AMD GPU[%d]: %s", index, name)
				index++
			}
		}
	}

	return gpus, nil
}

func (p *amdProvider) extractGPUName(line string) string {
	if idx := strings.Index(line, ":"); idx != -1 {
		return strings.TrimSpace(line[idx+1:])
	}
	return strings.TrimSpace(line)
}

func (p *amdProvider) parseUpdateOutput(output string, gpu *Info) error {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "VRAM Total Memory (B):") {
			parts := strings.Split(line, "VRAM Total Memory (B):")
			if len(parts) == 2 {
				valStr := strings.TrimSpace(parts[1])
				if val, err := strconv.ParseInt(valStr, 10, 64); err == nil {
					gpu.TotalMemory = val
				}
			}
		}
		if strings.Contains(line, "VRAM Total Used Memory (B):") {
			parts := strings.Split(line, "VRAM Total Used Memory (B):")
			if len(parts) == 2 {
				valStr := strings.TrimSpace(parts[1])
				if val, err := strconv.ParseInt(valStr, 10, 64); err == nil {
					gpu.UsedMemory = val
				}
			}
		}
		if strings.Contains(line, "GPU use (%):") {
			parts := strings.Split(line, "GPU use (%):")
			if len(parts) == 2 {
				valStr := strings.TrimSpace(parts[1])
				if val, err := strconv.ParseFloat(valStr, 64); err == nil {
					gpu.Utilization = val
				}
			}
		}
		if strings.Contains(line, "Temperature") && strings.Contains(line, "(C):") {
			parts := strings.Split(line, "(C):")
			if len(parts) == 2 {
				valStr := strings.TrimSpace(parts[1])
				if val, err := strconv.ParseFloat(valStr, 64); err == nil {
					gpu.Temperature = val
				}
			}
		}
	}
	return nil
}

// Sensor constants for AMD GPU detection
const (
	sensorTypeGPU      = 0 // GPU temperature sensor
	sensorTypeJunction = 1 // Junction temperature sensor
	metricTypeCurrent  = 0 // Current value metric
)

// Error constants for AMD SMI operations
const (
	uint16MaxError = 0xFFFF
	uint32MaxError = 0xFFFFFFFF
	uint64MaxError = 0xFFFFFFFFFFFFFFFF
)

// convertMicrowattsToWatts converts power from microwatts to watts
func convertMicrowattsToWatts(microwatts uint64) float64 {
	return float64(microwatts) / 1000000.0
}
