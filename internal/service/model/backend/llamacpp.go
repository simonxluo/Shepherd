package backend

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/utils"
)

// LlamaCppBackend implements Backend for llama.cpp (llama-server)
type LlamaCppBackend struct{}

// NewLlamaCppBackend creates a new llama.cpp backend instance
func NewLlamaCppBackend() *LlamaCppBackend {
	return &LlamaCppBackend{}
}

func (b *LlamaCppBackend) Type() BackendType { return BackendLlamaCpp }

// LlamaCppProbeResult contains llama.cpp installation probe details.
type LlamaCppProbeResult struct {
	Path      string   `json:"path"`
	Binary    string   `json:"binary"`
	Version   string   `json:"version"`
	Warnings  []string `json:"warnings"`
	Available bool     `json:"available"`
}

// ProbeLlamaCppInstallation probes a llama.cpp installation path for llama-server.
func ProbeLlamaCppInstallation(path string) (*LlamaCppProbeResult, error) {
	result := &LlamaCppProbeResult{
		Path:     path,
		Warnings: []string{},
	}

	serverBin := utils.FindLlamacppBinary(path, "server")
	if serverBin == "" {
		result.Warnings = append(result.Warnings, fmt.Sprintf("llama-server not found in path: %s", path))
		return result, nil
	}

	info, err := os.Stat(serverBin)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to stat llama-server: %v", err))
		return result, nil
	}
	if !info.Mode().IsRegular() {
		result.Warnings = append(result.Warnings, fmt.Sprintf("llama-server is not a regular file: %s", serverBin))
		return result, nil
	}
	if info.Mode().Perm()&0111 == 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("llama-server is not executable: %s", serverBin))
		return result, nil
	}

	result.Binary = serverBin
	result.Available = true

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, serverBin, "--version")
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		result.Warnings = append(result.Warnings, "llama-server --version timed out")
		return result, nil
	}
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to run llama-server --version: %v", err))
		return result, nil
	}

	result.Version = strings.TrimSpace(string(output))
	if result.Version == "" {
		result.Warnings = append(result.Warnings, "llama-server --version returned empty output")
	}

	return result, nil
}

// Discover validates that llama-server is available at the configured path
func (b *LlamaCppBackend) Discover(cfg *BackendConfig) (*BackendInfo, error) {
	info := &BackendInfo{
		Type: BackendLlamaCpp,
		Name: "LlamaCpp",
	}

	if cfg.BinPath == "" {
		// Try all configured paths first
		for _, p := range cfg.BinPaths {
			if p == "" {
				continue
			}
			probe, err := ProbeLlamaCppInstallation(p)
			if err != nil {
				return nil, err
			}
			if probe.Available {
				info.BinPath = p
				info.Version = probe.Version
				info.Available = true
				return info, nil
			}
		}
		// Try common locations
		commonPaths := []string{
			"/usr/local/bin",
			"/usr/bin",
			"./llama.cpp",
		}
		for _, p := range commonPaths {
			probe, err := ProbeLlamaCppInstallation(p)
			if err != nil {
				return nil, err
			}
			if probe.Available {
				info.BinPath = p
				info.Version = probe.Version
				info.Available = true
				return info, nil
			}
		}
		info.Available = false
		return info, nil
	}

	probe, err := ProbeLlamaCppInstallation(cfg.BinPath)
	if err != nil {
		return nil, err
	}
	if !probe.Available {
		info.Available = false
		return info, nil
	}

	info.BinPath = cfg.BinPath
	info.Version = probe.Version
	info.Available = true
	return info, nil
}

