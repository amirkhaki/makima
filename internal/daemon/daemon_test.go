package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/amirkhaki/makima/internal/engine"
	"github.com/amirkhaki/makima/internal/tracker"
)

type mockTracker struct {
	events chan tracker.Event
}

func (m *mockTracker) Name() string             { return "mock" }
func (m *mockTracker) Start(ctx context.Context) error { return nil }
func (m *mockTracker) Stop() error              { return nil }
func (m *mockTracker) Events() <-chan tracker.Event { return m.events }

func TestDaemonEventLoop(t *testing.T) {
	state := tracker.NewState()
	sessionMgr := engine.NewSessionManager()
	actionExecutor := engine.NewActionExecutor(state, nil)
	ruleEngine := engine.NewEngine(state, engine.NewSessionManager())

	mock := &mockTracker{
		events: make(chan tracker.Event, 10),
	}

	d := NewDaemon(state, sessionMgr, actionExecutor, ruleEngine, nil)
	d.AddTracker(mock)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := d.Run(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
