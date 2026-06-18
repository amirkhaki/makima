# Makima Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use compose:subagent (recommended) or compose:execute to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a personal assistant daemon with custom DSL rule engine, Chrome CDP and Hyprland tracking, hierarchical todos, and a DMS plugin UI.

**Architecture:** Go daemon manages trackers (Hyprland IPC, Chrome CDP), runs a custom DSL rule engine with time budgets and session logic, and exposes a Unix socket for CLI and DMS plugin communication. The DMS plugin provides full CRUD for rules, categories, and todos.

**Tech Stack:** Go 1.22+, Nix flakes, Chrome DevTools Protocol (WebSocket), Hyprland IPC (Unix socket), QML (DMS plugin)

---

## Phase 1: Project Setup & Core Infrastructure

### Task 1: Initialize Go Module

**Covers:** [S3, S8]

**Files:**
- Create: `go.mod`
- Create: `cmd/makima/main.go`

- [ ] **Step 1: Create go.mod**

```bash
cd /Projects/makima
go mod init github.com/makima/makima
```

- [ ] **Step 2: Create basic main.go**

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: makima <command>")
		fmt.Println("Commands: daemon, status, rule, category, todo")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "daemon":
		fmt.Println("Starting makima daemon...")
	case "status":
		fmt.Println("makima status")
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Verify it compiles**

```bash
go build -o makima ./cmd/makima
./makima
```

Expected: Usage message

- [ ] **Step 4: Commit**

```bash
git add go.mod cmd/makima/main.go
git commit -m "feat: initialize Go module and CLI entry point"
```

---

### Task 2: Set Up Nix Build System

**Covers:** [S8]

**Files:**
- Modify: `flake.nix`

- [ ] **Step 1: Update flake.nix**

```nix
{
  description = "makima - Personal assistant with rule-based automation";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        devShells.default = pkgs.mkShell {
          buildInputs = [
            pkgs.go
            pkgs.golangci-lint
            pkgs.gopls
          ];
        };

        packages.default = pkgs.buildGoModule {
          pname = "makima";
          version = "1.0.0";
          src = ./.;
          vendorHash = null;
          doCheck = true;
          checkPhase = ''
            go test ./...
          '';
        };
      }
    );
}
```

- [ ] **Step 2: Commit**

```bash
git add flake.nix
git commit -m "feat: set up Nix build system with Go tooling"
```

---

### Task 3: Create DSL AST Types

**Covers:** [S2]

**Files:**
- Create: `internal/dsl/ast.go`
- Create: `internal/dsl/ast_test.go`

- [ ] **Step 1: Write the failing test**

```go
package dsl

import (
	"testing"
	"time"
)

func TestASTTypes(t *testing.T) {
	// Test that we can create AST nodes
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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/dsl/ -v -run TestASTTypes
```

Expected: FAIL with "cannot refer to unexported name"

- [ ] **Step 3: Write minimal implementation**

```go
package dsl

import "time"

type Trigger string

const (
	TriggerEntering Trigger = "entering"
	TriggerWhen     Trigger = "when"
)

type Rule struct {
	Trigger   Trigger
	Condition Condition
	Actions   []Action
	Grace     time.Duration
	Cooldown  time.Duration
	Budget    *BudgetConfig
}

type Condition interface {
	conditionNode()
}

type CategoryCondition struct {
	Category string
}

func (CategoryCondition) conditionNode() {}

type URLCondition struct {
	Pattern string
}

func (URLCondition) conditionNode() {}

type AppCondition struct {
	Name    string
	Running bool
	Duration time.Duration
}

func (AppCondition) conditionNode() {}

type Action interface {
	actionNode()
}

type CDPAction struct {
	Command string
	Target  string
}

func (CDPAction) actionNode() {}

type HyprctlAction struct {
	Command string
}

func (HyprctlAction) actionNode() {}

type PopupAction struct {
	Message  string
	Duration time.Duration
}

func (PopupAction) actionNode() {}

type NotifyAction struct {
	Message string
}

func (NotifyAction) actionNode() {}

type ExecAction struct {
	Command string
}

func (ExecAction) actionNode() {}

type BudgetConfig struct {
	Message  string
	Options  []time.Duration
	Default  time.Duration
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/dsl/ -v -run TestASTTypes
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/dsl/ast.go internal/dsl/ast_test.go
git commit -m "feat: add DSL AST types for rules, conditions, and actions"
```

---

### Task 4: Implement DSL Lexer

**Covers:** [S2]

**Files:**
- Create: `internal/dsl/lexer.go`
- Create: `internal/dsl/lexer_test.go`

- [ ] **Step 1: Write the failing test**

```go
package dsl

import (
	"testing"
)

func TestLexerTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "simple rule",
			input: `when browser.url matches "*.game.com" then cdp close-tab`,
			expected: []Token{
				{Type: TokenWhen, Value: "when"},
				{Type: TokenBrowser, Value: "browser"},
				{Type: TokenDot, Value: "."},
				{Type: TokenIdent, Value: "url"},
				{Type: TokenMatches, Value: "matches"},
				{Type: TokenString, Value: "*.game.com"},
				{Type: TokenThen, Value: "then"},
				{Type: TokenCDP, Value: "cdp"},
				{Type: TokenIdent, Value: "close-tab"},
			},
		},
		{
			name:  "category definition",
			input: `category games { match "*.io" }`,
			expected: []Token{
				{Type: TokenCategory, Value: "category"},
				{Type: TokenIdent, Value: "games"},
				{Type: TokenLBrace, Value: "{"},
				{Type: TokenMatch, Value: "match"},
				{Type: TokenString, Value: "*.io"},
				{Type: TokenRBrace, Value: "}"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens := lexer.Tokenize()
			if len(tokens) != len(tt.expected) {
				t.Errorf("expected %d tokens, got %d", len(tt.expected), len(tokens))
				return
			}
			for i, tok := range tokens {
				if tok.Type != tt.expected[i].Type || tok.Value != tt.expected[i].Value {
					t.Errorf("token %d: expected %v, got %v", i, tt.expected[i], tok)
				}
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/dsl/ -v -run TestLexerTokenize
```

Expected: FAIL with "undefined: NewLexer"

- [ ] **Step 3: Write minimal implementation**

