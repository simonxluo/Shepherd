package gpu

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// nvidiaProvider implements GPU detection for NVIDIA GPUs using nvidia-smi.
type nvidiaProvider struct {
	logger Logger
}

// NewNvidiaProvider creates a new NVIDIA GPU provider.
func NewNvidiaProvider(logger Logger) Provider {
	return &nvidiaProvider{logger: logger}
}

func (p *nvidiaProvider) Name() string {
	return "nvidia"
}

func (p *nvidiaProvider) Vendor() string {
	return "NVIDIA"
}

func (p *nvidiaProvider) IsAvailable() bool {
	_, err := exec.LookPath("nvidia-smi")
	return err == nil
}

func (p *nvidiaProvider) Detect(ctx context.Context) ([]Info, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "nvidia-smi",
		"--query-gpu=index,name,memory.total,memory.used,temperature.gpu,utilization.gpu,power.draw,driver_version",
		"--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nvidia-smi failed: %w", err)
	}

	return p.parseOutput(string(output))
}

func (p *nvidiaProvider) Update(ctx context.Context, gpu *Info) error {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "nvidia-smi",
		"--query-gpu=memory.used,temperature.gpu,utilization.gpu,power.draw",
		"--format=csv,noheader,nounits",
		fmt.Sprintf("--id=%d", gpu.Index))
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("nvidia-smi update failed: %w", err)
	}

	return p.parseUpdateOutput(string(output), gpu)
}

func (p *nvidiaProvider) parseOutput(output string) ([]Info, error) {
	var gpus []Info
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) < 8 {
			continue
		}

		index, _ := strconv.Atoi(strings.TrimSpace(fields[0]))
		name := strings.TrimSpace(fields[1])
		totalMemory, _ := strconv.ParseInt(strings.TrimSpace(fields[2]), 10, 64)
		usedMemory, _ := strconv.ParseInt(strings.TrimSpace(fields[3]), 10, 64)
		temperature, _ := strconv.ParseFloat(strings.TrimSpace(fields[4]), 64)
		utilization, _ := strconv.ParseFloat(strings.TrimSpace(fields[5]), 64)
		powerUsage, _ := strconv.ParseFloat(strings.TrimSpace(fields[6]), 64)
		driverVersion := strings.TrimSpace(fields[7])

		gpus = append(gpus, Info{
			Index:         index,
			Name:          name,
			Vendor:        "NVIDIA",
			TotalMemory:   totalMemory * 1024 * 1024,
			UsedMemory:    usedMemory * 1024 * 1024,
			Temperature:   temperature,
			Utilization:   utilization,
			PowerUsage:    powerUsage,
			DriverVersion: driverVersion,
		})

		p.logger.Debugf("Detected NVIDIA GPU[%d]: %s", index, name)
	}

	return gpus, nil
}

func (p *nvidiaProvider) parseUpdateOutput(output string, gpu *Info) error {
	fields := strings.Split(strings.TrimSpace(output), ",")
	if len(fields) < 4 {
		return fmt.Errorf("unexpected output format: %s", output)
	}

	if usedMemory, err := strconv.ParseInt(strings.TrimSpace(fields[0]), 10, 64); err == nil {
		gpu.UsedMemory = usedMemory * 1024 * 1024
	}
	if temperature, err := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64); err == nil {
		gpu.Temperature = temperature
	}
	if utilization, err := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64); err == nil {
		gpu.Utilization = utilization
	}
	if powerUsage, err := strconv.ParseFloat(strings.TrimSpace(fields[3]), 64); err == nil {
		gpu.PowerUsage = powerUsage
	}

	return nil
}
