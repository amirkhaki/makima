package tracker

import (
	"testing"
)

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/page", "example.com"},
		{"https://sub.example.com/path", "sub.example.com"},
		{"https://another.com", "another.com"},
		{"http://test.co.uk/page", "test.co.uk"},
		{"https://localhost:3000", "localhost"},
		{"", ""},
	}

	for _, tt := range tests {
		got := extractDomain(tt.url)
		if got != tt.expected {
			t.Errorf("extractDomain(%q) = %q, want %q", tt.url, got, tt.expected)
		}
	}
}

func TestChromeTrackerState(t *testing.T) {
	state := NewState()

	tab := TabInfo{
		ID:     "1",
		URL:    "https://github.com/user/repo",
		Title:  "GitHub",
		Domain: "github.com",
	}

	tracker := &ChromeTracker{state: state}
	tracker.updateTab(tab)

	got := state.GetBrowser()
	if got.URL != "https://github.com/user/repo" {
		t.Errorf("expected URL https://github.com/user/repo, got %s", got.URL)
	}
	if got.Domain != "github.com" {
		t.Errorf("expected domain github.com, got %s", got.Domain)
	}
	if got.TabTitle != "GitHub" {
		t.Errorf("expected title GitHub, got %s", got.TabTitle)
	}
}

func TestChromeTrackerName(t *testing.T) {
	state := NewState()
	tracker := NewChromeTracker(state)

	if tracker.Name() != "chrome" {
		t.Errorf("expected name chrome, got %s", tracker.Name())
	}
}
