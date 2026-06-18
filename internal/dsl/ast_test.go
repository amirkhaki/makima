package dsl

import (
	"testing"
	"time"
)

func TestASTTypes(t *testing.T) {
	rule := &Rule{
		Trigger: TriggerEntering,
		Condition: &CategoryCondition{
			Category: "games",
		},
		Actions: []Action{
			&CDPAction{
				Command: "close-tab",
			},
		},
		Grace:    30 * time.Second,
		Cooldown: 5 * time.Minute,
	}

	if rule.Trigger != TriggerEntering {
		t.Errorf("expected TriggerEntering, got %v", rule.Trigger)
	}
}