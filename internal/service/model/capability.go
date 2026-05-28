package model

import (
	"strings"

	"github.com/simonxluo/Shepherd/internal/infra/gguf"
	"github.com/simonxluo/Shepherd/internal/infra/huggingface"
	"github.com/simonxluo/Shepherd/internal/comm/storage"
)

// 能力检测关键词常量
const (
	kwRerank                 = "rerank"
	kwReRank                 = "re-rank"
	kwReranker               = "reranker"
	kwRanker                 = "ranker"
	kwCrossEncoder           = "cross-encoder"
	kwCrossencoder           = "crossencoder"
	kwCrossUnderscoreEncoder = "cross_encoder"

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

	kwToolCall  = "tool_call"
	kwToolCalls = "tool_calls"
	kwTools     = "tools"
	kwMCP       = "mcp"
	kwFunction  = "function"

	kwEnableThinking  = "enable_thinking"
	kwThinking        = "thinking"
	kwReasoning       = "reasoning"
	kwDeepseekR1      = "deepseek-r1"
	kwDeepseekSpaceR1 = "deepseek r1"
	kwQWQ             = "qwq"
	kwQWQ32B          = "qwq-32b"

	kwTTS                        = "tts"
	kwTextToSpeech               = "text-to-speech"
	kwCosyVoice                  = "cosyvoice"
	kwChatTTS                    = "chattts"
	kwMelotts                    = "melotts"
	kwBark                       = "bark"
	kwSpeechT5                   = "speecht5"
	kwVITS                       = "vits"
	kwXTTS                       = "xtts"
	kwVoxCPM                     = "voxcpm"
	kwOmniVoice                  = "omnivoice"
	kwASR                        = "asr"
	kwWhisper                    = "whisper"
	kwSpeechToText               = "speech-to-text"
	kwAutomaticSpeechRecognition = "automatic-speech-recognition"
	kwWav2Vec                    = "wav2vec"
	kwHubert                     = "hubert"
	kwSenseVoice                 = "sense-voice"
	kwParaformer                 = "paraformer"
	kwStableDiffusion            = "stable-diffusion"
	kwStableDiffusionSDXL        = "sdxl"
	kwFLUX                       = "flux"
	kwDALL                       = "dall-e"
	kwImageGeneration            = "image-generation"
	kwTextToImage                = "text-to-image"
	kwKandinsky                  = "kandinsky"
	kwPixArt                     = "pixart"
	kwCogView                    = "cogview"
	kwJanus                      = "janus"

	kwMusicGen    = "musicgen"
	kwMusicGenAlt = "music-gen"
	kwMusicGenUS  = "music_gen"
	kwAudioGen    = "audiogen"
	kwAudioGenAlt = "audio-gen"
	kwAudioCraft  = "audiocraft"
	kwTextToMusic = "text-to-music"
	kwAceStep     = "acestep"
	kwAceStepAlt  = "ace-step"
)

