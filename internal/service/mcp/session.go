package mcp

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session represents an active MCP session for the MCP Server.
type Session struct {
	ID        string
	CreatedAt time.Time
	LastUsed  time.Time
}

// SessionManager manages MCP sessions for the server-side protocol.
type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewSessionManager creates a new session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// Create creates a new session and returns its ID.
func (sm *SessionManager) Create() string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	id := uuid.New().String()
	sm.sessions[id] = &Session{
		ID:        id,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}
	return id
}

// Get retrieves a session by ID.
func (sm *SessionManager) Get(id string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	s, ok := sm.sessions[id]
	if ok {
		s.LastUsed = time.Now()
	}
	return s, ok
}

// Delete removes a session.
func (sm *SessionManager) Delete(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, id)
}

// Cleanup removes sessions older than maxAge.
func (sm *SessionManager) Cleanup(maxAge time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for id, s := range sm.sessions {
		if s.LastUsed.Before(cutoff) {
			delete(sm.sessions, id)
		}
	}
}
