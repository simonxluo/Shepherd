package cli

import (
	"encoding/json"
	"fmt"

	"github.com/simonxluo/Shepherd/internal/cli/client"
	"github.com/spf13/cobra"
)

var systemCmd = &cobra.Command{
	Use:   "system",
	Short: "System information",
	Long:  `Display system-level information such as GPUs, backends, and resource usage.`,
}

// --- system gpus ---

var systemGPUsCmd = &cobra.Command{
	Use:   "gpus",
	Short: "Show GPU information",
	Long:  `Display detected GPUs and their memory information.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewClient(flagHost, flagPort)
		printer := client.NewPrinter(flagJSON)

		data, err := c.Get("/api/system/gpus")
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
			Devices []string         `json:"devices"`
			GPUs    []client.GPUInfo `json:"gpus"`
			Count   int              `json:"count"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return fmt.Errorf("failed to parse GPU info: %w", err)
		}

		if resp.Count == 0 {
			fmt.Println("No GPUs detected.")
			return nil
		}

		fmt.Printf("GPUs (%d detected)\n\n", resp.Count)

		headers := []string{"ID", "NAME", "TOTAL MEM", "FREE MEM", "AVAILABLE"}
		rows := make([][]string, 0, len(resp.GPUs))
		for _, g := range resp.GPUs {
			avail := "yes"
			if !g.Available {
				avail = "no"
			}
			totalMem := g.TotalMemory
			if totalMem == "" {
				totalMem = "-"
			}
			freeMem := g.FreeMemory
			if freeMem == "" {
				freeMem = "-"
			}
			rows = append(rows, []string{
				g.ID,
				g.Name,
				totalMem,
				freeMem,
				avail,
			})
		}

		printer.PrintTable(headers, rows)
		return nil
	},
}

// --- system backends ---

var systemBackendsCmd = &cobra.Command{
	Use:   "backends",
	Short: "Show inference backends",
	Long:  `Display available inference backends (llama.cpp, vLLM, etc.).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewClient(flagHost, flagPort)
		printer := client.NewPrinter(flagJSON)

		data, err := c.Get("/api/system/llamacpp-backends")
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
			Backends          []client.BackendInfo `json:"backends"`
			InferenceBackends []client.BackendInfo `json:"inferenceBackends"`
			Count             int                  `json:"count"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return fmt.Errorf("failed to parse backends: %w", err)
		}

		if len(resp.Backends) == 0 && len(resp.InferenceBackends) == 0 {
			fmt.Println("No backends configured.")
			return nil
		}

		if len(resp.Backends) > 0 {
			fmt.Println("Llama.cpp Backends")
			fmt.Println("------------------")
			headers := []string{"TYPE", "NAME", "PATH", "AVAILABLE"}
			rows := make([][]string, 0, len(resp.Backends))
			for _, b := range resp.Backends {
				avail := "yes"
				if !b.Available {
					avail = "no"
				}
				name := b.Name
				if name == "" {
					name = "-"
				}
				rows = append(rows, []string{
					b.Type,
					name,
					b.Path,
					avail,
				})
			}
			printer.PrintTable(headers, rows)
		}

		if len(resp.InferenceBackends) > 0 {
			fmt.Println("\nInference Backends")
			fmt.Println("------------------")
			headers := []string{"TYPE", "NAME", "CONDA ENV", "AVAILABLE"}
			rows := make([][]string, 0, len(resp.InferenceBackends))
			for _, b := range resp.InferenceBackends {
				avail := "yes"
				if !b.Available {
					avail = "no"
				}
				condaEnv := b.CondaEnv
				if condaEnv == "" {
					condaEnv = "-"
				}
				rows = append(rows, []string{
					b.Type,
					b.Name,
					condaEnv,
					avail,
				})
			}
			printer.PrintTable(headers, rows)
		}

		return nil
	},
}

// --- system resources ---

var systemResourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "Show system resource usage",
	Long:  `Display current CPU, memory, disk, and GPU resource usage.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewClient(flagHost, flagPort)
		printer := client.NewPrinter(flagJSON)

		data, err := c.Get("/api/system/resources")
		if err != nil {
			return err
		}

		if printer.Mode == client.OutputJSON {
			var raw interface{}
			json.Unmarshal(data, &raw)
			printer.PrintJSON(raw)
			return nil
		}

		var res client.ResourcesResponse
		if err := json.Unmarshal(data, &res); err != nil {
			return fmt.Errorf("failed to parse resources: %w", err)
		}

		fmt.Println("System Resources")
		fmt.Println("================")

		fmt.Println("\n[CPU]")
		printer.PrintKeyValue([]client.KeyValue{
			{Key: "Cores", Value: fmt.Sprintf("%d", res.CPU.Total/1000)},
			{Key: "Usage", Value: client.FormatPercent(res.CPU.Percent)},
		})

		fmt.Println("\n[Memory]")
		printer.PrintKeyValue([]client.KeyValue{
			{Key: "Used", Value: client.FormatSize(res.Memory.Used)},
			{Key: "Total", Value: client.FormatSize(res.Memory.Total)},
			{Key: "Usage", Value: client.FormatPercent(res.Memory.Percent)},
		})

		fmt.Println("\n[Disk]")
		printer.PrintKeyValue([]client.KeyValue{
			{Key: "Used", Value: client.FormatSize(res.Disk.Used)},
			{Key: "Total", Value: client.FormatSize(res.Disk.Total)},
			{Key: "Usage", Value: client.FormatPercent(res.Disk.Percent)},
		})

		if len(res.GPU) > 0 {
			fmt.Println("\n[GPU]")
			for _, g := range res.GPU {
				fmt.Printf("  GPU %d (%s):\n", g.Index, g.Name)
				fmt.Printf("    Memory: %s / %s\n", client.FormatSize(g.MemoryUsed), client.FormatSize(g.MemoryTotal))
			}
		}

		if len(res.Load) >= 3 {
			fmt.Println("\n[Load Average]")
			fmt.Printf("  1min: %.2f  5min: %.2f  15min: %.2f\n", res.Load[0], res.Load[1], res.Load[2])
		}

		fmt.Println("\n[System]")
		kvs := []client.KeyValue{
			{Key: "Uptime", Value: client.FormatDuration(res.Uptime)},
		}
		if res.Kernel != "" {
			kvs = append(kvs, client.KeyValue{Key: "Kernel", Value: res.Kernel})
		}
		if res.ROCm != "" {
			kvs = append(kvs, client.KeyValue{Key: "ROCm", Value: res.ROCm})
		}
		printer.PrintKeyValue(kvs)

		return nil
	},
}

func init() {
	systemCmd.AddCommand(systemGPUsCmd)
	systemCmd.AddCommand(systemBackendsCmd)
	systemCmd.AddCommand(systemResourcesCmd)
	rootCmd.AddCommand(systemCmd)
}
