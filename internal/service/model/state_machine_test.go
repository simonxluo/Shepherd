package model

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		name     string
		from     LoadState
		to       LoadState
		expected bool
	}{
		{"unloaded->loading", StateUnloaded, StateLoading, true},
		{"loading->loaded", StateLoading, StateLoaded, true},
		{"loading->error", StateLoading, StateError, true},
		{"loading->unloading", StateLoading, StateUnloading, true},
		{"loaded->unloading", StateLoaded, StateUnloading, true},
		{"loaded->error", StateLoaded, StateError, true},
		{"unloading->unloaded", StateUnloading, StateUnloaded, true},
		{"unloading->error", StateUnloading, StateError, true},
		{"error->unloaded", StateError, StateUnloaded, true},
		{"error->loading", StateError, StateLoading, true},

		{"unloaded->loaded", StateUnloaded, StateLoaded, false},
		{"unloaded->unloading", StateUnloaded, StateUnloading, false},
		{"unloaded->error", StateUnloaded, StateError, false},
		{"unloaded->unloaded", StateUnloaded, StateUnloaded, false},
		{"loaded->loading", StateLoaded, StateLoading, false},
		{"loaded->loaded", StateLoaded, StateLoaded, false},
		{"loading->unloaded", StateLoading, StateUnloaded, false},
		{"unloading->loading", StateUnloading, StateLoading, false},
		{"unloading->loaded", StateUnloading, StateLoaded, false},
		{"error->loaded", StateError, StateLoaded, false},
		{"error->unloading", StateError, StateUnloading, false},
		{"error->error", StateError, StateError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isValidTransition(tt.from, tt.to))
		})
	}
}

func TestSwapState(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		s := &ModelStatus{State: StateUnloaded}
		err := s.swapState(StateUnloaded, StateLoading)
		require.NoError(t, err)
		assert.Equal(t, StateLoading, s.State)
	})

	t.Run("wrong expected state", func(t *testing.T) {
		s := &ModelStatus{State: StateLoaded}
		err := s.swapState(StateUnloaded, StateLoading)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid state transition")
		assert.Equal(t, StateLoaded, s.State)
	})

	t.Run("forbidden transition", func(t *testing.T) {
		s := &ModelStatus{State: StateUnloaded}
		err := s.swapState(StateUnloaded, StateLoaded)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden state transition")
		assert.Equal(t, StateUnloaded, s.State)
	})

	t.Run("concurrent cas failure", func(t *testing.T) {
		s := &ModelStatus{State: StateUnloaded}
		var wg sync.WaitGroup
		successes := 0
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := s.swapState(StateUnloaded, StateLoading); err == nil {
					successes++
				}
			}()
		}
		wg.Wait()
		assert.Equal(t, 1, successes)
		assert.Equal(t, StateLoading, s.State)
	})
}

func TestTransitionTo(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		s := &ModelStatus{State: StateUnloaded}
		err := s.transitionTo(StateLoading)
		require.NoError(t, err)
		assert.Equal(t, StateLoading, s.State)
	})

	t.Run("forbidden transition", func(t *testing.T) {
		s := &ModelStatus{State: StateUnloaded}
		err := s.transitionTo(StateLoaded)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden state transition")
		assert.Equal(t, StateUnloaded, s.State)
	})

	t.Run("full lifecycle", func(t *testing.T) {
		s := &ModelStatus{State: StateUnloaded}
		require.NoError(t, s.transitionTo(StateLoading))
		assert.Equal(t, StateLoading, s.State)
		require.NoError(t, s.transitionTo(StateLoaded))
		assert.Equal(t, StateLoaded, s.State)
		require.NoError(t, s.transitionTo(StateUnloading))
		assert.Equal(t, StateUnloading, s.State)
		require.NoError(t, s.transitionTo(StateUnloaded))
		assert.Equal(t, StateUnloaded, s.State)
	})

	t.Run("error recovery", func(t *testing.T) {
		s := &ModelStatus{State: StateLoading}
		require.NoError(t, s.transitionTo(StateError))
		assert.Equal(t, StateError, s.State)
		require.NoError(t, s.transitionTo(StateUnloaded))
		assert.Equal(t, StateUnloaded, s.State)
		require.NoError(t, s.transitionTo(StateLoading))
		assert.Equal(t, StateLoading, s.State)
	})
}

func TestModelStatusInflight(t *testing.T) {
	s := &ModelStatus{ID: "test"}

	assert.Equal(t, int32(0), s.GetInflightCount())

	s.InflightAdd()
	assert.Equal(t, int32(1), s.GetInflightCount())

	s.InflightAdd()
	assert.Equal(t, int32(2), s.GetInflightCount())

	s.InflightDone()
	assert.Equal(t, int32(1), s.GetInflightCount())
	assert.False(t, s.GetLastRequestTime().IsZero())

	done := make(chan struct{})
	go func() {
		s.InflightWait()
		close(done)
	}()

	s.InflightDone()
	assert.Equal(t, int32(0), s.GetInflightCount())

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("InflightWait did not unblock")
	}
}

func TestModelStatusConcurrency(t *testing.T) {
	s := &ModelStatus{ID: "test"}
	s.InitConcurrency(3)
	assert.Equal(t, 3, s.ConcurrencyLimit)

	assert.True(t, s.AcquireSlot())
	assert.True(t, s.AcquireSlot())
	assert.True(t, s.AcquireSlot())

	s.ReleaseSlot()
	s.ReleaseSlot()

	assert.True(t, s.AcquireSlot())
	assert.True(t, s.AcquireSlot())

	s.ReleaseSlot()
	s.ReleaseSlot()
	s.ReleaseSlot()
}

func TestModelStatusConcurrencyExhaustion(t *testing.T) {
	s := &ModelStatus{ID: "test"}
	s.InitConcurrency(2)

	assert.True(t, s.AcquireSlot())
	assert.True(t, s.AcquireSlot())
	assert.False(t, s.AcquireSlot())

	s.ReleaseSlot()
	assert.True(t, s.AcquireSlot())
	assert.False(t, s.AcquireSlot())

	s.ReleaseSlot()
	s.ReleaseSlot()
}

func TestModelStatusConcurrencyNil(t *testing.T) {
	s := &ModelStatus{ID: "test"}

	assert.True(t, s.AcquireSlot())
	s.ReleaseSlot()
}

func TestModelStatusConcurrencyZero(t *testing.T) {
	s := &ModelStatus{ID: "test"}
	s.InitConcurrency(0)

	assert.Nil(t, s.ConcurrencySem)
	assert.True(t, s.AcquireSlot())
	s.ReleaseSlot()
}
