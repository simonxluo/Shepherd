// Package config provides configuration migration utilities
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Migrator handles configuration migration from old to new format
type Migrator struct {
	verbose bool // Enable verbose logging
}

// NewMigrator creates a new configuration migrator
func NewMigrator(verbose bool) *Migrator {
	return &Migrator{
		verbose: verbose,
	}
}

// OldMasterConfig represents the old master.config.yaml format
type OldMasterConfig struct {
	Master struct {
		Enabled         bool `yaml:"enabled"`
		Port            int  `yaml:"port"`
		ClientConfigDir string `yaml:"client_config_dir"`
	} `yaml:"master"`
}

// OldClientConfig represents the old client.config.yaml format
type OldClientConfig struct {
	Client struct {
		Enabled       bool `yaml:"enabled"`
		MasterAddress string `yaml:"master_address"`
	} `yaml:"client"`
}

// log 辅助日志方法
func (m *Migrator) log(format string, args ...interface{}) {
	if m.verbose {
		fmt.Printf("[Migrator] "+format+"\n", args...)
	}
}

// MigrateMasterConfig migrates old master.config.yaml to new unified format
func (m *Migrator) MigrateMasterConfig(oldPath, newPath string) error {
	m.log("开始迁移 Master 配置: %s -> %s", oldPath, newPath)

	// 读取旧配置
	oldData, err := os.ReadFile(oldPath)
	if err != nil {
		return fmt.Errorf("读取旧配置失败: %w", err)
	}

	var oldConfig OldMasterConfig
	if err := yaml.Unmarshal(oldData, &oldConfig); err != nil {
		return fmt.Errorf("解析旧配置失败: %w", err)
	}

	// 如果旧配置中 Master 未启用，返回警告
	if !oldConfig.Master.Enabled {
		return fmt.Errorf("旧配置中 Master 未启用，跳过迁移")
	}

	// 读取新配置模板（如果存在）
	newConfig := DefaultConfig()
	if fileInfo, err := os.Stat(newPath); err == nil && !fileInfo.IsDir() {
		// 新配置文件存在，读取它
		newData, err := os.ReadFile(newPath)
		if err == nil {
			if err := yaml.Unmarshal(newData, newConfig); err != nil {
				m.log("解析现有新配置失败，使用默认值: %v", err)
			}
		}
	}

	// 迁移配置
	newConfig.Node.Role = "master"
	newConfig.Node.MasterRole.Enabled = true
	newConfig.Node.MasterRole.Port = oldConfig.Master.Port

	// 保留旧的 Master.Enabled 以兼容性
	newConfig.Master.Enabled = true

	// 写入新配置
	newData, err := yaml.Marshal(newConfig)
	if err != nil {
		return fmt.Errorf("序列化新配置失败: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	if err := os.WriteFile(newPath, newData, 0644); err != nil {
		return fmt.Errorf("写入新配置失败: %w", err)
	}

	m.log("Master 配置迁移成功")
	return nil
}

// MigrateClientConfig migrates old client.config.yaml to new unified format
func (m *Migrator) MigrateClientConfig(oldPath, newPath string) error {
	m.log("开始迁移 Client 配置: %s -> %s", oldPath, newPath)

	// 读取旧配置
	oldData, err := os.ReadFile(oldPath)
	if err != nil {
		return fmt.Errorf("读取旧配置失败: %w", err)
	}

	var oldConfig OldClientConfig
	if err := yaml.Unmarshal(oldData, &oldConfig); err != nil {
		return fmt.Errorf("解析旧配置失败: %w", err)
	}

	// 如果旧配置中 Client 未启用，返回警告
	if !oldConfig.Client.Enabled {
		return fmt.Errorf("旧配置中 Client 未启用，跳过迁移")
	}

	// 读取新配置模板（如果存在）
	newConfig := DefaultConfig()
	if fileInfo, err := os.Stat(newPath); err == nil && !fileInfo.IsDir() {
		// 新配置文件存在，读取它
		newData, err := os.ReadFile(newPath)
		if err == nil {
			if err := yaml.Unmarshal(newData, newConfig); err != nil {
				m.log("解析现有新配置失败，使用默认值: %v", err)
			}
		}
	}

	// 迁移配置
	newConfig.Node.Role = "client"
	newConfig.Node.ClientRole.Enabled = true
	newConfig.Node.ClientRole.MasterAddress = oldConfig.Client.MasterAddress

	// 保留旧的 Client.Enabled 以兼容性
	newConfig.Client.Enabled = true
	newConfig.Client.MasterAddress = oldConfig.Client.MasterAddress

	// 写入新配置
	newData, err := yaml.Marshal(newConfig)
	if err != nil {
		return fmt.Errorf("序列化新配置失败: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	if err := os.WriteFile(newPath, newData, 0644); err != nil {
		return fmt.Errorf("写入新配置失败: %w", err)
	}

	m.log("Client 配置迁移成功")
	return nil
}

// BackupConfig backs up the old configuration file
func (m *Migrator) BackupConfig(configPath string) (string, error) {
	// 读取原文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 生成备份文件名
	baseName := filepath.Base(configPath)
	ext := filepath.Ext(configPath)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	backupName := fmt.Sprintf("%s.backup_%s%s", nameWithoutExt, timestamp(), ext)
	backupPath := filepath.Join(filepath.Dir(configPath), backupName)

	// 写入备份文件
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", fmt.Errorf("写入备份文件失败: %w", err)
	}

	m.log("配置已备份到: %s", backupPath)
	return backupPath, nil
}

// NeedsMigration checks if the configuration needs migration
func NeedsMigration(configPath string) bool {
	// 检查是否存在旧配置文件
	_, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		return false
	}

	// 读取并检查配置格式
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}

	// 检查是否包含旧的配置结构
	var config struct {
		Node struct {
			Role string `yaml:"role"`
		} `yaml:"node"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return false
	}

	// 如果没有 Node.Role 字段，则需要迁移
	return config.Node.Role == ""
}

// DetectOldConfigFile detects the old configuration file type
func DetectOldConfigFile(configPath string) string {
	baseName := strings.ToLower(filepath.Base(configPath))

	if strings.Contains(baseName, "master") {
		return "master"
	}
	if strings.Contains(baseName, "client") {
		return "client"
	}
	if strings.Contains(baseName, "server") || strings.Contains(baseName, "standalone") {
		return "standalone"
	}

	return "unknown"
}

// AutoMigrate automatically detects and migrates old configuration
func (m *Migrator) AutoMigrate(configPath string, mode string) error {
	// 检查是否需要迁移
	if !NeedsMigration(configPath) {
		m.log("配置已是新格式，无需迁移")
		return nil
	}

	configType := DetectOldConfigFile(configPath)
	m.log("检测到旧配置类型: %s", configType)

	// 备份原配置
	backupPath, err := m.BackupConfig(configPath)
	if err != nil {
		m.log("备份配置失败: %v", err)
	} else {
		m.log("原配置已备份到: %s", backupPath)
	}

	// 根据类型迁移
	var newConfigPath string
	switch configType {
	case "master":
		// 迁移到 server.config.yaml（standalone 格式但角色为 master）
		newConfigPath = filepath.Join(filepath.Dir(configPath), "server.config.yaml")
		if err := m.MigrateMasterConfig(configPath, newConfigPath); err != nil {
			return err
		}

	case "client":
		// 迁移到 server.config.yaml（standalone 格式但角色为 client）
		newConfigPath = filepath.Join(filepath.Dir(configPath), "server.config.yaml")
		if err := m.MigrateClientConfig(configPath, newConfigPath); err != nil {
			return err
		}

	default:
		return fmt.Errorf("未知的配置类型: %s", configType)
	}

	m.log("配置迁移完成: %s -> %s", configPath, newConfigPath)
	return nil
}

// timestamp 用于生成备份文件时间戳
func timestamp() string {
	return time.Now().Format("20060102_150405")
}

// MigrateToNodeConfig 将旧的 master/client 配置迁移到新的 node 配置
// 此方法在运行时调用，自动将旧配置同步到新配置结构
func (m *Migrator) MigrateToNodeConfig(cfg *Config) error {
	m.log("开始配置迁移：旧格式 -> 新 Node 格式")

	// 如果 node 配置已经设置且不是默认值，跳过
	if cfg.Node.Role != "" && cfg.Node.Role != "hybrid" {
		m.log("Node 配置已存在，跳过迁移")
		return nil
	}

	// 如果 node.role 为空，默认使用 hybrid
	if cfg.Node.Role == "" {
		cfg.Node.Role = "hybrid"
		m.log("设置 Node.Role = hybrid (默认)")
	}

	// 迁移 Client 配置
	if cfg.Client.Enabled {
		m.log("检测到旧的 Client 配置，开始迁移...")
		cfg.Node.ClientRole.Enabled = true
		if cfg.Node.ClientRole.MasterAddress == "" {
			cfg.Node.ClientRole.MasterAddress = cfg.Client.MasterAddress
			m.log("  迁移 MasterAddress: %s", cfg.Client.MasterAddress)
		}
		if cfg.Node.ClientRole.RegisterRetry == 0 {
			cfg.Node.ClientRole.RegisterRetry = cfg.Client.RegisterRetry
			m.log("  迁移 RegisterRetry: %d", cfg.Client.RegisterRetry)
		}
		// 优先使用较新的心跳间隔配置
		if cfg.Client.HeartbeatInterval > 0 {
			cfg.Node.ClientRole.HeartbeatInterval = cfg.Client.HeartbeatInterval
			m.log("  迁移 HeartbeatInterval: %d", cfg.Client.HeartbeatInterval)
		}
		if cfg.Client.HeartbeatTimeout > 0 {
			cfg.Node.ClientRole.HeartbeatTimeout = cfg.Client.HeartbeatTimeout
			m.log("  迁移 HeartbeatTimeout: %d", cfg.Client.HeartbeatTimeout)
		}
		// 检查 heartbeat 块配置
		if cfg.Client.Heartbeat.Interval > cfg.Node.ClientRole.HeartbeatInterval {
			cfg.Node.ClientRole.HeartbeatInterval = cfg.Client.Heartbeat.Interval
			m.log("  使用 heartbeat.interval: %d (更大)", cfg.Client.Heartbeat.Interval)
		}
		if cfg.Client.Heartbeat.Timeout > cfg.Node.ClientRole.HeartbeatTimeout {
			cfg.Node.ClientRole.HeartbeatTimeout = cfg.Client.Heartbeat.Timeout
			m.log("  使用 heartbeat.timeout: %d (更大)", cfg.Client.Heartbeat.Timeout)
		}
		m.log("Client 配置迁移完成")
	}

	// 迁移 Master 配置
	if cfg.Master.Enabled {
		m.log("检测到旧的 Master 配置，开始迁移...")
		cfg.Node.MasterRole.Enabled = true
		// Master Scheduler 配置保持不变
		// Scheduler 是 Master 专用配置，将在运行时从 Master 读取
		m.log("Master 配置迁移完成")
	}

	m.log("配置迁移成功")
	return nil
}
