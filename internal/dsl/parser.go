package dsl

import (
	"fmt"
	"strconv"
	"time"
)

type Parser struct {
	tokens []Token
	pos    int
}

func NewParser(input string) *Parser {
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()
	return &Parser{
		tokens: tokens,
		pos:    0,
	}
}

func (p *Parser) current() Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return Token{Type: TokenEOF}
}

func (p *Parser) advance() Token {
	tok := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *Parser) expect(tt TokenType) (Token, error) {
	tok := p.current()
	if tok.Type != tt {
		return tok, fmt.Errorf("expected token %v, got %v (value %q)", tt, tok.Type, tok.Value)
	}
	p.advance()
	return tok, nil
}

func (p *Parser) Parse() ([]*Rule, error) {
	var rules []*Rule

	for p.current().Type != TokenEOF {
		switch p.current().Type {
		case TokenWhen, TokenEntering:
			rule, err := p.parseRule()
			if err != nil {
				return nil, err
			}
			rules = append(rules, rule)
		default:
			p.advance()
		}
	}

	return rules, nil
}

func (p *Parser) ParseCategories() (map[string]*Category, error) {
	categories := make(map[string]*Category)

	for p.current().Type != TokenEOF {
		if p.current().Type == TokenCategory {
			cat, err := p.parseCategory()
			if err != nil {
				return nil, err
			}
			categories[cat.Name] = cat
		} else {
			p.advance()
		}
	}

	return categories, nil
}

func (p *Parser) parseRule() (*Rule, error) {
	rule := &Rule{}

	tok := p.advance()
	if tok.Type == TokenWhen {
		rule.Trigger = TriggerWhen
		if p.current().Type == TokenEntering {
			p.advance()
			rule.Trigger = TriggerEntering
		}
	} else if tok.Type == TokenEntering {
		rule.Trigger = TriggerEntering
	} else {
		return nil, fmt.Errorf("expected when or entering, got %v", tok.Type)
	}

	condition, err := p.parseCondition()
	if err != nil {
		return nil, err
	}
	rule.Condition = condition

	if p.current().Type == TokenLBrace {
		p.advance()

		for p.current().Type != TokenRBrace && p.current().Type != TokenEOF {
			switch p.current().Type {
			case TokenGrace:
				p.advance()
				dur, err := p.parseDuration()
				if err != nil {
					return nil, err
				}
				rule.Grace = dur
			case TokenCooldown:
				p.advance()
				dur, err := p.parseDuration()
				if err != nil {
					return nil, err
				}
				rule.Cooldown = dur
			case TokenBudget:
				budget, err := p.parseBudget()
				if err != nil {
					return nil, err
				}
				rule.Budget = budget
			case TokenThen:
				p.advance()
				actions, err := p.parseActions()
				if err != nil {
					return nil, err
				}
				rule.Actions = append(rule.Actions, actions...)
			default:
				p.advance()
			}
		}

		if _, err := p.expect(TokenRBrace); err != nil {
			return nil, err
		}
	} else if p.current().Type == TokenThen {
		p.advance()
		actions, err := p.parseActions()
		if err != nil {
			return nil, err
		}
		rule.Actions = actions
	}

	return rule, nil
}

func (p *Parser) parseCondition() (Condition, error) {
	if p.current().Type == TokenBrowser {
		p.advance()
		if _, err := p.expect(TokenDot); err != nil {
			return nil, err
		}

		propTok := p.advance()
		prop := propTok.Value

		if prop == "url" {
			if p.current().Type == TokenMatches {
				p.advance()
			} else if p.current().Type == TokenIdent && p.current().Value == "is" {
				p.advance()
			} else if p.current().Type == TokenIdent && p.current().Value == "matches" {
				p.advance()
			} else {
				return nil, fmt.Errorf("expected matches or is after browser.url, got %v", p.current().Type)
			}
			strTok, err := p.expect(TokenString)
			if err != nil {
				return nil, err
			}
			return &URLCondition{Pattern: strTok.Value}, nil
		} else if prop == "category" {
			if p.current().Type == TokenIdent && p.current().Value == "is" {
				p.advance()
			} else if p.current().Type == TokenMatches {
				p.advance()
			} else {
				return nil, fmt.Errorf("expected 'is' after browser.category, got %v %q", p.current().Type, p.current().Value)
			}
			catTok := p.advance()
			return &CategoryCondition{Category: catTok.Value}, nil
		}

		return nil, fmt.Errorf("unknown browser property: %s", prop)
	}

	if p.current().Type == TokenIdent && p.current().Value == "app" {
		p.advance()
		if _, err := p.expect(TokenDot); err != nil {
			return nil, err
		}
		appName := p.advance().Value
		if _, err := p.expect(TokenDot); err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenIdent); err != nil {
			return nil, err
		}
		return &AppCondition{Name: appName}, nil
	}

	return nil, fmt.Errorf("unexpected token %v %q", p.current().Type, p.current().Value)
}

