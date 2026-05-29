// inflight.go provides concurrency control and request tracking for loaded models.
//
// Key mechanisms:
//   - InflightCount: atomic counter tracking the number of active requests
//   - InflightWg: WaitGroup used during unload to wait for all requests to complete
//   - ConcurrencySem: buffered channel semaphore limiting max concurrent requests
//   - AddTokens: accumulates prompt/completion tokens for usage statistics

package model

import "time"

// InflightAdd marks the start of a new inflight request.
// Increments the atomic InflightCount counter and registers with the WaitGroup.
func (s *ModelStatus) InflightAdd() {
	s.InflightWg.Add(1)
	s.InflightCount.Add(1)
}

// InflightDone marks an inflight request as complete.
// Decrements the counter, signals the WaitGroup, and updates LastRequestTime.
func (s *ModelStatus) InflightDone() {
	s.InflightCount.Add(-1)
	s.InflightWg.Done()
	s.mu.Lock()
	s.LastRequestTime = time.Now()
	s.mu.Unlock()
}

// InflightWait blocks until all inflight requests have completed.
// Used before unloading a model to ensure all active requests are drained.
func (s *ModelStatus) InflightWait() {
	s.InflightWg.Wait()
}

// GetInflightCount returns the current number of inflight requests (atomic, lock-free).
func (s *ModelStatus) GetInflightCount() int32 {
	return s.InflightCount.Load()
}

// InitConcurrency initializes the concurrency limit for this model.
// If limit > 0, creates a semaphore channel of that capacity; limit = 0 means unlimited.
func (s *ModelStatus) InitConcurrency(limit int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ConcurrencyLimit = limit
	if limit > 0 {
		s.ConcurrencySem = make(chan struct{}, limit)
	}
}

// AcquireSlot attempts to acquire a concurrency slot.
// Returns true immediately if no limit is set; returns false (non-blocking) if at capacity.
func (s *ModelStatus) AcquireSlot() bool {
	if s.ConcurrencySem == nil {
		return true
	}
	select {
	case s.ConcurrencySem <- struct{}{}:
		return true
	default:
		return false
	}
}

// ReleaseSlot releases a concurrency slot.
// Safe to call even if no limit is set or the channel is empty.
func (s *ModelStatus) ReleaseSlot() {
	if s.ConcurrencySem == nil {
		return
	}
	select {
	case <-s.ConcurrencySem:
	default:
	}
}

// SetUnloadAfter sets the idle auto-unload duration.
// d = 0 means the model will never be auto-unloaded.
func (s *ModelStatus) SetUnloadAfter(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.UnloadAfter = d
}

// AddTokens accumulates prompt and completion token counts.
// Thread-safe via the dedicated tokenMu lock to avoid contention with ModelStatus.mu.
func (s *ModelStatus) AddTokens(prompt, completion int) {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	s.TotalPromptTokens += int64(prompt)
	s.TotalCompletionTokens += int64(completion)
}
