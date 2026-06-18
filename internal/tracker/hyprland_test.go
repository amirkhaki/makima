package tracker

import (
	"testing"
)

func TestHyprlandTrackerName(t *testing.T) {
	state := NewState()
	tracker := NewHyprlandTracker(state)

	if tracker.Name() != "hyprland" {
		t.Errorf("expected name hyprland, got %s", tracker.Name())
	}
}

func TestHyprlandTrackerEvents(t *testing.T) {
	state := NewState()
	tracker := NewHyprlandTracker(state)

	events := tracker.Events()
	if events == nil {
		t.Error("expected events channel to be non-nil")
	}
}
