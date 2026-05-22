package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/simonxluo/Shepherd/internal/cli/client"
	"github.com/spf13/cobra"
)

var modelCmd = &cobra.Command{
	Use:   "model",
	Short: "Manage models",
	Long:  `List, inspect, load, unload, and scan models on the Shepherd server.`,
}

// --- model list ---

var modelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available models",
	Long:  `Display all models discovered by the server.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewClient(flagHost, flagPort)
		printer := client.NewPrinter(flagJSON)

		data, err := c.Get("/api/models")
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
			Models []client.ModelItem `json:"models"`
			Total  int               `json:"total"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return fmt.Errorf("failed to parse models: %w", err)
		}

		if len(resp.Models) == 0 {
			fmt.Println("No models found.")
			return nil
		}

		headers := []string{"ID", "NAME", "SIZE", "STATUS", "BACKEND"}
		rows := make([][]string, 0, len(resp.Models))
		for _, m := range resp.Models {
			size := client.FormatSize(m.Size)
			if m.TotalSize > 0 && m.TotalSize != m.Size {
				size = client.FormatSize(m.TotalSize)
			}
			status := m.Status
			if m.IsLoaded {
				status = "loaded"
			}
			backend := m.BackendType
			if backend == "" {
				backend = "-"
			}
			rows = append(rows, []string{
				client.TruncateID(m.ID, 12),
				m.Name,
				size,
				status,
				backend,
			})
		}

		fmt.Printf("Models (%d total)\n\n", resp.Total)
		printer.PrintTable(headers, rows)
		return nil
	},
}

// --- model loaded ---

var modelLoadedCmd = &cobra.Command{
	Use:   "loaded",
	Short: "List loaded models",
	Long:  `Display models that are currently loaded or in the process of loading.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewClient(flagHost, flagPort)
		printer := client.NewPrinter(flagJSON)

		data, err := c.Get("/api/models/loaded")
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
			Models []client.LoadedModelItem `json:"models"`
			Total  int                      `json:"total"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return fmt.Errorf("failed to parse loaded models: %w", err)
		}

		if len(resp.Models) == 0 {
			fmt.Println("No models currently loaded.")
			return nil
		}

		headers := []string{"ID", "NAME", "STATE", "PORT", "CTX", "BACKEND", "LOADED AT"}
		rows := make([][]string, 0, len(resp.Models))
		for _, m := range resp.Models {
			backend := m.BackendType
			if backend == "" {
				backend = "-"
			}
			loadedAt := m.LoadedAt
			if loadedAt == "" {
				loadedAt = "-"
			}
			rows = append(rows, []string{
				client.TruncateID(m.ID, 12),
				m.Name,
				m.State,
				fmt.Sprintf("%d", m.Port),
				fmt.Sprintf("%d", m.CtxSize),
				backend,
				loadedAt,
			})
		}

		fmt.Printf("Loaded Models (%d)\n\n", resp.Total)
		printer.PrintTable(headers, rows)
		return nil
	},
}

// --- model info ---

var modelInfoCmd = &cobra.Command{
	Use:   "info <id>",
	Short: "Show model details",
	Long:  `Display detailed information about a specific model. Supports prefix matching on the model ID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewClient(flagHost, flagPort)
		printer := client.NewPrinter(flagJSON)

		modelID, err := resolveModelID(c, args[0])
		if err != nil {
			return err
		}

		data, err := c.Get("/api/models/" + modelID)
		if err != nil {
			return err
		}

		if printer.Mode == client.OutputJSON {
			var raw interface{}
			json.Unmarshal(data, &raw)
			printer.PrintJSON(raw)
			return nil
		}

		// Parse as generic map for flexible display
		var info map[string]interface{}
		if err := json.Unmarshal(data, &info); err != nil {
			return fmt.Errorf("failed to parse model info: %w", err)
		}

		// Try to extract the "model" field if present
		if modelData, ok := info["model"]; ok {
			info, _ = modelData.(map[string]interface{})
		}

		fmt.Println("Model Details")
		fmt.Println("=============")

		// Print common fields
		fields := []struct{ key, label string }{
			{"id", "ID"},
			{"name", "Name"},
			{"displayName", "Display Name"},
			{"alias", "Alias"},
			{"path", "Path"},
			{"status", "Status"},
			{"backendType", "Backend"},
			{"scannedAt", "Scanned At"},
		}
		for _, f := range fields {
			if v, ok := info[f.key]; ok && v != nil && v != "" {
				fmt.Printf("  %-14s %v\n", f.label+":", v)
			}
		}

		// Size
		if size, ok := info["size"].(float64); ok && size > 0 {
			fmt.Printf("  %-14s %s\n", "Size:", client.FormatSize(int64(size)))
		}
		if totalSize, ok := info["totalSize"].(float64); ok && totalSize > 0 {
			fmt.Printf("  %-14s %s\n", "Total Size:", client.FormatSize(int64(totalSize)))
		}

		// Metadata
		if meta, ok := info["metadata"].(map[string]interface{}); ok && len(meta) > 0 {
			fmt.Println("\n  Metadata:")
			for k, v := range meta {
				fmt.Printf("    %-20s %v\n", k+":", v)
			}
		}

		return nil
	},
}

// --- model load ---

var (
	modelLoadConfigName string
	modelLoadCtxSize    int
	modelLoadGPULayers  int
	modelLoadThreads    int
)

var modelLoadCmd = &cobra.Command{
	Use:   "load <id>",
	Short: "Load a model",
	Long:  `Load a model into memory. Supports prefix matching on the model ID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewClient(flagHost, flagPort)
		printer := client.NewPrinter(flagJSON)

		modelID, err := resolveModelID(c, args[0])
		if err != nil {
			return err
		}

		// Build load request body
		body := make(map[string]interface{})

		if modelLoadConfigName != "" {
			body["configName"] = modelLoadConfigName
		}
		if modelLoadCtxSize > 0 {
			body["ctxSize"] = modelLoadCtxSize
		}
		if modelLoadGPULayers >= 0 && cmd.Flags().Changed("gpu-layers") {
			body["gpuLayers"] = modelLoadGPULayers
		}
		if modelLoadThreads > 0 {
			body["threads"] = modelLoadThreads
		}

		data, err := c.Post("/api/models/"+modelID+"/load", body)
		if err != nil {
			return err
		}

		if printer.Mode == client.OutputJSON {
			var raw interface{}
			json.Unmarshal(data, &raw)
			printer.PrintJSON(raw)
			return nil
		}

		fmt.Printf("Model %s load initiated successfully.\n", client.TruncateID(modelID, 12))
		return nil
	},
}

