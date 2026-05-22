package cli

import (
	"encoding/json"
	"fmt"

	"github.com/simonxluo/Shepherd/internal/cli/client"
	"github.com/spf13/cobra"
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List running processes",
	Long:  `Display all model server processes currently running or loading.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewClient(flagHost, flagPort)
		printer := client.NewPrinter(flagJSON)

		data, err := c.Get("/api/processes")
		if err != nil {
			return err
		}

		if printer.Mode == client.OutputJSON {
			var raw interface{}
			json.Unmarshal(data, &raw)
			printer.PrintJSON(raw)
			return nil
		}

		var resp struct {
			Processes []client.ProcessInfo `json:"processes"`
			Stats     struct {
				Running int `json:"running"`
				Loading int `json:"loading"`
				Total   int `json:"total"`
			} `json:"stats"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return fmt.Errorf("failed to parse processes: %w", err)
		}

		if len(resp.Processes) == 0 {
			fmt.Println("No running processes.")
			return nil
		}

		headers := []string{"ID", "NAME", "PID", "PORT", "CTX", "STATUS"}
		rows := make([][]string, 0, len(resp.Processes))
		for _, p := range resp.Processes {
			status := "running"
			if p.Loading {
				status = "loading"
			} else if !p.Running {
				status = "stopped"
			}
			rows = append(rows, []string{
				client.TruncateID(p.ID, 12),
				p.Name,
				fmt.Sprintf("%d", p.PID),
				fmt.Sprintf("%d", p.Port),
				fmt.Sprintf("%d", p.CtxSize),
				status,
			})
		}

		fmt.Printf("Processes (running: %d, loading: %d)\n\n", resp.Stats.Running, resp.Stats.Loading)
		printer.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(psCmd)
}
