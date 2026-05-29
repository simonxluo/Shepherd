package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/simonxluo/Shepherd/internal/comm/config"
	"github.com/simonxluo/Shepherd/internal/comm/storage"
	"github.com/simonxluo/Shepherd/internal/service/model"
)

func TestResolveTTSAudioURL(t *testing.T) {
	os.Setenv("SHEPHERD_TEST", "1")
	cfg := config.DefaultConfig()

	storageMgr, err := storage.NewManager(&cfg.Storage)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storageMgr.Close()

	ctx := context.Background()
	testID := "tts-test-ref-audio-001"
	item := &storage.TTSHistoryItem{
		ID:        testID,
		Model:     "VoxCPM2",
		InputText: "测试参考音频",
		AudioPath: "tts-1779975533100193028.wav",
		Format:    "wav",
	}
	if err := storageMgr.GetStore().CreateTTSHistory(ctx, item); err != nil {
		t.Fatalf("Failed to create TTS history: %v", err)
	}

	projectRoot := filepath.Join(os.Getenv("HOME"), "workspace", "Shepherd")
	ttsDataDir := filepath.Join(projectRoot, "data", "tts")
	modelMgr := model.NewManager(cfg, nil, nil, nil, storageMgr)
	h := NewAudioHandler(modelMgr, storageMgr, ttsDataDir)

	tests := []struct {
		name     string
		input    string
		wantFile bool
	}{
		{"matching /api/tts/audio/<id> path", "/api/tts/audio/" + testID, true},
		{"data URI should pass through", "data:audio/wav;base64,UklGRiQAAABXQVZFZm10IBAAAAABAAEA", false},
		{"HTTP URL should pass through", "http://example.com/audio.wav", false},
		{"empty string should pass through", "", false},
		{"non-existent ID should pass through", "/api/tts/audio/nonexistent-id", false},
		{"bare /api/tts/audio/ with no ID should pass through", "/api/tts/audio/", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.resolveTTSAudioURL(tt.input)
			if tt.wantFile {
				if result[:7] != "file://" {
					t.Errorf("expected file:// URL, got: %s", result)
					return
				}
				absPath := result[7:]
				if _, err := os.Stat(absPath); err != nil {
					t.Errorf("resolved file does not exist: %s (%v)", absPath, err)
				}
				t.Logf("OK: %s -> %s", tt.input, result)
			} else {
				if result != tt.input {
					t.Errorf("expected passthrough (%q), got: %q", tt.input, result)
				} else {
					t.Logf("OK: %s (unchanged)", tt.input)
				}
			}
		})
	}

	t.Run("nil storageMgr returns original", func(t *testing.T) {
		h2 := &AudioHandler{
			Handler:    NewHandler(modelMgr),
			storageMgr: nil,
			ttsDataDir: ttsDataDir,
		}
		input := "/api/tts/audio/" + testID
		result := h2.resolveTTSAudioURL(input)
		if result != input {
			t.Errorf("expected passthrough with nil storageMgr, got: %s", result)
		}
		t.Logf("OK: nil storageMgr gracefully returns original: %s", result)
	})
}

func TestResolveTTSAudioURLAbsPath(t *testing.T) {
	os.Setenv("SHEPHERD_TEST", "1")
	cfg := config.DefaultConfig()

	storageMgr, err := storage.NewManager(&cfg.Storage)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storageMgr.Close()

	ctx := context.Background()
	tmpDir := t.TempDir()
	dummyAudio := filepath.Join(tmpDir, "test-audio.wav")
	if err := os.WriteFile(dummyAudio, []byte("RIFF fake wav"), 0644); err != nil {
		t.Fatal(err)
	}

	testID := "tts-abs-test-001"
	item := &storage.TTSHistoryItem{
		ID: testID, Model: "test", InputText: "abs path test",
		AudioPath: "test-audio.wav", Format: "wav",
	}
	if err := storageMgr.GetStore().CreateTTSHistory(ctx, item); err != nil {
		t.Fatalf("Failed to create TTS history: %v", err)
	}

	modelMgr := model.NewManager(cfg, nil, nil, nil, storageMgr)
	h := NewAudioHandler(modelMgr, storageMgr, tmpDir)

	result := h.resolveTTSAudioURL("/api/tts/audio/" + testID)
	if result[:7] != "file://" {
		t.Fatalf("expected file:// prefix, got: %s", result)
	}
	expectedAbs, _ := filepath.Abs(dummyAudio)
	if result[7:] != expectedAbs {
		t.Errorf("expected %s, got %s", expectedAbs, result[7:])
	}
	t.Logf("OK: resolved to correct absolute path: %s", result[7:])
}

func TestPrepareTTSRequestValidation(t *testing.T) {
	os.Setenv("SHEPHERD_TEST", "1")
	cfg := config.DefaultConfig()
	storageMgr, err := storage.NewManager(&cfg.Storage)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storageMgr.Close()

	modelMgr := model.NewManager(cfg, nil, nil, nil, storageMgr)
	h := NewAudioHandler(modelMgr, storageMgr, "./data/tts")

	t.Run("missing model returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech", strings.NewReader(`{"input": "hello"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		_, _, ok := h.prepareTTSRequest(c)
		if ok {
			t.Error("expected ok=false for missing model")
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
		t.Log("OK: missing model returns 400")
	})

	t.Run("missing input returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech", strings.NewReader(`{"model": "test"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		_, _, ok := h.prepareTTSRequest(c)
		if ok {
			t.Error("expected ok=false for missing input")
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
		t.Log("OK: missing input returns 400")
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech", strings.NewReader(`not json`))
		c.Request.Header.Set("Content-Type", "application/json")

		_, _, ok := h.prepareTTSRequest(c)
		if ok {
			t.Error("expected ok=false for invalid JSON")
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
		t.Log("OK: invalid JSON returns 400")
	})
}
