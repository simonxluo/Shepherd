// Package model provides model capability detection functionality
package model

import (
	"strings"

	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/gguf"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/huggingface"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/storage"
)

// 能力检测关键词常量
const (
	// Rerank 关键词
	kwRerank                 = "rerank"
	kwReRank                 = "re-rank"
	kwReranker               = "reranker"
	kwRanker                 = "ranker"
	kwCrossEncoder           = "cross-encoder"
	kwCrossencoder           = "crossencoder"
	kwCrossUnderscoreEncoder = "cross_encoder"

	// Embedding 关键词
	kwEmbedding     = "embedding"
	kwEmbeddings    = "embeddings"
	kwTextEmbedding = "text-embedding"
	kwEmbed         = "embed"
	kwE5            = "e5"
	kwGTE           = "gte"
	kwJina          = "jina"
	kwNomic         = "nomic"
	kwMxbai         = "mxbai"
	kwArcticEmbed   = "arctic-embed"
	kwBGE           = "bge"

	// Tools 关键词
	kwToolCall  = "tool_call"
	kwToolCalls = "tool_calls"
	kwTools     = "tools"
	kwMCP       = "mcp"
	kwFunction  = "function"

	// Thinking 关键词
	kwEnableThinking  = "enable_thinking"
	kwThinking        = "thinking"
	kwReasoning       = "reasoning"
	kwDeepseekR1      = "deepseek-r1"
	kwDeepseekSpaceR1 = "deepseek r1"
	kwQWQ             = "qwq"
	kwQWQ32B          = "qwq-32b"

	// TTS 关键词
	kwTTS          = "tts"
	kwTextToSpeech = "text-to-speech"
	kwCosyVoice    = "cosyvoice"
	kwChatTTS      = "chattts"
	kwMelotts      = "melotts"
	kwBark         = "bark"
	kwSpeechT5     = "speecht5"
	kwVITS         = "vits"
	kwXTTS         = "xtts"

	// ASR 关键词
	kwASR           = "asr"
	kwWhisper       = "whisper"
	kwSpeechToText  = "speech-to-text"
	kwAutomaticSpeechRecognition = "automatic-speech-recognition"
	kwWav2Vec       = "wav2vec"
	kwHubert        = "hubert"
	kwSenseVoice    = "sense-voice"
	kwParaformer    = "paraformer"

	// Image Generation 关键词
	kwStableDiffusion = "stable-diffusion"
	kwStableDiffusionSDXL = "sdxl"
	kwFLUX            = "flux"
	kwDALL            = "dall-e"
	kwImageGeneration = "image-generation"
	kwTextToImage     = "text-to-image"
	kwKandinsky       = "kandinsky"
	kwPixArt          = "pixart"
	kwCogView         = "cogview"
	kwJanus           = "janus"
)

