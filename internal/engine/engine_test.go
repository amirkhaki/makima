package engine

import (
	"testing"

	"github.com/makima/makima/internal/dsl"
	"github.com/makima/makima/internal/tracker"
)

func TestRuleEvaluation(t *testing.T) {
	state := tracker.NewState()
	state.UpdateBrowser(tracker.BrowserState{
		URL:      "https://example.com",
		Category: "games",
	})

	eng := NewEngine(state)

	rule := &dsl.Rule{
		Trigger: dsl.TriggerWhen,
		Condition: &dsl.CategoryCondition{
			Category: "games",
		},
		Actions: []dsl.Action{
			&dsl.HyprctlAction{Command: "dispatch workspace 2"},
		},
	}

	eng.AddRule(rule)

	events := eng.Evaluate()

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Rule != rule {
		t.Error("expected event rule to match added rule")
	}

	if len(events[0].Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(events[0].Actions))
	}

	action, ok := events[0].Actions[0].(*dsl.HyprctlAction)
	if !ok {
		t.Fatal("expected HyprAction")
	}

	if action.Command != "dispatch workspace 2" {
		t.Errorf("expected command 'dispatch workspace 2', got '%s'", action.Command)
	}
}
