package server

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/simonxluo/Shepherd/internal/comm/types"
	api "github.com/simonxluo/Shepherd/internal/handler"
)

const samplingConfigFile = "config/node/sampling.json"

// samplingStore holds global sampling configurations and per-model selections.
type samplingStore struct {
	mu         sync.RWMutex
	Configs    map[string]map[string]interface{} `json:"configs"`
	Selections map[string]string                 `json:"selections"` // modelID -> configName
}

var (
	samplingStoreInstance *samplingStore
	samplingStoreOnce    sync.Once
)

func getSamplingStore() *samplingStore {
	samplingStoreOnce.Do(func() {
		samplingStoreInstance = &samplingStore{
			Configs:    make(map[string]map[string]interface{}),
			Selections: make(map[string]string),
		}
		samplingStoreInstance.load()
	})
	return samplingStoreInstance
}

func (ss *samplingStore) load() {
	data, err := os.ReadFile(samplingConfigFile)
	if err != nil {
		return // File doesn't exist yet, use empty store
	}
	_ = json.Unmarshal(data, ss)
	if ss.Configs == nil {
		ss.Configs = make(map[string]map[string]interface{})
	}
	if ss.Selections == nil {
		ss.Selections = make(map[string]string)
	}
}

func (ss *samplingStore) save() error {
	if err := ensureDir("config/node"); err != nil {
		return err
	}
	data, err := json.MarshalIndent(ss, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(samplingConfigFile, data, 0644)
}

// HandleListSamplingConfigs returns all global sampling presets.
func (s *Server) HandleListSamplingConfigs(c *gin.Context) {
	store := getSamplingStore()
	store.mu.RLock()
	defer store.mu.RUnlock()

	api.Success(c, gin.H{
		"configs": store.Configs,
	})
}

// HandleSaveSamplingConfig adds or updates a named sampling configuration.
func (s *Server) HandleSaveSamplingConfig(c *gin.Context) {
	var req struct {
		SamplingConfigName string                 `json:"samplingConfigName"`
		Sampling           map[string]interface{} `json:"sampling"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "请求体解析失败")
		return
	}
	if req.SamplingConfigName == "" {
		api.BadRequest(c, "samplingConfigName不能为空")
		return
	}
	if req.Sampling == nil {
		req.Sampling = make(map[string]interface{})
	}

	store := getSamplingStore()
	store.mu.Lock()
	store.Configs[req.SamplingConfigName] = req.Sampling
	err := store.save()
	store.mu.Unlock()

	if err != nil {
		api.Error(c, types.ErrInternalError, fmt.Sprintf("保存采样配置失败: %v", err))
		return
	}

	api.Success(c, gin.H{
		"samplingConfigName": req.SamplingConfigName,
		"sampling":           req.Sampling,
	})
}

// HandleDeleteSamplingConfig deletes a named sampling configuration.
func (s *Server) HandleDeleteSamplingConfig(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		api.BadRequest(c, "配置名称不能为空")
		return
	}

	store := getSamplingStore()
	store.mu.Lock()
	_, existed := store.Configs[name]
	delete(store.Configs, name)

	// Clear selections pointing to this config
	for modelID, sel := range store.Selections {
		if sel == name {
			delete(store.Selections, modelID)
		}
	}

	err := store.save()
	store.mu.Unlock()

	if err != nil {
		api.Error(c, types.ErrInternalError, fmt.Sprintf("删除采样配置失败: %v", err))
		return
	}

	api.Success(c, gin.H{
		"deleted": existed,
	})
}

// HandleGetSamplingSelection returns which sampling config is selected for a model.
func (s *Server) HandleGetSamplingSelection(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	store := getSamplingStore()
	store.mu.RLock()
	configName := store.Selections[id]
	store.mu.RUnlock()

	api.Success(c, gin.H{
		"modelId":            id,
		"samplingConfigName": configName,
	})
}

// HandleSetSamplingSelection sets the sampling config selection for a model.
func (s *Server) HandleSetSamplingSelection(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	var req struct {
		SamplingConfigName string `json:"samplingConfigName"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "请求体解析失败")
		return
	}

	store := getSamplingStore()
	store.mu.Lock()
	if req.SamplingConfigName == "" {
		delete(store.Selections, id)
	} else {
		store.Selections[id] = req.SamplingConfigName
	}
	err := store.save()
	store.mu.Unlock()

	if err != nil {
		api.Error(c, types.ErrInternalError, fmt.Sprintf("保存采样选择失败: %v", err))
		return
	}

	api.Success(c, gin.H{
		"modelId":            id,
		"samplingConfigName": req.SamplingConfigName,
	})
}
