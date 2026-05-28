package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/simonxluo/Shepherd/internal/comm/types"
	api "github.com/simonxluo/Shepherd/internal/handler"
	"github.com/simonxluo/Shepherd/internal/infra/gguf"
)

const (
	templateDir = "config/node/templates"
	kwargsDir   = "config/node/kwargs"
)

// ensureDir creates a directory if it doesn't exist.
func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// HandleGetChatTemplate returns the saved custom chat template for a model.
func (s *Server) HandleGetChatTemplate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	filePath := filepath.Join(templateDir, id+".tpl")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			api.Success(c, gin.H{
				"modelId":      id,
				"exists":       false,
				"chatTemplate": "",
			})
			return
		}
		api.Error(c, types.ErrInternalError, fmt.Sprintf("读取模板文件失败: %v", err))
		return
	}

	api.Success(c, gin.H{
		"modelId":      id,
		"exists":       true,
		"chatTemplate": string(data),
	})
}

// HandleSaveChatTemplate saves a custom chat template for a model.
func (s *Server) HandleSaveChatTemplate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	var req struct {
		ChatTemplate string `json:"chatTemplate"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "请求体解析失败")
		return
	}
	if req.ChatTemplate == "" {
		api.BadRequest(c, "chatTemplate不能为空，如需删除请使用DELETE")
		return
	}

	if err := ensureDir(templateDir); err != nil {
		api.Error(c, types.ErrInternalError, fmt.Sprintf("创建目录失败: %v", err))
		return
	}

	filePath := filepath.Join(templateDir, id+".tpl")
	if err := os.WriteFile(filePath, []byte(req.ChatTemplate), 0644); err != nil {
		api.Error(c, types.ErrInternalError, fmt.Sprintf("保存模板失败: %v", err))
		return
	}

	api.Success(c, gin.H{
		"modelId": id,
		"saved":   true,
	})
}

// HandleDeleteChatTemplate deletes the saved custom chat template for a model.
func (s *Server) HandleDeleteChatTemplate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	filePath := filepath.Join(templateDir, id+".tpl")
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		api.Error(c, types.ErrInternalError, fmt.Sprintf("删除模板失败: %v", err))
		return
	}

	api.Success(c, gin.H{
		"modelId": id,
		"deleted": err == nil,
	})
}

// HandleGetDefaultChatTemplate returns the default chat template embedded in the GGUF model.
func (s *Server) HandleGetDefaultChatTemplate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	m, exists := s.modelMgr.GetModel(id)
	if !exists {
		api.NotFound(c, "模型")
		return
	}

	// Try to read chat template from GGUF metadata
	chatTemplate := ""
	if m.Metadata != nil && m.Metadata.ChatTemplate != "" {
		chatTemplate = m.Metadata.ChatTemplate
	} else {
		// Try parsing directly from file
		parser, err := gguf.NewParser(m.Path)
		if err == nil {
			defer func() { _ = parser.Close() }()
			meta, err := parser.GetMetadata()
			if err == nil && meta.ChatTemplate != "" {
				chatTemplate = meta.ChatTemplate
			}
		}
	}

	api.Success(c, gin.H{
		"modelId":      id,
		"exists":       chatTemplate != "",
		"chatTemplate": chatTemplate,
	})
}

// HandleGetChatTemplateKwargs returns the saved kwargs JSON for a model.
func (s *Server) HandleGetChatTemplateKwargs(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	filePath := filepath.Join(kwargsDir, id+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			api.Success(c, gin.H{
				"modelId":              id,
				"chat_template_kwargs": map[string]interface{}{},
			})
			return
		}
		api.Error(c, types.ErrInternalError, fmt.Sprintf("读取kwargs失败: %v", err))
		return
	}

	var kwargs map[string]interface{}
	if err := json.Unmarshal(data, &kwargs); err != nil {
		api.Error(c, types.ErrInternalError, fmt.Sprintf("解析kwargs JSON失败: %v", err))
		return
	}

	api.Success(c, gin.H{
		"modelId":              id,
		"chat_template_kwargs": kwargs,
	})
}

// HandleSaveChatTemplateKwargs saves kwargs JSON for a model.
func (s *Server) HandleSaveChatTemplateKwargs(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	var req struct {
		ChatTemplateKwargs map[string]interface{} `json:"chat_template_kwargs"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "请求体解析失败")
		return
	}
	if req.ChatTemplateKwargs == nil {
		api.BadRequest(c, "chat_template_kwargs不能为空")
		return
	}

	if err := ensureDir(kwargsDir); err != nil {
		api.Error(c, types.ErrInternalError, fmt.Sprintf("创建目录失败: %v", err))
		return
	}

	data, err := json.MarshalIndent(req.ChatTemplateKwargs, "", "  ")
	if err != nil {
		api.Error(c, types.ErrInternalError, fmt.Sprintf("序列化JSON失败: %v", err))
		return
	}

	filePath := filepath.Join(kwargsDir, id+".json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		api.Error(c, types.ErrInternalError, fmt.Sprintf("保存kwargs失败: %v", err))
		return
	}

	api.Success(c, gin.H{
		"modelId": id,
		"saved":   true,
	})
}

// HandleDeleteChatTemplateKwargs deletes the kwargs JSON for a model.
func (s *Server) HandleDeleteChatTemplateKwargs(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	filePath := filepath.Join(kwargsDir, id+".json")
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		api.Error(c, types.ErrInternalError, fmt.Sprintf("删除kwargs失败: %v", err))
		return
	}

	api.Success(c, gin.H{
		"modelId": id,
		"deleted": err == nil,
	})
}