// 架构→能力映射关键词表
// 参考 llama.cpp (llama-arch.h) 和 LM Studio (architectureStylizations.ts) 的架构分类
var (
	embeddingArchKeywords = []string{
		"bert", "nomic-bert", "modern-bert", "neo-bert",
		"jina-bert", "gemma-embedding", "pangu-embed",
		"llama-embed", "arctic", "eurobert",
	}

	ttsArchKeywords = []string{
		"tts", "talker", "cosyvoice", "speecht5",
		"vits", "bark", "xtts", "melotts", "chattts",
		"voice", "voxcpm", "omnivoice",
	}

	asrArchKeywords = []string{
		"asr", "whisper", "wav2vec", "hubert",
		"sensevoice", "paraformer", "speech_to_text",
		"automatic-speech-recognition",
	}

	imageGenArchKeywords = []string{
		"flux", "stable-diffusion", "sdxl",
		"kandinsky", "pixart", "cogview", "wan",
		"dall", "text-to-image", "image-generation",
	}

	musicArchKeywords = []string{
		"musicgen", "audiogen", "audiocraft",
		"music", "music-gen",
	}

	rerankNameKeywords = []string{
		kwRerank, kwReRank, kwReranker, kwRanker,
		kwCrossEncoder, kwCrossencoder, kwCrossUnderscoreEncoder,
	}

	embeddingNameKeywords = []string{
		kwEmbedding, kwEmbeddings, kwTextEmbedding, kwEmbed,
		kwE5, kwGTE, kwJina, kwNomic, kwMxbai, kwArcticEmbed, kwBGE,
	}

	ttsNameKeywords = []string{
		kwTTS, kwTextToSpeech, kwCosyVoice, kwChatTTS,
		kwMelotts, kwBark, kwSpeechT5, kwVITS, kwXTTS, kwVoxCPM, kwOmniVoice,
	}

	asrNameKeywords = []string{
		kwASR, kwWhisper, kwSpeechToText,
		kwAutomaticSpeechRecognition, kwWav2Vec, kwHubert,
		kwSenseVoice, kwParaformer,
	}

	imageNameKeywords = []string{
		kwStableDiffusion, kwStableDiffusionSDXL, kwFLUX,
		kwDALL, kwImageGeneration, kwTextToImage,
		kwKandinsky, kwPixArt, kwCogView, kwJanus,
	}

	musicNameKeywords = []string{
		kwMusicGen, kwMusicGenAlt, kwMusicGenUS,
		kwAudioGen, kwAudioGenAlt, kwAudioCraft,
		kwTextToMusic, kwAceStep, kwAceStepAlt,
	}

	toolTemplateKeywords = []string{
		kwToolCall, kwToolCalls, kwTools, kwMCP, kwFunction,
	}

	thinkingTemplateKeywords = []string{
		kwEnableThinking, kwThinking, kwReasoning,
	}

	thinkingNameKeywords = []string{
		kwDeepseekR1, kwDeepseekSpaceR1, kwQWQ, kwQWQ32B,
	}
)

// DetectCapabilities 自动检测 GGUF 模型能力
//
// 检测优先级:
// P1: general.type 过滤（非 model 类型直接返回空能力）
// P2: pooling_type（GGUF 结构化元数据，最可靠的 embedding/rerank 检测信号）
// P3: architecture 名称映射（已知架构关键词匹配）
// P4: 模型名称关键词 fallback
// P5: chat_template 检测（tools/thinking）
func DetectCapabilities(meta *gguf.Metadata) *storage.Capabilities {
	if meta == nil {
		return &storage.Capabilities{}
	}

	caps := &storage.Capabilities{}

	if meta.Type != "" && meta.Type != "model" {
		return caps
	}

	// P2: pooling_type 检测 embedding / rerank
	switch meta.PoolingType {
	case 1, 2, 3:
		caps.Embedding = true
	case 4:
		caps.Rerank = true
	}

	archLower := strings.ToLower(meta.Architecture)
	combinedLower := strings.ToLower(meta.Name + " " + meta.Architecture)

	// P3: architecture 名称映射
	if !caps.Embedding && !caps.Rerank {
		if containsAny(archLower, ttsArchKeywords) {
			caps.TTS = true
		}
		if containsAny(archLower, asrArchKeywords) {
			caps.ASR = true
		}
		if containsAny(archLower, imageGenArchKeywords) {
			caps.ImageGeneration = true
		}
		if containsAny(archLower, musicArchKeywords) {
			caps.Music = true
		}
		if containsAny(archLower, embeddingArchKeywords) {
			caps.Embedding = true
		}
	}

	// P4: 名称关键词 fallback（仅当 P2+P3 都未命中时）
	if !caps.TTS && !caps.ASR && !caps.ImageGeneration && !caps.Music && !caps.Embedding && !caps.Rerank {
		if containsAny(combinedLower, ttsNameKeywords) {
			caps.TTS = true
		}
		if containsAny(combinedLower, asrNameKeywords) {
			caps.ASR = true
		}
		if containsAny(combinedLower, imageNameKeywords) {
			caps.ImageGeneration = true
		}
		if containsAny(combinedLower, musicNameKeywords) {
			caps.Music = true
		}
		if containsAny(combinedLower, rerankNameKeywords) {
			caps.Rerank = true
		}
		if containsAny(combinedLower, embeddingNameKeywords) {
			caps.Embedding = true
		}
	}

	// P5: chat_template 检测 tools / thinking
	var chatTemplate string
	if meta.ChatTemplate != "" {
		chatTemplate = strings.ToLower(meta.ChatTemplate)
	}

	if chatTemplate != "" {
		caps.Tools = containsAny(chatTemplate, toolTemplateKeywords)
	}

	thinkingFromTemplate := false
	if chatTemplate != "" {
		thinkingFromTemplate = containsAny(chatTemplate, thinkingTemplateKeywords)
	}
	thinkingFromName := containsAny(combinedLower, thinkingNameKeywords)
	caps.Thinking = thinkingFromTemplate || thinkingFromName

	caps.ApplyConstraints()

	return caps
}

