package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate config files to the new schema",
	Long: `Migrate server.config.yaml and launch_config.json to the new plugin-based schema.

Changes applied:
  - Move top-level 'llamacpp' into 'backends.llamacpp'
  - Rename 'backends.vllm_omni' to 'backends.vllmomni'
  - Move 'backends.multimodal_paths' to top-level 'multimodal_paths'
  - Rename backend_type values in launch_config.json

A timestamped backup is created before any changes are written.
Idempotent: skips if the config is already in the new format.`,
	RunE: runConfigMigrate,
}

var (
	migrateDryRun    bool
	migrateConfigDir string
)

func init() {
	configMigrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "Show what would change without writing")
	configMigrateCmd.Flags().StringVar(&migrateConfigDir, "config-dir", "", "Override config directory (default: auto-detect)")
	configCmd.AddCommand(configMigrateCmd)
}

func runConfigMigrate(cmd *cobra.Command, args []string) error {
	cfgDir := migrateConfigDir
	if cfgDir == "" {
		cfgDir = config.GetConfigDir()
	}

	yamlPath := filepath.Join(cfgDir, config.DefaultConfigFile)
	launchPath := filepath.Join(cfgDir, config.DefaultLaunchConfigFile)

	yamlChanged, err := migrateServerConfig(yamlPath)
	if err != nil {
		return fmt.Errorf("server config migration failed: %w", err)
	}

	launchChanged, err := migrateLaunchConfig(launchPath)
	if err != nil {
		return fmt.Errorf("launch config migration failed: %w", err)
	}

	if !yamlChanged && !launchChanged {
		fmt.Println("Config is already in the new format. Nothing to do.")
	}
	return nil
}

// migrateServerConfig migrates server.config.yaml to the new schema.
func migrateServerConfig(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("  %s: not found, skipping\n", path)
			return false, nil
		}
		return false, err
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return false, fmt.Errorf("failed to parse YAML: %w", err)
	}

	changed := false

	// 1. Move top-level 'llamacpp' into 'backends.llamacpp'
	if llamacppVal, ok := raw["llamacpp"]; ok {
		backends := ensureMapKey(raw, "backends")
		if _, exists := backends["llamacpp"]; !exists {
			backends["llamacpp"] = llamacppVal
		} else {
			// Merge: existing backends.llamacpp takes precedence
			fmt.Println("  WARNING: both top-level 'llamacpp' and 'backends.llamacpp' exist; keeping backends.llamacpp")
		}
		delete(raw, "llamacpp")
		changed = true
		fmt.Println("  MOVE: llamacpp → backends.llamacpp")
	}

	// 2. Rename 'backends.vllm_omni' → 'backends.vllmomni'
	if backendsVal, ok := raw["backends"]; ok {
		backends, _ := backendsVal.(map[string]any)
		if backends != nil {
			if omniVal, ok := backends["vllm_omni"]; ok {
				if _, exists := backends["vllmomni"]; !exists {
					backends["vllmomni"] = omniVal
				}
				delete(backends, "vllm_omni")
				changed = true
				fmt.Println("  RENAME: backends.vllm_omni → backends.vllmomni")
			}

			// 3. Move 'backends.multimodal_paths' → top-level
			if mmVal, ok := backends["multimodal_paths"]; ok {
				if _, exists := raw["multimodal_paths"]; !exists {
					raw["multimodal_paths"] = mmVal
				}
				delete(backends, "multimodal_paths")
				changed = true
				fmt.Println("  MOVE: backends.multimodal_paths → multimodal_paths")
			}
		}
	}

	if !changed {
		return false, nil
	}

	if migrateDryRun {
		fmt.Printf("  DRY RUN: would write %s\n", path)
		return true, nil
	}

	// Backup
	backupPath := fmt.Sprintf("%s.bak.%s", path, time.Now().Format("20060102-150405"))
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return false, fmt.Errorf("failed to create backup: %w", err)
	}
	fmt.Printf("  BACKUP: %s\n", backupPath)

	// Write migrated config
	out, err := yaml.Marshal(raw)
	if err != nil {
		return false, fmt.Errorf("failed to marshal YAML: %w", err)
	}
	if err := atomicWrite(path, out); err != nil {
		return false, err
	}
	fmt.Printf("  WRITTEN: %s\n", path)
	return true, nil
}

// migrateLaunchConfig migrates launch_config.json backend_type values.
func migrateLaunchConfig(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	var configs map[string]any
	if err := json.Unmarshal(data, &configs); err != nil {
		return false, fmt.Errorf("failed to parse launch config: %w", err)
	}

	renames := map[string]string{
		"llama.cpp": "llamacpp",
		"vllm_omni": "vllmomni",
	}

	changed := false
	for modelID, v := range configs {
		entry, ok := v.(map[string]any)
		if !ok {
			continue
		}
		bt, _ := entry["backend_type"].(string)
		if newBT, ok := renames[bt]; ok {
			entry["backend_type"] = newBT
			changed = true
			fmt.Printf("  RENAME: %s backend_type %q → %q\n", modelID, bt, newBT)
		}
	}

	if !changed {
		return false, nil
	}

	if migrateDryRun {
		fmt.Printf("  DRY RUN: would write %s\n", path)
		return true, nil
	}

	// Backup
	backupPath := fmt.Sprintf("%s.bak.%s", path, time.Now().Format("20060102-150405"))
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return false, fmt.Errorf("failed to create backup: %w", err)
	}
	fmt.Printf("  BACKUP: %s\n", backupPath)

	out, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return false, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	if err := atomicWrite(path, out); err != nil {
		return false, err
	}
	fmt.Printf("  WRITTEN: %s\n", path)
	return true, nil
}

// ensureMapKey returns m[key] as map[string]any, creating it if absent.
func ensureMapKey(m map[string]any, key string) map[string]any {
	v, ok := m[key]
	if !ok {
		sub := make(map[string]any)
		m[key] = sub
		return sub
	}
	sub, _ := v.(map[string]any)
	if sub == nil {
		sub = make(map[string]any)
		m[key] = sub
	}
	return sub
}

// atomicWrite writes data to path via a temp file + rename.
func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}
	return nil
}
