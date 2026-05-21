package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Persistent flags for CLI-to-server communication
	flagHost string
	flagPort int
	flagJSON bool
)

var rootCmd = &cobra.Command{
	Use:   "shepherd",
	Short: "Shepherd - llama.cpp model management system",
	Long: `Shepherd is a distributed llama.cpp model management system written in Go.

Supports model auto-discovery, load/unload, multi-protocol compatible API
(OpenAI/Anthropic/Ollama/LM Studio), distributed node management and task scheduling.

Quick start:
  shepherd                # Default start (hybrid mode)
  shepherd serve --web    # Start with frontend dev server
  shepherd serve --build  # Build frontend then start`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	// Persistent flags available to all subcommands
	rootCmd.PersistentFlags().StringVar(&flagHost, "host", getEnvOrDefault("SHEPHERD_HOST", "localhost"), "Server host (env: SHEPHERD_HOST)")
	rootCmd.PersistentFlags().IntVar(&flagPort, "port", getEnvOrDefaultInt("SHEPHERD_PORT", 9190), "Server port (env: SHEPHERD_PORT)")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output in JSON format")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// getEnvOrDefault returns the value of an environment variable or a default.
func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// getEnvOrDefaultInt returns the int value of an env variable or a default.
func getEnvOrDefaultInt(key string, defaultValue int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
		return n
	}
	return defaultValue
}