// DetectCapabilitiesFromHF 检测 HuggingFace 模型能力
//
// 检测优先级:
// P1: diffusers 类名检测
// P2: HF architectures 字段映射
// P3: HF model_type 字段映射
// P4: 名称关键词 fallback（使用 DirName + Name + ModelType + Architectures）
func DetectCapabilitiesFromHF(hfInfo *huggingface.HFModelInfo) *storage.Capabilities {
	caps := &storage.Capabilities{}

	if hfInfo == nil {
		return caps
	}

	// P1: diffusers 模型检测
	if hfInfo.IsDiffusers {
		diffuserLower := strings.ToLower(hfInfo.DiffuserClass)
		switch {
		case containsAny(diffuserLower, []string{"stable", "flux", "sdxl", "kandinsky", "pixart", "image"}):
			caps.ImageGeneration = true
		case containsAny(diffuserLower, []string{"speech", "tts", "voice", "audio"}):
			caps.TTS = true
		}
	}

	// P2: architectures 字段映射
	archLower := strings.ToLower(strings.Join(hfInfo.Architectures, " "))
	if !caps.TTS && containsAny(archLower, ttsArchKeywords) {
		caps.TTS = true
	}
	if !caps.ASR && containsAny(archLower, asrArchKeywords) {
		caps.ASR = true
	}
	if !caps.ImageGeneration && containsAny(archLower, imageGenArchKeywords) {
		caps.ImageGeneration = true
	}
	if !caps.Music && containsAny(archLower, musicArchKeywords) {
		caps.Music = true
	}
	if !caps.Embedding && containsAny(archLower, embeddingArchKeywords) {
		caps.Embedding = true
	}

	// P3: model_type 字段映射
	modelTypeLower := strings.ToLower(hfInfo.ModelType)
	switch {
	case !caps.ASR && containsAny(modelTypeLower, []string{"whisper", "wav2vec", "hubert", "speech_to_text", "sense_voice", "paraformer"}):
		caps.ASR = true
	case !caps.TTS && containsAny(modelTypeLower, []string{"speecht5", "vits", "bark", "xtts", "tts", "voxcpm", "omnivoice"}):
		caps.TTS = true
	}

	// P4: 名称关键词 fallback
	if !caps.ASR && !caps.TTS && !caps.ImageGeneration && !caps.Music && !caps.Embedding && !caps.Rerank {
		combined := strings.ToLower(hfInfo.DirName + " " + hfInfo.Name + " " + hfInfo.ModelType + " " + strings.Join(hfInfo.Architectures, " "))
		if containsAny(combined, asrNameKeywords) {
			caps.ASR = true
		}
		if containsAny(combined, ttsNameKeywords) {
			caps.TTS = true
		}
		if containsAny(combined, imageNameKeywords) {
			caps.ImageGeneration = true
		}
		if containsAny(combined, musicNameKeywords) {
			caps.Music = true
		}
		if containsAny(combined, rerankNameKeywords) {
			caps.Rerank = true
		}
		if containsAny(combined, embeddingNameKeywords) {
			caps.Embedding = true
		}
	}

	caps.ApplyConstraints()

	return caps
}

func containsAny(s string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}
