package engine

import (
	"sync"
	"time"
)

type Session struct {
	key       string
	grace     time.Duration
	cooldown  time.Duration
	started   time.Time
	fired     time.Time
	inGrace   bool
}

func NewSession(key string, grace, cooldown time.Duration) *Session {
	s := &Session{
		key:      key,
		grace:    grace,
		cooldown: cooldown,
		started:  time.Now(),
		inGrace:  grace > 0,
	}
	return s
}

func (s *Session) Key() string {
	return s.key
}

func (s *Session) InGrace() bool {
	if !s.inGrace {
		return false
	}
	return time.Since(s.started) < s.grace
}

func (s *Session) FireAction() {
	s.fired = time.Now()
	s.inGrace = false
}

func (s *Session) InCooldown() bool {
	if s.cooldown == 0 {
		return false
	}
	if s.fired.IsZero() {
		return false
	}
	return time.Since(s.fired) < s.cooldown
}

type SessionManager struct {
	mu            sync.RWMutex
	sessions      map[string]*Session
	budgetMinutes int
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

func (m *SessionManager) GetOrCreate(key string, grace, cooldown time.Duration) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.sessions[key]; ok {
		return s
	}

	s := NewSession(key, grace, cooldown)
	m.sessions[key] = s
	return s
}

func (m *SessionManager) SetBudget(minutes int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store budget in minutes for current session
	// This will be used by the engine to schedule tab close
	m.budgetMinutes = minutes
}

func (m *SessionManager) GetBudget() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.budgetMinutes
}
