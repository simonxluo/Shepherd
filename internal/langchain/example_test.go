// Package langchain provides usage examples for LangChainGo integration
package langchain_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/langchain"
	"github.com/tmc/langchaingo/llms"
)

// ExampleSimplePrompt 演示简单的文本生成
func DemoSimplePrompt() {
	// 创建 LLM 实例
	llm, err := langchain.NewLlamaCPP(
		"http://localhost:8080",   // llama.cpp server URL
		"Qwen3.5-0.8B-UD-Q8_K_XL", // 模型 ID
		langchain.WithTemperature(0.7),
		langchain.WithMaxTokens(200),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 生成文本
	ctx := context.Background()
	response, err := llm.Call(ctx, "What is the capital of France?")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response) // Output: ...
}

// ExampleChatCompletion 演示聊天完成
func DemoChatCompletion() {
	// 创建 LLM 实例
	llm, err := langchain.NewLlamaCPP(
		"http://localhost:8080",
		"Qwen3.5-0.8B-UD-Q8_K_XL",
	)
	if err != nil {
		log.Fatal(err)
	}

	// 构建对话消息
	messages := []llms.MessageContent{
		{
			Role: llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{
				llms.TextPart("You are a helpful assistant that answers questions concisely."),
			},
		},
		{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextPart("What is Go programming language?"),
			},
		},
	}

	// 生成响应
	ctx := context.Background()
	response, err := llm.GenerateContent(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	if len(response.Choices) > 0 {
		fmt.Println(response.Choices[0].Content)
	}
}

// ExampleStreamingCompletion 演示流式生成
func DemoStreamingCompletion() {
	// 创建 LLM 实例
	llm, err := langchain.NewLlamaCPP(
		"http://localhost:8080",
		"Qwen3.5-0.8B-UD-Q8_K_XL",
		langchain.WithTemperature(0.8),
		langchain.WithMaxTokens(500),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 构建消息
	messages := []llms.MessageContent{
		{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextPart("Write a short poem about programming"),
			},
		},
	}

	// 流式生成
	ctx := context.Background()
	respChan, err := llm.GenerateContentStream(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	// 接收流式响应
	for response := range respChan {
		if len(response.Choices) > 0 {
			fmt.Print(response.Choices[0].Content)
		}
	}
	fmt.Println("\n[Done]")
}

// ExampleWithOptions 演示使用各种选项
func DemoWithOptions() {
	llm, err := langchain.NewLlamaCPP(
		"http://localhost:8080",
		"Qwen3.5-0.8B-UD-Q8_K_XL",
		langchain.WithTemperature(0.5), // 温度：控制随机性
		langchain.WithMaxTokens(1000),  // 最大 token 数
		langchain.WithTopP(0.9),        // Top-p 采样
		langchain.WithTopK(40),         // Top-k 采样
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	response, err := llm.Call(ctx, "Explain quantum computing in simple terms")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response) // Output: ...
}

// ExampleWithTimeout 演示使用超时
func DemoWithTimeout() {
	llm, err := langchain.NewLlamaCPP(
		"http://localhost:8080",
		"Qwen3.5-0.8B-UD-Q8_K_XL",
	)
	if err != nil {
		log.Fatal(err)
	}

	// 设置 5 秒超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := llm.Call(ctx, "Write a very short answer")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response) // Output: ...
}

// ExampleMultiTurnConversation 演示多轮对话
func DemoMultiTurnConversation() {
	llm, err := langchain.NewLlamaCPP(
		"http://localhost:8080",
		"Qwen3.5-0.8B-UD-Q8_K_XL",
	)
	if err != nil {
		log.Fatal(err)
	}

	// 构建多轮对话
	messages := []llms.MessageContent{
		{
			Role: llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{
				llms.TextPart("You are a helpful assistant."),
			},
		},
		{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextPart("My name is Alice"),
			},
		},
		{
			Role: llms.ChatMessageTypeAI,
			Parts: []llms.ContentPart{
				llms.TextPart("Nice to meet you, Alice!"),
			},
		},
		{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextPart("What's my name?"),
			},
		},
	}

	ctx := context.Background()
	response, err := llm.GenerateContent(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	if len(response.Choices) > 0 {
		fmt.Println(response.Choices[0].Content)
	}
}

// ExampleAPIUsage 演示 API 调用
func DemoAPIUsage() {
	// 简单提示 API
	// POST /api/langchain/prompt
	// {
	//   "model_id": "Qwen3.5-0.8B-UD-Q8_K_XL",
	//   "prompt": "What is the capital of {country}?",
	//   "input": {
	//     "country": "France"
	//   },
	//   "options": {
	//     "temperature": 0.7,
	//     "max_tokens": 200
	//   }
	// }

	// 聊天 API
	// POST /api/langchain/chat
	// {
	//   "model_id": "Qwen3.5-0.8B-UD-Q8_K_XL",
	//   "messages": [
	//     {"role": "system", "content": "You are a helpful assistant"},
	//     {"role": "user", "content": "Hello"}
	//   ],
	//   "options": {
	//     "temperature": 0.7
	//   }
	// }

	// 流式 API
	// POST /api/langchain/stream
	// {
	//   "model_id": "Qwen3.5-0.8B-UD-Q8_K_XL",
	//   "messages": [
	//     {"role": "user", "content": "Tell me a story"}
	//   ]
	// }

	fmt.Println("API examples require a running Shepherd server")
}

// ExampleErrorHandling 演示错误处理
func DemoErrorHandling() {
	llm, err := langchain.NewLlamaCPP(
		"http://localhost:8080",
		"Qwen3.5-0.8B-UD-Q8_K_XL",
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	response, err := llm.Call(ctx, "Hello")

	if err != nil {
		// 处理错误
		if err.Error() == "model not loaded" {
			fmt.Println("Please load the model first")
			return
		}
		log.Fatal(err)
	}

	fmt.Println(response) // Output: ...
}

// ExampleCustomHTTPClient 演示使用自定义 HTTP 客户端
func DemoCustomHTTPClient() {
	// 创建自定义 HTTP 客户端（例如，设置超时）
	customClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	llm, err := langchain.NewLlamaCPP(
		"http://localhost:8080",
		"Qwen3.5-0.8B-UD-Q8_K_XL",
		langchain.WithHTTPClient(customClient),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	response, err := llm.Call(ctx, "Hello")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response) // Output: ...
}
