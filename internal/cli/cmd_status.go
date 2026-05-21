package cli

import (
	"encoding/json"
	"fmt"

	"github.com/shepherd-project/shepherd/Shepherd/internal/cli/client"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show server status",
	Long:  `Check whether the Shepherd server is running and display basic information.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewClient(flagHost, flagPort)
		printer := client.NewPrinter(flagJSON)

		data, err := c.Get("/api/info")
		if err != nil {
			return err
		}

		if printer.Mode == client.OutputJSON {
			var raw interface{}
			json.Unmarshal(data, &raw)
			printer.PrintJSON(raw)
			return nil
		}

		var info client.ServerInfo
		if err := json.Unmarshal(data, &info); err != nil {
			return fmt.Errorf("failed to parse server info: %w", err)
		}

		fmt.Println("Shepherd Server Status")
		fmt.Println("======================")
		printer.PrintKeyValue([]client.KeyValue{
			{Key: "Status", Value: info.Status},
			{Key: "Version", Value: info.Version},
			{Key: "Build Time", Value: info.BuildTime},
			{Key: "Git Commit", Value: info.GitCommit},
			{Key: "Role", Value: info.Role},
			{Key: "Web Port", Value: fmt.Sprintf("%d", info.Ports.Web)},
			{Key: "Anthropic Port", Value: fmt.Sprintf("%d", info.Ports.Anthropic)},
			{Key: "Ollama Port", Value: fmt.Sprintf("%d", info.Ports.Ollama)},
			{Key: "LM Studio Port", Value: fmt.Sprintf("%d", info.Ports.LMStudio)},
		})

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
