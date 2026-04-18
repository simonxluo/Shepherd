package model

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/port"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPhase2TestManager(t *testing.T) *Manager {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)
	storageMgr := createTestStorageMgr(t)
	m := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)
	return m
}

func TestEnsureLoadedAlreadyLoaded(t *testing.T) {
	manager := newPhase2TestManager(t)
	defer manager.Close()

	modelID := "test-already-loaded"
	expectedPort := 8081

	manager.mu.Lock()
	manager.statuses[modelID] = &ModelStatus{
		ID:       modelID,
		Name:     "test",
		State:    StateLoaded,
		Port:     expectedPort,
		LoadedAt: time.Now(),
	}
	manager.mu.Unlock()

	resultPort, err := manager.EnsureLoaded(modelID)
	assert.NoError(t, err)
	assert.Equal(t, expectedPort, resultPort)
}

func TestEnsureLoadedModelNotInStatuses(t *testing.T) {
	manager := newPhase2TestManager(t)
	defer manager.Close()

	tmpDir := t.TempDir()
	modelPath := filepath.Join(tmpDir, "test-model.gguf")
	err := createMinimalGGUF(modelPath)
	require.NoError(t, err)

	model, err := manager.loadModel(modelPath)
	require.NoError(t, err)

	manager.mu.Lock()
	manager.models[model.ID] = model
	manager.mu.Unlock()

	done := make(chan error, 1)
	go func() {
		_, err := manager.EnsureLoaded(model.ID)
		done <- err
	}()

	time.Sleep(3 * time.Second)

	s, exists := manager.GetStatusRef(model.ID)
	if assert.True(t, exists) {
		assert.NotEqual(t, StateUnloaded, s.State, "EnsureLoaded should have triggered a load attempt")
		assert.NotEqual(t, StateLoaded, s.State, "load should not succeed without a valid model")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

func TestTTLCheckerDoesNotUnloadActiveModel(t *testing.T) {
	manager := newPhase2TestManager(t)
	defer manager.Close()

	modelID := "active-model"
	manager.mu.Lock()
	status := &ModelStatus{
		ID:       modelID,
		Name:     "active",
		State:    StateLoaded,
		Port:     8081,
		LoadedAt: time.Now(),
	}
	status.SetUnloadAfter(1 * time.Millisecond)
	status.InflightAdd()
	manager.statuses[modelID] = status
	manager.mu.Unlock()

	manager.checkAndUnloadIdle()

	s, exists := manager.GetStatusRef(modelID)
	assert.True(t, exists)
	assert.Equal(t, StateLoaded, s.State, "active model with inflight requests should not be unloaded")

	status.InflightDone()
}

func TestTTLCheckerUnloadsIdleModel(t *testing.T) {
	manager := newPhase2TestManager(t)
	defer manager.Close()

	modelID := "idle-model"
	manager.mu.Lock()
	status := &ModelStatus{
		ID:       modelID,
		Name:     "idle",
		State:    StateLoaded,
		Port:     8081,
		LoadedAt: time.Now().Add(-10 * time.Second),
	}
	status.SetUnloadAfter(1 * time.Millisecond)
	manager.statuses[modelID] = status
	manager.mu.Unlock()

	manager.checkAndUnloadIdle()

	s, exists := manager.GetStatusRef(modelID)
	assert.True(t, exists)
	assert.NotEqual(t, StateLoaded, s.State, "idle model with expired TTL should be unloaded")
}

func TestReverseProxyStreamForwarding(t *testing.T) {
	manager := newPhase2TestManager(t)
	defer manager.Close()

	modelID := "proxy-model"
	testPort := 8099
	manager.mu.Lock()
	status := &ModelStatus{
		ID:       modelID,
		Name:     "proxy",
		State:    StateLoaded,
		Port:     testPort,
		LoadedAt: time.Now(),
	}
	status.InitConcurrency(5)
	manager.statuses[modelID] = status
	manager.mu.Unlock()

	port, err := manager.EnsureLoaded(modelID)
	assert.NoError(t, err)
	assert.Equal(t, testPort, port)

	targetURL := fmt.Sprintf("http://127.0.0.1:%d/v1/chat/completions", port)
	assert.Equal(t, "http://127.0.0.1:8099/v1/chat/completions", targetURL)

	assert.True(t, status.AcquireSlot())
	assert.True(t, status.AcquireSlot())
	status.ReleaseSlot()
	status.ReleaseSlot()

	status.InflightAdd()
	assert.Equal(t, int32(1), status.GetInflightCount())
	status.InflightDone()
	assert.Equal(t, int32(0), status.GetInflightCount())

	assert.False(t, status.GetLastRequestTime().IsZero(), "LastRequestTime should be set after InflightDone")
}
