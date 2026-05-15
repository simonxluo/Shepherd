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
func (s *Server) HandleLMStudioChat(c *gin.Context) {
	s.handlers.LMStudio.HandleChatCompletions(c)
}
func (s *Server) HandleLMStudioComplete(c *gin.Context) {
	s.handlers.LMStudio.HandleCompletions(c)
}
func (s *Server) HandleLMStudioModels(c *gin.Context) {
	s.handlers.LMStudio.HandleModels(c)
}
func (s *Server) HandleLMStudioEmbeddings(c *gin.Context) {
	s.handlers.LMStudio.HandleEmbeddings(c)
}
func (s *Server) HandleCreateSpeech(c *gin.Context) {
	s.handlers.Audio.HandleCreateSpeech(c)
}
func (s *Server) HandleCreateTranscription(c *gin.Context) {
	s.handlers.Audio.HandleCreateTranscription(c)
}
func (s *Server) HandleCreateTranslation(c *gin.Context) {
	s.handlers.Audio.HandleCreateTranslation(c)
}
func (s *Server) HandleCreateImage(c *gin.Context) {
	s.handlers.Image.HandleCreateImage(c)
}
func (s *Server) HandleCreateMusic(c *gin.Context) {
	s.handlers.Music.HandleCreateMusic(c)
}
func (s *Server) HandleListVoices(c *gin.Context) {
	s.handlers.Audio.HandleListVoices(c)
}