// BuildStartConfig constructs the llama-server command line
func (b *LlamaCppBackend) BuildStartConfig(info *BackendInfo, req *LoadRequest) (*StartConfig, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if info.BinPath == "" {
		return nil, fmt.Errorf("llama.cpp binary path not set")
	}

	serverBin := utils.FindLlamacppBinary(info.BinPath, "server")
	if serverBin == "" {
		return nil, fmt.Errorf("llama-server not found in path: %s", info.BinPath)
	}

	bindHost := req.BindHost
	if bindHost == "" {
		bindHost = "0.0.0.0"
	}

	args := []string{
		"-m", req.ModelPath,
		"--port", strconv.Itoa(req.Port),
		"--host", bindHost,
	}

	p := req.LlamacppParams
	if p == nil {
		p = &LlamacppLoadParams{}
	}

	// Context and batch size
	if req.CtxSize > 0 {
		args = append(args, "-c", strconv.Itoa(req.CtxSize))
	}
	if p.BatchSize > 0 {
		args = append(args, "-b", strconv.Itoa(p.BatchSize))
	}
	if req.Threads > 0 {
		args = append(args, "-t", strconv.Itoa(req.Threads))
	}
	if p.ThreadsBatch > 0 {
		args = append(args, "-tb", strconv.Itoa(p.ThreadsBatch))
	}

	// GPU configuration
	if req.GPULayers > 0 {
		args = append(args, "-ngl", strconv.Itoa(req.GPULayers))
	}

	// GPU device selection
	if len(req.Devices) > 0 {
		if p.SplitMode != "" {
			args = append(args, "-sm", p.SplitMode)
		} else if len(req.Devices) == 1 {
			args = append(args, "-sm", "none")
		}
		args = append(args, "-dev", strings.Join(req.Devices, ","))
		if len(req.Devices) == 1 {
			args = append(args, "-mg", strconv.Itoa(p.MainGPU))
		}
	}
	if p.TensorSplit != "" {
		args = append(args, "-ts", p.TensorSplit)
	}

	// Vision/Multimodal support
	if p.MmprojPath != "" {
		args = append(args, "--mmproj", p.MmprojPath)
	}
	if p.MmprojOffload {
		args = append(args, "--mmproj-offload")
	}

	// Performance feature flags
	if p.FlashAttention {
		args = append(args, "-fa", "on")
	}
	if p.NoMmap {
		args = append(args, "--no-mmap")
	}
	if p.LockMemory {
		args = append(args, "--mlock")
	}
	if p.DirectIO != "" {
		args = append(args, "--direct-io", p.DirectIO)
	}

	// Server feature flags
	if p.NoWebUI {
		args = append(args, "--no-webui")
	}
	if p.EnableMetrics {
		args = append(args, "--metrics")
	}
	if p.SlotSavePath != "" {
		args = append(args, "--slot-save-path", p.SlotSavePath)
	}
	if p.CacheRAM != 0 {
		args = append(args, "--cache-ram", strconv.Itoa(p.CacheRAM))
	}

	// Chat template
	if p.ChatTemplateFile != "" {
		args = append(args, "--chat-template-file", p.ChatTemplateFile)
	}

	// Batch processing
	if p.UBatchSize > 0 {
		args = append(args, "--ubatch-size", strconv.Itoa(p.UBatchSize))
	}
	if p.ParallelSlots > 0 {
		args = append(args, "--parallel", strconv.Itoa(p.ParallelSlots))
	}

	// KV cache configuration
	if p.KVCacheTypeK != "" {
		args = append(args, "-ctk", p.KVCacheTypeK)
	}
	if p.KVCacheTypeV != "" {
		args = append(args, "-ctv", p.KVCacheTypeV)
	}
	if p.KVCacheUnified {
		args = append(args, "-kvu")
	}
	if p.KVOffload {
		args = append(args, "--kv-offload")
	}
	if p.CacheIdleSlots {
		args = append(args, "--cache-idle-slots")
	}
	if p.CacheReuse > 0 {
		args = append(args, "--cache-reuse", strconv.Itoa(p.CacheReuse))
	}
	if p.CtxCheckpoints > 0 {
		args = append(args, "--ctx-checkpoints", strconv.Itoa(p.CtxCheckpoints))
	}
	if p.CheckpointMinStep > 0 {
		args = append(args, "--checkpoint-min-step", strconv.Itoa(p.CheckpointMinStep))
	}
	// Runtime configuration
	if p.Timeout > 0 {
		args = append(args, "--timeout", strconv.Itoa(p.Timeout))
	}
	if p.Alias != "" {
		args = append(args, "--alias", p.Alias)
	}

	// Basic sampling
	if p.Temperature != 0 {
		args = append(args, "--temp", fmt.Sprintf("%.4f", p.Temperature))
	}
	if p.TopP != 0 {
		args = append(args, "--top-p", fmt.Sprintf("%.4f", p.TopP))
	}
	if p.TopK > 0 {
		args = append(args, "--top-k", strconv.Itoa(p.TopK))
	}
	if p.MinP > 0 {
		args = append(args, "--min-p", fmt.Sprintf("%.4f", p.MinP))
	}
	if p.TopNSigma > 0 {
		args = append(args, "--top-n-sigma", fmt.Sprintf("%.4f", p.TopNSigma))
	}
	if p.TypicalP > 0 && p.TypicalP != 1.0 {
		args = append(args, "--typical-p", fmt.Sprintf("%.4f", p.TypicalP))
	}
	if p.Seed != 0 {
		args = append(args, "--seed", strconv.Itoa(p.Seed))
	}
	if p.NPredict > 0 {
		args = append(args, "-n", strconv.Itoa(p.NPredict))
	}
	if p.Samplers != "" {
		args = append(args, "--samplers", p.Samplers)
	}

	// Penalties
	if p.RepeatPenalty != 0 && p.RepeatPenalty != 1.0 {
		args = append(args, "--repeat-penalty", fmt.Sprintf("%.4f", p.RepeatPenalty))
	}
	if p.RepeatLastN != 0 {
		args = append(args, "--repeat-last-n", strconv.Itoa(p.RepeatLastN))
	}
	if p.PresencePenalty != 0 {
		args = append(args, "--presence-penalty", fmt.Sprintf("%.4f", p.PresencePenalty))
	}
	if p.FrequencyPenalty != 0 {
		args = append(args, "--frequency-penalty", fmt.Sprintf("%.4f", p.FrequencyPenalty))
	}
	if p.IgnoreEOS {
		args = append(args, "--ignore-eos")
	}

	// DRY sampling
	if p.DryMultiplier > 0 {
		args = append(args, "--dry-multiplier", fmt.Sprintf("%.4f", p.DryMultiplier))
		if p.DryBase > 0 {
			args = append(args, "--dry-base", fmt.Sprintf("%.4f", p.DryBase))
		}
		if p.DryAllowedLength > 0 {
			args = append(args, "--dry-allowed-length", strconv.Itoa(p.DryAllowedLength))
		}
		if p.DryPenaltyLastN != 0 {
			args = append(args, "--dry-penalty-last-n", strconv.Itoa(p.DryPenaltyLastN))
		}
		if p.DrySequenceBreakers != "" {
			args = append(args, "--dry-sequence-breaker", p.DrySequenceBreakers)
		}
	}

	// Mirostat sampling
	if p.Mirostat > 0 {
		args = append(args, "--mirostat", strconv.Itoa(p.Mirostat))
		if p.MirostatLR > 0 {
			args = append(args, "--mirostat-lr", fmt.Sprintf("%.4f", p.MirostatLR))
		}
		if p.MirostatEnt > 0 {
			args = append(args, "--mirostat-ent", fmt.Sprintf("%.4f", p.MirostatEnt))
		}
	}

	// Dynamic temperature
	if p.DynaTempRange > 0 {
		args = append(args, "--dynatemp-range", fmt.Sprintf("%.4f", p.DynaTempRange))
		if p.DynaTempExp != 0 && p.DynaTempExp != 1.0 {
			args = append(args, "--dynatemp-exp", fmt.Sprintf("%.4f", p.DynaTempExp))
		}
	}

	// XTC sampling
	if p.XTCProbability > 0 {
		args = append(args, "--xtc-probability", fmt.Sprintf("%.4f", p.XTCProbability))
		if p.XTCThreshold > 0 && p.XTCThreshold != 1.0 {
			args = append(args, "--xtc-threshold", fmt.Sprintf("%.4f", p.XTCThreshold))
		}
	}

	// Chat template
	if p.DisableJinja {
		args = append(args, "--no-jinja")
	}
	if p.ChatTemplate != "" {
		args = append(args, "--chat-template", p.ChatTemplate)
	}
	if p.ChatTemplateKwargs != "" {
		args = append(args, "--chat-template-kwargs", p.ChatTemplateKwargs)
	}
	if p.ContextShift {
		args = append(args, "--context-shift")
	}

	// Server optimization
	if p.ContBatching {
		args = append(args, "--cont-batching")
	} else if !p.ContBatching && p.ExtraParams == "" {
		args = append(args, "--no-cont-batching")
	}
	if p.CachePrompt {
		args = append(args, "--cache-prompt")
	}
	if p.ReusePort {
		args = append(args, "--reuse-port")
	}
	if p.SleepIdleSeconds != 0 {
		args = append(args, "--sleep-idle-seconds", strconv.Itoa(p.SleepIdleSeconds))
	}
	if p.ThreadsHTTP > 0 {
		args = append(args, "--threads-http", strconv.Itoa(p.ThreadsHTTP))
	}
	if p.SlotPromptSimilarity > 0 {
		args = append(args, "--slot-prompt-similarity", fmt.Sprintf("%.4f", p.SlotPromptSimilarity))
	}

	// Reasoning / thinking
	if p.Reasoning != "" {
		args = append(args, "--reasoning", p.Reasoning)
	}
	if p.ReasoningFormat != "" {
		args = append(args, "--reasoning-format", p.ReasoningFormat)
	}
	if p.ReasoningBudget != 0 {
		args = append(args, "--reasoning-budget", strconv.Itoa(p.ReasoningBudget))
	}

	// Embedding / reranking
	if p.LogitsAll {
		args = append(args, "--logits-all")
	}
	if p.Reranking {
		args = append(args, "--reranking")
	}
	if p.Pooling != "" {
		args = append(args, "--pooling", p.Pooling)
	}
	if p.EmbdNormalize != 0 {
		args = append(args, "--embd-normalize", strconv.Itoa(p.EmbdNormalize))
	}

	// Structured generation
	if p.Grammar != "" {
		args = append(args, "--grammar", p.Grammar)
	}
	if p.GrammarFile != "" {
		args = append(args, "--grammar-file", p.GrammarFile)
	}
	if p.JSONSchema != "" {
		args = append(args, "--json-schema", p.JSONSchema)
	}
	if p.JSONSchemaFile != "" {
		args = append(args, "--json-schema-file", p.JSONSchemaFile)
	}

	// LoRA adapters
	if p.Lora != "" {
		args = append(args, "--lora", p.Lora)
	}
	if p.LoraScaled != "" {
		args = append(args, "--lora-scaled", p.LoraScaled)
	}

	// RoPE scaling
	if p.RopeScaling != "" {
		args = append(args, "--rope-scaling", p.RopeScaling)
	}
	if p.RopeScale > 0 {
		args = append(args, "--rope-scale", fmt.Sprintf("%.4f", p.RopeScale))
	}
	if p.RopeFreqBase > 0 {
		args = append(args, "--rope-freq-base", fmt.Sprintf("%.4f", p.RopeFreqBase))
	}
	if p.RopeFreqScale > 0 {
		args = append(args, "--rope-freq-scale", fmt.Sprintf("%.4f", p.RopeFreqScale))
	}

	// YaRN extended context
	if p.YarnOrigCtx > 0 {
		args = append(args, "--yarn-orig-ctx", strconv.Itoa(p.YarnOrigCtx))
	}
	if p.YarnExtFactor != 0 {
		args = append(args, "--yarn-ext-factor", fmt.Sprintf("%.4f", p.YarnExtFactor))
	}
	if p.YarnAttnFactor != 0 {
		args = append(args, "--yarn-attn-factor", fmt.Sprintf("%.4f", p.YarnAttnFactor))
	}
	if p.YarnBetaSlow != 0 {
		args = append(args, "--yarn-beta-slow", fmt.Sprintf("%.4f", p.YarnBetaSlow))
	}
	if p.YarnBetaFast != 0 {
		args = append(args, "--yarn-beta-fast", fmt.Sprintf("%.4f", p.YarnBetaFast))
	}

	// CPU affinity & NUMA
	if p.CpuMask != "" {
		args = append(args, "--cpu-mask", p.CpuMask)
	}
	if p.CpuRange != "" {
		args = append(args, "--cpu-range", p.CpuRange)
	}
	if p.Priority != 0 {
		args = append(args, "--prio", strconv.Itoa(p.Priority))
	}
	if p.NumaStrategy != "" {
		args = append(args, "--numa", p.NumaStrategy)
	}

	// MoE CPU override
	if p.CpuMoe {
		args = append(args, "--cpu-moe")
	} else if p.NCpuMoe > 0 {
		args = append(args, "--n-cpu-moe", strconv.Itoa(p.NCpuMoe))
	}

	// Speculative decoding
	if req.SpecDecoding != nil && req.SpecDecoding.SpecType != "" && req.SpecDecoding.SpecType != "none" {
		args = append(args, "--spec-type", req.SpecDecoding.SpecType)
		switch req.SpecDecoding.SpecType {
		case "draft", "eagle3":
			if req.SpecDecoding.SpecDraftModelPath != "" {
				args = append(args, "-md", req.SpecDecoding.SpecDraftModelPath)
			}
			if req.SpecDecoding.SpecDraftNMax > 0 {
				args = append(args, "--spec-draft-n-max", strconv.Itoa(req.SpecDecoding.SpecDraftNMax))
			}
			if req.SpecDecoding.SpecDraftNMin > 0 {
				args = append(args, "--spec-draft-n-min", strconv.Itoa(req.SpecDecoding.SpecDraftNMin))
			}
			if req.SpecDecoding.SpecDraftPSplit > 0 {
				args = append(args, "--spec-draft-p-split", fmt.Sprintf("%.2f", req.SpecDecoding.SpecDraftPSplit))
			}
			if req.SpecDecoding.SpecDraftPMin > 0 {
				args = append(args, "--spec-draft-p-min", fmt.Sprintf("%.2f", req.SpecDecoding.SpecDraftPMin))
			}
			if req.SpecDecoding.SpecDraftCtxSize > 0 {
				args = append(args, "--spec-draft-ctx-size", strconv.Itoa(req.SpecDecoding.SpecDraftCtxSize))
			}
			if req.SpecDecoding.SpecDraftNGL > 0 {
				args = append(args, "--spec-draft-ngl", strconv.Itoa(req.SpecDecoding.SpecDraftNGL))
			}
			if req.SpecDecoding.SpecDraftDevice != "" {
				args = append(args, "--spec-draft-device", req.SpecDecoding.SpecDraftDevice)
			}
		case "ngram-simple":
			if req.SpecDecoding.SpecNgramSimpleSizeN > 0 {
				args = append(args, "--spec-ngram-simple-size-n", strconv.Itoa(req.SpecDecoding.SpecNgramSimpleSizeN))
			}
			if req.SpecDecoding.SpecNgramSimpleSizeM > 0 {
				args = append(args, "--spec-ngram-simple-size-m", strconv.Itoa(req.SpecDecoding.SpecNgramSimpleSizeM))
			}
			if req.SpecDecoding.SpecNgramSimpleMinHits > 0 {
				args = append(args, "--spec-ngram-simple-min-hits", strconv.Itoa(req.SpecDecoding.SpecNgramSimpleMinHits))
			}
		case "ngram-mod":
			if req.SpecDecoding.SpecNgramModNMin > 0 {
				args = append(args, "--spec-ngram-mod-n-min", strconv.Itoa(req.SpecDecoding.SpecNgramModNMin))
			}
			if req.SpecDecoding.SpecNgramModNMax > 0 {
				args = append(args, "--spec-ngram-mod-n-max", strconv.Itoa(req.SpecDecoding.SpecNgramModNMax))
			}
			if req.SpecDecoding.SpecNgramModNMatch > 0 {
				args = append(args, "--spec-ngram-mod-n-match", strconv.Itoa(req.SpecDecoding.SpecNgramModNMatch))
			}
		case "ngram-map-k":
			if req.SpecDecoding.SpecNgramMapKSizeN > 0 {
				args = append(args, "--spec-ngram-map-k-size-n", strconv.Itoa(req.SpecDecoding.SpecNgramMapKSizeN))
			}
			if req.SpecDecoding.SpecNgramMapKSizeM > 0 {
				args = append(args, "--spec-ngram-map-k-size-m", strconv.Itoa(req.SpecDecoding.SpecNgramMapKSizeM))
			}
			if req.SpecDecoding.SpecNgramMapKMinHits > 0 {
				args = append(args, "--spec-ngram-map-k-min-hits", strconv.Itoa(req.SpecDecoding.SpecNgramMapKMinHits))
			}
		case "ngram-map-k4v":
			if req.SpecDecoding.SpecNgramMapK4VSizeN > 0 {
				args = append(args, "--spec-ngram-map-k4v-size-n", strconv.Itoa(req.SpecDecoding.SpecNgramMapK4VSizeN))
			}
			if req.SpecDecoding.SpecNgramMapK4VSizeM > 0 {
				args = append(args, "--spec-ngram-map-k4v-size-m", strconv.Itoa(req.SpecDecoding.SpecNgramMapK4VSizeM))
			}
			if req.SpecDecoding.SpecNgramMapK4VMinHits > 0 {
				args = append(args, "--spec-ngram-map-k4v-min-hits", strconv.Itoa(req.SpecDecoding.SpecNgramMapK4VMinHits))
			}
		case "ngram-cache":
			if req.SpecDecoding.LookupCacheStatic != "" {
				args = append(args, "--lookup-cache-static", req.SpecDecoding.LookupCacheStatic)
			}
			if req.SpecDecoding.LookupCacheDynamic != "" {
				args = append(args, "--lookup-cache-dynamic", req.SpecDecoding.LookupCacheDynamic)
			}
		}
	}

	cmdSpec := NewCommandSpec(serverBin, args, nil, "")

	// Append custom command if provided
	cmdSpec = cmdSpec.AppendRaw(p.CustomCmd)

	// Append extra params if provided
	cmdSpec = cmdSpec.AppendRaw(p.ExtraParams)

	return &StartConfig{
		Command:     cmdSpec.RedactedPreview,
		CommandSpec: &cmdSpec,
		BinPath:     info.BinPath,
		BackendType: BackendLlamaCpp,
	}, nil
}

