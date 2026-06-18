package tracker

import "sync"

type BrowserState struct {
	URL        string
	TabTitle   string
	Domain     string
	Category   string
	TimeOnSite int
}

type HyprlandState struct {
	ActiveWorkspace int
	WorkspaceCount  int
	WindowClass     string
	WindowTitle     string
}

type AppStatus struct {
	Running bool
	Uptime  int
}

type State struct {
	mu      sync.RWMutex
	browser BrowserState
	hypr    HyprlandState
	app     AppStatus
}

func NewState() *State {
	return &State{}
}

func (s *State) UpdateBrowser(b BrowserState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.browser = b
}

func (s *State) UpdateHyprland(h HyprlandState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hypr = h
}

func (s *State) UpdateApp(a AppStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.app = a
}

func (s *State) GetBrowser() BrowserState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.browser
}

func (s *State) GetHyprland() HyprlandState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hypr
}

func (s *State) GetApp() AppStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.app
}