func (p *Parser) parseActions() ([]Action, error) {
	var actions []Action

	for {
		tok := p.current()
		switch tok.Type {
		case TokenCDP:
			p.advance()
			cmd := p.advance().Value
			actions = append(actions, &CDPAction{Command: cmd})
		case TokenHyprctl:
			p.advance()
			cmd := p.advance().Value
			actions = append(actions, &HyprctlAction{Command: cmd})
		case TokenPopup:
			p.advance()
			title := ""
			body := ""
			if p.current().Type == TokenString {
				title = p.advance().Value
			}
			if p.current().Type == TokenString {
				body = p.advance().Value
			}
			actions = append(actions, &PopupAction{Title: title, Message: body})
		case TokenNotify:
			p.advance()
			summary := ""
			body := ""
			if p.current().Type == TokenString {
				summary = p.advance().Value
			}
			if p.current().Type == TokenString {
				body = p.advance().Value
			}
			actions = append(actions, &NotifyAction{Summary: summary, Body: body})
		case TokenExec:
			p.advance()
			cmd := p.advance().Value
			var args []string
			for p.current().Type == TokenString {
				args = append(args, p.advance().Value)
			}
			actions = append(actions, &ExecAction{Command: cmd, Args: args})
		default:
			return actions, nil
		}
	}
}

func (p *Parser) parseDuration() (time.Duration, error) {
	tok := p.current()
	if tok.Type == TokenDuration {
		p.advance()
		return p.parseDurationString(tok.Value)
	}
	return 0, fmt.Errorf("expected duration, got %v %q", tok.Type, tok.Value)
}

func (p *Parser) parseDurationString(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}

	numStr := s[:len(s)-1]
	unit := s[len(s)-1]

	n, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid duration number: %s", numStr)
	}

	switch unit {
	case 's':
		return time.Duration(n) * time.Second, nil
	case 'm':
		return time.Duration(n) * time.Minute, nil
	case 'h':
		return time.Duration(n) * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid duration unit: %c", unit)
	}
}

func (p *Parser) parseBudget() (*BudgetConfig, error) {
	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}

	budget := &BudgetConfig{}

	for p.current().Type != TokenRBrace && p.current().Type != TokenEOF {
		tok := p.advance()
		switch tok.Type {
		case TokenMaxPerHour:
			if _, err := p.expect(TokenNumber); err != nil {
				return nil, err
			}
			n, _ := strconv.Atoi(p.tokens[p.pos-1].Value)
			budget.MaxPerHour = n
		case TokenMaxPerDay:
			if _, err := p.expect(TokenNumber); err != nil {
				return nil, err
			}
			n, _ := strconv.Atoi(p.tokens[p.pos-1].Value)
			budget.MaxPerDay = n
		case TokenMaxPerWeek:
			if _, err := p.expect(TokenNumber); err != nil {
				return nil, err
			}
			n, _ := strconv.Atoi(p.tokens[p.pos-1].Value)
			budget.MaxPerWeek = n
		}
	}

	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}

	return budget, nil
}

func (p *Parser) parseCategory() (*Category, error) {
	if _, err := p.expect(TokenCategory); err != nil {
		return nil, err
	}

	nameTok := p.advance()
	cat := &Category{
		Name: nameTok.Value,
	}

	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}

	for p.current().Type != TokenRBrace && p.current().Type != TokenEOF {
		if p.current().Type == TokenMatch {
			p.advance()
			strTok, err := p.expect(TokenString)
			if err != nil {
				return nil, err
			}
			cat.Patterns = append(cat.Patterns, strTok.Value)
		} else {
			p.advance()
		}
	}

	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}

	return cat, nil
}

func (p *Parser) parseAction() (Action, error) {
	actions, err := p.parseActions()
	if err != nil {
		return nil, err
	}
	if len(actions) > 0 {
		return actions[0], nil
	}
	return nil, fmt.Errorf("expected action, got %v", p.current().Type)
}
