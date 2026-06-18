package dsl

import (
	"testing"
)

func stripEOF(tokens []Token) []Token {
	if len(tokens) > 0 && tokens[len(tokens)-1].Type == TokenEOF {
		return tokens[:len(tokens)-1]
	}
	return tokens
}

func TestTokenizeSimpleRule(t *testing.T) {
	input := `when browser.url matches "*.game.com" then cdp close-tab`
	lexer := NewLexer(input)
	tokens := stripEOF(lexer.Tokenize())

	expected := []Token{
		{Type: TokenWhen, Value: "when"},
		{Type: TokenBrowser, Value: "browser"},
		{Type: TokenDot, Value: "."},
		{Type: TokenIdent, Value: "url"},
		{Type: TokenMatches, Value: "matches"},
		{Type: TokenString, Value: "*.game.com"},
		{Type: TokenThen, Value: "then"},
		{Type: TokenCDP, Value: "cdp"},
		{Type: TokenIdent, Value: "close-tab"},
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		if tokens[i].Type != exp.Type || tokens[i].Value != exp.Value {
			t.Errorf("token %d: expected %v %q, got %v %q", i, exp.Type, exp.Value, tokens[i].Type, tokens[i].Value)
		}
	}
}

func TestTokenizeCategoryDefinition(t *testing.T) {
	input := `category games { match "*.io" }`
	lexer := NewLexer(input)
	tokens := stripEOF(lexer.Tokenize())

	expected := []Token{
		{Type: TokenCategory, Value: "category"},
		{Type: TokenIdent, Value: "games"},
		{Type: TokenLBrace, Value: "{"},
		{Type: TokenMatch, Value: "match"},
		{Type: TokenString, Value: "*.io"},
		{Type: TokenRBrace, Value: "}"},
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		if tokens[i].Type != exp.Type || tokens[i].Value != exp.Value {
			t.Errorf("token %d: expected %v %q, got %v %q", i, exp.Type, exp.Value, tokens[i].Type, tokens[i].Value)
		}
	}
}

func TestTokenizeDuration(t *testing.T) {
	input := `when browser.url matches "*.game.com" then cdp close-tab grace 30s cooldown 5m`
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	var durationTokens []Token
	for _, tok := range tokens {
		if tok.Type == TokenDuration {
			durationTokens = append(durationTokens, tok)
		}
	}

	if len(durationTokens) != 2 {
		t.Fatalf("expected 2 duration tokens, got %d", len(durationTokens))
	}

	if durationTokens[0].Value != "30s" {
		t.Errorf("expected duration '30s', got %q", durationTokens[0].Value)
	}
	if durationTokens[1].Value != "5m" {
		t.Errorf("expected duration '5m', got %q", durationTokens[1].Value)
	}
}

func TestTokenizeComment(t *testing.T) {
	input := `# this is a comment
when browser.url matches "*.game.com" then cdp close-tab`
	lexer := NewLexer(input)
	tokens := stripEOF(lexer.Tokenize())

	expected := []Token{
		{Type: TokenWhen, Value: "when"},
		{Type: TokenBrowser, Value: "browser"},
		{Type: TokenDot, Value: "."},
		{Type: TokenIdent, Value: "url"},
		{Type: TokenMatches, Value: "matches"},
		{Type: TokenString, Value: "*.game.com"},
		{Type: TokenThen, Value: "then"},
		{Type: TokenCDP, Value: "cdp"},
		{Type: TokenIdent, Value: "close-tab"},
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		if tokens[i].Type != exp.Type || tokens[i].Value != exp.Value {
			t.Errorf("token %d: expected %v %q, got %v %q", i, exp.Type, exp.Value, tokens[i].Type, tokens[i].Value)
		}
	}
}

func TestTokenizeEnterAction(t *testing.T) {
	input := `when entering browser.url matches "*.game.com" then cdp close-tab`
	lexer := NewLexer(input)
	tokens := stripEOF(lexer.Tokenize())

	if tokens[1].Type != TokenEntering {
		t.Errorf("expected TokenEntering, got %v", tokens[1].Type)
	}
}
