package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ParseLlamacppDeviceList parses the output of llama-bench --list-devices and returns a deduplicated device list.
//
// Example input:
// Available devices:
//
//	ROCm0: AMD Radeon Graphics (122880 MiB, 114915 MiB free)
//	CUDA0: NVIDIA GeForce RTX 3090 (24576 MiB, 20321 MiB free)
//
// Note: llama.cpp has a bug that may output duplicate devices, so deduplication is required.
//
// Parameters:
//   - output: the output of the llama-bench command
//
// Returns:
//   - []string: deduplicated device info list, each line formatted as "ROCm0: AMD Radeon Graphics (122880 MiB, 114915 MiB free)"
func ParseLlamacppDeviceList(output string) []string {
	var devices []string
	lines := strings.Split(output, "\n")

	// Used for deduplication to prevent llama-bench from outputting the same device multiple times
	seenDeviceIDs := make(map[string]bool)

	// Look for the "Available devices:" marker followed by the device list
	// Must match "Available devices:" exactly to avoid matching debug info containing "found"
	inDeviceList := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Detect device list start - must contain the complete "Available devices:" marker
		if strings.Contains(trimmedLine, "Available devices:") {
			inDeviceList = true
			continue
		}

		// Empty line ends the device list
		if inDeviceList && trimmedLine == "" {
			break
		}

		// Parse device line
		if inDeviceList {
			// Match format: "ROCm0: AMD Radeon Graphics (122880 MiB, 114915 MiB free)"
			// or "CUDA0: NVIDIA GeForce RTX 3090 (24576 MiB, 20321 MiB free)"
			// Must contain a device type prefix (ROCm/CUDA etc.) and colon
			if strings.Contains(trimmedLine, ":") {
				parts := strings.SplitN(trimmedLine, ":", 2)
				if len(parts) == 2 {
					deviceID := strings.TrimSpace(parts[0])
					// Validate device prefix format: ROCm0, CUDA0, Vulkan0, Metal0 etc.
					if isValidLlamacppDevicePrefix(deviceID) {
						// Check if this device ID has already been processed (dedup)
						if seenDeviceIDs[deviceID] {
							// Skip duplicate device
							continue
						}
						seenDeviceIDs[deviceID] = true

						// Keep the full device info line for frontend display
						devices = append(devices, trimmedLine)
					}
				}
			}
		}
	}

	return devices
}

// devicePrefixRe matches known llama.cpp device prefixes: CUDA0, ROCm0, Vulkan0, Metal0, SYCL0, CPU0, etc.
var devicePrefixRe = regexp.MustCompile(`^(CUDA|ROCm|Vulkan|Metal|SYCL|CPU)[0-9]*$`)

// isValidLlamacppDevicePrefix validates whether a device prefix is valid.
// Valid prefix format: CUDA, ROCm, Vulkan, Metal, SYCL, or CPU followed by an optional number.
func isValidLlamacppDevicePrefix(prefix string) bool {
	return devicePrefixRe.MatchString(prefix)
}

// GetLlamacppDeviceList uses llama-bench to retrieve the device list.
//
// Parameters:
//   - benchPath: path to the llama-bench executable
//
// Returns:
//   - []string: deduplicated device info list
//   - error: error information
func GetLlamacppDeviceList(benchPath string) ([]string, error) {
	// Execute llama-bench --list-devices
	cmd := exec.Command(benchPath, "--list-devices")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute %s --list-devices: %w, output: %s", benchPath, err, string(output))
	}

	// Parse output to get device list
	devices := ParseLlamacppDeviceList(string(output))

	return devices, nil
}

// FindLlamacppBinary searches for a llama.cpp executable in the specified path.
//
// Parameters:
//   - binPath: llama.cpp path (can be a directory or executable file)
//   - binaryType: binary file type ("server", "cli", "bench", etc.)
//
// Returns:
//   - string: full path of the found executable, or empty string if not found
//
// Search logic:
//  1. If binPath is an executable file, return it directly
//  2. If binPath is a directory, search for the corresponding binary in it
//  3. If not found, try searching in bin/ and build/bin/ subdirectories
//
// Supported binary names:
//   - server: llama-server, server
//   - cli: llama-cli, cli, main
//   - bench: llama-bench, bench
func FindLlamacppBinary(binPath string, binaryType string) string {
	// Define possible binary names for each type
	var possibleNames []string
	switch strings.ToLower(binaryType) {
	case "server":
		possibleNames = []string{"llama-server", "server"}
	case "cli":
		possibleNames = []string{"llama-cli", "cli", "main"}
	case "bench":
		possibleNames = []string{"llama-bench", "bench"}
	default:
		// Default: search for all possible binaries
		possibleNames = []string{"llama-server", "server", "llama-cli", "cli", "main", "llama-bench", "bench"}
	}

	// 1. Check if the path is a file
	if info, err := os.Stat(binPath); err == nil && info.Mode().IsRegular() {
		// If it's a file and executable, return directly
		if info.Mode().Perm()&0111 != 0 {
			return binPath
		}
	}

	// 2. If it's a directory, search for binaries within it
	if info, err := os.Stat(binPath); err == nil && info.IsDir() {
		// Search in the current directory first
		for _, name := range possibleNames {
			candidatePath := filepath.Join(binPath, name)
			if info, err := os.Stat(candidatePath); err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0 {
				return candidatePath
			}
		}

		// 3. Try searching in bin/ and build/bin/ subdirectories
		for _, subDir := range []string{"bin", "build/bin"} {
			dirPath := filepath.Join(binPath, subDir)
			for _, name := range possibleNames {
				candidatePath := filepath.Join(dirPath, name)
				if info, err := os.Stat(candidatePath); err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0 {
					return candidatePath
				}
			}
		}

	}

	return ""
}
