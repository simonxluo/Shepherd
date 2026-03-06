package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, 9190, config.Server.WebPort)
	assert.Equal(t, 9170, config.Server.AnthropicPort)
	assert.Equal(t, 11434, config.Server.OllamaPort)
	assert.Equal(t, 1234, config.Server.LMStudioPort)
	// 注意：在测试环境中，AutoScan 被设为 false 以避免扫描模型文件
	// 这是预期行为，参见 DefaultConfig() 中的测试环境检测逻辑
	assert.Equal(t, 4, config.Download.MaxConcurrent)
	assert.False(t, config.Security.APIKeyEnabled)
	assert.True(t, config.Compatibility.Ollama.Enabled)
	assert.False(t, config.Compatibility.LMStudio.Enabled)
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:   "Valid config",
			config: DefaultConfig(),
		},
		{
			name: "Invalid web port",
			config: &Config{
				Server: ServerConfig{WebPort: -1},
			},
			wantErr: true,
			errMsg:  "invalid web port",
		},
		{
			name: "Invalid anthropic port",
			config: &Config{
				Server: ServerConfig{WebPort: 8080, AnthropicPort: 70000},
			},
			wantErr: true,
			errMsg:  "invalid anthropic port",
		},
		{
			name: "Port conflict",
			config: &Config{
				Server: ServerConfig{
					WebPort:       8080,
					AnthropicPort: 8070,
					OllamaPort:    8080, // Conflict
					LMStudioPort:  1234,
				},
				Compatibility: CompatibilityConfig{
					Ollama: OllamaConfig{Enabled: true},
				},
				Download: DownloadConfig{MaxConcurrent: 4, ChunkSize: 1024},
			},
			wantErr: true,
			errMsg:  "port conflict",
		},
		{
			name: "Max concurrent too low",
			config: &Config{
				Server: ServerConfig{
					WebPort:       8080,
					AnthropicPort: 8070,
					OllamaPort:    11434,
					LMStudioPort:  1234,
				},
				Download: DownloadConfig{
					MaxConcurrent: 0,
					ChunkSize:     1024,
				},
			},
			wantErr: true,
			errMsg:  "max concurrent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManagerLoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	manager := &Manager{
		configPath:       filepath.Join(tmpDir, "config.yaml"),
		modelsConfigPath: filepath.Join(tmpDir, "models.json"),
		launchConfigPath: filepath.Join(tmpDir, "launch.json"),
	}

	t.Run("Load creates default config when file doesn't exist", func(t *testing.T) {
		config, err := manager.Load()
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, 9190, config.Server.WebPort)

		// Check that file was created
		_, err = os.Stat(manager.configPath)
		assert.NoError(t, err)
	})

	t.Run("Save and Load config", func(t *testing.T) {
		config := DefaultConfig()
		config.Server.WebPort = 9090

		err := manager.Save(config)
		require.NoError(t, err)

		// Create new manager and load
		manager2 := &Manager{
			configPath:       manager.configPath,
			modelsConfigPath: manager.modelsConfigPath,
			launchConfigPath: manager.launchConfigPath,
		}
		loaded, err := manager2.Load()
		require.NoError(t, err)

		assert.Equal(t, 9090, loaded.Server.WebPort)
	})

	t.Run("Save validates config", func(t *testing.T) {
		config := &Config{
			Server: ServerConfig{WebPort: -1},
		}

		err := manager.Save(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid config")
	})
}

func TestModelsConfig(t *testing.T) {
	tmpDir := t.TempDir()
	manager := &Manager{
		configPath:       filepath.Join(tmpDir, "config.yaml"),
		modelsConfigPath: filepath.Join(tmpDir, "models.json"),
		launchConfigPath: filepath.Join(tmpDir, "launch.json"),
	}

	t.Run("Save and load models config", func(t *testing.T) {
		models := []ModelConfigEntry{
			{
				ModelID:   "model-1",
				Path:      "/path/to/model.gguf",
				Size:      1234567890,
				Alias:     "test-model",
				Favourite: true,
			},
			{
				ModelID:   "model-2",
				Favourite: false,
			},
		}

		err := manager.SaveModelsConfig(models)
		require.NoError(t, err)

		loaded, err := manager.LoadModelsConfig()
		require.NoError(t, err)

		assert.Len(t, loaded, 2)
		assert.Equal(t, "model-1", loaded[0].ModelID)
		assert.Equal(t, "test-model", loaded[0].Alias)
		assert.True(t, loaded[0].Favourite)
		assert.Equal(t, "model-2", loaded[1].ModelID)
		assert.False(t, loaded[1].Favourite)
	})

	t.Run("Load empty models config", func(t *testing.T) {
		manager2 := &Manager{
			configPath:       filepath.Join(tmpDir, "config2.yaml"),
			modelsConfigPath: filepath.Join(tmpDir, "models2.json"),
			launchConfigPath: filepath.Join(tmpDir, "launch2.json"),
		}

		models, err := manager2.LoadModelsConfig()
		require.NoError(t, err)
		assert.Empty(t, models)
	})
}

