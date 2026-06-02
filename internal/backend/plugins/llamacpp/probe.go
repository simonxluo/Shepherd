package llamacpp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/utils"
)

// ProbeResult contains llama.cpp installation probe details.
type ProbeResult struct {
	Path      string   `json:"path"`
	Binary    string   `json:"binary"`
	Version   string   `json:"version"`
	Warnings  []string `json:"warnings"`
	Available bool     `json:"available"`
}

// ProbeInstallation probes a llama.cpp installation path for llama-server.
func ProbeInstallation(path string) (*ProbeResult, error) {
	result := &ProbeResult{
		Path:     path,
		Warnings: []string{},
	}

	serverBin := utils.FindLlamacppBinary(path, "server")
	if serverBin == "" {
		result.Warnings = append(result.Warnings, fmt.Sprintf("llama-server not found in path: %s", path))
		return result, nil
	}

	info, err := os.Stat(serverBin)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to stat llama-server: %v", err))
		return result, nil
	}
	if !info.Mode().IsRegular() {
		result.Warnings = append(result.Warnings, fmt.Sprintf("llama-server is not a regular file: %s", serverBin))
		return result, nil
	}
	if info.Mode().Perm()&0111 == 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("llama-server is not executable: %s", serverBin))
		return result, nil
	}

	result.Binary = serverBin
	result.Available = true

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, serverBin, "--version")
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		result.Warnings = append(result.Warnings, "llama-server --version timed out")
		return result, nil
	}
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to run llama-server --version: %v", err))
		return result, nil
	}

	result.Version = strings.TrimSpace(string(output))
	if result.Version == "" {
		result.Warnings = append(result.Warnings, "llama-server --version returned empty output")
	}

	return result, nil
}
