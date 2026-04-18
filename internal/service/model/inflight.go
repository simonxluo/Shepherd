package model

import "time"

func (s *ModelStatus) InflightAdd() {
	s.InflightWg.Add(1)
	s.InflightCount.Add(1)
}

func (s *ModelStatus) InflightDone() {
	s.InflightCount.Add(-1)
	s.InflightWg.Done()
	s.mu.Lock()
	s.LastRequestTime = time.Now()
	s.mu.Unlock()
}

func (s *ModelStatus) InflightWait() {
	s.InflightWg.Wait()
}

func (s *ModelStatus) GetInflightCount() int32 {
	return s.InflightCount.Load()
}

func (s *ModelStatus) GetLastRequestTime() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.LastRequestTime
}

func (s *ModelStatus) InitConcurrency(limit int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ConcurrencyLimit = limit
	if limit > 0 {
		s.ConcurrencySem = make(chan struct{}, limit)
	}
}

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

func (s *ModelStatus) ReleaseSlot() {
	if s.ConcurrencySem == nil {
		return
	}
	select {
	case <-s.ConcurrencySem:
	default:
	}
}

func (s *ModelStatus) SetUnloadAfter(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.UnloadAfter = d
}

func (s *ModelStatus) AddTokens(prompt, completion int) {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	s.TotalPromptTokens += int64(prompt)
	s.TotalCompletionTokens += int64(completion)
}

func (s *ModelStatus) GetTokenCounts() (prompt, completion int64) {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	return s.TotalPromptTokens, s.TotalCompletionTokens
}