```go
package dsl

type TokenType int

const (
	TokenWhen TokenType = iota
	TokenEntering
	TokenThen
	TokenAnd
	TokenOr
	TokenNot
	TokenMatches
	TokenIs
	TokenBrowser
	TokenApp
	TokenWorkspace
	TokenWindow
	TokenTime
	TokenCDP
	TokenHyprctl
	TokenPopup
	TokenNotify
	TokenExec
	TokenCategory
	TokenMatch
	TokenDot
	TokenIdent
	TokenString
	TokenNumber
	TokenDuration
	TokenLBrace
	TokenRBrace
	TokenLParen
	TokenRParen
	TokenLBracket
	TokenRBracket
	TokenComma
	TokenColon
	TokenEOF
)

type Token struct {
	Type  TokenType
	Value string
}

type Lexer struct {
	input  string
	pos    int
	tokens []Token
}

func NewLexer(input string) *Lexer {
	return &Lexer{input: input}
}

func (l *Lexer) Tokenize() []Token {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		switch {
		case ch == ' ' || ch == '\t' || ch == '\n':
			l.pos++
		case ch == '#':
			l.skipComment()
		case ch == '"':
			l.readString()
		case ch == '{':
			l.tokens = append(l.tokens, Token{Type: TokenLBrace, Value: "{"})
			l.pos++
		case ch == '}':
			l.tokens = append(l.tokens, Token{Type: TokenRBrace, Value: "}"})
			l.pos++
		case ch == '(':
			l.tokens = append(l.tokens, Token{Type: TokenLParen, Value: "("})
			l.pos++
		case ch == ')':
			l.tokens = append(l.tokens, Token{Type: TokenRParen, Value: ")"})
			l.pos++
		case ch == '[':
			l.tokens = append(l.tokens, Token{Type: TokenLBracket, Value: "["})
			l.pos++
		case ch == ']':
			l.tokens = append(l.tokens, Token{Type: TokenRBracket, Value: "]"})
			l.pos++
		case ch == ',':
			l.tokens = append(l.tokens, Token{Type: TokenComma, Value: ","})
			l.pos++
		case ch == ':':
			l.tokens = append(l.tokens, Token{Type: TokenColon, Value: ":"})
			l.pos++
		case ch == '.':
			l.tokens = append(l.tokens, Token{Type: TokenDot, Value: "."})
			l.pos++
		case isDigit(ch):
			l.readNumber()
		case isAlpha(ch):
			l.readIdent()
		default:
			l.pos++
		}
	}
	l.tokens = append(l.tokens, Token{Type: TokenEOF})
	return l.tokens
}

func (l *Lexer) skipComment() {
	for l.pos < len(l.input) && l.input[l.pos] != '\n' {
		l.pos++
	}
}

func (l *Lexer) readString() {
	l.pos++ // skip opening quote
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != '"' {
		if l.input[l.pos] == '\\' {
			l.pos++
		}
		l.pos++
	}
	l.tokens = append(l.tokens, Token{Type: TokenString, Value: l.input[start:l.pos]})
	l.pos++ // skip closing quote
}

func (l *Lexer) readNumber() {
	start := l.pos
	for l.pos < len(l.input) && (isDigit(l.input[l.pos]) || l.input[l.pos] == '.') {
		l.pos++
	}
	// Check for duration suffix
	if l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == 's' || ch == 'm' || ch == 'h' {
			l.tokens = append(l.tokens, Token{Type: TokenDuration, Value: l.input[start:l.pos+1]})
			l.pos++
			return
		}
	}
	l.tokens = append(l.tokens, Token{Type: TokenNumber, Value: l.input[start:l.pos]})
}

func (l *Lexer) readIdent() {
	start := l.pos
	for l.pos < len(l.input) && (isAlphaNumeric(l.input[l.pos]) || l.input[l.pos] == '-' || l.input[l.pos] == '_') {
		l.pos++
	}
	word := l.input[start:l.pos]
	switch word {
	case "when":
		l.tokens = append(l.tokens, Token{Type: TokenWhen, Value: word})
	case "entering":
		l.tokens = append(l.tokens, Token{Type: TokenEntering, Value: word})
	case "then":
		l.tokens = append(l.tokens, Token{Type: TokenThen, Value: word})
	case "and":
		l.tokens = append(l.tokens, Token{Type: TokenAnd, Value: word})
	case "or":
		l.tokens = append(l.tokens, Token{Type: TokenOr, Value: word})
	case "not":
		l.tokens = append(l.tokens, Token{Type: TokenNot, Value: word})
	case "matches":
		l.tokens = append(l.tokens, Token{Type: TokenMatches, Value: word})
	case "is":
		l.tokens = append(l.tokens, Token{Type: TokenIs, Value: word})
	case "browser":
		l.tokens = append(l.tokens, Token{Type: TokenBrowser, Value: word})
	case "app":
		l.tokens = append(l.tokens, Token{Type: TokenApp, Value: word})
	case "workspace":
		l.tokens = append(l.tokens, Token{Type: TokenWorkspace, Value: word})
	case "window":
		l.tokens = append(l.tokens, Token{Type: TokenWindow, Value: word})
	case "time":
		l.tokens = append(l.tokens, Token{Type: TokenTime, Value: word})
	case "cdp":
		l.tokens = append(l.tokens, Token{Type: TokenCDP, Value: word})
	case "hyprctl":
		l.tokens = append(l.tokens, Token{Type: TokenHyprctl, Value: word})
	case "popup":
		l.tokens = append(l.tokens, Token{Type: TokenPopup, Value: word})
	case "notify":
		l.tokens = append(l.tokens, Token{Type: TokenNotify, Value: word})
	case "exec":
		l.tokens = append(l.tokens, Token{Type: TokenExec, Value: word})
	case "category":
		l.tokens = append(l.tokens, Token{Type: TokenCategory, Value: word})
	case "match":
		l.tokens = append(l.tokens, Token{Type: TokenMatch, Value: word})
	default:
		l.tokens = append(l.tokens, Token{Type: TokenIdent, Value: word})
	}
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isAlpha(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isAlphaNumeric(ch byte) bool {
	return isAlpha(ch) || isDigit(ch)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/dsl/ -v -run TestLexerTokenize
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/dsl/lexer.go internal/dsl/lexer_test.go
git commit -m "feat: implement DSL lexer with tokenization"
```

---

### Task 5: Implement DSL Parser

**Covers:** [S2]

**Files:**
- Create: `internal/dsl/parser.go`
- Create: `internal/dsl/parser_test.go`

- [ ] **Step 1: Write the failing test**

```go
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
	if _, ok := rule.Condition.(*URLCondition); !ok {
		t.Errorf("expected URLCondition, got %T", rule.Condition)
	}
	if len(rule.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(rule.Actions))
	}
	if _, ok := rule.Actions[0].(*CDPAction); !ok {
		t.Errorf("expected CDPAction, got %T", rule.Actions[0])
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
	if rule.Grace != 30*time.Second {
		t.Errorf("expected 30s grace, got %v", rule.Grace)
	}
}

func TestParserCategoryDefinition(t *testing.T) {
	input := `category games { match "*.io" match "*steam*" }`
	parser := NewParser(input)
	categories, err := parser.ParseCategories()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}
	cat := categories["games"]
	if len(cat.Patterns) != 2 {
		t.Fatalf("expected 2 patterns, got %d", len(cat.Patterns))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/dsl/ -v -run TestParser
```

Expected: FAIL with "undefined: NewParser"

- [ ] **Step 3: Write minimal implementation**

```go
package dsl

import (
	"fmt"
	"strconv"
	"time"
)

type Parser struct {
	lexer  *Lexer
	tokens []Token
	pos    int
}

func NewParser(input string) *Parser {
	lexer := NewLexer(input)
	return &Parser{
		lexer:  lexer,
		tokens: lexer.Tokenize(),
	}
}

func (p *Parser) Parse() ([]*Rule, error) {
	var rules []*Rule
	for p.current().Type != TokenEOF {
		if p.current().Type == TokenCategory {
			// Skip category definitions for now
			p.parseCategoryDef()
			continue
		}
		rule, err := p.parseRule()
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (p *Parser) ParseCategories() (map[string]*Category, error) {
	categories := make(map[string]*Category)
	for p.current().Type != TokenEOF {
		if p.current().Type == TokenCategory {
			cat, err := p.parseCategoryDef()
			if err != nil {
				return nil, err
			}
			categories[cat.Name] = cat
		} else {
			p.advance() // skip rules
		}
	}
	return categories, nil
}

func (p *Parser) parseRule() (*Rule, error) {
	rule := &Rule{}

	// Parse trigger
	if p.current().Type == TokenWhen {
		p.advance()
		if p.current().Type == TokenEntering {
			rule.Trigger = TriggerEntering
			p.advance()
		} else {
			rule.Trigger = TriggerWhen
		}
	}

	// Parse condition
	condition, err := p.parseCondition()
	if err != nil {
		return nil, err
	}
	rule.Condition = condition

	// Parse optional block
	if p.current().Type == TokenLBrace {
		p.advance()
		for p.current().Type != TokenRBrace && p.current().Type != TokenEOF {
			if p.current().Type == TokenGrace {
				p.advance()
				dur, err := p.parseDuration()
				if err != nil {
					return nil, err
				}
				rule.Grace = dur
			} else if p.current().Type == TokenCooldown {
				p.advance()
				dur, err := p.parseDuration()
				if err != nil {
					return nil, err
				}
				rule.Cooldown = dur
			} else if p.current().Type == TokenThen {
				action, err := p.parseAction()
				if err != nil {
					return nil, err
				}
				rule.Actions = append(rule.Actions, action)
			} else {
				p.advance()
			}
		}
		if p.current().Type == TokenRBrace {
			p.advance()
		}
	} else if p.current().Type == TokenThen {
		action, err := p.parseAction()
		if err != nil {
			return nil, err
		}
		rule.Actions = append(rule.Actions, action)
	}

	return rule, nil
}

func (p *Parser) parseCondition() (Condition, error) {
	if p.current().Type == TokenBrowser {
		p.advance()
		if p.current().Type == TokenDot {
			p.advance()
			field := p.current().Value
			p.advance()
			if p.current().Type == TokenMatches {
				p.advance()
				pattern := p.current().Value
				p.advance()
				return &URLCondition{Pattern: pattern}, nil
			} else if p.current().Type == TokenIs {
				p.advance()
				category := p.current().Value
				p.advance()
				return &CategoryCondition{Category: category}, nil
			}
		}
	} else if p.current().Type == TokenApp {
		p.advance()
		name := p.current().Value
		p.advance()
		if p.current().Type == TokenDot {
			p.advance()
			if p.current().Type == TokenIdent && p.current().Value == "running" {
				p.advance()
				return &AppCondition{Name: name, Running: true}, nil
			}
		}
	}
	return nil, fmt.Errorf("unexpected token: %v", p.current())
}

func (p *Parser) parseAction() (Action, error) {
	if p.current().Type == TokenThen {
		p.advance()
	}

	switch p.current().Type {
	case TokenCDP:
		p.advance()
		command := p.current().Value
		p.advance()
		target := ""
		if p.current().Type != TokenEOF && p.current().Type != TokenRBrace && p.current().Type != TokenThen {
			target = p.current().Value
			p.advance()
		}
		return &CDPAction{Command: command, Target: target}, nil
	case TokenHyprctl:
		p.advance()
		command := p.current().Value
		p.advance()
		return &HyprctlAction{Command: command}, nil
	case TokenPopup:
		p.advance()
		message := p.current().Value
		p.advance()
		duration := time.Duration(0)
		if p.current().Type == TokenFor {
			p.advance()
			dur, err := p.parseDuration()
			if err != nil {
				return nil, err
			}
			duration = dur
		}
		return &PopupAction{Message: message, Duration: duration}, nil
	case TokenNotify:
		p.advance()
		message := p.current().Value
		p.advance()
		return &NotifyAction{Message: message}, nil
	case TokenExec:
		p.advance()
		command := p.current().Value
		p.advance()
		return &ExecAction{Command: command}, nil
	}

	return nil, fmt.Errorf("unexpected token for action: %v", p.current())
}

func (p *Parser) parseDuration() (time.Duration, error) {
	if p.current().Type == TokenDuration {
		val := p.current().Value
		p.advance()
		return parseDurationString(val)
	}
	return 0, fmt.Errorf("expected duration, got %v", p.current())
}

func parseDurationString(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}
	val, err := strconv.Atoi(s[:len(s)-1])
	if err != nil {
		return 0, err
	}
	switch s[len(s)-1] {
	case 's':
		return time.Duration(val) * time.Second, nil
	case 'm':
		return time.Duration(val) * time.Minute, nil
	case 'h':
		return time.Duration(val) * time.Hour, nil
	}
	return 0, fmt.Errorf("unknown duration suffix: %c", s[len(s)-1])
}

func (p *Parser) parseCategoryDef() (*Category, error) {
	p.advance() // skip 'category'
	name := p.current().Value
	p.advance()
	if p.current().Type != TokenLBrace {
		return nil, fmt.Errorf("expected '{', got %v", p.current())
	}
	p.advance()

	cat := &Category{Name: name}
	for p.current().Type != TokenRBrace && p.current().Type != TokenEOF {
		if p.current().Type == TokenMatch {
			p.advance()
			pattern := p.current().Value
			p.advance()
			cat.Patterns = append(cat.Patterns, pattern)
		} else {
			p.advance()
		}
	}
	if p.current().Type == TokenRBrace {
		p.advance()
	}
	return cat, nil
}

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/dsl/ -v -run TestParser
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/dsl/parser.go internal/dsl/parser_test.go
git commit -m "feat: implement DSL parser for rules and categories"
```

