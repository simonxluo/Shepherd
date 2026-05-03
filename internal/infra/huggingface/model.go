package huggingface

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type HFModelConfig struct {
	ModelType      string   `json:"model_type"`
	Architectures  []string `json:"architectures"`
	NameOrPath     string   `json:"_name_or_path"`
	IsEncoderDecoder bool   `json:"is_encoder_decoder"`
}

type HFModelIndex struct {
	ClassName        string `json:"_class_name"`
	DiffusersVersion string `json:"_diffusers_version"`
}

type HFModelInfo struct {
	Name           string
	DirName        string
	ModelType      string
	Architectures  []string
	PipelineTag    string
	IsDiffusers    bool
	DiffuserClass  string
	DirPath        string
	TotalSize      int64
}

func ReadModelInfo(dirPath string) (*HFModelInfo, error) {
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, fmt.Errorf("无法访问模型目录: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("路径不是目录: %s", dirPath)
	}

	hfInfo := &HFModelInfo{
		DirPath: dirPath,
		Name:    filepath.Base(dirPath),
		DirName: filepath.Base(dirPath),
	}

	hfInfo.TotalSize, _ = calcDirSize(dirPath)

	configPath := filepath.Join(dirPath, "config.json")
	if data, err := os.ReadFile(configPath); err == nil {
		var cfg HFModelConfig
		if err := json.Unmarshal(data, &cfg); err == nil {
			hfInfo.ModelType = cfg.ModelType
			hfInfo.Architectures = cfg.Architectures
			if cfg.NameOrPath != "" {
				hfInfo.Name = cfg.NameOrPath
			}
		}
	}

	indexConfigPath := filepath.Join(dirPath, "model_index.json")
	if data, err := os.ReadFile(indexConfigPath); err == nil {
		var idx HFModelIndex
		if err := json.Unmarshal(data, &idx); err == nil {
			hfInfo.IsDiffusers = true
			hfInfo.DiffuserClass = idx.ClassName
		}
	}

	hfInfo.Name = cleanModelName(hfInfo.Name, dirPath)

	return hfInfo, nil
}

func IsHuggingFaceModelDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}

	configPath := filepath.Join(path, "config.json")
	if _, err := os.Stat(configPath); err == nil {
		return true
	}

	modelIndexPath := filepath.Join(path, "model_index.json")
	if _, err := os.Stat(modelIndexPath); err == nil {
		if hasSafetensorsFiles(path) {
			return true
		}
	}

	return false
}

func hasSafetensorsFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".safetensors") {
			return true
		}
	}
	return false
}

func calcDirSize(dir string) (int64, error) {
	var size int64
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		size += fi.Size()
	}
	return size, nil
}

func cleanModelName(name string, dirPath string) string {
	if name == "" || name == "." {
		name = filepath.Base(dirPath)
	}

	if strings.Contains(name, "models--") {
		parts := strings.SplitN(name, "--", 3)
		if len(parts) >= 3 {
			name = parts[len(parts)-1]
		}
	}

	name = strings.ReplaceAll(name, "--", "/")

	return name
}