func TestLaunchConfig(t *testing.T) {
	tmpDir := t.TempDir()
	manager := &Manager{
		configPath:       filepath.Join(tmpDir, "config.yaml"),
		modelsConfigPath: filepath.Join(tmpDir, "models.json"),
		launchConfigPath: filepath.Join(tmpDir, "launch.json"),
	}

	t.Run("Save and load launch config", func(t *testing.T) {
		config := &LaunchConfig{
			CtxSize:     8192,
			Temperature: 0.8,
			GPULayers:   99,
		}

		err := manager.SaveLaunchConfig("model-1", config)
		require.NoError(t, err)

		loaded, err := manager.LoadLaunchConfig("model-1")
		require.NoError(t, err)

		assert.Equal(t, 8192, loaded.CtxSize)
		assert.Equal(t, 0.8, loaded.Temperature)
		assert.Equal(t, 99, loaded.GPULayers)
	})

	t.Run("Load default launch config for non-existent model", func(t *testing.T) {
		config, err := manager.LoadLaunchConfig("non-existent")
		require.NoError(t, err)

		defaultConfig := DefaultLaunchConfig()
		assert.Equal(t, defaultConfig.CtxSize, config.CtxSize)
	})

	t.Run("Load all launch configs", func(t *testing.T) {
		// Save multiple configs
		config1 := &LaunchConfig{CtxSize: 4096}
		config2 := &LaunchConfig{CtxSize: 8192}

		err := manager.SaveLaunchConfig("model-1", config1)
		require.NoError(t, err)
		err = manager.SaveLaunchConfig("model-2", config2)
		require.NoError(t, err)

		configs, err := manager.LoadLaunchConfigs()
		require.NoError(t, err)

		assert.Len(t, configs, 2)
		assert.Equal(t, 4096, configs["model-1"].CtxSize)
		assert.Equal(t, 8192, configs["model-2"].CtxSize)
	})
}

func TestAliasAndFavourite(t *testing.T) {
	tmpDir := t.TempDir()
	manager := &Manager{
		configPath:       filepath.Join(tmpDir, "config.yaml"),
		modelsConfigPath: filepath.Join(tmpDir, "models.json"),
		launchConfigPath: filepath.Join(tmpDir, "launch.json"),
	}

	t.Run("Save and load alias", func(t *testing.T) {
		err := manager.SaveModelAlias("model-1", "my-alias")
		require.NoError(t, err)

		aliases, err := manager.LoadAliasMap()
		require.NoError(t, err)

		assert.Equal(t, "my-alias", aliases["model-1"])
	})

	t.Run("Update existing alias", func(t *testing.T) {
		err := manager.SaveModelAlias("model-1", "new-alias")
		require.NoError(t, err)

		aliases, err := manager.LoadAliasMap()
		require.NoError(t, err)

		assert.Equal(t, "new-alias", aliases["model-1"])
	})

	t.Run("Save and load favourite", func(t *testing.T) {
		err := manager.SaveModelFavourite("model-2", true)
		require.NoError(t, err)

		favourites, err := manager.LoadFavouriteMap()
		require.NoError(t, err)

		assert.True(t, favourites["model-2"])
	})

	t.Run("Update favourite status", func(t *testing.T) {
		err := manager.SaveModelFavourite("model-2", false)
		require.NoError(t, err)

		favourites, err := manager.LoadFavouriteMap()
		require.NoError(t, err)

		assert.False(t, favourites["model-2"])
	})
}

func TestCachedModelsConfig(t *testing.T) {
	tmpDir := t.TempDir()
	manager := &Manager{
		configPath:       filepath.Join(tmpDir, "config.yaml"),
		modelsConfigPath: filepath.Join(tmpDir, "models.json"),
		launchConfigPath: filepath.Join(tmpDir, "launch.json"),
	}

	t.Run("Cache works", func(t *testing.T) {
		models := []ModelConfigEntry{
			{ModelID: "model-1", Favourite: true},
		}

		err := manager.SaveModelsConfig(models)
		require.NoError(t, err)

		// First load - reads from file
		loaded1, err := manager.LoadModelsConfigCached()
		require.NoError(t, err)
		assert.Len(t, loaded1, 1)

		// Second load - uses cache
		loaded2, err := manager.LoadModelsConfigCached()
		require.NoError(t, err)
		assert.Len(t, loaded2, 1)
		assert.Equal(t, loaded1[0].ModelID, loaded2[0].ModelID)
	})

	t.Run("Invalidate cache", func(t *testing.T) {
		manager.InvalidateCache()

		models, err := manager.LoadModelsConfigCached()
		require.NoError(t, err)
		assert.NotNil(t, models)
	})
}

func TestDefaultLaunchConfig(t *testing.T) {
	config := DefaultLaunchConfig()

	assert.Equal(t, 4096, config.CtxSize)
	assert.Equal(t, 512, config.BatchSize)
	assert.Equal(t, 8, config.Threads)
	assert.Equal(t, 99, config.GPULayers)
	assert.Equal(t, 0.7, config.Temperature)
	assert.Equal(t, 0.9, config.TopP)
	assert.Equal(t, 40, config.TopK)
	assert.InDelta(t, 1.1, config.RepeatPenalty, 0.01)
	assert.Equal(t, -1, config.Seed)
	assert.Equal(t, -1, config.NPredict)
}

func TestGetConfigDir(t *testing.T) {
	// Test default
	dir := GetConfigDir()
	assert.Equal(t, "config", dir)

	// Test environment variable override
	t.Setenv("SHEPHERD_CONFIG_DIR", "/custom/config")
	dir = GetConfigDir()
	assert.Equal(t, "/custom/config", dir)
}

func TestEnsureConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SHEPHERD_CONFIG_DIR", filepath.Join(tmpDir, "test-config"))

	err := EnsureConfigDir()
	require.NoError(t, err)

	// Check directory was created
	info, err := os.Stat(filepath.Join(tmpDir, "test-config"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}