---

### Task 6: Implement Category Loader

**Covers:** [S2]

**Files:**
- Create: `internal/dsl/categories.go`
- Create: `internal/dsl/categories_test.go`

- [ ] **Step 1: Write the failing test**

```go
package dsl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCategoryLoader(t *testing.T) {
	dir := t.TempDir()
	content := `category games {
  match "*.game.com"
  match "*.io"
}

category social {
  match "*.twitter.com"
  match "*.reddit.com"
}
`
	err := os.WriteFile(filepath.Join(dir, "categories.makima"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	loader := NewCategoryLoader(dir)
	categories, err := loader.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(categories))
	}

	games := categories["games"]
	if len(games.Patterns) != 2 {
		t.Errorf("expected 2 patterns for games, got %d", len(games.Patterns))
	}
}

func TestCategoryMatch(t *testing.T) {
	cat := &Category{
		Name:     "games",
		Patterns: []string{"*.game.com", "*.io"},
	}

	tests := []struct {
		url     string
		matches bool
	}{
		{"https://game.com/play", true},
		{"https://example.io", true},
		{"https://twitter.com", false},
	}

	for _, tt := range tests {
		if got := cat.Matches(tt.url); got != tt.matches {
			t.Errorf("Matches(%q) = %v, want %v", tt.url, got, tt.matches)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/dsl/ -v -run TestCategory
```

Expected: FAIL with "undefined: NewCategoryLoader"

- [ ] **Step 3: Write minimal implementation**

```go
package dsl

import (
	"os"
	"path/filepath"
	"strings"
)

type Category struct {
	Name     string
	Patterns []string
}

func (c *Category) Matches(url string) bool {
	for _, pattern := range c.Patterns {
		if matchGlob(pattern, url) {
			return true
		}
	}
	return false
}

func matchGlob(pattern, s string) bool {
	// Simple glob matching: * matches any sequence of characters
	if pattern == "*" {
		return true
	}
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		return strings.Contains(s, pattern[1:len(pattern)-1])
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(s, pattern[1:])
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(s, pattern[:len(pattern)-1])
	}
	return s == pattern
}

type CategoryLoader struct {
	dir string
}

func NewCategoryLoader(dir string) *CategoryLoader {
	return &CategoryLoader{dir: dir}
}

func (l *CategoryLoader) Load() (map[string]*Category, error) {
	path := filepath.Join(l.dir, "categories.makima")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	parser := NewParser(string(data))
	return parser.ParseCategories()
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/dsl/ -v -run TestCategory
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/dsl/categories.go internal/dsl/categories_test.go
git commit -m "feat: implement category loader with glob matching"
```

---

### Task 7: Create State Management Types

**Covers:** [S1, S7]

**Files:**
- Create: `internal/tracker/state.go`
- Create: `internal/tracker/state_test.go`

- [ ] **Step 1: Write the failing test**

```go
package tracker

import (
	"testing"
	"time"
)

func TestStateUpdate(t *testing.T) {
	state := NewState()

	// Update browser state
	state.UpdateBrowser(BrowserState{
		URL:       "https://game.com/play",
		TabTitle:  "Play Game",
		Domain:    "game.com",
		Category:  "games",
		TimeOnSite: 30 * time.Second,
	})

	if state.Browser.URL != "https://game.com/play" {
		t.Errorf("expected URL to be updated")
	}
	if state.Browser.Category != "games" {
		t.Errorf("expected category to be games")
	}

	// Update Hyprland state
	state.UpdateHyprland(HyprlandState{
		ActiveWorkspace: 3,
		WorkspaceCount:  5,
		WindowClass:     "firefox",
		WindowTitle:     "Game Site",
	})

	if state.Hyprland.ActiveWorkspace != 3 {
		t.Errorf("expected workspace to be 3")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/tracker/ -v -run TestState
```

Expected: FAIL with "undefined: NewState"

- [ ] **Step 3: Write minimal implementation**

```go
package tracker

import (
	"sync"
	"time"
)

type BrowserState struct {
	URL         string
	TabTitle    string
	Domain      string
	Category    string
	TimeOnSite  time.Duration
}

type HyprlandState struct {
	ActiveWorkspace int
	WorkspaceCount  int
	WindowClass     string
	WindowTitle     string
}

type AppStatus struct {
	Running bool
	Uptime  time.Duration
}

type State struct {
	mu        sync.RWMutex
	Browser   BrowserState
	Hyprland  HyprlandState
	Apps      map[string]AppStatus
}

func NewState() *State {
	return &State{
		Apps: make(map[string]AppStatus),
	}
}

func (s *State) UpdateBrowser(b BrowserState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Browser = b
}

func (s *State) UpdateHyprland(h HyprlandState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Hyprland = h
}

func (s *State) UpdateApp(name string, status AppStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Apps[name] = status
}

func (s *State) GetBrowser() BrowserState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Browser
}

func (s *State) GetHyprland() HyprlandState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Hyprland
}

func (s *State) GetApp(name string) (AppStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status, ok := s.Apps[name]
	return status, ok
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/tracker/ -v -run TestState
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tracker/state.go internal/tracker/state_test.go
git commit -m "feat: implement thread-safe state management"
```

---

### Task 8: Implement Unix Socket IPC

**Covers:** [S1, S3]

**Files:**
- Create: `internal/daemon/socket.go`
- Create: `internal/daemon/socket_test.go`

- [ ] **Step 1: Write the failing test**

