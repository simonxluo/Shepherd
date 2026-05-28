package cli

import (
	"encoding/json"
	"fmt"

	"github.com/simonxluo/Shepherd/internal/cli/client"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage server configuration",
	Long:  `View and manage the running Shepherd server configuration.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current server configuration",
	Long:  `Display the current configuration of the running Shepherd server.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewClient(flagHost, flagPort)
		printer := client.NewPrinter(flagJSON)

		data, err := c.Get("/api/config")
		if err != nil {
			return err
		}

		if printer.Mode == client.OutputJSON {
			var raw interface{}
			json.Unmarshal(data, &raw)
			printer.PrintJSON(raw)
			return nil
		}

		var cfg client.ConfigResponse
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}

		fmt.Println("Server Configuration")
		fmt.Println("====================")

		fmt.Println("\n[Server]")
		printer.PrintKeyValue([]client.KeyValue{
			{Key: "Host", Value: cfg.Server.Host},
			{Key: "Web Port", Value: fmt.Sprintf("%d", cfg.Server.WebPort)},
			{Key: "Anthropic Port", Value: fmt.Sprintf("%d", cfg.Server.AnthropicPort)},
		})

		fmt.Println("\n[Node]")
		printer.PrintKeyValue([]client.KeyValue{
			{Key: "Role", Value: cfg.Node.Role},
			{Key: "ID", Value: cfg.Node.ID},
			{Key: "Name", Value: cfg.Node.Name},
		})

		fmt.Println("\n[Models]")
		printer.PrintKeyValue([]client.KeyValue{
			{Key: "Auto Scan", Value: fmt.Sprintf("%v", cfg.Models.AutoScan)},
		})
		if len(cfg.Models.Paths) > 0 {
			fmt.Println("  Paths:")
			for _, p := range cfg.Models.Paths {
				fmt.Printf("    - %s\n", p)
			}
		}

		fmt.Println("\n[Storage]")
		printer.PrintKeyValue([]client.KeyValue{
			{Key: "Type", Value: cfg.Storage.Type},
		})

		if len(cfg.Llamacpp.Paths) > 0 {
			fmt.Println("\n[Llama.cpp]")
			for _, p := range cfg.Llamacpp.Paths {
				name := p.Name
				if name == "" {
					name = "(unnamed)"
				}
				fmt.Printf("  %s: %s\n", name, p.Path)
			}
		}

		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}
