// Package chat provides the internal chat API handler for the web frontend.
// It replaces the heavyweight LangChainGo integration with a lightweight proxy
// that reuses the existing compat.BaseHandler reverse-proxy infrastructure.
package chat

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/handler/compat"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/model"
)

// Handler serves the internal chat API used by the web frontend.
type Handler struct {
	*compat.BaseHandler
}

// NewHandler creates a new chat handler.
func NewHandler(modelMgr *model.Manager) *Handler {
	return &Handler{
		BaseHandler: compat.NewBaseHandler(modelMgr),
	}
}

// ChatModelInfo represents a model's info for the chat UI.
type ChatModelInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Alias    string `json:"alias,omitempty"`
	State    string `json:"state"`
	IsLoaded bool   `json:"isLoaded"`
	IsVision bool   `json:"isVision"`
	Port     int    `json:"port,omitempty"`
}

// HandleChatModels returns all models with their loaded status for the chat UI.
// @Summary      List chat models
// @Description  Returns all models with their loaded status and capabilities for the chat interface
// @Tags         Chat
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /api/langchain/chat/models [get]
func (h *Handler) HandleChatModels(c *gin.Context) {
	models := h.ModelMgr.ListModels()
	statuses := h.ModelMgr.ListStatus()

	result := make([]ChatModelInfo, 0, len(models))
	for _, mdl := range models {
		info := ChatModelInfo{
			ID:       mdl.ID,
			Name:     mdl.Name,
			IsVision: mdl.MmprojPath != "",
		}
		if mdl.Alias != "" {
			info.Alias = mdl.Alias
		}
		if st, ok := statuses[mdl.ID]; ok {
			info.State = st.State.String()
			info.IsLoaded = st.State == model.StateLoaded
			info.Port = st.Port
			if st.Name != "" {
				info.Name = st.Name
			}
		} else {
			info.State = "stopped"
			info.IsLoaded = false
		}
		result = append(result, info)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"models":  result,
		"total":   len(result),
	})
}

// HandleChatCompletions proxies a chat completion request to the loaded model backend.
// @Summary      Chat completions
// @Description  Proxies an OpenAI-compatible streaming chat request to the loaded model backend with lazy-load support
// @Tags         Chat
// @Accept       json
// @Produce      json
// @Param        request body object true "Chat completion request with model and messages"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Router       /api/langchain/chat/completions [post]
func (h *Handler) HandleChatCompletions(c *gin.Context) {
	var req struct {
		Model       string      `json:"model"`
		Messages    interface{} `json:"messages"`
		Stream      bool        `json:"stream"`
		Temperature float64     `json:"temperature,omitempty"`
		MaxTokens   int         `json:"max_tokens,omitempty"`
		TopP        float64     `json:"top_p,omitempty"`
		Stop        []string    `json:"stop,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	if req.Model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model is required"})
		return
	}

	// Force stream=true for the lazy-load SSE flow, then delegate to the same
	// infrastructure used by /v1/chat/completions.
	req.Stream = true
	h.StreamWithLazyLoad(c, req.Model, "/v1/chat/completions", &req)
}

// RegisterRoutes registers the chat API routes.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	// New canonical paths
	chat := router.Group("/chat")
	{
		chat.GET("/models", h.HandleChatModels)
		chat.POST("/completions", h.HandleChatCompletions)
	}

	// Legacy /langchain/chat/* paths kept as aliases for backward compatibility
	langchain := router.Group("/langchain")
	{
		legacyChat := langchain.Group("/chat")
		{
			legacyChat.GET("/models", h.HandleChatModels)
			legacyChat.POST("/completions", h.HandleChatCompletions)
		}
	}
}
