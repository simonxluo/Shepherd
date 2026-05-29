package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/simonxluo/Shepherd/internal/comm/logger"
)

// isSplitGGUF checks whether the filename represents a split GGUF shard.
// Returns: isSplit, baseName, partNumber, totalParts.
func isSplitGGUF(filename string) (bool, string, int, int) {
	// Pattern: "name-00001-of-00006.gguf"
	matches := reSplitGGUF.FindStringSubmatch(filename)
	if len(matches) == 4 {
		partNum, _ := strconv.Atoi(matches[2])
		totalParts, _ := strconv.Atoi(matches[3])
		return true, matches[1], partNum, totalParts
	}
	return false, "", 0, 0
}

// extractModelName extracts the model name from a filename, removing the split shard suffix.
func extractModelName(filename string) string {
	name := strings.TrimSuffix(filename, ".gguf")
	name = strings.TrimSuffix(name, ".GGUF")
	name = reSplitSuffix.ReplaceAllString(name, "")
	return name
}

// generateUnifiedModelID generates a unified model ID for a split/sharded model.
func generateUnifiedModelID(baseName string, partsCount int) string {
	hash := sha256.Sum256([]byte(baseName))
	hashStr := hex.EncodeToString(hash[:8])
	return fmt.Sprintf("%s-%dparts-%s", baseName, partsCount, hashStr)
}

// mergeSplitModels merges split GGUF shards into a single unified model entry.
// Must be called under m.mu write lock. Returns the number of merged groups.
func (m *Manager) mergeSplitModels() int {
	// Group by directory + base name
	groups := make(map[string][]*Model)

	for _, model := range m.models {
		if isSplit, baseName, _, totalParts := isSplitGGUF(filepath.Base(model.Path)); isSplit {
			groupKey := fmt.Sprintf("%s/%s-%dparts", filepath.Dir(model.Path), baseName, totalParts)
			groups[groupKey] = append(groups[groupKey], model)
		}
	}

	mergedCount := 0

	for groupKey, models := range groups {
		logger.Debugf("mergeSplitModels: processing group: groupKey=%s, found=%d", groupKey, len(models))

		if len(models) < 2 {
			logger.Debug("mergeSplitModels: skipping group with fewer than 2 shards")
			continue
		}

		// Sort by part number (bubble sort for small N)
		for i := 0; i < len(models); i++ {
			for j := i + 1; j < len(models); j++ {
				_, _, pi, _ := isSplitGGUF(filepath.Base(models[i].Path))
				_, _, pj, _ := isSplitGGUF(filepath.Base(models[j].Path))
				if pi > pj {
					models[i], models[j] = models[j], models[i]
				}
			}
		}

		// Validate shard contiguity
		_, _, _, totalParts := isSplitGGUF(filepath.Base(models[0].Path))
		isComplete := true
		for i, model := range models {
			_, _, partNum, _ := isSplitGGUF(filepath.Base(model.Path))
			expectedPart := i + 1
			if partNum != expectedPart {
				isComplete = false
				logger.Warnf("shard file not contiguous: path=%s, expected=%d, actual=%d", model.Path, expectedPart, partNum)
			}
		}

		if !isComplete {
			logger.Warnf("shard group incomplete: groupKey=%s, found=%d, expected=%d", groupKey, len(models), totalParts)
		}

		// Use the first shard as the primary model
		primary := models[0]

		// Calculate total size across all shards
		totalSize := int64(0)
		shardFiles := make([]string, len(models))
		for i, m := range models {
			totalSize += m.Size
			shardFiles[i] = m.Path
		}

		// Find and include mmproj file size
		mmprojSize, mmprojPath := m.findMmprojForSplit(models)

		totalSizeWithMmproj := totalSize + mmprojSize

		// Update primary model attributes
		primary.Name = extractModelName(filepath.Base(primary.Path))
		primary.TotalSize = totalSizeWithMmproj
		primary.ShardCount = len(models)
		primary.ShardFiles = shardFiles
		if mmprojPath != "" {
			primary.MmprojPath = mmprojPath
		}

		logger.Debugf("mergeSplitModels: merged: name=%s, shardCount=%d, totalSizeGB=%.2f, mmprojSizeGB=%.2f",
			primary.Name, primary.ShardCount,
			float64(totalSize)/(1024*1024*1024),
			float64(mmprojSize)/(1024*1024*1024),
		)

		// Remove other shard entries
		for i := 1; i < len(models); i++ {
			delete(m.models, models[i].ID)
		}

		// Update primary model ID to a unified one
		newID := generateUnifiedModelID(primary.Name, len(models))
		m.models[newID] = primary
		delete(m.models, primary.ID)
		primary.ID = newID

		mergedCount++
		logger.Infof("mergeSplitModels: merged shards: name=%s, shardCount=%d, totalSizeGB=%.2f",
			primary.Name, len(models),
			float64(totalSize)/(1024*1024*1024),
		)
	}

	return mergedCount
}

// findMmprojForSplit searches for a multimodal projector file associated with split model shards.
// Tries multiple naming patterns, falls back to directory search.
func (m *Manager) findMmprojForSplit(models []*Model) (int64, string) {
	if len(models) == 0 {
		return 0, ""
	}

	dir := filepath.Dir(models[0].Path)
	baseName := extractModelName(filepath.Base(models[0].Path))

	// Try specific naming patterns (ordered by priority)
	candidates := []string{
		filepath.Join(dir, "mmproj-"+baseName+".gguf"),
		filepath.Join(dir, baseName+"-mmproj.gguf"),
		filepath.Join(dir, baseName+"-mmproj-F32.gguf"),
		filepath.Join(dir, baseName+"-mmproj-f32.gguf"),
		filepath.Join(dir, baseName+"-mmproj-F16.gguf"),
		filepath.Join(dir, baseName+"-mmproj-f16.gguf"),
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil {
			logger.Infof("found mmproj file: name=%s, sizeGB=%.2f", filepath.Base(candidate), float64(info.Size())/(1024*1024*1024))
			return info.Size(), candidate
		}
	}

	// Fallback: search directory for any file containing "mmproj"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, ""
	}
	for _, entry := range entries {
		nameLower := strings.ToLower(entry.Name())
		if !entry.IsDir() && strings.Contains(nameLower, "mmproj") && strings.HasSuffix(nameLower, ".gguf") {
			fullPath := filepath.Join(dir, entry.Name())
			if info, err := os.Stat(fullPath); err == nil {
				logger.Infof("found mmproj via directory search: name=%s, sizeGB=%.2f", entry.Name(), float64(info.Size())/(1024*1024*1024))
				return info.Size(), fullPath
			}
		}
	}

	return 0, ""
}
