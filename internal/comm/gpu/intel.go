package gpu

import (
	"context"
	"os/exec"
	"strings"

	"github.com/shirou/gopsutil/v3/host"
)

// intelProvider implements GPU detection for Intel GPUs.
type intelProvider struct {
	logger Logger
}

// NewIntelProvider creates a new Intel GPU provider.
func NewIntelProvider(logger Logger) Provider {
	return &intelProvider{logger: logger}
}

func (p *intelProvider) Name() string {
	return "intel"
}

func (p *intelProvider) Vendor() string {
	return "Intel"
}

func (p *intelProvider) IsAvailable() bool {
	// Check for Intel GPU kernel module
	hostStat, err := host.Info()
	if err != nil {
		return false
	}

	kernelInfo := strings.ToLower(hostStat.KernelVersion)
	return strings.Contains(kernelInfo, "i915") || strings.Contains(kernelInfo, "intel")
}

func (p *intelProvider) Detect(ctx context.Context) ([]Info, error) {
	if !p.IsAvailable() {
		return nil, nil
	}

	var gpus []Info

	if name := p.detectGPUName(); name != "" {
		gpus = append(gpus, Info{
			Index:       0,
			Name:        name,
			Vendor:      "Intel",
			TotalMemory: 0,
			UsedMemory:  0,
			Temperature: 0,
			Utilization: 0,
		})
		p.logger.Debugf("Detected Intel GPU: %s", name)
	} else {
		gpus = append(gpus, Info{
			Index:       0,
			Name:        "Intel Integrated Graphics",
			Vendor:      "Intel",
			TotalMemory: 0,
			UsedMemory:  0,
			Temperature: 0,
			Utilization: 0,
		})
		p.logger.Debugf("Detected Intel Integrated Graphics")
	}

	return gpus, nil
}

func (p *intelProvider) Update(ctx context.Context, gpu *Info) error {
	return nil
}

func (p *intelProvider) detectGPUName() string {
	cmd := exec.Command("lspci", "-nn")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "vga") && strings.Contains(lower, "intel") {
				if idx := strings.Index(line, ": "); idx != -1 {
					return strings.TrimSpace(line[idx+2:])
				}
			}
		}
	}
	return ""
}