```go
package daemon

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSocketIPC(t *testing.T) {
	dir := t.TempDir()
	sockPath := filepath.Join(dir, "makima.sock")

	server, err := NewSocketServer(sockPath)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer server.Close()

	go server.Serve()

	// Connect as client
	time.Sleep(10 * time.Millisecond)
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Send request
	req := Request{
		Method: "status",
		Params: nil,
	}
	encoder := json.NewEncoder(conn)
	encoder.Encode(req)

	// Read response
	decoder := json.NewDecoder(conn)
	var resp Response
	if err := decoder.Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != "" {
		t.Errorf("unexpected error: %s", resp.Error)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/daemon/ -v -run TestSocket
```

Expected: FAIL with "undefined: NewSocketServer"

- [ ] **Step 3: Write minimal implementation**

```go
package daemon

import (
	"encoding/json"
	"net"
	"os"
)

type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
	ID     int             `json:"id"`
}

type Response struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

type Handler func(req Request) Response

type SocketServer struct {
	sockPath string
	listener net.Listener
	handler  Handler
}

func NewSocketServer(sockPath string) (*SocketServer, error) {
	os.Remove(sockPath)
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, err
	}
	return &SocketServer{
		sockPath: sockPath,
		listener: listener,
	}, nil
}

func (s *SocketServer) SetHandler(h Handler) {
	s.handler = h
}

func (s *SocketServer) Serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *SocketServer) handleConn(conn net.Conn) {
	defer conn.Close()
	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		var req Request
		if err := decoder.Decode(&req); err != nil {
			return
		}

		var resp Response
		if s.handler != nil {
			resp = s.handler(req)
		}
		resp.ID = req.ID
		encoder.Encode(resp)
	}
}

func (s *SocketServer) Close() {
	s.listener.Close()
	os.Remove(s.sockPath)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/daemon/ -v -run TestSocket
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/daemon/socket.go internal/daemon/socket_test.go
git commit -m "feat: implement Unix socket IPC server"
```

---

## Phase 2: Trackers

### Task 9: Implement Hyprland IPC Tracker

**Covers:** [S1, S7]

**Files:**
- Create: `internal/tracker/hyprland.go`
- Create: `internal/tracker/hyprland_test.go`

- [ ] **Step 1: Write the failing test**

```go
package tracker

import (
	"testing"
)

func TestHyprlandParser(t *testing.T) {
	// Test workspace event parsing
	event := `workspace>>3`
	state, err := ParseHyprlandEvent(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.ActiveWorkspace != 3 {
		t.Errorf("expected workspace 3, got %d", state.ActiveWorkspace)
	}

	// Test window focus event
	event = `focuswindow>>firefox`
	state, err = ParseHyprlandEvent(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.WindowClass != "firefox" {
		t.Errorf("expected window class firefox, got %s", state.WindowClass)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/tracker/ -v -run TestHyprland
```

Expected: FAIL with "undefined: ParseHyprlandEvent"

- [ ] **Step 3: Write minimal implementation**

```go
package tracker

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

type HyprlandTracker struct {
	events chan Event
	state  *State
}

func NewHyprlandTracker(state *State) *HyprlandTracker {
	return &HyprlandTracker{
		events: make(chan Event, 100),
		state:  state,
	}
}

func (t *HyprlandTracker) Name() string {
	return "hyprland"
}

func (t *HyprlandTracker) Start(ctx context.Context) error {
	socketPath := "/tmp/hypr/" + getHyprSocket() + "/.socket.sock"
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to Hyprland socket: %w", err)
	}

	go func() {
		defer conn.Close()
		buf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn.SetReadDeadline(time.Now().Add(time.Second))
				n, err := conn.Read(buf)
				if err != nil {
					continue
				}
				lines := strings.Split(string(buf[:n]), "\n")
				for _, line := range lines {
					if line == "" {
						continue
					}
					state, err := ParseHyprlandEvent(line)
					if err == nil {
						t.state.UpdateHyprland(*state)
						t.events <- Event{Type: "hyprland", Data: state}
					}
				}
			}
		}
	}()

	return nil
}

func (t *HyprlandTracker) Stop() error {
	return nil
}

func (t *HyprlandTracker) Events() <-chan Event {
	return t.events
}

func ParseHyprlandEvent(event string) (*HyprlandState, error) {
	state := &HyprlandState{}

	if strings.HasPrefix(event, "workspace>>") {
		var ws int
		_, err := fmt.Sscanf(event, "workspace>>%d", &ws)
		if err != nil {
			return nil, err
		}
		state.ActiveWorkspace = ws
	} else if strings.HasPrefix(event, "focuswindow>>") {
		class := strings.TrimPrefix(event, "focuswindow>>")
		state.WindowClass = class
	}

	return state, nil
}

func getHyprSocket() string {
	// In real implementation, read from environment
	return "instance0"
}

type Event struct {
	Type string
	Data interface{}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/tracker/ -v -run TestHyprland
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tracker/hyprland.go internal/tracker/hyprland_test.go
git commit -m "feat: implement Hyprland IPC tracker"
```

---

### Task 10: Implement Chrome CDP Tracker

**Covers:** [S1, S7]

**Files:**
- Create: `internal/tracker/chrome.go`
- Create: `internal/tracker/chrome_test.go`

- [ ] **Step 1: Write the failing test**

```go
package tracker

import (
	"testing"
	"time"
)

func TestChromeCDPParser(t *testing.T) {
	// Test tab info parsing
	tabJSON := `{"url":"https://game.com/play","title":"Play Game","id":1}`
	tab, err := ParseTabInfo([]byte(tabJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tab.URL != "https://game.com/play" {
		t.Errorf("expected URL, got %s", tab.URL)
	}
	if tab.Title != "Play Game" {
		t.Errorf("expected title, got %s", tab.Title)
	}
}

func TestChromeTrackerState(t *testing.T) {
	state := NewState()
	tracker := NewChromeTracker(state)

	// Simulate tab update
	tracker.updateTab(TabInfo{
		URL:    "https://example.io/game",
		Title:  "IO Game",
		ID:     1,
		Domain: "example.io",
	})

	browser := state.GetBrowser()
	if browser.URL != "https://example.io/game" {
		t.Errorf("expected URL to be updated")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/tracker/ -v -run TestChrome
```

Expected: FAIL with "undefined: ParseTabInfo"

- [ ] **Step 3: Write minimal implementation**

```go
package tracker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TabInfo struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	ID      int    `json:"id"`
	Domain  string
}

type ChromeTracker struct {
	events   chan Event
	state    *State
	debugURL string
}

func NewChromeTracker(state *State) *ChromeTracker {
	return &ChromeTracker{
		events:   make(chan Event, 100),
		state:    state,
		debugURL: "http://localhost:9222",
	}
}

func (t *ChromeTracker) Name() string {
	return "chrome"
}

func (t *ChromeTracker) Start(ctx context.Context) error {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				tabs, err := t.getTabs()
				if err != nil {
					continue
				}
				for _, tab := range tabs {
					t.updateTab(tab)
				}
			}
		}
	}()

	return nil
}

func (t *ChromeTracker) Stop() error {
	return nil
}

func (t *ChromeTracker) Events() <-chan Event {
	return t.events
}

func (t *ChromeTracker) getTabs() ([]TabInfo, error) {
	resp, err := http.Get(t.debugURL + "/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tabs []TabInfo
	if err := json.NewDecoder(resp.Body).Decode(&tabs); err != nil {
		return nil, err
	}

	return tabs, nil
}

func (t *ChromeTracker) updateTab(tab TabInfo) {
	t.state.UpdateBrowser(BrowserState{
		URL:    tab.URL,
		TabTitle: tab.Title,
		Domain: extractDomain(tab.URL),
	})
	t.events <- Event{Type: "chrome", Data: tab}
}

func ParseTabInfo(data []byte) (*TabInfo, error) {
	var tab TabInfo
	if err := json.Unmarshal(data, &tab); err != nil {
		return nil, err
	}
	tab.Domain = extractDomain(tab.URL)
	return &tab, nil
}

func extractDomain(url string) string {
	// Simple domain extraction
	if len(url) > 8 && url[:8] == "https://" {
		url = url[8:]
	} else if len(url) > 7 && url[:7] == "http://" {
		url = url[7:]
	}
	for i, ch := range url {
		if ch == '/' || ch == ':' {
			return url[:i]
		}
	}
	return url
}

func (t *ChromeTracker) CloseTab(tabID int) error {
	// Send CDP command to close tab
	_, err := http.Get(fmt.Sprintf("%s/json/close/%d", t.debugURL, tabID))
	return err
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/tracker/ -v -run TestChrome
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tracker/chrome.go internal/tracker/chrome_test.go
git commit -m "feat: implement Chrome CDP tracker"
```

