package engine

import (
	"github.com/amirkhaki/makima/internal/dsl"
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
	e.rules = append(e.rules, rule)
}

func (e *Engine) GetRules() []*dsl.Rule {
	return e.rules
}

func (e *Engine) SetCategories(categories map[string]*dsl.Category) {
	e.categories = categories
}

func (e *Engine) AddCategory(name string, category *dsl.Category) {
	e.categories[name] = category
}

func (e *Engine) GetCategories() map[string]*dsl.Category {
	return e.categories
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

func (e *Engine) evaluateCondition(cond dsl.Condition) bool {
	switch c := cond.(type) {
	case *dsl.CategoryCondition:
		browser := e.state.GetBrowser()
		return matchGlob(c.Category, browser.Category)
	case *dsl.URLCondition:
		browser := e.state.GetBrowser()
		return matchGlob(c.Pattern, browser.URL)
	case *dsl.AppCondition:
		hypr := e.state.GetHyprland()
		return matchGlob(c.Name, hypr.WindowClass)
	default:
		return false
	}
}

func matchGlob(pattern, s string) bool {
	return pattern == s
}
