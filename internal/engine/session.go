package engine

import (
	"sync"
	"time"
)

type Session struct {
	mu        sync.RWMutex
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.key
}

func (s *Session) InGrace() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.inGrace {
		return false
	}
	return time.Since(s.started) < s.grace
}

func (s *Session) FireAction() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fired = time.Now()
	s.inGrace = false
}

func (s *Session) InCooldown() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cooldown == 0 {
		return false
	}
	if s.fired.IsZero() {
		return false
	}
	return time.Since(s.fired) < s.cooldown
}

type Budget struct {
	Minutes   int
	StartTime time.Time
	EndTime   time.Time
}

func (b *Budget) Expired() bool {
	return time.Now().After(b.EndTime)
}

func (b *Budget) Remaining() time.Duration {
	remaining := time.Until(b.EndTime)
	if remaining < 0 {
		return 0
	}
	return remaining
}

type SessionManager struct {
	mu            sync.RWMutex
	sessions      map[string]*Session
	budget        *Budget
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

	if minutes <= 0 {
		m.budget = nil
		return
	}

	now := time.Now()
	m.budget = &Budget{
		Minutes:   minutes,
		StartTime: now,
		EndTime:   now.Add(time.Duration(minutes) * time.Minute),
	}
}

func (m *SessionManager) GetBudget() *Budget {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.budget
}

func (m *SessionManager) BudgetExpired() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.budget == nil {
		return false
	}
	return m.budget.Expired()
}
