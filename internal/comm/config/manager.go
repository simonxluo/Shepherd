package config

import (
	"encoding/json"
	"fmt"
	"github.com/simonxluo/Shepherd/internal/comm/utils"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Manager manages configuration loading and saving
type Manager struct {
	config           *Config
	configPath       string
	modelsConfigPath string
	launchConfigPath string
	mu               sync.RWMutex
	cachedModels     []ModelConfigEntry
	cachedModelsTime int64
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	configDir := GetConfigDir()
	return &Manager{
		configPath:       filepath.Join(configDir, DefaultConfigFile),
		modelsConfigPath: filepath.Join(configDir, DefaultModelsConfigFile),
		launchConfigPath: filepath.Join(configDir, DefaultLaunchConfigFile),
	}
}

// NewManagerWithPath creates a new configuration manager with a custom config path
func NewManagerWithPath(configPath string) *Manager {
	configDir := filepath.Dir(configPath)
	modelsDir := filepath.Join(GetConfigDir(), "node")
	return &Manager{
		configPath:       configPath,
		modelsConfigPath: filepath.Join(modelsDir, "models.json"),
		launchConfigPath: filepath.Join(configDir, DefaultLaunchConfigFile),
	}
}

// GetConfigPath returns the main configuration file path
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// GetModelsConfigPath returns the models configuration file path
func (m *Manager) GetModelsConfigPath() string {
	return m.modelsConfigPath
}

// GetLaunchConfigPath returns the launch configuration file path
func (m *Manager) GetLaunchConfigPath() string {
	return m.launchConfigPath
}

// Load loads the main configuration from file
// If the file doesn't exist, creates a default config file
func (m *Manager) Load() (*Config, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure config directory exists
	if err := EnsureConfigDir(); err != nil {
		return nil, err
	}

	// Try to read config file
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		// If file doesn't exist, create default config
		if os.IsNotExist(err) {
			config := DefaultConfig()
			// Save without re-acquiring lock
			m.mu.Unlock()
			if saveErr := m.saveUnsafe(config); saveErr != nil {
				m.mu.Lock()
				return nil, fmt.Errorf("failed to create default config: %w", saveErr)
			}
			m.mu.Lock()
			m.config = config
			return config, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// Unmarshal YAML
	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	m.config = config
	return config, nil
}

// saveUnsafe saves config without locking (internal use)
func (m *Manager) saveUnsafe(config *Config) error {
	// Validate before saving
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Ensure directory exists
	configDir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config to temp file first (atomic write)
	tempPath := m.configPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, m.configPath); err != nil {
		utils.RemoveQuietly(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename config file: %w", err)
	}

	m.config = config
	return nil
}

// Save saves the configuration to file
func (m *Manager) Save(config *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate before saving
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Ensure directory exists
	configDir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config to temp file first (atomic write)
	tempPath := m.configPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, m.configPath); err != nil {
		utils.RemoveQuietly(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename config file: %w", err)
	}

	m.config = config
	return nil
}

// Get returns the currently loaded configuration
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return DefaultConfig()
	}

	// Return a copy to prevent concurrent modification
	config := *m.config
	return &config
}

// LoadModelsConfig loads the models configuration
func (m *Manager) LoadModelsConfig() ([]ModelConfigEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check file modification time for caching
	info, err := os.Stat(m.modelsConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ModelConfigEntry{}, nil
		}
		return nil, err
	}

	modTime := info.ModTime().Unix()
	if m.cachedModels != nil && modTime == m.cachedModelsTime {
		return m.cachedModels, nil
	}

	// Read and parse JSON file
	data, err := os.ReadFile(m.modelsConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ModelConfigEntry{}, nil
		}
		return nil, err
	}

	var models []ModelConfigEntry
	if err := json.Unmarshal(data, &models); err != nil {
		return nil, fmt.Errorf("failed to parse models config: %w", err)
	}

	m.cachedModels = models
	m.cachedModelsTime = modTime

	return models, nil
}

// SaveModelsConfig saves the models configuration
func (m *Manager) SaveModelsConfig(models []ModelConfigEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure directory exists
	configDir := filepath.Dir(m.modelsConfigPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(models, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal models config: %w", err)
	}

	// Write to temp file first
	tempPath := m.modelsConfigPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write models config: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, m.modelsConfigPath); err != nil {
		utils.RemoveQuietly(tempPath)
		return fmt.Errorf("failed to rename models config file: %w", err)
	}

	// Update cache
	m.cachedModels = models
	info, _ := os.Stat(m.modelsConfigPath)
	if info != nil {
		m.cachedModelsTime = info.ModTime().Unix()
	}

	return nil
}

// LoadModelsConfigCached loads models configuration with caching
func (m *Manager) LoadModelsConfigCached() ([]ModelConfigEntry, error) {
	return m.LoadModelsConfig()
}

