package tracker

import (
	"encoding/json"
	"testing"
)

func TestChromeCDPParser(t *testing.T) {
	jsonData := `[
		{
			"id": "ABC123",
			"title": "Example Page",
			"url": "https://example.com/page",
			"type": "page"
		},
		{
			"id": "DEF456",
			"title": "Another Page",
			"url": "https://another.com",
			"type": "page"
		}
	]`

	var rawTabs []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &rawTabs); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	tabs := ParseTabInfo(rawTabs)
	if len(tabs) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(tabs))
	}

	if tabs[0].ID != "ABC123" {
		t.Errorf("expected ID ABC123, got %s", tabs[0].ID)
	}
	if tabs[0].Title != "Example Page" {
		t.Errorf("expected title Example Page, got %s", tabs[0].Title)
	}
	if tabs[0].URL != "https://example.com/page" {
		t.Errorf("expected URL https://example.com/page, got %s", tabs[0].URL)
	}
	if tabs[0].Domain != "example.com" {
		t.Errorf("expected domain example.com, got %s", tabs[0].Domain)
	}

	if tabs[1].Domain != "another.com" {
		t.Errorf("expected domain another.com, got %s", tabs[1].Domain)
	}
}

func TestChromeTrackerState(t *testing.T) {
	state := NewState()

	tabs := []TabInfo{
		{ID: "1", URL: "https://github.com/user/repo", Title: "GitHub", Domain: "github.com"},
		{ID: "2", URL: "https://stackoverflow.com/questions", Title: "SO", Domain: "stackoverflow.com"},
	}

	tracker := &ChromeTracker{state: state}
	tracker.updateTab(tabs[0])

	got := state.GetBrowser()
	if got.URL != "https://github.com/user/repo" {
		t.Errorf("expected URL github.com, got %s", got.URL)
	}
	if got.Domain != "github.com" {
		t.Errorf("expected domain github.com, got %s", got.Domain)
	}
	if got.TabTitle != "GitHub" {
		t.Errorf("expected title GitHub, got %s", got.TabTitle)
	}
}
