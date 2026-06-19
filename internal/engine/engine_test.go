package engine

import (
	"testing"

	"github.com/amirkhaki/makima/internal/dsl"
	"github.com/amirkhaki/makima/internal/tracker"
)

func TestRuleEvaluation(t *testing.T) {
	state := tracker.NewState()
	state.UpdateBrowser(tracker.BrowserState{
		URL: "https://game.com/play",
	})

	eng := NewEngine(state)

	// Load categories
	eng.SetCategories(map[string]*dsl.Category{
		"games": {
			Name:     "games",
			Patterns: []string{"*.game.com"},
		},
	})

	rule := &dsl.Rule{
		Trigger: dsl.TriggerWhen,
		Condition: &dsl.CategoryCondition{
			Category: "games",
		},
		Actions: []dsl.Action{
			&dsl.HyprctlAction{Command: "dispatch workspace 2"},
		},
		Enabled: true,
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
