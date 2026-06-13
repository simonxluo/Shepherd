package vllm

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/simonxluo/Shepherd/internal/backend"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
)

// DiscoverVariant is the shared discovery logic for vLLM-family backends.
// binaryName is the CLI binary to search for (e.g. "vllm", "vllm-omni").
// Exported for reuse by vllmomni.
func DiscoverVariant(cfg *backend.Config, pluginID backend.ID, displayName, binaryName string) (*backend.Info, error) {
	info := &backend.Info{
		ID:          pluginID,
		DisplayName: displayName,
	}
	if cfg == nil {
		return info, nil
	}

	// Carry global extra_args through to BuildStartConfig.
	if raw, ok := cfg.Raw["extra_args"]; ok {
		if s, ok := raw.(string); ok {
			info.GlobalExtraArgs = s
		}
	}

	env := backend.BuildEnvWithVars(cfgEnvVars(cfg))

	// 1. Try ServeBin (explicitly configured binary path).
	serveBin := cfgString(cfg, "serve_bin")
	if serveBin != "" {
		cmd := exec.Command(serveBin, "--version")
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err == nil {
			info.Version = strings.TrimSpace(string(out))
			info.Available = true
			info.BinPath = serveBin
			return info, nil
		}
		logger.Warnf("%s ServeBin not available: path=%s", displayName, serveBin)
	}

	// 2. Try BinPaths (configured directories or executables).
	for _, p := range cfg.BinPaths {
		if fi, err := os.Stat(p); err == nil && fi.Mode().IsRegular() && fi.Mode().Perm()&0111 != 0 {
			cmd := exec.Command(p, "--version")
			cmd.Env = env
			if out, err := cmd.CombinedOutput(); err == nil {
				info.Version = strings.TrimSpace(string(out))
				info.Available = true
				info.BinPath = p
				return info, nil
			}
		}
		candidate := filepath.Join(p, binaryName)
		cmd := exec.Command(candidate, "--version")
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err == nil {
			info.Version = strings.TrimSpace(string(out))
			info.Available = true
			info.BinPath = candidate
			return info, nil
		}
	}

	// 3. Try conda environment.
	condaEnv := cfgString(cfg, "conda_env")
	if condaEnv == "" {
		return info, nil
	}
	condaPath := cfgString(cfg, "conda_path")
	if condaPath == "" {
		condaPath = "conda"
	}

	cmd := exec.Command(condaPath, "run", "--no-banner", "-n", condaEnv, binaryName, "--version")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Warnf("%s discovery failed: condaEnv=%s, error=%v, output=%s", displayName, condaEnv, err, string(out))
		return info, nil
	}

	info.Version = strings.TrimSpace(string(out))
	info.Available = true
	info.CondaEnv = condaEnv
	info.CondaPath = condaPath
	return info, nil
}

// cfgString reads a string value from Config.Raw.
func cfgString(cfg *backend.Config, key string) string {
	if cfg.Raw == nil {
		return ""
	}
	v, _ := cfg.Raw[key].(string)
	return v
}

// cfgEnvVars reads the env list from Config.Raw.
func cfgEnvVars(cfg *backend.Config) []string {
	if cfg.Raw == nil {
		return nil
	}
	raw, ok := cfg.Raw["env"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
