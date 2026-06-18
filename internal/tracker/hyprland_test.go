package tracker

import (
	"testing"
)

func TestHyprlandParser(t *testing.T) {
	// Test workspace event parsing
	event := `workspace>>3`
	state, err := ParseHyprlandEvent(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.ActiveWorkspace != 3 {
		t.Errorf("expected workspace 3, got %d", state.ActiveWorkspace)
	}

	// Test window focus event
	event = `focuswindow>>firefox`
	state, err = ParseHyprlandEvent(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.WindowClass != "firefox" {
		t.Errorf("expected window class firefox, got %s", state.WindowClass)
	}
}
