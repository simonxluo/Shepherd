package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	flagConfig string
	flagWeb    bool
	flagBuild  bool
	flagHost   string
	flagPort   int
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "启动 Shepherd 服务",
	Long: `启动 Shepherd 后端服务。

默认以 hybrid 模式启动（兼具 master 和 client 能力）。
角色由配置文件的 node.role 字段决定。

支持同时启动前端开发服务器，实现一键启动前后端。`,
	Example: `  shepherd serve                      # 默认启动
  shepherd serve --web                # 同时启动前端开发服务器
  shepherd serve --build --web        # 先编译再启动，同时启动前端
  shepherd serve --config custom.yaml # 指定配置文件
  shepherd serve --port 8080          # 指定端口`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().StringVarP(&flagConfig, "config", "c", "", "配置文件路径")
	serveCmd.Flags().BoolVarP(&flagWeb, "web", "w", false, "同时启动前端开发服务器")
	serveCmd.Flags().BoolVarP(&flagBuild, "build", "b", false, "启动前先编译")
	serveCmd.Flags().StringVar(&flagHost, "host", "", "监听地址 (覆盖配置文件)")
	serveCmd.Flags().IntVar(&flagPort, "port", 0, "监听端口 (覆盖配置文件)")

	rootCmd.AddCommand(serveCmd)
	rootCmd.SetOut(os.Stdout)
}

func runServe(cmd *cobra.Command, args []string) error {
	projectDir := getProjectDir()

	if flagBuild {
		if err := buildBinary(projectDir); err != nil {
			return fmt.Errorf("编译失败: %w", err)
		}
		if err := buildFrontend(projectDir); err != nil {
			return fmt.Errorf("前端编译失败: %w", err)
		}
	}

	binaryPath := filepath.Join(projectDir, "build", "shepherd")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("二进制文件不存在: %s\n请先运行: shepherd build", binaryPath)
	}

	configPath := resolveConfigPath(projectDir, flagConfig)
	if configPath == "" {
		return fmt.Errorf("未找到配置文件\n请确保 config/example/server.config.yaml 或 config/node/server.config.yaml 存在")
	}

	if err := cleanupStaleProcesses(); err != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "警告: 清理残留进程失败: %v\n", err)
	}

	if flagWeb {
		if err := startWebDevServer(projectDir); err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "警告: 前端启动失败: %v\n", err)
		}
	}

	fmt.Println()
	fmt.Println("---")
	fmt.Println("  Shepherd")
	fmt.Println("---")
	fmt.Printf("  配置文件: %s\n", configPath)
	fmt.Printf("  节点角色: (从配置文件读取)\n")
	if flagWeb {
		fmt.Println("  前端服务器: 已启动")
	}
	fmt.Println("---")
	fmt.Println()

	var execArgs []string
	execArgs = append(execArgs, "run-server")
	execArgs = append(execArgs, "--config="+configPath)
	if flagHost != "" {
		execArgs = append(execArgs, "--host="+flagHost)
	}
	if flagPort > 0 {
		execArgs = append(execArgs, fmt.Sprintf("--port=%d", flagPort))
	}

	execCmd := exec.Command(binaryPath, execArgs...)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Dir = projectDir
	execCmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	return execCmd.Run()
}

func getProjectDir() string {
	exePath, err := os.Executable()
	if err != nil {
		wd, _ := os.Getwd()
		return wd
	}

	resolved, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		resolved = exePath
	}

	dir := filepath.Dir(resolved)

	if filepath.Base(dir) == "build" {
		parent := filepath.Dir(dir)
		if _, err := os.Stat(filepath.Join(parent, "go.mod")); err == nil {
			return parent
		}
	}

	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return dir
	}

	wd, _ := os.Getwd()
	return wd
}