---

## Phase 3: Rule Engine

### Task 11: Implement Rule Evaluation

**Covers:** [S2, S7]

**Files:**
- Create: `internal/engine/engine.go`
- Create: `internal/engine/engine_test.go`

- [ ] **Step 1: Write the failing test**

```go
package engine

import (
	"testing"
	"time"

	"github.com/makima/makima/internal/dsl"
	"github.com/makima/makima/internal/tracker"
)

func TestRuleEvaluation(t *testing.T) {
	state := tracker.NewState()
	state.UpdateBrowser(tracker.BrowserState{
		URL:      "https://game.com/play",
		Category: "games",
	})

	rule := &dsl.Rule{
		Trigger: dsl.TriggerWhen,
		Condition: &dsl.CategoryCondition{
			Category: "games",
		},
		Actions: []dsl.Action{
			&dsl.CDPAction{Command: "close-tab"},
		},
	}

	engine := NewEngine(state)
	engine.AddRule(rule)

	// Evaluate should trigger the rule
	events := engine.Evaluate()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Rule != rule {
		t.Errorf("expected rule to match")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/engine/ -v -run TestRuleEvaluation
```

Expected: FAIL with "undefined: NewEngine"

- [ ] **Step 3: Write minimal implementation**

```go
package engine

import (
	"github.com/makima/makima/internal/dsl"
	"github.com/makima/makima/internal/tracker"
)

type RuleEvent struct {
	Rule    *dsl.Rule
	Actions []dsl.Action
}

type Engine struct {
	state *tracker.State
	rules []*dsl.Rule
}

func NewEngine(state *tracker.State) *Engine {
	return &Engine{
		state: state,
	}
}

func (e *Engine) AddRule(rule *dsl.Rule) {
	e.rules = append(e.rules, rule)
}

func (e *Engine) Evaluate() []RuleEvent {
	var events []RuleEvent

	for _, rule := range e.rules {
		if e.evaluateCondition(rule.Condition) {
			events = append(events, RuleEvent{
				Rule:    rule,
				Actions: rule.Actions,
			})
		}
	}

	return events
}

func (e *Engine) evaluateCondition(condition dsl.Condition) bool {
	switch c := condition.(type) {
	case *dsl.CategoryCondition:
		browser := e.state.GetBrowser()
		return browser.Category == c.Category
	case *dsl.URLCondition:
		browser := e.state.GetBrowser()
		return matchGlob(c.Pattern, browser.URL)
	case *dsl.AppCondition:
		status, ok := e.state.GetApp(c.Name)
		if !ok {
			return false
		}
		if c.Running && !status.Running {
			return false
		}
		if c.Duration > 0 && status.Uptime < c.Duration {
			return false
		}
		return true
	default:
		return false
	}
}

func matchGlob(pattern, s string) bool {
	if pattern == "*" {
		return true
	}
	if len(pattern) > 0 && pattern[0] == '*' && pattern[len(pattern)-1] == '*' {
		return len(s) > 0
	}
	if len(pattern) > 0 && pattern[0] == '*' {
		return len(s) >= len(pattern)-1 && s[len(s)-len(pattern)+1:] == pattern[1:]
	}
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		return len(s) >= len(pattern)-1 && s[:len(pattern)-1] == pattern[:len(pattern)-1]
	}
	return s == pattern
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/engine/ -v -run TestRuleEvaluation
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/engine/engine.go internal/engine/engine_test.go
git commit -m "feat: implement rule evaluation engine"
```

---

### Task 12: Implement Session Management

**Covers:** [S2]

**Files:**
- Create: `internal/engine/session.go`
- Create: `internal/engine/session_test.go`

- [ ] **Step 1: Write the failing test**

```go
package engine

import (
	"testing"
	"time"
)

func TestSessionGracePeriod(t *testing.T) {
	session := NewSession("games", 30*time.Second, 5*time.Minute)

	// Initially not in grace
	if session.InGrace() {
		t.Error("should not be in grace initially")
	}

	// Start grace
	session.StartGrace()
	if !session.InGrace() {
		t.Error("should be in grace after starting")
	}

	// Still in grace before timeout
	time.Sleep(10 * time.Millisecond)
	if !session.InGrace() {
		t.Error("should still be in grace")
	}
}

func TestSessionCooldown(t *testing.T) {
	session := NewSession("games", 30*time.Second, 1*time.Second)

	session.StartGrace()
	session.FireAction()

	if !session.InCooldown() {
		t.Error("should be in cooldown after firing action")
	}

	// Wait for cooldown
	time.Sleep(1100 * time.Millisecond)
	if session.InCooldown() {
		t.Error("should not be in cooldown after waiting")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/engine/ -v -run TestSession
```

Expected: FAIL with "undefined: NewSession"

- [ ] **Step 3: Write minimal implementation**

```go
package engine

import (
	"sync"
	"time"
)

type Session struct {
	mu        sync.RWMutex
	key       string
	grace     time.Duration
	cooldown  time.Duration
	started   time.Time
	fired     time.Time
	inGrace   bool
}

func NewSession(key string, grace, cooldown time.Duration) *Session {
	return &Session{
		key:      key,
		grace:    grace,
		cooldown: cooldown,
	}
}

func (s *Session) StartGrace() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.started = time.Now()
	s.inGrace = true
}

func (s *Session) InGrace() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.inGrace {
		return false
	}
	return time.Since(s.started) < s.grace
}

func (s *Session) FireAction() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fired = time.Now()
	s.inGrace = false
}

func (s *Session) InCooldown() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.fired.IsZero() {
		return false
	}
	return time.Since(s.fired) < s.cooldown
}

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

func (m *SessionManager) GetOrCreate(key string, grace, cooldown time.Duration) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[key]; ok {
		return session
	}

	session := NewSession(key, grace, cooldown)
	m.sessions[key] = session
	return session
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/engine/ -v -run TestSession
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/engine/session.go internal/engine/session_test.go
git commit -m "feat: implement session management with grace and cooldown"
```

---

### Task 13: Implement Action Executors

**Covers:** [S2]

**Files:**
- Create: `internal/engine/actions.go`
- Create: `internal/engine/actions_test.go`

- [ ] **Step 1: Write the failing test**

```go
package engine

import (
	"testing"

	"github.com/makima/makima/internal/dsl"
)

func TestActionExecutor(t *testing.T) {
	executor := NewActionExecutor(nil)

	// Test CDP action
	cdpAction := &dsl.CDPAction{
		Command: "close-tab",
		Target:  "",
	}
	err := executor.Execute(cdpAction)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test notify action
	notifyAction := &dsl.NotifyAction{
		Message: "Test notification",
	}
	err = executor.Execute(notifyAction)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/engine/ -v -run TestActionExecutor
```

Expected: FAIL with "undefined: NewActionExecutor"

- [ ] **Step 3: Write minimal implementation**

```go
package engine

import (
	"fmt"
	"os/exec"

	"github.com/makima/makima/internal/dsl"
	"github.com/makima/makima/internal/tracker"
)

type ActionExecutor struct {
	state *tracker.State
}

func NewActionExecutor(state *tracker.State) *ActionExecutor {
	return &ActionExecutor{state: state}
}

func (e *ActionExecutor) Execute(action dsl.Action) error {
	switch a := action.(type) {
	case *dsl.CDPAction:
		return e.executeCDP(a)
	case *dsl.HyprctlAction:
		return e.executeHyprctl(a)
	case *dsl.PopupAction:
		return e.executePopup(a)
	case *dsl.NotifyAction:
		return e.executeNotify(a)
	case *dsl.ExecAction:
		return e.executeExec(a)
	default:
		return fmt.Errorf("unknown action type: %T", action)
	}
}

func (e *ActionExecutor) executeCDP(action *dsl.CDPAction) error {
	// In real implementation, send CDP command
	fmt.Printf("CDP: %s %s\n", action.Command, action.Target)
	return nil
}

func (e *ActionExecutor) executeHyprctl(action *dsl.HyprctlAction) error {
	cmd := exec.Command("hyprctl", "dispatch", action.Command)
	return cmd.Run()
}

func (e *ActionExecutor) executePopup(action *dsl.PopupAction) error {
	// In real implementation, show popup via DMS
	fmt.Printf("Popup: %s (for %v)\n", action.Message, action.Duration)
	return nil
}

func (e *ActionExecutor) executeNotify(action *dsl.NotifyAction) error {
	cmd := exec.Command("notify-send", action.Message)
	return cmd.Run()
}

func (e *ActionExecutor) executeExec(action *dsl.ExecAction) error {
	cmd := exec.Command("sh", "-c", action.Command)
	return cmd.Run()
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/engine/ -v -run TestActionExecutor
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/engine/actions.go internal/engine/actions_test.go
git commit -m "feat: implement action executors for CDP, hyprctl, notifications"
```

