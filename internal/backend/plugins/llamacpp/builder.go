package llamacpp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/simonxluo/Shepherd/internal/backend"
	"github.com/simonxluo/Shepherd/internal/comm/utils"
)

// buildArgs constructs the llama-server argument list from plugin params
// and the common LoadRequest.
func buildArgs(p *Params, req *backend.LoadRequest, serverBin string) (backend.CommandSpec, error) {
	args := []string{
		"-m", req.ModelPath,
		"--port", strconv.Itoa(req.Port),
		"--host", req.BindHost,
	}

	// Context and batch
	if p.CtxSize > 0 {
		args = append(args, "-c", strconv.Itoa(p.CtxSize))
	}
	if p.BatchSize > 0 {
		args = append(args, "-b", strconv.Itoa(p.BatchSize))
	}
	if p.Threads > 0 {
		args = append(args, "-t", strconv.Itoa(p.Threads))
	}
	if p.ThreadsBatch > 0 {
		args = append(args, "-tb", strconv.Itoa(p.ThreadsBatch))
	}

	// GPU offloading
	if p.GPULayers > 0 {
		args = append(args, "-ngl", strconv.Itoa(p.GPULayers))
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

	// Multimodal
	if p.MmprojPath != "" {
		args = append(args, "--mmproj", p.MmprojPath)
	}
	if p.MmprojOffload {
		args = append(args, "--mmproj-offload")
	}

	// Performance
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

	// Server features
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

	// KV cache
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

	// Runtime
	if p.Timeout > 0 {
		args = append(args, "--timeout", strconv.Itoa(p.Timeout))
	}
	if p.Alias != "" {
		args = append(args, "--alias", p.Alias)
	}

	// Sampling: basic
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

	// Mirostat
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

	// XTC
	if p.XTCProbability > 0 {
		args = append(args, "--xtc-probability", fmt.Sprintf("%.4f", p.XTCProbability))
		if p.XTCThreshold > 0 && p.XTCThreshold != 1.0 {
			args = append(args, "--xtc-threshold", fmt.Sprintf("%.4f", p.XTCThreshold))
		}
	}

	// Chat template (inline)
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

	// Reasoning
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

	// LoRA
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

	// YaRN
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
	args = appendSpecDecodingArgs(args, p.SpecDecoding)

	cmdSpec := backend.NewCommandSpec(serverBin, args, nil, "")
	cmdSpec = cmdSpec.AppendRaw(p.CustomCmd)
	cmdSpec = cmdSpec.AppendRaw(p.ExtraParams)
	return cmdSpec, nil
}

// findServerBin locates the llama-server binary under the given base path.
func findServerBin(basePath string) (string, error) {
	bin := utils.FindLlamacppBinary(basePath, "server")
	if bin == "" {
		return "", fmt.Errorf("llama-server not found in path: %s", basePath)
	}
	return bin, nil
}
