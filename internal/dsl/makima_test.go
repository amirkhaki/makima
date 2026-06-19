package dsl

import (
	"testing"
	"time"
)

func TestParseCategoryLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantPat  []string
	}{
		{
			name:     "simple category",
			input:    "category games: *.game.com, *.io",
			wantName: "games",
			wantPat:  []string{"*.game.com", "*.io"},
		},
		{
			name:     "single pattern",
			input:    "category social: *.twitter.com",
			wantName: "social",
			wantPat:  []string{"*.twitter.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat, err := parseCategoryLine(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cat.Name != tt.wantName {
				t.Errorf("name = %q, want %q", cat.Name, tt.wantName)
			}
			if len(cat.Patterns) != len(tt.wantPat) {
				t.Fatalf("patterns count = %d, want %d", len(cat.Patterns), len(tt.wantPat))
			}
			for i, p := range cat.Patterns {
				if p != tt.wantPat[i] {
					t.Errorf("patterns[%d] = %q, want %q", i, p, tt.wantPat[i])
				}
			}
		})
	}
}

func TestParseRuleLine(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantTrigger Trigger
		wantAction  string
	}{
		{
			name:        "simple when rule",
			input:       "when browser.category is games then popup \"Take a break!\"",
			wantTrigger: TriggerWhen,
			wantAction:  "popup",
		},
		{
			name:        "entering rule",
			input:       "when entering browser.url matches \"*.game.com\" then cdp close-tab",
			wantTrigger: TriggerEntering,
			wantAction:  "cdp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := parseRuleLine(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rule.Trigger != tt.wantTrigger {
				t.Errorf("trigger = %v, want %v", rule.Trigger, tt.wantTrigger)
			}
			if len(rule.Actions) == 0 {
				t.Fatal("expected at least one action")
			}
		})
	}
}

func TestParseMakimaFile(t *testing.T) {
	content := `
# Categories
category games: *.game.com, *.io
category social: *.twitter.com

# Rules
when browser.category is games then popup "Take a break!"
when entering browser.url matches "*.game.com" then cdp close-tab
`

	file, err := ParseMakimaFile(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(file.Categories) != 2 {
		t.Errorf("expected 2 categories, got %d", len(file.Categories))
	}

	if len(file.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(file.Rules))
	}
}

func TestParsePopupAction(t *testing.T) {
	action, err := parsePopupAction("popup \"Take a break!\" for 30s")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	popup, ok := action.(*PopupAction)
	if !ok {
		t.Fatalf("expected PopupAction, got %T", action)
	}

	if popup.Message != "Take a break!" {
		t.Errorf("message = %q, want %q", popup.Message, "Take a break!")
	}
}

func TestParseHyprctlAction(t *testing.T) {
	action, err := parseHyprctlAction("hyprctl \"dispatch workspace 2\"")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hyprctl, ok := action.(*HyprctlAction)
	if !ok {
		t.Fatalf("expected HyprctlAction, got %T", action)
	}

	if hyprctl.Command != "dispatch workspace 2" {
		t.Errorf("command = %q, want %q", hyprctl.Command, "dispatch workspace 2")
	}
}

func TestBrowserCondition(t *testing.T) {
	condition, err := parseBrowserCondition("browser.category is games")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	catCond, ok := condition.(*CategoryCondition)
	if !ok {
		t.Fatalf("expected CategoryCondition, got %T", condition)
	}

	if catCond.Category != "games" {
		t.Errorf("category = %q, want %q", catCond.Category, "games")
	}
}

func TestURLCondition(t *testing.T) {
	condition, err := parseBrowserCondition("browser.url matches \"*.game.com\"")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	urlCond, ok := condition.(*URLCondition)
	if !ok {
		t.Fatalf("expected URLCondition, got %T", condition)
	}

	if urlCond.Pattern != "*.game.com" {
		t.Errorf("pattern = %q, want %q", urlCond.Pattern, "*.game.com")
	}
}

func TestAppCondition(t *testing.T) {
	condition, err := parseAppCondition("app.mpv running")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	appCond, ok := condition.(*AppCondition)
	if !ok {
		t.Fatalf("expected AppCondition, got %T", condition)
	}

	if appCond.Name != "mpv" {
		t.Errorf("name = %q, want %q", appCond.Name, "mpv")
	}
}

func TestDurationParsing(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"30s", 30 * time.Second},
		{"5m", 5 * time.Minute},
		{"1h", 1 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			dur, err := time.ParseDuration(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dur != tt.want {
				t.Errorf("duration = %v, want %v", dur, tt.want)
			}
		})
	}
}
