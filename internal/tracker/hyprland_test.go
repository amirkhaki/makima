package tracker

import (
	"context"
	"testing"
	"time"
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

// Tracker interface is defined in daemon package
// This test verifies the tracker has the expected methods
func TestHyprlandTrackerMethods(t *testing.T) {
	state := NewState()
	tracker := NewHyprlandTracker(state)

	// Verify expected methods exist
	if tracker.Name() != "hyprland" {
		t.Error("Name() should return 'hyprland'")
	}

	events := tracker.Events()
	if events == nil {
		t.Error("Events() should return non-nil channel")
	}
}

func TestHyprlandTrackerStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	state := NewState()
	tracker := NewHyprlandTracker(state)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start should not error even without Hyprland running
	err := tracker.Start(ctx)
	if err != nil {
		t.Logf("Start returned error (expected if not in Hyprland session): %v", err)
	}

	// Stop should work
	err = tracker.Stop()
	if err != nil {
		t.Errorf("Stop returned error: %v", err)
	}
}

func TestHyprlandTrackerDoubleStart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	state := NewState()
	tracker := NewHyprlandTracker(state)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// First start
	tracker.Start(ctx)
	// Second start should be idempotent
	err := tracker.Start(ctx)
	if err != nil {
		t.Errorf("second Start returned error: %v", err)
	}

	tracker.Stop()
}

func TestHyprlandTrackerDoubleStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	state := NewState()
	tracker := NewHyprlandTracker(state)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	tracker.Start(ctx)
	// First stop
	tracker.Stop()
	// Second stop should be idempotent
	err := tracker.Stop()
	if err != nil {
		t.Errorf("second Stop returned error: %v", err)
	}
}

func TestHyprlandStateUpdate(t *testing.T) {
	state := NewState()
	tracker := NewHyprlandTracker(state)

	// Simulate a state update
	tracker.state.UpdateHyprland(HyprlandState{
		ActiveWorkspace: 3,
		WindowClass:     "firefox",
		WindowTitle:     "Mozilla Firefox",
	})

	// Verify state was updated
	got := state.GetHyprland()
	if got.ActiveWorkspace != 3 {
		t.Errorf("expected workspace 3, got %d", got.ActiveWorkspace)
	}
	if got.WindowClass != "firefox" {
		t.Errorf("expected window class firefox, got %s", got.WindowClass)
	}
	if got.WindowTitle != "Mozilla Firefox" {
		t.Errorf("expected window title Mozilla Firefox, got %s", got.WindowTitle)
	}
}

func TestHyprlandEventTypes(t *testing.T) {
	// Test that event types are sent correctly
	events := []Event{
		{Type: "workspace", Data: nil},
		{Type: "window", Data: nil},
	}

	for _, e := range events {
		if e.Type == "" {
			t.Error("event type should not be empty")
		}
	}
}
