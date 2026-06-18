package tracker

import "testing"

func TestStateUpdate(t *testing.T) {
	s := NewState()

	s.UpdateBrowser(BrowserState{
		URL:         "https://example.com",
		TabTitle:    "Example Page",
		Domain:      "example.com",
		Category:    "work",
		TimeOnSite:  120,
	})

	s.UpdateHyprland(HyprlandState{
		ActiveWorkspace: 3,
		WorkspaceCount:  5,
		WindowClass:     "firefox",
		WindowTitle:     "Example Page - Firefox",
	})

	browser := s.GetBrowser()
	if browser.URL != "https://example.com" {
		t.Errorf("expected URL 'https://example.com', got '%s'", browser.URL)
	}
	if browser.Domain != "example.com" {
		t.Errorf("expected Domain 'example.com', got '%s'", browser.Domain)
	}
	if browser.Category != "work" {
		t.Errorf("expected Category 'work', got '%s'", browser.Category)
	}

	hypr := s.GetHyprland()
	if hypr.ActiveWorkspace != 3 {
		t.Errorf("expected ActiveWorkspace 3, got %d", hypr.ActiveWorkspace)
	}
	if hypr.WindowClass != "firefox" {
		t.Errorf("expected WindowClass 'firefox', got '%s'", hypr.WindowClass)
	}
}