---

## Phase 4: CLI

### Task 14: Set Up CLI Framework

**Covers:** [S6]

**Files:**
- Modify: `cmd/makima/main.go`

- [ ] **Step 1: Write the failing test**

```bash
go build -o makima ./cmd/makima
./makima rule list
```

Expected: "Unknown command: rule"

- [ ] **Step 2: Implement CLI subcommands**

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "daemon":
		startDaemon()
	case "status":
		showStatus()
	case "rule":
		handleRule(os.Args[2:])
	case "category":
		handleCategory(os.Args[2:])
	case "todo":
		handleTodo(os.Args[2:])
	case "config":
		showConfig()
	case "log":
		showLog()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: makima <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  daemon              Start the daemon")
	fmt.Println("  status              Show current status")
	fmt.Println("  rule <subcommand>   Manage rules")
	fmt.Println("  category <subcommand> Manage categories")
	fmt.Println("  todo <subcommand>   Manage todos")
	fmt.Println("  config              Show configuration")
	fmt.Println("  log                 Show recent events")
}

func startDaemon() {
	fmt.Println("Starting makima daemon...")
	// TODO: Implement daemon startup
}

func showStatus() {
	fmt.Println("makima status")
	// TODO: Connect to daemon and show status
}

func handleRule(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: makima rule <list|add|remove|enable|disable>")
		return
	}
	switch args[0] {
	case "list":
		fmt.Println("Listing rules...")
	case "add":
		fmt.Println("Adding rule...")
	case "remove":
		fmt.Println("Removing rule...")
	case "enable":
		fmt.Println("Enabling rule...")
	case "disable":
		fmt.Println("Disabling rule...")
	default:
		fmt.Printf("Unknown rule subcommand: %s\n", args[0])
	}
}

func handleCategory(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: makima category <list|add|remove>")
		return
	}
	switch args[0] {
	case "list":
		fmt.Println("Listing categories...")
	case "add":
		fmt.Println("Adding category...")
	case "remove":
		fmt.Println("Removing category...")
	default:
		fmt.Printf("Unknown category subcommand: %s\n", args[0])
	}
}

func handleTodo(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: makima todo <list|add|done|remove|tree>")
		return
	}
	switch args[0] {
	case "list":
		fmt.Println("Listing todos...")
	case "add":
		fmt.Println("Adding todo...")
	case "done":
		fmt.Println("Marking todo done...")
	case "remove":
		fmt.Println("Removing todo...")
	case "tree":
		fmt.Println("Showing todo tree...")
	default:
		fmt.Printf("Unknown todo subcommand: %s\n", args[0])
	}
}

func showConfig() {
	fmt.Println("makima config")
}

func showLog() {
	fmt.Println("makima log")
}
```

- [ ] **Step 3: Verify it works**

```bash
go build -o makima ./cmd/makima
./makima rule list
./makima category list
./makima todo list
```

Expected: "Listing rules...", "Listing categories...", "Listing todos..."

- [ ] **Step 4: Commit**

```bash
git add cmd/makima/main.go
git commit -m "feat: implement CLI framework with subcommands"
```

---

### Task 15: Implement Rule Commands

**Covers:** [S6]

**Files:**
- Create: `internal/cli/rule.go`
- Create: `internal/cli/rule_test.go`

- [ ] **Step 1: Write the failing test**

```go
package cli

import (
	"testing"
)

