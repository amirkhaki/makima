package dsl

import (
	"testing"
	"time"
)

func TestParserSimpleRule(t *testing.T) {
	input := `when browser.url matches "*.game.com" then cdp close-tab`
	parser := NewParser(input)

	rules, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}

	rule := rules[0]

	if rule.Trigger != TriggerWhen {
		t.Errorf("expected TriggerWhen, got %v", rule.Trigger)
	}

	cond, ok := rule.Condition.(*URLCondition)
	if !ok {
		t.Fatalf("expected *URLCondition, got %T", rule.Condition)
	}

	if cond.Pattern != "*.game.com" {
		t.Errorf("expected pattern '*.game.com', got %q", cond.Pattern)
	}

	if len(rule.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(rule.Actions))
	}

	cdpAction, ok := rule.Actions[0].(*CDPAction)
	if !ok {
		t.Fatalf("expected *CDPAction, got %T", rule.Actions[0])
	}

	if cdpAction.Command != "close-tab" {
		t.Errorf("expected command 'close-tab', got %q", cdpAction.Command)
	}

	if rule.Grace != 0 {
		t.Errorf("expected grace 0, got %v", rule.Grace)
	}
}

func TestParserRuleWithGrace(t *testing.T) {
	input := `when entering browser.category is games { grace 30s then cdp close-tab }`
	parser := NewParser(input)

	rules, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}

	rule := rules[0]

	if rule.Trigger != TriggerEntering {
		t.Errorf("expected TriggerEntering, got %v", rule.Trigger)
	}

	cond, ok := rule.Condition.(*CategoryCondition)
	if !ok {
		t.Fatalf("expected *CategoryCondition, got %T", rule.Condition)
	}

	if cond.Category != "games" {
		t.Errorf("expected category 'games', got %q", cond.Category)
	}

	if rule.Grace != 30*time.Second {
		t.Errorf("expected grace 30s, got %v", rule.Grace)
	}

	if len(rule.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(rule.Actions))
	}

	cdpAction, ok := rule.Actions[0].(*CDPAction)
	if !ok {
		t.Fatalf("expected *CDPAction, got %T", rule.Actions[0])
	}

	if cdpAction.Command != "close-tab" {
		t.Errorf("expected command 'close-tab', got %q", cdpAction.Command)
	}
}

func TestParserCategoryDefinition(t *testing.T) {
	input := `category games { match "*.io" match "*steam*" }`
	parser := NewParser(input)

	categories, err := parser.ParseCategories()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cat, ok := categories["games"]
	if !ok {
		t.Fatalf("expected category 'games', got none")
	}

	if len(cat.Patterns) != 2 {
		t.Fatalf("expected 2 patterns, got %d", len(cat.Patterns))
	}

	if cat.Patterns[0] != "*.io" {
		t.Errorf("expected pattern '*.io', got %q", cat.Patterns[0])
	}

	if cat.Patterns[1] != "*steam*" {
		t.Errorf("expected pattern '*steam*', got %q", cat.Patterns[1])
	}
}
