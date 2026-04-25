package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "v0.6.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Long:  `显示 Shepherd 的版本号、构建时间和 Git 提交哈希。`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Shepherd %s\n", Version)
		fmt.Printf("构建时间: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
