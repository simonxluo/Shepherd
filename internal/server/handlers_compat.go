// Package server provides compatibility API handler methods (OpenAI, Anthropic, Ollama).
package server

import "github.com/gin-gonic/gin"

func (s *Server) HandleEvents(c *gin.Context) {
	s.wsMgr.HandleWebSocket(c)
}

func (s *Server) HandleOpenAIChat(c *gin.Context) {
	s.handlers.OpenAI.HandleChatCompletions(c)
}
func (s *Server) HandleOpenAIComplete(c *gin.Context) {
	s.handlers.OpenAI.HandleCompletions(c)
}
func (s *Server) HandleOpenAIModels(c *gin.Context) {
	s.handlers.OpenAI.HandleModels(c)
}
func (s *Server) HandleAnthropicMessages(c *gin.Context) {
	s.handlers.Anthropic.HandleMessages(c)
}
func (s *Server) HandleOllamaChat(c *gin.Context) {
	s.handlers.Ollama.HandleChat(c)
}
func (s *Server) HandleOllamaTags(c *gin.Context) {
	s.handlers.Ollama.HandleTags(c)
}
