package ollama

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/simonxluo/Shepherd/internal/handler/compat"
	"github.com/simonxluo/Shepherd/internal/service/model"
)

type Handler struct {
	*compat.BaseHandler
}

func NewHandler(modelMgr *model.Manager) *Handler {
	return &Handler{
		BaseHandler: compat.NewBaseHandler(modelMgr),
	}
}

type ChatRequest struct {
	Model    string            `json:"model"`
	Messages []ChatMessage     `json:"messages"`
	Stream   bool              `json:"stream,omitempty"`
	Options  *GenerationParams `json:"options,omitempty"`
	Format   string            `json:"format,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GenerationParams struct {
	Temperature   float64  `json:"temperature,omitempty"`
	TopP          float64  `json:"top_p,omitempty"`
	TopK          int      `json:"top_k,omitempty"`
	NumPredict    int      `json:"num_predict,omitempty"`
	RepeatPenalty float64  `json:"repeat_penalty,omitempty"`
	Stop          []string `json:"stop,omitempty"`
}

type ChatResponse struct {
	Model     string      `json:"model"`
	CreatedAt string      `json:"created_at,omitempty"`
	Message   ChatMessage `json:"message,omitempty"`
	Done      bool        `json:"done,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// @Summary      Ollama Chat
// @Description  Ollama 兼容的聊天接口，内部转换为 OpenAI 格式转发
// @Tags         Ollama
// @Accept       json
// @Produce      json
// @Param        request  body  ChatRequest  true  "Ollama chat request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /api/chat [post]
func (h *Handler) HandleChat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.SendSimpleError(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Model == "" {
		h.SendSimpleError(c, http.StatusBadRequest, "model is required")
		return
	}

	if len(req.Messages) == 0 {
		h.SendSimpleError(c, http.StatusBadRequest, "messages array is empty")
		return
	}

	actualModelID, err := h.FindModel(req.Model)
	if err != nil {
		h.SendSimpleError(c, http.StatusNotFound, err.Error())
		return
	}

	port, err := h.GetModelPort(actualModelID)
	if err != nil {
		h.SendSimpleError(c, http.StatusInternalServerError, err.Error())
		return
	}

	openaiReq := h.convertToOpenAI(actualModelID, req)
	h.ForwardRequest(c, port, "/v1/chat/completions", actualModelID, openaiReq)
}

// @Summary      Ollama Tags
// @Description  获取模型标签列表（Ollama 格式）
// @Tags         Ollama
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/tags [post]
func (h *Handler) HandleTags(c *gin.Context) {
	c.JSON(http.StatusOK, map[string]interface{}{
		"models": []interface{}{},
	})
}

func (h *Handler) convertToOpenAI(modelID string, ollamaReq ChatRequest) map[string]interface{} {
	messages := make([]map[string]interface{}, len(ollamaReq.Messages))
	for i, msg := range ollamaReq.Messages {
		messages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	openaiReq := map[string]interface{}{
		"model":    modelID,
		"messages": messages,
		"stream":   ollamaReq.Stream,
	}

	if ollamaReq.Options != nil {
		if ollamaReq.Options.Temperature > 0 {
			openaiReq["temperature"] = ollamaReq.Options.Temperature
		}
		if ollamaReq.Options.TopP > 0 {
			openaiReq["top_p"] = ollamaReq.Options.TopP
		}
		if ollamaReq.Options.TopK > 0 {
			openaiReq["top_k"] = ollamaReq.Options.TopK
		}
	}

	return openaiReq
}
