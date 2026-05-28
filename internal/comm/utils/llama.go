package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ParseLlamacppDeviceList 解析 llama-bench --list-devices 输出，返回去重后的设备列表
//
// 输入示例:
// Available devices:
//
//	ROCm0: AMD Radeon Graphics (122880 MiB, 114915 MiB free)
//	CUDA0: NVIDIA GeForce RTX 3090 (24576 MiB, 20321 MiB free)
//
// 注意: llama.cpp 存在 bug，可能会输出重复的设备，因此需要去重
//
// 参数:
//   - output: llama-bench 命令的输出
//
// 返回:
//   - []string: 去重后的设备信息列表，每行格式为 "ROCm0: AMD Radeon Graphics (122880 MiB, 114915 MiB free)"
func ParseLlamacppDeviceList(output string) []string {
	var devices []string
	lines := strings.Split(output, "\n")

	// 用于去重，防止 llama-bench 重复输出同一设备
	seenDeviceIDs := make(map[string]bool)

	// 查找 "Available devices:" 标记后的设备列表
	// 必须精确匹配 "Available devices:"，避免匹配调试信息中的 "found"
	inDeviceList := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// 检测设备列表开始 - 必须包含完整的 "Available devices:" 标记
		if strings.Contains(trimmedLine, "Available devices:") {
			inDeviceList = true
			continue
		}

		// 空行结束设备列表
		if inDeviceList && trimmedLine == "" {
			break
		}

		// 解析设备行
		if inDeviceList {
			// 匹配格式: "ROCm0: AMD Radeon Graphics (122880 MiB, 114915 MiB free)"
			// 或 "CUDA0: NVIDIA GeForce RTX 3090 (24576 MiB, 20321 MiB free)"
			// 必须包含设备类型前缀 (ROCm/CUDA 等) 和冒号
			if strings.Contains(trimmedLine, ":") {
				parts := strings.SplitN(trimmedLine, ":", 2)
				if len(parts) == 2 {
					deviceID := strings.TrimSpace(parts[0])
					// 验证设备前缀格式: ROCm0, CUDA0, Vulkan0, Metal0 等
					if isValidLlamacppDevicePrefix(deviceID) {
						// 检查是否已经处理过该设备 ID（去重）
						if seenDeviceIDs[deviceID] {
							// 跳过重复设备
							continue
						}
						seenDeviceIDs[deviceID] = true

						// 保留完整的设备信息行，以便前端显示
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

// isValidLlamacppDevicePrefix 验证设备前缀是否有效
// 有效的前缀格式: CUDA, ROCm, Vulkan, Metal, SYCL, CPU 后跟可选数字
func isValidLlamacppDevicePrefix(prefix string) bool {
	return devicePrefixRe.MatchString(prefix)
}

// GetLlamacppDeviceList 使用 llama-bench 获取设备列表
//
// 参数:
//   - benchPath: llama-bench 可执行文件路径
//
// 返回:
//   - []string: 去重后的设备信息列表
//   - error: 错误信息
func GetLlamacppDeviceList(benchPath string) ([]string, error) {
	// 执行 llama-bench --list-devices
	cmd := exec.Command(benchPath, "--list-devices")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute %s --list-devices: %w, output: %s", benchPath, err, string(output))
	}

	// 解析输出获取设备列表
	devices := ParseLlamacppDeviceList(string(output))

	return devices, nil
}

// FindLlamacppBinary 在指定路径中查找 llama.cpp 可执行文件
//
// 参数:
//   - binPath: llama.cpp 路径（可以是目录或可执行文件）
//   - binaryType: 二进制文件类型 ("server", "cli", "bench" 等)
//
// 返回:
//   - string: 找到的可执行文件完整路径，如果未找到返回空字符串
//
// 查找逻辑:
//  1. 如果 binPath 是文件且可执行，直接返回
//  2. 如果 binPath 是目录，在目录中查找对应的二进制文件
//  3. 如果未找到，尝试在 bin/ 子目录中查找
//
// 支持的二进制文件名:
//   - server: llama-server, server
//   - cli: llama-cli, cli, main
//   - bench: llama-bench, bench
func FindLlamacppBinary(binPath string, binaryType string) string {
	// 定义不同类型的二进制文件名
	var possibleNames []string
	switch strings.ToLower(binaryType) {
	case "server":
		possibleNames = []string{"llama-server", "server"}
	case "cli":
		possibleNames = []string{"llama-cli", "cli", "main"}
	case "bench":
		possibleNames = []string{"llama-bench", "bench"}
	default:
		// 默认查找所有可能的二进制文件
		possibleNames = []string{"llama-server", "server", "llama-cli", "cli", "main", "llama-bench", "bench"}
	}

	// 1. 检查路径是否为文件
	if info, err := os.Stat(binPath); err == nil && info.Mode().IsRegular() {
		// 如果是文件且可执行，直接返回
		if info.Mode().Perm()&0111 != 0 {
			return binPath
		}
	}

	// 2. 如果是目录，在目录中查找二进制文件
	if info, err := os.Stat(binPath); err == nil && info.IsDir() {
		// 先在当前目录中查找
		for _, name := range possibleNames {
			candidatePath := filepath.Join(binPath, name)
			if info, err := os.Stat(candidatePath); err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0 {
				return candidatePath
			}
		}

		// 3. 尝试在 bin 和 build/bin 子目录中查找
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
