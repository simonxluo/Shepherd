package cli

import (
	"encoding/json"
	"fmt"

	"github.com/simonxluo/Shepherd/internal/cli/client"
	"github.com/spf13/cobra"
)

var backendsCmd = &cobra.Command{
	Use:   "backends",
	Short: "Manage backend plugins",
	Long:  `Inspect backend plugins registered in the running Shepherd server.`,
}

// backends list
var backendsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered backend plugins",
	Long:  `Display every backend plugin registered in the plugin registry (id + display name).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewClient(flagHost, flagPort)
		printer := client.NewPrinter(flagJSON)

		data, err := c.Get("/api/backends")
		if err != nil {
			return err
		}

		if printer.Mode == client.OutputJSON {
			var raw interface{}
			_ = json.Unmarshal(data, &raw)
			printer.PrintJSON(raw)
			return nil
		}

		var resp struct {
			Backends []struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"backends"`
			Count int `json:"count"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return fmt.Errorf("failed to parse backends: %w", err)
		}

		if resp.Count == 0 {
			fmt.Println("No backend plugins registered.")
			return nil
		}

		fmt.Printf("Backend plugins (%d registered)\n\n", resp.Count)
		headers := []string{"ID", "DISPLAY NAME"}
		rows := make([][]string, 0, len(resp.Backends))
		for _, b := range resp.Backends {
			rows = append(rows, []string{b.ID, b.DisplayName})
		}
		printer.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	backendsCmd.AddCommand(backendsListCmd)
	rootCmd.AddCommand(backendsCmd)
}
