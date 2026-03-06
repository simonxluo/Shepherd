package gguf

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestParserRealFiles 测试真实的 GGUF 文件
func TestParserRealFiles(t *testing.T) {
	// 从配置文件读取模型路径
	modelPaths := []string{
		"/home/user/.cache/huggingface/hub/Oss-120b/gpt-oss-120b-Derestricted.MXFP4_MOE.gguf",
		"/home/user/workspace/LlamacppServer/build/models/DavidAU/GLM-4.7-Flash-Uncensored-Heretic-NEO-CODE-Imatrix-MAX-GGUF/GLM-4.7-Flash-Uncen-Hrt-NEO-CODE-MAX-imat-D_AU-Q8_0/GLM-4.7-Flash-Uncen-Q8_0.gguf",
		"/home/user/workspace/LlamacppServer/build/models/unsloth/Qwen3-Coder-Next-GGUF/Qwen3-Coder-Next-MXFP4_MOE/Qwen3-Coder-Next-MXFP4_MOE.gguf",
		"/home/user/workspace/LlamacppServer/build/models/Qwen/Qwen3-Embedding-8B-GGUF/Qwen3-Embedding-8B-Q8_0/Qwen3-Embedding-8B-Q8_0.gguf",
	}

	for _, path := range modelPaths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Skipf("文件不存在: %s", path)
				return
			}

			parser, err := NewParser(path)
			if err != nil {
				t.Fatalf("创建解析器失败: %v", err)
			}
			defer parser.Close()

			meta, err := parser.GetMetadata()
			if err != nil {
				t.Fatalf("获取元数据失败: %v", err)
			}

			// 输出元数据用于调试
			fmt.Printf("\n========== %s ==========\n", filepath.Base(path))
			fmt.Printf("名称: %s\n", meta.Name)
			fmt.Printf("架构: %s\n", meta.Architecture)
			fmt.Printf("文件类型: %d\n", meta.FileType)
			fmt.Printf("量化: %s\n", meta.Quantization)
			fmt.Printf("参数量: %.0f\n", meta.Parameters)
			fmt.Printf("上下文长度: %d\n", meta.ContextLength)
			fmt.Printf("嵌入维度: %d\n", meta.EmbeddingLength)
			fmt.Printf("Tokenizer: %s\n", meta.TokenizerModel)
			fmt.Printf("BOS Token ID: %d\n", meta.BosTokenID)
			fmt.Printf("EOS Token ID: %d\n", meta.EosTokenID)
			fmt.Printf("文件大小: %d 字节\n", parser.GetFileSize())

			// 验证关键字段是否正确读取
			if meta.ContextLength == 0 {
				t.Errorf("❌ contextLength 为 0，架构=%s", meta.Architecture)
			}
			if meta.Architecture == "" {
				t.Errorf("❌ architecture 为空")
			}
			if meta.Parameters == 0 {
				t.Errorf("❌ parameters 为 0")
			}
		})
	}
}

