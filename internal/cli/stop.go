package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var flagForce bool

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止 Shepherd 进程",
	Long: `停止所有正在运行的 Shepherd 相关进程。

包括后端服务进程、前端开发服务器、Vite 开发服务器等。

使用 --force 参数会立即强制终止（SIGKILL），否则先尝试优雅停止（SIGTERM）。`,
	Example: `  shepherd stop          # 优雅停止
  shepherd stop --force  # 强制停止`,
	RunE: runStop,
}

func init() {
	stopCmd.Flags().BoolVarP(&flagForce, "force", "f", false, "强制终止 (SIGKILL)")
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	fmt.Println("停止 Shepherd 进程...")

	stopped := 0

	stopped += stopShepherdProcesses(flagForce)

	stopped += stopWebDevProcess(flagForce)

	stopped += stopViteProcesses(flagForce)

	if stopped == 0 {
		fmt.Println("没有正在运行的 Shepherd 进程")
	} else {
		fmt.Printf("已停止 %d 个进程\n", stopped)
	}

	return nil
}

func stopShepherdProcesses(force bool) int {
	cmd := exec.Command("pgrep", "-f", "shepherd")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	pids := strings.Fields(strings.TrimSpace(string(output)))
	selfPid := os.Getpid()
	stopped := 0

	for _, pidStr := range pids {
		var pid int
		fmt.Sscanf(pidStr, "%d", &pid)
		if pid == selfPid || pid == 0 {
			continue
		}

		if force {
			fmt.Printf("强制终止 Shepherd 进程 (PID: %d)\n", pid)
			syscall.Kill(pid, syscall.SIGKILL)
		} else {
			fmt.Printf("停止 Shepherd 进程 (PID: %d)\n", pid)
			syscall.Kill(pid, syscall.SIGTERM)
		}
		stopped++
	}

	if !force && stopped > 0 {
		time.Sleep(3 * time.Second)
		for _, pidStr := range pids {
			var pid int
			fmt.Sscanf(pidStr, "%d", &pid)
			if pid == selfPid || pid == 0 {
				continue
			}
			if processExists(pid) {
				fmt.Printf("进程 %d 仍在运行，强制终止\n", pid)
				syscall.Kill(pid, syscall.SIGKILL)
			}
		}
	}

	return stopped
}

func stopWebDevProcess(force bool) int {
	pidFile := fmt.Sprintf("%s/shepherd-web-dev.pid", os.TempDir())
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0
	}

	var pid int
	fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid)
	if pid == 0 {
		return 0
	}

	if !processExists(pid) {
		os.Remove(pidFile)
		return 0
	}

	sig := syscall.SIGTERM
	if force {
		sig = syscall.SIGKILL
	}

	fmt.Printf("停止前端开发服务器 (PID: %d)\n", pid)
	syscall.Kill(pid, sig)
	os.Remove(pidFile)

	return 1
}

func stopViteProcesses(force bool) int {
	cmd := exec.Command("pgrep", "-f", "vite")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	pids := strings.Fields(strings.TrimSpace(string(output)))
	stopped := 0

	sig := syscall.SIGTERM
	if force {
		sig = syscall.SIGKILL
	}

	for _, pidStr := range pids {
		var pid int
		fmt.Sscanf(pidStr, "%d", &pid)
		if pid == 0 {
			continue
		}
		fmt.Printf("停止 Vite 进程 (PID: %d)\n", pid)
		syscall.Kill(pid, sig)
		stopped++
	}

	return stopped
}

func processExists(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil
}