// LoadLaunchConfigs loads all launch configurations
func (m *Manager) LoadLaunchConfigs() (map[string]*LaunchConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Read file
	data, err := os.ReadFile(m.launchConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]*LaunchConfig{}, nil
		}
		return nil, err
	}

	// Parse JSON
	var configs map[string]*LaunchConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to parse launch config: %w", err)
	}

	if configs == nil {
		return map[string]*LaunchConfig{}, nil
	}

	return configs, nil
}

// SaveLaunchConfig saves a launch configuration for a model
func (m *Manager) SaveLaunchConfig(modelID string, config *LaunchConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load existing configs
	configs, err := m.loadLaunchConfigsUnsafe()
	if err != nil {
		return err
	}

	// Add/update config for this model
	if configs == nil {
		configs = make(map[string]*LaunchConfig)
	}
	configs[modelID] = config

	// Save all configs
	return m.saveLaunchConfigsUnsafe(configs)
}

// loadLaunchConfigsUnsafe loads launch configs without locking (internal use)
func (m *Manager) loadLaunchConfigsUnsafe() (map[string]*LaunchConfig, error) {
	data, err := os.ReadFile(m.launchConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]*LaunchConfig{}, nil
		}
		return nil, err
	}

	var configs map[string]*LaunchConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to parse launch config: %w", err)
	}

	if configs == nil {
		return map[string]*LaunchConfig{}, nil
	}

	return configs, nil
}

// saveLaunchConfigsUnsafe saves launch configs without locking (internal use)
func (m *Manager) saveLaunchConfigsUnsafe(configs map[string]*LaunchConfig) error {
	// Ensure directory exists
	configDir := filepath.Dir(m.launchConfigPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal launch config: %w", err)
	}

	// Write to temp file
	tempPath := m.launchConfigPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write launch config: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, m.launchConfigPath); err != nil {
		utils.RemoveQuietly(tempPath)
		return fmt.Errorf("failed to rename launch config file: %w", err)
	}

	return nil
}

// LoadLaunchConfig loads a specific model's launch configuration
func (m *Manager) LoadLaunchConfig(modelID string) (*LaunchConfig, error) {
	configs, err := m.LoadLaunchConfigs()
	if err != nil {
		return nil, err
	}

	config, exists := configs[modelID]
	if !exists {
		return DefaultLaunchConfig(), nil
	}

	return config, nil
}

// SaveModelAlias saves or updates a model's alias
func (m *Manager) SaveModelAlias(modelID, alias string) error {
	models, err := m.LoadModelsConfig()
	if err != nil {
		return err
	}

	// Find and update existing model, or add new entry
	found := false
	for i := range models {
		if models[i].ModelID == modelID {
			models[i].Alias = alias
			found = true
			break
		}
	}

	if !found {
		models = append(models, ModelConfigEntry{
			ModelID:   modelID,
			Alias:     alias,
			Favourite: false,
		})
	}

	return m.SaveModelsConfig(models)
}

// LoadAliasMap loads all model aliases as a map
func (m *Manager) LoadAliasMap() (map[string]string, error) {
	models, err := m.LoadModelsConfigCached()
	if err != nil {
		return nil, err
	}

	aliases := make(map[string]string)
	for _, m := range models {
		if m.Alias != "" {
			aliases[m.ModelID] = m.Alias
		}
	}

	return aliases, nil
}

// SaveModelFavourite saves or updates a model's favourite status
func (m *Manager) SaveModelFavourite(modelID string, favourite bool) error {
	models, err := m.LoadModelsConfig()
	if err != nil {
		return err
	}

	// Find and update existing model, or add new entry
	found := false
	for i := range models {
		if models[i].ModelID == modelID {
			models[i].Favourite = favourite
			found = true
			break
		}
	}

	if !found {
		models = append(models, ModelConfigEntry{
			ModelID:   modelID,
			Favourite: favourite,
		})
	}

	return m.SaveModelsConfig(models)
}

// LoadFavouriteMap loads all model favourite statuses as a map
func (m *Manager) LoadFavouriteMap() (map[string]bool, error) {
	models, err := m.LoadModelsConfigCached()
	if err != nil {
		return nil, err
	}

	favourites := make(map[string]bool)
	for _, m := range models {
		favourites[m.ModelID] = m.Favourite
	}

	return favourites, nil
}

// InvalidateCache invalidates the internal cache
func (m *Manager) InvalidateCache() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cachedModels = nil
	m.cachedModelsTime = 0
}

// GetConfigModTime returns the modification time of the config file
func (m *Manager) GetConfigModTime() (time.Time, error) {
	info, err := os.Stat(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// WatchConfig watches for config file changes (basic implementation)
// In a production system, you might use fsnotify for more efficient watching
func (m *Manager) WatchConfig(interval time.Duration, onChange func(*Config, error)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	lastModTime, _ := m.GetConfigModTime()

	for range ticker.C {
		currentModTime, err := m.GetConfigModTime()
		if err != nil {
			onChange(nil, err)
			continue
		}

		if currentModTime.After(lastModTime) {
			config, err := m.Load()
			onChange(config, err)
			lastModTime = currentModTime
		}
	}
}
