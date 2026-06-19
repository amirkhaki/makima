package engine

import (
	"strings"

	"github.com/amirkhaki/makima/internal/dsl"
	"github.com/amirkhaki/makima/internal/log"
	"github.com/amirkhaki/makima/internal/tracker"
)

type RuleEvent struct {
	Rule    *dsl.Rule
	Actions []dsl.Action
}

type Engine struct {
	state      *tracker.State
	rules      []*dsl.Rule
	categories map[string]*dsl.Category
}

func NewEngine(state *tracker.State) *Engine {
	return &Engine{
		state:      state,
		categories: make(map[string]*dsl.Category),
	}
}

func (e *Engine) AddRule(rule *dsl.Rule) {
	log.Event("engine", "rule loaded: condition=%T enabled=%v", rule.Condition, rule.Enabled)
	e.rules = append(e.rules, rule)
}

func (e *Engine) GetRules() []*dsl.Rule {
	return e.rules
}

func (e *Engine) SetCategories(categories map[string]*dsl.Category) {
	for name, cat := range categories {
		log.Event("engine", "category loaded: %s -> %v", name, cat.Patterns)
	}
	e.categories = categories
}

func (e *Engine) AddCategory(name string, category *dsl.Category) {
	log.Event("engine", "category loaded: %s -> %v", name, category.Patterns)
	e.categories[name] = category
}

func (e *Engine) GetCategories() map[string]*dsl.Category {
	return e.categories
}

func (e *Engine) Evaluate() []RuleEvent {
	var events []RuleEvent

	// Update category from URL before evaluating
	e.updateCategory()

	browser := e.state.GetBrowser()
	log.Debug("engine", "evaluating: url=%s category=%s", browser.URL, browser.Category)

	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}
		if e.evaluateCondition(rule.Condition) {
			log.Event("engine", "rule matched: %T", rule.Condition)
			events = append(events, RuleEvent{
				Rule:    rule,
				Actions: rule.Actions,
			})
		}
	}

	if len(events) == 0 {
		log.Debug("engine", "no rules matched")
	}

	return events
}

func (e *Engine) updateCategory() {
	browser := e.state.GetBrowser()
	if browser.URL == "" {
		return
	}

	// Find matching category
	for name, cat := range e.categories {
		if cat.Matches(browser.URL) {
			if browser.Category != name {
				log.Event("engine", "category matched: %s for %s", name, browser.URL)
			}
			browser.Category = name
			e.state.UpdateBrowser(browser)
			return
		}
	}

	// No category matched
	if browser.Category != "" {
		log.Debug("engine", "no category match for %s", browser.URL)
	}
	browser.Category = ""
	e.state.UpdateBrowser(browser)
}

func (e *Engine) evaluateCondition(cond dsl.Condition) bool {
	switch c := cond.(type) {
	case *dsl.CategoryCondition:
		browser := e.state.GetBrowser()
		match := strings.EqualFold(c.Category, browser.Category)
		log.Debug("engine", "category check: rule=%s state=%s match=%v", c.Category, browser.Category, match)
		return match
	case *dsl.URLCondition:
		browser := e.state.GetBrowser()
		match := matchGlob(c.Pattern, browser.URL)
		log.Debug("engine", "url check: pattern=%s url=%s match=%v", c.Pattern, browser.URL, match)
		return match
	case *dsl.AppCondition:
		hypr := e.state.GetHyprland()
		match := matchGlob(c.Name, hypr.WindowClass)
		log.Debug("engine", "app check: name=%s window=%s match=%v", c.Name, hypr.WindowClass, match)
		return match
	default:
		return false
	}
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
