package gpu

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// ROCmGPUInfo contains GPU information parsed from rocminfo output.
type ROCmGPUInfo struct {
	DeviceIndex   int
	Name          string
	MarketingName string
	TotalKB       int64 // Total memory in KB
}

// ParseROCmInfo parses the output of the `rocminfo` command and returns
// detected GPU devices with their memory information.
func ParseROCmInfo(output string) []ROCmGPUInfo {
	lines := strings.Split(output, "\n")
	var gpuMemories []ROCmGPUInfo

	currentAgentType := ""
	var currentGPU ROCmGPUInfo
	inPoolInfo := false
	poolIndex := 0

	agentRe := regexp.MustCompile(`^Agent\s+\d+\s*$`)
	deviceTypeGPURe := regexp.MustCompile(`^\s*Device Type:\s+GPU`)
	nameRe := regexp.MustCompile(`^\s*Name:\s+(\S+)`)
	poolInfoRe := regexp.MustCompile(`^\s*Pool Info:\s*$`)
	poolRe := regexp.MustCompile(`Pool\s+(\d+)`)
	sizeRe := regexp.MustCompile(`^\s*Size:\s+(\d+)\s*\(0x[0-9a-fA-F]+\)\s*KB`)

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Detect new Agent
		if agentRe.MatchString(trimmedLine) {
			// Save previous GPU info if applicable
			if currentAgentType == "GPU" && currentGPU.TotalKB > 0 {
				currentGPU.DeviceIndex = len(gpuMemories)
				gpuMemories = append(gpuMemories, currentGPU)
			}
			// Reset state
			currentAgentType = ""
			currentGPU = ROCmGPUInfo{}
			inPoolInfo = false
			poolIndex = 0
		}

		// Detect Device Type: GPU
		if deviceTypeGPURe.MatchString(line) {
			currentAgentType = "GPU"
		}

		// Parse Name
		if matches := nameRe.FindStringSubmatch(line); len(matches) > 1 {
			currentGPU.Name = matches[1]
		}

		// Parse Marketing Name
		if strings.Contains(line, "Marketing Name:") {
			marketingName := strings.ReplaceAll(line, "Marketing Name:", "")
			currentGPU.MarketingName = strings.TrimSpace(marketingName)
		}

		// Detect Pool Info start
		if poolInfoRe.MatchString(line) {
			inPoolInfo = true
			poolIndex = 0
		}

		// Detect Pool number
		if inPoolInfo {
			if matches := poolRe.FindStringSubmatch(trimmedLine); len(matches) > 1 {
				if idx, err := strconv.Atoi(matches[1]); err == nil {
					poolIndex = idx
				}
			}
		}

		// Parse Pool Size (only in Pool 1)
		if inPoolInfo && poolIndex == 1 {
			if matches := sizeRe.FindStringSubmatch(line); len(matches) > 1 {
				if kb, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
					currentGPU.TotalKB = kb
				}
			}
		}
	}

	// Save last GPU info
	if currentAgentType == "GPU" && currentGPU.TotalKB > 0 {
		currentGPU.DeviceIndex = len(gpuMemories)
		gpuMemories = append(gpuMemories, currentGPU)
	}

	return gpuMemories
}

// DetectROCmGPUs runs rocminfo and parses the output to detect AMD GPUs.
// Returns nil, nil if rocminfo is not available or produces no output.
func DetectROCmGPUs() ([]ROCmGPUInfo, error) {
	cmd := exec.Command("rocminfo")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	gpus := ParseROCmInfo(string(output))
	return gpus, nil
}
