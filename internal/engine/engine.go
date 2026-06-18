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
	return &Engine{state: state}
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