// TestShardedModelSizes 测试分卷模型的文件大小读取
func TestShardedModelSizes(t *testing.T) {
	// Qwen3.5-397B-A17B 的 6 个分卷文件
	shardDir := "/home/user/workspace/LlamacppServer/build/models/unsloth/Qwen3.5-397B-A17B-GGUF/Qwen3.5-397B-A17B-MXFP4_MOE"

	shards := []string{
		"Qwen3.5-397B-A17B-MXFP4_MOE-00001-of-00006.gguf",
		"Qwen3.5-397B-A17B-MXFP4_MOE-00002-of-00006.gguf",
		"Qwen3.5-397B-A17B-MXFP4_MOE-00003-of-00006.gguf",
		"Qwen3.5-397B-A17B-MXFP4_MOE-00004-of-00006.gguf",
		"Qwen3.5-397B-A17B-MXFP4_MOE-00005-of-00006.gguf",
		"Qwen3.5-397B-A17B-MXFP4_MOE-00006-of-00006.gguf",
	}

	var totalSize int64 = 0
	shardSizes := make([]int64, len(shards))

	for i, shard := range shards {
		path := filepath.Join(shardDir, shard)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("无法读取文件 %s: %v", path, err)
		}
		shardSizes[i] = info.Size()
		totalSize += info.Size()

		fmt.Printf("分卷 %d: %s\n", i+1, shard)
		fmt.Printf("  大小: %d 字节 (%.2f GB)\n", info.Size(), float64(info.Size())/(1024*1024*1024))
	}

	fmt.Printf("\n========== 总计 ==========\n")
	fmt.Printf("分卷数量: %d\n", len(shards))
	fmt.Printf("总大小: %d 字节 (%.2f GB)\n", totalSize, float64(totalSize)/(1024*1024*1024))

	// 验证总大小应该约为 200GB
	expectedMinGB := 180.0 // 最少 180GB
	actualGB := float64(totalSize) / (1024 * 1024 * 1024)
	if actualGB < expectedMinGB {
		t.Errorf("❌ 分卷模型总大小太小: %.2f GB (期望至少 %.2f GB)", actualGB, expectedMinGB)
	}

	// 测试每个分卷的元数据读取
	fmt.Printf("\n========== 测试每个分卷的元数据 ==========\n")
	for i, shard := range shards {
		path := filepath.Join(shardDir, shard)

		parser, err := NewParser(path)
		if err != nil {
			t.Logf("分卷 %d (%s) 解析失败: %v", i+1, shard, err)
			continue
		}
		defer parser.Close()

		meta, err := parser.GetMetadata()
		if err != nil {
			t.Logf("分卷 %d (%s) 获取元数据失败: %v", i+1, shard, err)
			continue
		}

		fileSize := parser.GetFileSize()
		fmt.Printf("分卷 %d (%s):\n", i+1, shard)
		fmt.Printf("  GetFileSize(): %d 字节 (%.2f GB)\n", fileSize, float64(fileSize)/(1024*1024*1024))
		fmt.Printf("  Meta.FileSize: %d 字节\n", meta.FileSize)
		fmt.Printf("  Meta.ModelSize: %d 字节\n", meta.ModelSize)
		fmt.Printf("  Meta.Parameters: %.0f\n", meta.Parameters)
		fmt.Printf("  Meta.Architecture: %s\n", meta.Architecture)
		fmt.Printf("  Meta.Quantization: %s\n", meta.Quantization)

		// 验证 GetFileSize() 返回正确的大小
		if fileSize != shardSizes[i] {
			t.Errorf("❌ 分卷 %d GetFileSize() 不匹配: 得到 %d, 期望 %d", i+1, fileSize, shardSizes[i])
		}
	}
}

// TestGetQuantizationString 测试量化字符串映射
func TestGetQuantizationString(t *testing.T) {
	testCases := []struct {
		fileType uint32
		expected string
	}{
		{0, "F32"},
		{1, "F16"},
		{2, "Q4_0"},
		{3, "Q4_1"},
		{6, "Q5_0"},
		{7, "Q5_1"},
		{8, "Q8_0"},
		{9, "Q8_1"},
		{10, "Q2_K"},
		{11, "Q3_K_S"},
		{12, "Q3_K_M"},
		{13, "Q3_K_L"},
		{14, "Q4_K_S"},
		{15, "Q4_K_M"},
		{16, "Q5_K_S"},
		{17, "Q5_K_M"},
		{18, "Q6_K"},
		{28, "IQ4_NL"},
		{41, "MXFP4"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			meta := &Metadata{FileType: tc.fileType}
			result := meta.GetQuantizationString()
			if result != tc.expected {
				t.Errorf("FileType=%d: 期望 %s, 得到 %s", tc.fileType, tc.expected, result)
			}
		})
	}
}