// IsLoadComplete detects llama.cpp load completion from stdout
func (b *LlamaCppBackend) IsLoadComplete(outputLine string) bool {
	return strings.Contains(outputLine, "all slots are idle")
}

// CheckHealth performs an HTTP health check against the llama.cpp server
func (b *LlamaCppBackend) CheckHealth(port int) (*HealthResult, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/health", port))
	if err != nil {
		return &HealthResult{Healthy: false}, err
	}
	defer utils.CloseQuietly(resp.Body)

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return &HealthResult{Healthy: false, Body: string(body)}, nil
	}

	healthy := strings.Contains(string(body), `"status":"ok"`)
	return &HealthResult{Healthy: healthy, Body: string(body)}, nil
}

// SupportsModel returns true for GGUF files
func (b *LlamaCppBackend) SupportsModel(modelPath string) bool {
	return IsGGUFModel(modelPath)
}

// SupportedEndpoints returns the endpoints supported by llama.cpp
func (b *LlamaCppBackend) SupportedEndpoints() map[string]bool {
	return endpointsWithoutAudio()
}

// quoteAndJoin joins arguments into a command string with proper quoting
func quoteAndJoin(args []string) string {
	var result string
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		if needsQuoting(arg) {
			result += `"` + escapeQuotes(arg) + `"`
		} else {
			result += arg
		}
	}
	return result
}

func needsQuoting(arg string) bool {
	for _, c := range arg {
		if c == ' ' || c == '\t' || c == '"' || c == '\'' || c == '\\' {
			return true
		}
	}
	return false
}

func escapeQuotes(s string) string {
	var result strings.Builder
	for _, c := range s {
		if c == '"' || c == '\\' {
			result.WriteRune('\\')
		}
		result.WriteRune(c)
	}
	return result.String()
}