func resolveConfigPath(projectDir, userConfig string) string {
	if userConfig != "" {
		if filepath.IsAbs(userConfig) {
			if _, err := os.Stat(userConfig); err == nil {
				return userConfig
			}
		}
		candidate := filepath.Join(projectDir, userConfig)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		return ""
	}

	nodeConfig := filepath.Join(projectDir, "config", "node", "server.config.yaml")
	if _, err := os.Stat(nodeConfig); err == nil {
		return nodeConfig
	}

	exampleConfig := filepath.Join(projectDir, "config", "example", "server.config.yaml")
	if _, err := os.Stat(exampleConfig); err == nil {
		os.MkdirAll(filepath.Dir(nodeConfig), 0755)
		copyFile(exampleConfig, nodeConfig)
		return nodeConfig
	}

	return ""
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func buildBinary(projectDir string) error {
	fmt.Println("编译项目...")

	goPath := "/home/user/sdk/go/bin/go"
	if _, err := os.Stat(goPath); os.IsNotExist(err) {
		goPath = "go"
	}

	ldflags := fmt.Sprintf(
		`-X main.Version=dev -X main.BuildTime=%s -X main.GitCommit=%s -s -w`,
		time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		getGitCommit(projectDir),
	)

	cmd := exec.Command(goPath, "build", "-mod=mod",
		"-ldflags", ldflags,
		"-o", filepath.Join("build", "shepherd"),
		"./cmd/shepherd/",
	)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "GOPROXY=https://goproxy.cn,direct")

	if err := cmd.Run(); err != nil {
		return err
	}

	fmt.Println("编译完成")
	return nil
}

func buildFrontend(projectDir string) error {
	webDir := filepath.Join(projectDir, "web")
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		return nil
	}

	if err := ensureNodeModules(webDir); err != nil {
		return err
	}

	npmCmd := getNpmCmd()
	buildCmd := exec.Command(npmCmd, "run", "build")
	buildCmd.Dir = webDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	fmt.Println("编译前端...")
	if err := buildCmd.Run(); err != nil {
		return err
	}

	fmt.Println("前端编译完成")
	return nil
}

func getGitCommit(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func startWebDevServer(projectDir string) error {
	webDir := filepath.Join(projectDir, "web")
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		return fmt.Errorf("前端目录不存在: %s", webDir)
	}

	npmCmd := "npm"
	if runtime.GOOS == "windows" {
		npmCmd = "npm.cmd"
	}

	if _, err := os.Stat(filepath.Join(webDir, "node_modules")); os.IsNotExist(err) {
		fmt.Println("安装前端依赖...")
		installCmd := exec.Command(npmCmd, "install")
		installCmd.Dir = webDir
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("安装前端依赖失败: %w", err)
		}
	}

	syncScript := filepath.Join(projectDir, "scripts", "linux", "sync-web-config.sh")
	if _, err := os.Stat(syncScript); err == nil {
		exec.Command("/bin/bash", syncScript).Run()
	}

	fmt.Println("启动前端开发服务器...")
	devCmd := exec.Command(npmCmd, "run", "dev")
	devCmd.Dir = webDir

	logFile, err := os.CreateTemp("", "shepherd-web-dev-*.log")
	if err != nil {
		return fmt.Errorf("创建日志文件失败: %w", err)
	}
	devCmd.Stdout = logFile
	devCmd.Stderr = logFile

	if err := devCmd.Start(); err != nil {
		return fmt.Errorf("启动前端失败: %w", err)
	}

	pidFile := filepath.Join(os.TempDir(), "shepherd-web-dev.pid")
	os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", devCmd.Process.Pid)), 0644)

	fmt.Printf("前端开发服务器已启动 (PID: %d)\n", devCmd.Process.Pid)
	fmt.Printf("日志文件: %s\n", logFile.Name())

	time.Sleep(2 * time.Second)

	return nil
}

func cleanupStaleProcesses() error {
	cmd := exec.Command("pgrep", "-f", "shepherd")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	pids := strings.Fields(strings.TrimSpace(string(output)))
	selfPid := os.Getpid()

	for _, pidStr := range pids {
		var pid int
		fmt.Sscanf(pidStr, "%d", &pid)
		if pid == selfPid || pid == 0 {
			continue
		}
		syscall.Kill(pid, syscall.SIGTERM)
	}

	if len(pids) > 0 {
		time.Sleep(500 * time.Millisecond)
	}

	return nil
}
