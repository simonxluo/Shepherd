package cli

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
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

	var webDevCmd *exec.Cmd
	if flagWeb {
		var err error
		webDevCmd, err = startWebDevServer(projectDir)
		if err != nil {
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
	if cmd.Flags().Changed("host") || rootCmd.PersistentFlags().Changed("host") {
		execArgs = append(execArgs, "--host="+flagHost)
	}
	if cmd.Flags().Changed("port") || rootCmd.PersistentFlags().Changed("port") {
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

	if err := execCmd.Start(); err != nil {
		return fmt.Errorf("启动服务进程失败: %w", err)
	}

	// 拦截信号，转发给子进程，确保 Ctrl+C 能正确关闭服务
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	// 等待子进程退出或收到信号
	done := make(chan error, 1)
	go func() {
		done <- execCmd.Wait()
	}()

	select {
	case err := <-done:
		// 子进程自行退出
		cleanupWebDevServer(webDevCmd)
		return err
	case sig := <-sigChan:
		// 收到信号，转发给子进程
		fmt.Printf("\n收到信号 %v，正在停止服务...\n", sig)

		// 向子进程发送信号，触发其优雅关闭
		if execCmd.Process != nil {
			execCmd.Process.Signal(sig)
		}

		// 等待子进程优雅退出，超时后强制终止
		select {
		case <-done:
			// 子进程已退出
		case <-time.After(15 * time.Second):
			fmt.Println("子进程关闭超时，强制终止...")
			if execCmd.Process != nil {
				execCmd.Process.Kill()
			}
			<-done
		}

		// 清理前端开发服务器
		cleanupWebDevServer(webDevCmd)
		return nil
	}
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

func startWebDevServer(projectDir string) (*exec.Cmd, error) {
	webDir := filepath.Join(projectDir, "web")
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("前端目录不存在: %s", webDir)
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
			return nil, fmt.Errorf("安装前端依赖失败: %w", err)
		}
	}

	syncScript := filepath.Join(projectDir, "scripts", "linux", "sync-web-config.sh")
	if _, err := os.Stat(syncScript); err == nil {
		exec.Command("/bin/bash", syncScript).Run()
	}

	fmt.Println("启动前端开发服务器...")
	devCmd := exec.Command(npmCmd, "run", "dev")
	devCmd.Dir = webDir
	devCmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	logFile, err := os.CreateTemp("", "shepherd-web-dev-*.log")
	if err != nil {
		return nil, fmt.Errorf("创建日志文件失败: %w", err)
	}
	devCmd.Stdout = logFile
	devCmd.Stderr = logFile

	if err := devCmd.Start(); err != nil {
		return nil, fmt.Errorf("启动前端失败: %w", err)
	}

	pidFile := filepath.Join(os.TempDir(), "shepherd-web-dev.pid")
	os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", devCmd.Process.Pid)), 0644)

	fmt.Printf("前端开发服务器已启动 (PID: %d)\n", devCmd.Process.Pid)
	fmt.Printf("日志文件: %s\n", logFile.Name())

	time.Sleep(2 * time.Second)

	return devCmd, nil
}

// cleanupWebDevServer stops the frontend dev server process.
func cleanupWebDevServer(devCmd *exec.Cmd) {
	if devCmd == nil || devCmd.Process == nil {
		return
	}

	pid := devCmd.Process.Pid
	fmt.Printf("停止前端开发服务器 (PID: %d)...\n", pid)

	// 向进程组发送 SIGTERM，确保 npm 及其子进程都被终止
	syscall.Kill(-pid, syscall.SIGTERM)

	// 等待进程退出
	done := make(chan struct{})
	go func() {
		devCmd.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		// 超时强制终止
		syscall.Kill(-pid, syscall.SIGKILL)
		<-done
	}

	// 清理 PID 文件
	pidFile := filepath.Join(os.TempDir(), "shepherd-web-dev.pid")
	os.Remove(pidFile)
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
