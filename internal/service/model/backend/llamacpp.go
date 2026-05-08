package backend

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/utils"
)

// LlamaCppBackend implements Backend for llama.cpp (llama-server)
type LlamaCppBackend struct{}

// NewLlamaCppBackend creates a new llama.cpp backend instance
func NewLlamaCppBackend() *LlamaCppBackend {
	return &LlamaCppBackend{}
}

func (b *LlamaCppBackend) Type() BackendType { return BackendLlamaCpp }

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
			if bin := utils.FindLlamacppBinary(p, "server"); bin != "" {
				info.BinPath = p
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
			if _, err := os.Stat(filepath.Join(p, "llama-server")); err == nil {
				info.BinPath = p
				info.Available = true
				return info, nil
			}
		}
		info.Available = false
		return info, nil
	}

	serverBin := utils.FindLlamacppBinary(cfg.BinPath, "server")
	if serverBin == "" {
		info.Available = false
		return info, nil
	}

	info.BinPath = cfg.BinPath
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

	args := []string{
		serverBin,
		"-m", req.ModelPath,
		"--port", strconv.Itoa(req.Port),
		"--host", "0.0.0.0",
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
	// Runtime configuration
	if p.Timeout > 0 {
		args = append(args, "--timeout", strconv.Itoa(p.Timeout))
	}
	if p.Alias != "" {
		args = append(args, "--alias", p.Alias)
	}

	// Sampling parameters
	if p.Temperature != 0 {
		args = append(args, "--temp", fmt.Sprintf("%.2f", p.Temperature))
	}
	if p.TopP != 0 {
		args = append(args, "--top-p", fmt.Sprintf("%.2f", p.TopP))
	}
	if p.TopK > 0 {
		args = append(args, "--top-k", strconv.Itoa(p.TopK))
	}
	if p.RepeatPenalty != 0 {
		args = append(args, "--repeat-penalty", fmt.Sprintf("%.2f", p.RepeatPenalty))
	}
	if p.Seed != 0 {
		args = append(args, "--seed", strconv.Itoa(p.Seed))
	}
	if p.NPredict > 0 {
		args = append(args, "-n", strconv.Itoa(p.NPredict))
	}

	// Additional sampling parameters
	if p.Reranking {
		args = append(args, "--reranking")
	}
	if p.MinP > 0 {
		args = append(args, "--min-p", fmt.Sprintf("%.2f", p.MinP))
	}
	if p.PresencePenalty != 0 {
		args = append(args, "--presence-penalty", fmt.Sprintf("%.2f", p.PresencePenalty))
	}
	if p.FrequencyPenalty != 0 {
		args = append(args, "--frequency-penalty", fmt.Sprintf("%.2f", p.FrequencyPenalty))
	}

	// Template and processing
	if p.DisableJinja {
		args = append(args, "--no-jinja")
	}
	if p.ChatTemplate != "" {
		args = append(args, "--chat-template", p.ChatTemplate)
	}
	if p.ContextShift {
		args = append(args, "--context-shift")
	}

	// Extended sampling parameters
	if p.RepeatLastN != 0 {
		args = append(args, "--repeat-last-n", strconv.Itoa(p.RepeatLastN))
	}
	if p.TypicalP > 0 {
		args = append(args, "--typical-p", fmt.Sprintf("%.2f", p.TypicalP))
	}
	if p.IgnoreEOS {
		args = append(args, "--ignore-eos")
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

	// Structured generation
	if p.Grammar != "" {
		args = append(args, "--grammar", p.Grammar)
	}
	if p.GrammarFile != "" {
		args = append(args, "--grammar-file", p.GrammarFile)
	}

	// LoRA adapter support
	if p.Lora != "" {
		args = append(args, "--lora", p.Lora)
	}
	if p.LoraScaled != "" {
		args = append(args, "--lora-scaled", p.LoraScaled)
	}

	// Chat template kwargs
	if p.ChatTemplateKwargs != "" {
		args = append(args, "--chat-template-kwargs", p.ChatTemplateKwargs)
	}

	// RoPE scaling
	if p.RopeScaling != "" {
		args = append(args, "--rope-scaling", p.RopeScaling)
	}
	if p.RopeScale > 0 {
		args = append(args, "--rope-scale", fmt.Sprintf("%.2f", p.RopeScale))
	}
	if p.RopeFreqBase > 0 {
		args = append(args, "--rope-freq-base", fmt.Sprintf("%.2f", p.RopeFreqBase))
	}
	if p.RopeFreqScale > 0 {
		args = append(args, "--rope-freq-scale", fmt.Sprintf("%.2f", p.RopeFreqScale))
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

	// Build command string
	cmd := quoteAndJoin(args)

	// Append custom command if provided
	if p.CustomCmd != "" {
		cmd += " " + strings.TrimSpace(p.CustomCmd)
	}

	// Append extra params if provided
	if p.ExtraParams != "" {
		cmd += " " + strings.TrimSpace(p.ExtraParams)
	}

	return &StartConfig{
		Command:     cmd,
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
