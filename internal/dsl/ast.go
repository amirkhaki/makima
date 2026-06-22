package dsl

import "time"

type Trigger int

const (
	TriggerWhen      Trigger = iota
	TriggerEntering
)

type Rule struct {
	ID        string
	Trigger   Trigger
	Condition Condition
	Actions   []Action
	Grace     time.Duration
	Cooldown  time.Duration
	Enabled   bool
}

func NewRule() *Rule {
	return &Rule{Enabled: true}
}

type Condition interface {
	conditionNode()
}

type CategoryCondition struct {
	Category string
}

func (*CategoryCondition) conditionNode() {}

type URLCondition struct {
	Pattern string
}

func (*URLCondition) conditionNode() {}

type AppCondition struct {
	Name string
}

func (*AppCondition) conditionNode() {}

type Action interface {
	actionNode()
}

type CDPAction struct {
	Command string
	Target  string
}

func (*CDPAction) actionNode() {}

type CDPNewTabAction struct {
	URL string
}

func (*CDPNewTabAction) actionNode() {}

type CDPMuteTabAction struct{}

func (*CDPMuteTabAction) actionNode() {}

type CDPCloseDomainAction struct {
	Domain string
}

func (*CDPCloseDomainAction) actionNode() {}

type HyprctlAction struct {
	Command string
}

func (*HyprctlAction) actionNode() {}

type PopupAction struct {
	Title   string
	Message string
	Budget  []int
}

func (*PopupAction) actionNode() {}

type NotifyAction struct {
	Summary string
	Body    string
}

func (*NotifyAction) actionNode() {}

type ExecAction struct {
	Command string
	Args    []string
}

func (*ExecAction) actionNode() {}

type Category struct {
	Name     string
	Patterns []string
}

type TabTitleCondition struct {
	Pattern string
}

func (*TabTitleCondition) conditionNode() {}

type DomainCondition struct {
	Pattern string
}

func (*DomainCondition) conditionNode() {}

type WindowClassCondition struct {
	Pattern string
}

func (*WindowClassCondition) conditionNode() {}

type TimeOnSiteCondition struct {
	Duration time.Duration
	Operator string
}

func (*TimeOnSiteCondition) conditionNode() {}

type WorkspaceCountCondition struct {
	Operator string
	Count    int
}

func (*WorkspaceCountCondition) conditionNode() {}

type TimeBetweenCondition struct {
	Start string
	End   string
}

func (*TimeBetweenCondition) conditionNode() {}