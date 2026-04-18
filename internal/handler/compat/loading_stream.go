package compat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type loadingMessage struct {
	ID      string          `json:"id"`
	Object  string          `json:"object"`
	Created int64           `json:"created"`
	Model   string          `json:"model"`
	Choices []loadingChoice `json:"choices"`
}

type loadingChoice struct {
	Index        int          `json:"index"`
	Delta        loadingDelta `json:"delta"`
	FinishReason *string      `json:"finish_reason"`
}

type loadingDelta struct {
	Role             string `json:"role,omitempty"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
	Content          string `json:"content,omitempty"`
}

func sendLoadingSSE(c *gin.Context, modelID string, cancelCtx context.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return
	}

	messages := []string{
		fmt.Sprintf("Loading model %s...", modelID),
		"Preparing GPU memory...",
		"Loading model weights...",
		"Initializing inference engine...",
		"Almost ready...",
	}

	roleMsg := loadingMessage{
		Object: "chat.completion.chunk",
		Model:  modelID,
		Choices: []loadingChoice{{
			Delta: loadingDelta{Role: "assistant"},
		}},
	}
	writeSSE(c, flusher, roleMsg)

	for i, msg := range messages {
		select {
		case <-cancelCtx.Done():
			return
		case <-c.Request.Context().Done():
			return
		default:
		}

		chunk := loadingMessage{
			Object: "chat.completion.chunk",
			Model:  modelID,
			Choices: []loadingChoice{{
				Delta: loadingDelta{ReasoningContent: msg + "\n"},
			}},
		}
		writeSSE(c, flusher, chunk)

		if i < len(messages)-1 {
			select {
			case <-cancelCtx.Done():
				return
			case <-c.Request.Context().Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}
}

func writeSSE(c *gin.Context, flusher http.Flusher, data interface{}) {
	jsonBytes, _ := json.Marshal(data)
	fmt.Fprintf(c.Writer, "data: %s\n\n", string(jsonBytes))
	flusher.Flush()
}
