package backend

import "github.com/simonxluo/Shepherd/internal/comm/config"

// SyncFromConfig populates the registry's Config entries from the application
// config.Config. This bridges the current YAML layout (top-level llamacpp +
// backends.vllm/vllm_omni) with the plugin-based Config model.
func (r *Registry) SyncFromConfig(cfg *config.Config) {
	bindHost := cfg.Server.ModelBindHost
	if bindHost == "" {
		bindHost = "0.0.0.0"
	}

	// llamacpp
	llamaCfg := &Config{
		ID:          IDLlamaCpp,
		DisplayName: "llama.cpp",
		BindHost:    bindHost,
	}
	if len(cfg.Llamacpp.Paths) > 0 {
		llamaCfg.BinPaths = make([]string, 0, len(cfg.Llamacpp.Paths))
		for _, p := range cfg.Llamacpp.Paths {
			llamaCfg.BinPaths = append(llamaCfg.BinPaths, p.Path)
		}
	}
	r.Configure(IDLlamaCpp, llamaCfg)

	// vllm
	if cfg.Backends.VLLM != nil && cfg.Backends.VLLM.Enabled {
		r.Configure(IDVLLM, vllmConfigFromYAML(IDVLLM, "vLLM", cfg.Backends.VLLM, bindHost))
	}

	// vllmomni
	if cfg.Backends.VLLMOmni != nil && cfg.Backends.VLLMOmni.Enabled {
		r.Configure(IDVLLMOmni, vllmConfigFromYAML(IDVLLMOmni, "vLLM-Omni", cfg.Backends.VLLMOmni, bindHost))
	}
}

// vllmConfigFromYAML converts a VLLMBackendConfig into a backend.Config.
func vllmConfigFromYAML(id ID, displayName string, vc *config.VLLMBackendConfig, bindHost string) *Config {
	c := &Config{
		ID:          id,
		DisplayName: displayName,
		BindHost:    bindHost,
		Raw: map[string]any{
			"conda_env":  vc.CondaEnv,
			"conda_path": vc.CondaPath,
			"serve_bin":  vc.ServeBin,
			"extra_args": vc.ExtraArgs,
			"env":        vc.Env,
		},
	}
	if len(vc.Paths) > 0 {
		c.BinPaths = make([]string, 0, len(vc.Paths))
		for _, p := range vc.Paths {
			c.BinPaths = append(c.BinPaths, p.Path)
		}
	}
	return c
}