// DetectCapabilities 自动检测模型能力
// 参考 LlamacppServer 的 resolveModelType() 实现
// 基于 GGUF 元数据中的模型名称、架构和 chat_template 进行检测
func DetectCapabilities(meta *gguf.Metadata) *storage.Capabilities {
	if meta == nil {
		return &storage.Capabilities{}
	}

	caps := &storage.Capabilities{}

	// 组合模型名称和架构用于关键词检测（转换为小写一次性）
	var combined strings.Builder
	combined.Grow(len(meta.Name) + len(meta.Architecture) + 1)
	combined.WriteString(strings.ToLower(meta.Name))
	combined.WriteString(" ")
	combined.WriteString(strings.ToLower(meta.Architecture))
	combinedStr := combined.String()

	// Chat template 转换为小写（如果存在）
	var chatTemplate string
	if meta.ChatTemplate != "" {
		chatTemplate = strings.ToLower(meta.ChatTemplate)
	}

	// Rerank 检测：通过模型名称和架构关键词
	caps.Rerank = containsAny(combinedStr, []string{
		kwRerank, kwReRank, kwReranker, kwRanker,
		kwCrossEncoder, kwCrossencoder, kwCrossUnderscoreEncoder,
	})

	// Embedding 检测：通过模型名称和架构关键词
	caps.Embedding = containsAny(combinedStr, []string{
		kwEmbedding, kwEmbeddings, kwTextEmbedding, kwEmbed,
		kwE5, kwGTE, kwJina, kwNomic, kwMxbai, kwArcticEmbed, kwBGE,
	})

	// Tools 检测：通过 chat_template 关键词（仅当 chat_template 存在时）
	if chatTemplate != "" {
		caps.Tools = containsAny(chatTemplate, []string{
			kwToolCall, kwToolCalls, kwTools, kwMCP, kwFunction,
		})
	}

	// Thinking 检测：通过 chat_template 和模型名称
	thinkingFromTemplate := false
	if chatTemplate != "" {
		thinkingFromTemplate = containsAny(chatTemplate, []string{
			kwEnableThinking, kwThinking, kwReasoning,
		})
	}
	thinkingFromName := containsAny(combinedStr, []string{
		kwDeepseekR1, kwDeepseekSpaceR1, kwQWQ, kwQWQ32B,
	})
	caps.Thinking = thinkingFromTemplate || thinkingFromName

	// TTS 检测：通过模型名称关键词
	caps.TTS = containsAny(combinedStr, []string{
		kwTTS, kwTextToSpeech, kwCosyVoice, kwChatTTS,
		kwMelotts, kwBark, kwSpeechT5, kwVITS, kwXTTS,
	})

	// ASR 检测：通过模型名称关键词
	caps.ASR = containsAny(combinedStr, []string{
		kwASR, kwWhisper, kwSpeechToText,
		kwAutomaticSpeechRecognition, kwWav2Vec, kwHubert,
		kwSenseVoice, kwParaformer,
	})

	// Image Generation 检测：通过模型名称关键词
	caps.ImageGeneration = containsAny(combinedStr, []string{
		kwStableDiffusion, kwStableDiffusionSDXL, kwFLUX,
		kwDALL, kwImageGeneration, kwTextToImage,
		kwKandinsky, kwPixArt, kwCogView, kwJanus,
	})

	// 应用互斥约束（使用统一的 ApplyConstraints 方法）
	caps.ApplyConstraints()

	return caps
}

// containsAny 检查字符串是否包含任一关键词
// 这是一个优化版本，避免创建临时字符串
func containsAny(s string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

// DetectCapabilitiesFromHF detects model capabilities from HuggingFace model info
func DetectCapabilitiesFromHF(hfInfo *huggingface.HFModelInfo) *storage.Capabilities {
	caps := &storage.Capabilities{}

	if hfInfo == nil {
		return caps
	}

	combined := strings.ToLower(hfInfo.Name + " " + hfInfo.ModelType + " " + strings.Join(hfInfo.Architectures, " "))

	if hfInfo.IsDiffusers {
		diffuserLower := strings.ToLower(hfInfo.DiffuserClass)
		switch {
		case containsAny(diffuserLower, []string{"stable", "flux", "sdxl", "kandinsky", "pixart", "image"}):
			caps.ImageGeneration = true
		case containsAny(diffuserLower, []string{"speech", "tts", "voice", "audio"}):
			caps.TTS = true
		}
	}

	modelTypeLower := strings.ToLower(hfInfo.ModelType)
	switch {
	case containsAny(modelTypeLower, []string{"whisper", "wav2vec", "hubert", "speech_to_text", "sense_voice", "paraformer"}):
		caps.ASR = true
	case containsAny(modelTypeLower, []string{"speecht5", "vits", "bark", "xtts"}):
		caps.TTS = true
	}

	archLower := strings.ToLower(strings.Join(hfInfo.Architectures, " "))
	switch {
	case containsAny(archLower, []string{"whisper", "wav2vec", "hubert", "sensevoice", "paraformer"}):
		caps.ASR = true
	case containsAny(archLower, []string{"speecht5", "vits", "bark"}):
		caps.TTS = true
	}

	if !caps.ASR && !caps.TTS && !caps.ImageGeneration {
		if containsAny(combined, []string{kwASR, kwWhisper, kwSpeechToText, kwAutomaticSpeechRecognition, kwWav2Vec, kwHubert, kwSenseVoice, kwParaformer}) {
			caps.ASR = true
		}
		if containsAny(combined, []string{kwTTS, kwTextToSpeech, kwCosyVoice, kwChatTTS, kwMelotts, kwBark, kwSpeechT5, kwVITS, kwXTTS}) {
			caps.TTS = true
		}
		if containsAny(combined, []string{kwStableDiffusion, kwStableDiffusionSDXL, kwFLUX, kwDALL, kwImageGeneration, kwTextToImage, kwKandinsky, kwPixArt, kwCogView, kwJanus}) {
			caps.ImageGeneration = true
		}
	}

	caps.ApplyConstraints()

	return caps
}
