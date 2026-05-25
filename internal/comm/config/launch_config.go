package config

// LaunchConfig represents model launch parameters
type LaunchConfig struct {
	CtxSize       int     `mapstructure:"ctx_size" yaml:"ctx_size" json:"ctxSize"`
	BatchSize     int     `mapstructure:"batch_size" yaml:"batch_size" json:"batchSize"`
	Threads       int     `mapstructure:"threads" yaml:"threads" json:"threads"`
	GPULayers     int     `mapstructure:"gpu_layers" yaml:"gpu_layers" json:"gpuLayers"`
	Temperature   float64 `mapstructure:"temperature" yaml:"temperature" json:"temperature"`
	TopP          float64 `mapstructure:"top_p" yaml:"top_p" json:"topP"`
	TopK          int     `mapstructure:"top_k" yaml:"top_k" json:"topK"`
	RepeatPenalty float64 `mapstructure:"repeat_penalty" yaml:"repeat_penalty" json:"repeatPenalty"`
	Seed          int     `mapstructure:"seed" yaml:"seed" json:"seed"`
	NPredict      int     `mapstructure:"n_predict" yaml:"n_predict" json:"nPredict"`
}

// DefaultLaunchConfig returns default launch parameters
func DefaultLaunchConfig() *LaunchConfig {
	return &LaunchConfig{
		CtxSize:       4096,
		BatchSize:     512,
		Threads:       -1, // Auto-detect
		GPULayers:     99,
		Temperature:   0.7,
		TopP:          0.9,
		TopK:          40,
		RepeatPenalty: 1.1,
		Seed:          -1, // Random
		NPredict:      -1, // Unlimited
	}
}
