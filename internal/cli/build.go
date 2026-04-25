package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"time"

	"github.com/spf13/cobra"
)

var (
	buildOutput    string
	buildGoos      string
	buildGoarch    string
	buildVersion   string
	buildCross     bool
	buildUniversal bool
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "编译 Shepherd",
	Long: `编译 Shepherd 后端二进制文件。

支持交叉编译和版本注入。默认编译当前平台的二进制文件。`,
	Example: `  shepherd build                          # 编译当前平台
  shepherd build --version v1.0.0         # 指定版本号
  shepherd build --cross --goos linux --goarch arm64  # 交叉编译
  shepherd build --universal              # macOS 通用二进制`,
	RunE: runBuild,
}

func init() {
	buildCmd.Flags().StringVarP(&buildOutput, "output", "o", "", "输出文件路径")
	buildCmd.Flags().StringVar(&buildGoos, "goos", runtime.GOOS, "目标操作系统")
	buildCmd.Flags().StringVar(&buildGoarch, "goarch", runtime.GOARCH, "目标架构")
	buildCmd.Flags().StringVarP(&buildVersion, "version", "v", "dev", "版本号")
	buildCmd.Flags().BoolVar(&buildCross, "cross", false, "交叉编译")
	buildCmd.Flags().BoolVar(&buildUniversal, "universal", false, "编译通用二进制 (macOS)")

	rootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) error {
	projectDir := getProjectDir()

	if buildUniversal {
		return buildUniversalBinary(projectDir)
	}

	return buildSingle(projectDir)
}

func buildSingle(projectDir string) error {
	goPath := findGoBinary()

	fmt.Printf("编译 Shepherd %s...\n", buildVersion)

	outputPath := buildOutput
	if outputPath == "" {
		outputPath = filepath.Join(projectDir, "build", "shepherd")
		if buildGoos == "windows" || (buildCross && buildGoos == "windows") {
			outputPath += ".exe"
		}
	}

	os.MkdirAll(filepath.Dir(outputPath), 0755)

	commit := getGitCommit(projectDir)
	ldflags := fmt.Sprintf(
		`-X main.Version=%s -X main.BuildTime=%s -X main.GitCommit=%s -s -w`,
		buildVersion,
		time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		commit,
	)

	buildArgs := []string{"build", "-mod=mod", "-ldflags", ldflags, "-o", outputPath, "./cmd/shepherd/"}

	buildCmd := exec.Command(goPath, buildArgs...)
	buildCmd.Dir = projectDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	env := os.Environ()
	if buildCross {
		env = append(env,
			"GOOS="+buildGoos,
			"GOARCH="+buildGoarch,
			"CGO_ENABLED=0",
		)
	}
	env = append(env, "GOPROXY=https://goproxy.cn,direct")
	buildCmd.Env = env

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("编译失败: %w", err)
	}

	fmt.Printf("编译完成: %s\n", outputPath)

	verifyCmd := exec.Command(goPath, "version", "-m", outputPath)
	verifyCmd.Output()

	return nil
}

func buildUniversalBinary(projectDir string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("通用二进制仅支持 macOS")
	}

	goPath := findGoBinary()
	commit := getGitCommit(projectDir)
	buildDir := filepath.Join(projectDir, "build")
	os.MkdirAll(buildDir, 0755)

	ldflags := fmt.Sprintf(
		`-X main.Version=%s -X main.BuildTime=%s -X main.GitCommit=%s -s -w`,
		buildVersion,
		time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		commit,
	)

	for _, arch := range []string{"arm64", "amd64"} {
		output := filepath.Join(buildDir, fmt.Sprintf("shepherd-darwin-%s", arch))
		fmt.Printf("编译 darwin/%s...\n", arch)

		cmd := exec.Command(goPath, "build", "-mod=mod", "-ldflags", ldflags, "-o", output, "./cmd/shepherd/")
		cmd.Dir = projectDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(os.Environ(),
			"GOOS=darwin",
			"GOARCH="+arch,
			"CGO_ENABLED=0",
			"GOPROXY=https://goproxy.cn,direct",
		)

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("编译 darwin/%s 失败: %w", arch, err)
		}
	}

	universalPath := filepath.Join(buildDir, "shepherd-darwin-universal")
	arm64Bin := filepath.Join(buildDir, "shepherd-darwin-arm64")
	amd64Bin := filepath.Join(buildDir, "shepherd-darwin-amd64")

	lipoCmd := exec.Command("lipo", "-create", "-output", universalPath, arm64Bin, amd64Bin)
	if err := lipoCmd.Run(); err != nil {
		fmt.Println("警告: lipo 合并失败，通用二进制未创建")
		fmt.Println("分别编译的二进制文件仍然可用:")
		fmt.Printf("  %s\n", arm64Bin)
		fmt.Printf("  %s\n", amd64Bin)
		return nil
	}

	os.Remove(arm64Bin)
	os.Remove(amd64Bin)

	fmt.Printf("通用二进制编译完成: %s\n", universalPath)
	return nil
}

func findGoBinary() string {
	candidates := []string{
		"/home/user/sdk/go/bin/go",
		"/usr/local/go/bin/go",
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	if p, err := exec.LookPath("go"); err == nil {
		return p
	}

	return "go"
}