func TestRuleList(t *testing.T) {
	// Test that rule list connects to daemon
	client, err := NewClient("/tmp/makima-test.sock")
	if err != nil {
		t.Skip("daemon not running")
	}
	defer client.Close()

	rules, err := client.RuleList()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("Found %d rules", len(rules))
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/cli/ -v -run TestRuleList
```

Expected: FAIL with "undefined: NewClient"

- [ ] **Step 3: Write minimal implementation**

```go
package cli

import (
	"encoding/json"
	"net"
)

type Client struct {
	conn net.Conn
}

func NewClient(sockPath string) (*Client, error) {
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) send(method string, params interface{}) (json.RawMessage, error) {
	req := map[string]interface{}{
		"method": method,
		"params": params,
	}
	if err := json.NewEncoder(c.conn).Encode(req); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(c.conn).Decode(&resp); err != nil {
		return nil, err
	}

	if err, ok := resp["error"].(string); ok && err != "" {
		return nil, &RuleError{Message: err}
	}

	result, _ := json.Marshal(resp["result"])
	return result, nil
}

type RuleError struct {
	Message string
}

func (e *RuleError) Error() string {
	return e.Message
}

func (c *Client) RuleList() ([]map[string]interface{}, error) {
	result, err := c.send("rule.list", nil)
	if err != nil {
		return nil, err
	}

	var rules []map[string]interface{}
	if err := json.Unmarshal(result, &rules); err != nil {
		return nil, err
	}

	return rules, nil
}

func (c *Client) RuleAdd(rule map[string]interface{}) error {
	_, err := c.send("rule.add", rule)
	return err
}

func (c *Client) RuleRemove(id string) error {
	_, err := c.send("rule.remove", map[string]string{"id": id})
	return err
}

func (c *Client) RuleEnable(id string) error {
	_, err := c.send("rule.enable", map[string]string{"id": id})
	return err
}

func (c *Client) RuleDisable(id string) error {
	_, err := c.send("rule.disable", map[string]string{"id": id})
	return err
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/cli/ -v -run TestRuleList
```

Expected: PASS (or skip if daemon not running)

- [ ] **Step 5: Commit**

```bash
git add internal/cli/rule.go internal/cli/rule_test.go
git commit -m "feat: implement CLI client for rule commands"
```

---

### Task 16: Implement Category Commands

**Covers:** [S6]

**Files:**
- Create: `internal/cli/category.go`
- Create: `internal/cli/category_test.go`

- [ ] **Step 1: Write the failing test**

```go
package cli

import (
	"testing"
)

func TestCategoryList(t *testing.T) {
	client, err := NewClient("/tmp/makima-test.sock")
	if err != nil {
		t.Skip("daemon not running")
	}
	defer client.Close()

	categories, err := client.CategoryList()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("Found %d categories", len(categories))
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/cli/ -v -run TestCategoryList
```

Expected: FAIL (or pass if client works)

- [ ] **Step 3: Write minimal implementation**

```go
package cli

import (
	"encoding/json"
)

func (c *Client) CategoryList() (map[string][]string, error) {
	result, err := c.send("category.list", nil)
	if err != nil {
		return nil, err
	}

	var categories map[string][]string
	if err := json.Unmarshal(result, &categories); err != nil {
		return nil, err
	}

	return categories, nil
}

func (c *Client) CategoryAdd(name string, patterns []string) error {
	_, err := c.send("category.add", map[string]interface{}{
		"name":     name,
		"patterns": patterns,
	})
	return err
}

func (c *Client) CategoryRemove(name string) error {
	_, err := c.send("category.remove", map[string]string{"name": name})
	return err
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/cli/ -v -run TestCategoryList
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/category.go internal/cli/category_test.go
git commit -m "feat: implement CLI client for category commands"
```

---

### Task 17: Implement Todo Commands

**Covers:** [S6]

**Files:**
- Create: `internal/cli/todo.go`
- Create: `internal/cli/todo_test.go`

- [ ] **Step 1: Write the failing test**

```go
package cli

import (
	"testing"
)

func TestTodoList(t *testing.T) {
	client, err := NewClient("/tmp/makima-test.sock")
	if err != nil {
		t.Skip("daemon not running")
	}
	defer client.Close()

	todos, err := client.TodoList()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("Found %d todos", len(todos))
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/cli/ -v -run TestTodoList
```

Expected: FAIL (or pass if client works)

- [ ] **Step 3: Write minimal implementation**

```go
package cli

import (
	"encoding/json"
)

type TodoItem struct {
	ID        string      `json:"id"`
	Text      string      `json:"text"`
	Completed bool        `json:"completed"`
	Children  []*TodoItem `json:"children"`
}

func (c *Client) TodoList() ([]*TodoItem, error) {
	result, err := c.send("todo.list", nil)
	if err != nil {
		return nil, err
	}

	var todos []*TodoItem
	if err := json.Unmarshal(result, &todos); err != nil {
		return nil, err
	}

	return todos, nil
}

func (c *Client) TodoAdd(text string, parentID string) error {
	params := map[string]string{
		"text": text,
	}
	if parentID != "" {
		params["parent"] = parentID
	}
	_, err := c.send("todo.add", params)
	return err
}

func (c *Client) TodoDone(id string) error {
	_, err := c.send("todo.done", map[string]string{"id": id})
	return err
}

func (c *Client) TodoRemove(id string) error {
	_, err := c.send("todo.remove", map[string]string{"id": id})
	return err
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/cli/ -v -run TestTodoList
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/todo.go internal/cli/todo_test.go
git commit -m "feat: implement CLI client for hierarchical todo commands"
```

---

## Phase 5: DMS Plugin

### Task 18: Create Plugin Manifest

**Covers:** [S5]

**Files:**
- Create: `plugin/plugin.json`

- [ ] **Step 1: Create plugin.json**

```json
{
  "id": "makima",
  "name": "Makima",
  "description": "Personal assistant with rule-based automation",
  "version": "1.0.0",
  "author": "makima",
  "type": "composite",
  "capabilities": ["daemon", "dankbar-widget"],
  "components": {
    "daemon": "./MakimaDaemon.qml",
    "widget": "./MakimaWidget.qml"
  },
  "settings": "./MakimaSettings.qml",
  "requires_dms": ">=0.1.0",
  "permissions": ["settings_read", "settings_write"]
}
```

- [ ] **Step 2: Commit**

```bash
git add plugin/plugin.json
git commit -m "feat: create DMS plugin manifest"
```

---

### Task 19: Implement Daemon Connection

**Covers:** [S5]

**Files:**
- Create: `plugin/MakimaDaemon.qml`

- [ ] **Step 1: Create MakimaDaemon.qml**

```qml
import QtQuick
import Quickshell.Io
import qs.Common
import qs.Modules.Plugins

PluginComponent {
    id: root

    property var popoutService: null
    property bool isConnected: false
    property var status: ({})
    property var rules: []
    property var categories: ({})
    property var todos: []

    DankSocket {
        id: socket
        path: "/tmp/makima.sock"

        onConnectionStateChanged: {
            root.isConnected = connected
            if (connected) {
                requestStatus()
            }
        }

        parser: SplitParser {
            onRead: line => {
                if (!line || line.length === 0) return
                try {
                    const response = JSON.parse(line)
                    handleResponse(response)
                } catch (e) {
                    console.error("Failed to parse response:", e)
                }
            }
        }
    }

    function requestStatus() {
        socket.send({method: "status"})
    }

    function requestRules() {
        socket.send({method: "rule.list"})
    }

    function requestCategories() {
        socket.send({method: "category.list"})
    }

    function requestTodos() {
        socket.send({method: "todo.list"})
    }

    function handleResponse(response) {
        if (response.error) {
            console.error("Daemon error:", response.error)
            return
        }

        switch (response.method) {
        case "status":
            root.status = response.result
            break
        case "rule.list":
            root.rules = response.result
            break
        case "category.list":
            root.categories = response.result
            break
        case "todo.list":
            root.todos = response.result
            break
        }
    }

    function addRule(rule) {
        socket.send({method: "rule.add", params: rule})
    }

    function removeRule(id) {
        socket.send({method: "rule.remove", params: {id: id}})
    }

    function addCategory(name, patterns) {
        socket.send({method: "category.add", params: {name: name, patterns: patterns}})
    }

    function addTodo(text, parentId) {
        socket.send({method: "todo.add", params: {text: text, parent: parentId}})
    }

    function completeTodo(id) {
        socket.send({method: "todo.done", params: {id: id}})
    }

    Component.onCompleted: {
        socket.connected = true
    }
}
```

- [ ] **Step 2: Commit**

```bash
git add plugin/MakimaDaemon.qml
git commit -m "feat: implement DMS plugin daemon connection"
```

---

### Task 20: Implement Widget

**Covers:** [S5]

**Files:**
- Create: `plugin/MakimaWidget.qml`

- [ ] **Step 1: Create MakimaWidget.qml**

```qml
import QtQuick
import qs.Common
import qs.Widgets
import qs.Modules.Plugins

PluginComponent {
    id: root

    property var popoutService: null
    property bool isConnected: false
    property var status: ({})

    horizontalBarPill: Component {
        StyledRect {
            width: content.implicitWidth + Theme.spacingM * 2
            height: parent.widgetThickness
            radius: Theme.cornerRadius
            color: Theme.surfaceContainerHigh

            Row {
                id: content
                anchors.centerIn: parent
                spacing: Theme.spacingS

                DankIcon {
                    icon: root.isConnected ? "check_circle" : "error"
                    color: root.isConnected ? Theme.primary : Theme.error
                    size: 16
                }

                StyledText {
                    text: root.status.browser?.category || "No browser"
                    color: Theme.surfaceText
                    font.pixelSize: Theme.fontSizeSmall
                }
            }
        }
    }

    verticalBarPill: Component {
        StyledRect {
            width: parent.widgetThickness
            height: content.implicitWidth + Theme.spacingM * 2
            radius: Theme.cornerRadius
            color: Theme.surfaceContainerHigh

            Column {
                id: content
                anchors.centerIn: parent
                spacing: Theme.spacingS

                DankIcon {
                    icon: root.isConnected ? "check_circle" : "error"
                    color: root.isConnected ? Theme.primary : Theme.error
                    size: 16
                }

                StyledText {
                    text: root.status.browser?.category || "None"
                    color: Theme.surfaceText
                    font.pixelSize: Theme.fontSizeSmall
                    rotation: 90
                }
            }
        }
    }

    popoutContent: Component {
        PopoutComponent {
            headerText: "Makima"
            detailsText: root.isConnected ? "Connected" : "Disconnected"
            showCloseButton: true

            Column {
                width: parent.width
                spacing: Theme.spacingM

                // Status section
                StyledText {
                    text: "Status"
                    font.pixelSize: Theme.fontSizeLarge
                    font.weight: Font.Bold
                    color: Theme.surfaceText
                }

                StyledText {
                    text: "Browser: " + (root.status.browser?.url || "None")
                    color: Theme.surfaceText
                    wrapMode: Text.WordWrap
                }

                StyledText {
                    text: "Category: " + (root.status.browser?.category || "None")
                    color: Theme.surfaceText
                }

                StyledText {
                    text: "Workspace: " + (root.status.hyprland?.activeWorkspace || "Unknown")
                    color: Theme.surfaceText
                }

                // Rules section
                StyledText {
                    text: "Rules (" + root.rules.length + ")"
                    font.pixelSize: Theme.fontSizeLarge
                    font.weight: Font.Bold
                    color: Theme.surfaceText
                    topPadding: Theme.spacingM
                }

                Repeater {
                    model: root.rules
                    delegate: StyledText {
                        text: modelData.condition + " → " + modelData.action
                        color: Theme.surfaceText
                        wrapMode: Text.WordWrap
                    }
                }
            }
        }
    }

    popoutWidth: 400
    popoutHeight: 500
}
```

- [ ] **Step 2: Commit**

```bash
git add plugin/MakimaWidget.qml
git commit -m "feat: implement DMS plugin widget with status display"
```

---

### Task 21: Implement Settings

**Covers:** [S5]

**Files:**
- Create: `plugin/MakimaSettings.qml`

- [ ] **Step 1: Create MakimaSettings.qml**

```qml
import QtQuick
import qs.Common
import qs.Widgets
import qs.Modules.Plugins

PluginSettings {
    pluginId: "makima"

    StringSetting {
        settingKey: "socketPath"
        label: "Daemon Socket Path"
        description: "Path to the makima daemon Unix socket"
        placeholder: "/tmp/makima.sock"
        defaultValue: "/tmp/makima.sock"
    }

    ToggleSetting {
        settingKey: "browserTracking"
        label: "Browser Tracking"
        description: "Enable Chrome CDP browser tracking"
        defaultValue: true
    }

    ToggleSetting {
        settingKey: "hyprlandTracking"
        label: "Hyprland Tracking"
        description: "Enable Hyprland IPC tracking"
        defaultValue: true
    }
}
```

- [ ] **Step 2: Commit**

```bash
git add plugin/MakimaSettings.qml
git commit -m "feat: implement DMS plugin settings UI"
```

---

## Phase 6: Integration

### Task 22: Wire Up Daemon Event Loop

**Covers:** [S1, S7]

**Files:**
- Create: `internal/daemon/daemon.go`
- Create: `internal/daemon/daemon_test.go`

- [ ] **Step 1: Write the failing test**

```go
package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/makima/makima/internal/engine"
	"github.com/makima/makima/internal/tracker"
)

func TestDaemonEventLoop(t *testing.T) {
	state := tracker.NewState()
	sessionMgr := engine.NewSessionManager()
	actionExecutor := engine.NewActionExecutor(state)
	ruleEngine := engine.NewEngine(state)

	daemon := NewDaemon(state, sessionMgr, actionExecutor, ruleEngine)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go daemon.Run(ctx)

	// Send browser event
	state.UpdateBrowser(tracker.BrowserState{
		URL:      "https://game.com",
		Category: "games",
	})

	time.Sleep(50 * time.Millisecond)
	// Daemon should be running without errors
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/daemon/ -v -run TestDaemonEventLoop
```

Expected: FAIL with "undefined: NewDaemon"

- [ ] **Step 3: Write minimal implementation**

```go
package daemon

import (
	"context"

	"github.com/makima/makima/internal/engine"
	"github.com/makima/makima/internal/tracker"
)

type Daemon struct {
	state          *tracker.State
	sessionMgr     *engine.SessionManager
	actionExecutor *engine.ActionExecutor
	ruleEngine     *engine.Engine
	trackers       []tracker.Tracker
}

func NewDaemon(
	state *tracker.State,
	sessionMgr *engine.SessionManager,
	actionExecutor *engine.ActionExecutor,
	ruleEngine *engine.Engine,
) *Daemon {
	return &Daemon{
		state:          state,
		sessionMgr:     sessionMgr,
		actionExecutor: actionExecutor,
		ruleEngine:     ruleEngine,
	}
}

func (d *Daemon) AddTracker(t tracker.Tracker) {
	d.trackers = append(d.trackers, t)
}

func (d *Daemon) Run(ctx context.Context) error {
	// Start all trackers
	for _, t := range d.trackers {
		if err := t.Start(ctx); err != nil {
			return err
		}
	}

	// Event loop
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-d.eventChan():
			d.handleEvent(event)
		}
	}
}

func (d *Daemon) eventChan() <-chan tracker.Event {
	// Merge events from all trackers
	ch := make(chan tracker.Event, 100)
	go func() {
		for _, t := range d.trackers {
			go func(tr tracker.Tracker) {
				for event := range tr.Events() {
					ch <- event
				}
			}(t)
		}
	}()
	return ch
}

func (d *Daemon) handleEvent(event tracker.Event) {
	// Evaluate rules based on new state
	events := d.ruleEngine.Evaluate()
	for _, e := range events {
		for _, action := range e.Actions {
			d.actionExecutor.Execute(action)
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/daemon/ -v -run TestDaemonEventLoop
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/daemon/daemon.go internal/daemon/daemon_test.go
git commit -m "feat: wire up daemon event loop with trackers and rule engine"
```

---

### Task 23: Implement Todo Store

**Covers:** [S6]

**Files:**
- Create: `internal/todo/store.go`
- Create: `internal/todo/store_test.go`

- [ ] **Step 1: Write the failing test**

```go
package todo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTodoStore(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "todos.json"))

	// Add todo
	todo, err := store.Add("Read Dune", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if todo.Text != "Read Dune" {
		t.Errorf("expected text 'Read Dune', got %s", todo.Text)
	}

	// Add child
	child, err := store.Add("Chapter 1", todo.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if child.ParentID != todo.ID {
		t.Errorf("expected parent ID %s, got %s", todo.ID, child.ParentID)
	}

	// List
	todos, err := store.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 root todo, got %d", len(todos))
	}
	if len(todos[0].Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(todos[0].Children))
	}

	// Complete child
	err = store.Complete(child.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check parent progress
	todos, _ = store.List()
	if todos[0].Progress != "1/1" {
		t.Errorf("expected progress '1/1', got %s", todos[0].Progress)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/todo/ -v -run TestTodoStore
```

Expected: FAIL with "undefined: NewStore"

- [ ] **Step 3: Write minimal implementation**

```go
package todo

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Todo struct {
	ID        string  `json:"id"`
	Text      string  `json:"text"`
	Completed bool    `json:"completed"`
	ParentID  string  `json:"parentId,omitempty"`
	Children  []*Todo `json:"children,omitempty"`
	Progress  string  `json:"progress,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type Store struct {
	mu       sync.RWMutex
	filePath string
	todos    []*Todo
}

func NewStore(filePath string) *Store {
	s := &Store{filePath: filePath}
	s.load()
	return s
}

func (s *Store) load() {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		s.todos = []*Todo{}
		return
	}
	json.Unmarshal(data, &s.todos)
	s.updateProgress()
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.todos, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}

func (s *Store) Add(text string, parentID string) (*Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo := &Todo{
		ID:        generateID(),
		Text:      text,
		ParentID:  parentID,
		CreatedAt: time.Now(),
	}

	if parentID == "" {
		s.todos = append(s.todos, todo)
	} else {
		parent := s.findTodo(parentID)
		if parent == nil {
			return nil, fmt.Errorf("parent not found: %s", parentID)
		}
		parent.Children = append(parent.Children, todo)
	}

	s.updateProgress()
	if err := s.save(); err != nil {
		return nil, err
	}

	return todo, nil
}

func (s *Store) Complete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo := s.findTodo(id)
	if todo == nil {
		return fmt.Errorf("todo not found: %s", id)
	}

	s.completeRecursive(todo)
	s.updateProgress()
	return s.save()
}

func (s *Store) completeRecursive(todo *Todo) {
	todo.Completed = true
	for _, child := range todo.Children {
		s.completeRecursive(child)
	}
}

func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.todos = s.removeTodo(s.todos, id)
	s.updateProgress()
	return s.save()
}

func (s *Store) removeTodo(todos []*Todo, id string) []*Todo {
	var result []*Todo
	for _, t := range todos {
		if t.ID != id {
			t.Children = s.removeTodo(t.Children, id)
			result = append(result, t)
		}
	}
	return result
}

func (s *Store) List() ([]*Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.todos, nil
}

func (s *Store) findTodo(id string) *Todo {
	return s.findTodoInSlice(s.todos, id)
}

func (s *Store) findTodoInSlice(todos []*Todo, id string) *Todo {
	for _, t := range todos {
		if t.ID == id {
			return t
		}
		if found := s.findTodoInSlice(t.Children, id); found != nil {
			return found
		}
	}
	return nil
}

func (s *Store) updateProgress() {
	for _, t := range s.todos {
		s.updateTodoProgress(t)
	}
}

func (s *Store) updateTodoProgress(todo *Todo) {
	if len(todo.Children) == 0 {
		todo.Progress = ""
		return
	}

	completed := 0
	for _, child := range todo.Children {
		s.updateTodoProgress(child)
		if child.Completed {
			completed++
		}
	}

	todo.Progress = fmt.Sprintf("%d/%d", completed, len(todo.Children))
}

func generateID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

func (s *Store) TreeString() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sb strings.Builder
	s.printTree(s.todos, &sb, 0)
	return sb.String()
}

func (s *Store) printTree(todos []*Todo, sb *strings.Builder, depth int) {
	for _, t := range todos {
		indent := strings.Repeat("  ", depth)
		status := " "
		if t.Completed {
			status = "x"
		}
		progress := ""
		if t.Progress != "" {
			progress = " (" + t.Progress + ")"
		}
		sb.WriteString(fmt.Sprintf("%s[%s] %s%s\n", indent, status, t.Text, progress))
		s.printTree(t.Children, sb, depth+1)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/todo/ -v -run TestTodoStore
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/todo/store.go internal/todo/store_test.go
git commit -m "feat: implement hierarchical todo store with persistence"
```

---

### Task 24: Run All Tests

**Covers:** [S4]

**Files:**
- None (verification step)

- [ ] **Step 1: Run all tests**

```bash
go test ./... -v
```

Expected: All tests pass

- [ ] **Step 2: Run tests with race detector**

```bash
go test ./... -race -v
```

Expected: No race conditions detected

- [ ] **Step 3: Run tests with coverage**

```bash
go test ./... -cover
```

Expected: Coverage report shows adequate coverage

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "test: verify all tests pass with race detection"
```

---

## Final Verification

- [ ] **Step 1: Build the project**

```bash
go build -o makima ./cmd/makima
```

Expected: Binary created successfully

- [ ] **Step 2: Run the binary**

```bash
./makima
```

Expected: Usage message displayed

- [ ] **Step 3: Run Nix build**

```bash
nix build
```

Expected: Build succeeds

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "feat: makima personal assistant v1.0.0"
```