// --- model unload ---

var modelUnloadCmd = &cobra.Command{
	Use:   "unload <id>",
	Short: "Unload a model",
	Long:  `Unload a model from memory. Supports prefix matching on the model ID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewClient(flagHost, flagPort)
		printer := client.NewPrinter(flagJSON)

		modelID, err := resolveModelID(c, args[0])
		if err != nil {
			return err
		}

		data, err := c.Post("/api/models/"+modelID+"/unload", nil)
		if err != nil {
			return err
		}

		if printer.Mode == client.OutputJSON {
			var raw interface{}
			json.Unmarshal(data, &raw)
			printer.PrintJSON(raw)
			return nil
		}

		fmt.Printf("Model %s unloaded successfully.\n", client.TruncateID(modelID, 12))
		return nil
	},
}

// --- model scan ---

var modelScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Trigger model scan",
	Long:  `Trigger a rescan of all configured model directories.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewClient(flagHost, flagPort)
		printer := client.NewPrinter(flagJSON)

		data, err := c.Post("/api/model/scan", nil)
		if err != nil {
			return err
		}

		if printer.Mode == client.OutputJSON {
			var raw interface{}
			json.Unmarshal(data, &raw)
			printer.PrintJSON(raw)
			return nil
		}

		// Try to extract scan results
		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err == nil {
			if total, ok := result["total"]; ok {
				fmt.Printf("Scan complete. Found %v model(s).\n", total)
				return nil
			}
		}

		fmt.Println("Model scan triggered successfully.")
		return nil
	},
}

func init() {
	// Model load flags
	modelLoadCmd.Flags().StringVar(&modelLoadConfigName, "config", "", "Named load configuration to use")
	modelLoadCmd.Flags().IntVar(&modelLoadCtxSize, "ctx-size", 0, "Context size (0 = use default)")
	modelLoadCmd.Flags().IntVar(&modelLoadGPULayers, "gpu-layers", -1, "Number of GPU layers (-1 = auto)")
	modelLoadCmd.Flags().IntVar(&modelLoadThreads, "threads", 0, "Number of threads (0 = auto)")

	modelCmd.AddCommand(modelListCmd)
	modelCmd.AddCommand(modelLoadedCmd)
	modelCmd.AddCommand(modelInfoCmd)
	modelCmd.AddCommand(modelLoadCmd)
	modelCmd.AddCommand(modelUnloadCmd)
	modelCmd.AddCommand(modelScanCmd)
	rootCmd.AddCommand(modelCmd)
}

// resolveModelID resolves a prefix to a full model ID.
// If the prefix matches exactly one model, returns that ID.
// If it matches multiple, prints the matches and returns an error.
func resolveModelID(c *client.Client, prefix string) (string, error) {
	// First try the prefix directly (server may handle it)
	data, err := c.Get("/api/models")
	if err != nil {
		return "", err
	}

	var resp struct {
		Models []client.ModelItem `json:"models"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("failed to parse models: %w", err)
	}

	// Exact match first
	for _, m := range resp.Models {
		if m.ID == prefix {
			return m.ID, nil
		}
	}

	// Prefix match
	var matches []client.ModelItem
	lowerPrefix := strings.ToLower(prefix)
	for _, m := range resp.Models {
		if strings.HasPrefix(strings.ToLower(m.ID), lowerPrefix) {
			matches = append(matches, m)
		}
	}

	// Also try matching against name
	if len(matches) == 0 {
		for _, m := range resp.Models {
			if strings.Contains(strings.ToLower(m.Name), lowerPrefix) {
				matches = append(matches, m)
			}
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no model found matching '%s'", prefix)
	case 1:
		return matches[0].ID, nil
	default:
		fmt.Printf("Multiple models match '%s':\n\n", prefix)
		for _, m := range matches {
			fmt.Printf("  %s  %s\n", client.TruncateID(m.ID, 16), m.Name)
		}
		fmt.Println("\nPlease provide a more specific ID.")
		return "", fmt.Errorf("ambiguous model ID '%s' (%d matches)", prefix, len(matches))
	}
}
