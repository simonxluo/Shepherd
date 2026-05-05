package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "管理前端",
	Long:  `管理 Shepherd Web 前端的开发、编译和预览。`,
	Example: `  shepherd web dev       # 启动前端开发服务器
  shepherd web build     # 编译前端生产版本
  shepherd web preview   # 预览生产构建`,
}

var webDevCmd = &cobra.Command{
	Use:   "dev",
	Short: "启动前端开发服务器",
	RunE:  runWebDev,
}

var webBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "编译前端生产版本",
	RunE:  runWebBuild,
}

var webPreviewCmd = &cobra.Command{
	Use:   "preview",
	Short: "预览生产构建",
	RunE:  runWebPreview,
}

func init() {
	webCmd.AddCommand(webDevCmd)
	webCmd.AddCommand(webBuildCmd)
	webCmd.AddCommand(webPreviewCmd)
	rootCmd.AddCommand(webCmd)
}

func runWebDev(cmd *cobra.Command, args []string) error {
	projectDir := getProjectDir()
	webDir := projectDir + "/web"

	if err := ensureNodeModules(webDir); err != nil {
		return err
	}

	syncScript := projectDir + "/scripts/linux/sync-web-config.sh"
	if _, err := os.Stat(syncScript); err == nil {
		exec.Command("/bin/bash", syncScript).Run()
	}

	npmCmd := getNpmCmd()
	devCmd := exec.Command(npmCmd, "run", "dev")
	devCmd.Dir = webDir
	devCmd.Stdin = os.Stdin
	devCmd.Stdout = os.Stdout
	devCmd.Stderr = os.Stderr

	fmt.Println("启动前端开发服务器...")
	return devCmd.Run()
}

func runWebBuild(cmd *cobra.Command, args []string) error {
	projectDir := getProjectDir()
	webDir := projectDir + "/web"

	if err := ensureNodeModules(webDir); err != nil {
		return err
	}

	npmCmd := getNpmCmd()
	buildCmd := exec.Command(npmCmd, "run", "build")
	buildCmd.Dir = webDir
	buildCmd.Stdin = os.Stdin
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	fmt.Println("编译前端...")
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("前端编译失败: %w", err)
	}

	fmt.Println("前端编译完成:", webDir+"/dist")
	return nil
}

func runWebPreview(cmd *cobra.Command, args []string) error {
	projectDir := getProjectDir()
	webDir := projectDir + "/web"

	distDir := webDir + "/dist"
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		return fmt.Errorf("前端未编译，请先运行: shepherd web build")
	}

	npmCmd := getNpmCmd()
	previewCmd := exec.Command(npmCmd, "run", "preview")
	previewCmd.Dir = webDir
	previewCmd.Stdin = os.Stdin
	previewCmd.Stdout = os.Stdout
	previewCmd.Stderr = os.Stderr

	fmt.Println("启动前端预览服务器...")
	return previewCmd.Run()
}

func ensureNodeModules(webDir string) error {
	if _, err := os.Stat(webDir + "/node_modules"); os.IsNotExist(err) {
		fmt.Println("安装前端依赖...")
		npmCmd := getNpmCmd()
		installCmd := exec.Command(npmCmd, "install")
		installCmd.Dir = webDir
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		return installCmd.Run()
	}
	return nil
}

func getNpmCmd() string {
	if runtime.GOOS == "windows" {
		return "npm.cmd"
	}
	return "npm"
}
