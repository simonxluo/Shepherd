package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "shepherd",
	Short: "Shepherd - llama.cpp 模型管理系统",
	Long: `Shepherd 是一个 Go 语言编写的分布式 llama.cpp 模型管理系统。

支持模型自动发现、加载/卸载、多协议兼容 API（OpenAI/Anthropic/Ollama/LM Studio）、
分布式节点管理和任务调度。

快速启动:
  shepherd                # 默认启动（hybrid 模式）
  shepherd serve --web    # 同时启动前端开发服务器
  shepherd serve --build  # 先编译再启动`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}
