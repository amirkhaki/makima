package dsl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCategoryLoader(t *testing.T) {
	dir := t.TempDir()
	catFile := filepath.Join(dir, "categories.makima")
	content := `category games: *.game.com, *.io
category social: *.facebook.com, *.twitter.com`
	if err := os.WriteFile(catFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewCategoryLoader(catFile)
	categories, err := loader.Load()
	if err != nil {
		t.Fatal(err)
	}

	if len(categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(categories))
	}

	games, ok := categories["games"]
	if !ok {
		t.Fatal("expected 'games' category")
	}
	if len(games.Patterns) != 2 {
		t.Fatalf("expected 2 patterns in games, got %d", len(games.Patterns))
	}
	if games.Patterns[0] != "*.game.com" {
		t.Errorf("expected pattern '*.game.com', got %q", games.Patterns[0])
	}
	if games.Patterns[1] != "*.io" {
		t.Errorf("expected pattern '*.io', got %q", games.Patterns[1])
	}

	social, ok := categories["social"]
	if !ok {
		t.Fatal("expected 'social' category")
	}
	if len(social.Patterns) != 2 {
		t.Fatalf("expected 2 patterns in social, got %d", len(social.Patterns))
	}
}

func TestCategoryMatch(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		url      string
		expected bool
	}{
		{"exact subdomain match", "*.game.com", "https://game.com/play", true},
		{"exact subdomain match 2", "*.game.com", "https://sub.game.com/play", true},
		{"no match different domain", "*.game.com", "https://notgame.com/play", false},
		{"io match", "*.io", "https://agar.io", true},
		{"io match subdomain", "*.io", "https://slither.io", true},
		{"io no match", "*.io", "https://example.com", false},
		{"facebook match", "*.facebook.com", "https://www.facebook.com", true},
		{"exact pattern", "game.com", "https://game.com", true},
		{"exact pattern no match", "game.com", "https://notgame.com", false},
		{"empty pattern", "", "", true},
		{"empty url", "*.game.com", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat := &Category{
				Name:     "test",
				Patterns: []string{tt.pattern},
			}
			result := cat.Matches(tt.url)
			if result != tt.expected {
				t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.url, result, tt.expected)
			}
		})
	}
}
