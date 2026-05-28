package config

// ModelConfigEntry represents a model configuration entry in models.json
type ModelConfigEntry struct {
	ModelID   string `json:"modelId"`
	Path      string `json:"path,omitempty"`
	Size      int64  `json:"size,omitempty"`
	Alias     string `json:"alias,omitempty"`
	Favourite bool   `json:"favourite"`
	// 分卷模型相关字段
	TotalSize    int64             `json:"totalSize,omitempty"`  // 所有分卷的总大小
	ShardCount   int               `json:"shardCount,omitempty"` // 分卷数量
	ShardFiles   []string          `json:"shardFiles,omitempty"` // 所有分卷文件路径
	PrimaryModel *PrimaryModelInfo `json:"primaryModel,omitempty"`
	Mmproj       *MmprojInfo       `json:"mmproj,omitempty"`
}

// PrimaryModelInfo contains information about the primary model
type PrimaryModelInfo struct {
	FileName        string `json:"fileName"`
	Name            string `json:"name,omitempty"`
	Architecture    string `json:"architecture,omitempty"`
	ContextLength   int    `json:"contextLength,omitempty"`
	EmbeddingLength int    `json:"embeddingLength,omitempty"`
}

// MmprojInfo contains information about the multimodal projector
type MmprojInfo struct {
	FileName     string `json:"fileName"`
	Size         int64  `json:"size,omitempty"` // mmproj 文件大小（字节）
	Name         string `json:"name,omitempty"`
	Architecture string `json:"architecture,omitempty"`
}
