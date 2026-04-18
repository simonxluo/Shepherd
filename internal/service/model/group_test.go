package model

import (
	"testing"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/port"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)
	storageMgr := createTestStorageMgr(t)
	return NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)
}

func TestSetGetGroups(t *testing.T) {
	m := newTestManager(t)

	groups := []*ModelGroup{
		{ID: "g1", Models: []string{"m1", "m2"}, Swap: true},
		{ID: "g2", Models: []string{"m3"}, Swap: false, Persistent: true},
	}
	m.SetGroups(groups)

	result := m.GetGroups()
	assert.Len(t, result, 2)
	assert.NotNil(t, result["g1"])
	assert.NotNil(t, result["g2"])
	assert.True(t, result["g1"].Swap)
	assert.False(t, result["g1"].Persistent)
	assert.True(t, result["g2"].Persistent)

	m.SetGroups([]*ModelGroup{})
	result = m.GetGroups()
	assert.Empty(t, result)
}

func TestFindGroupForModel(t *testing.T) {
	m := newTestManager(t)

	groups := []*ModelGroup{
		{ID: "chat", Models: []string{"llama-8b", "mistral-7b"}, Swap: true},
		{ID: "embed", Models: []string{"bge-large"}, Persistent: true},
	}
	m.SetGroups(groups)

	g := m.findGroupForModel("llama-8b")
	require.NotNil(t, g)
	assert.Equal(t, "chat", g.ID)

	g = m.findGroupForModel("bge-large")
	require.NotNil(t, g)
	assert.Equal(t, "embed", g.ID)

	g = m.findGroupForModel("unknown")
	assert.Nil(t, g)
}

func addLoadedStatus(m *Manager, modelID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statuses[modelID] = &ModelStatus{
		ID:       modelID,
		Name:     modelID,
		State:    StateLoaded,
		Port:     8081,
		LoadedAt: time.Now(),
	}
}

func TestSwapBeforeLoad(t *testing.T) {
	m := newTestManager(t)

	m.SetGroups([]*ModelGroup{
		{ID: "swap-group", Models: []string{"model-a", "model-b"}, Swap: true},
	})

	addLoadedStatus(m, "model-a")

	err := m.swapBeforeLoad("model-b")
	assert.NoError(t, err)

	status, exists := m.GetStatus("model-a")
	assert.True(t, exists)
	assert.NotEqual(t, StateLoaded, status.State)
}

func TestExclusiveGroup(t *testing.T) {
	m := newTestManager(t)

	m.SetGroups([]*ModelGroup{
		{ID: "exclusive", Models: []string{"ex-a"}, Swap: true, Exclusive: true},
		{ID: "normal", Models: []string{"norm-b"}, Swap: false, Persistent: false},
	})

	addLoadedStatus(m, "norm-b")

	err := m.swapBeforeLoad("ex-a")
	assert.NoError(t, err)

	status, exists := m.GetStatus("norm-b")
	assert.True(t, exists)
	assert.NotEqual(t, StateLoaded, status.State)
}

func TestPersistentGroupNotEvicted(t *testing.T) {
	m := newTestManager(t)

	m.SetGroups([]*ModelGroup{
		{ID: "exclusive", Models: []string{"ex-a"}, Swap: true, Exclusive: true},
		{ID: "persist", Models: []string{"per-b"}, Swap: false, Persistent: true},
	})

	addLoadedStatus(m, "per-b")

	err := m.swapBeforeLoad("ex-a")
	assert.NoError(t, err)

	status, exists := m.GetStatus("per-b")
	assert.True(t, exists)
	assert.Equal(t, StateLoaded, status.State)
}

func TestNoGroupNoSwap(t *testing.T) {
	m := newTestManager(t)

	addLoadedStatus(m, "standalone")

	err := m.swapBeforeLoad("standalone")
	assert.NoError(t, err)

	status, exists := m.GetStatus("standalone")
	assert.True(t, exists)
	assert.Equal(t, StateLoaded, status.State)
}
