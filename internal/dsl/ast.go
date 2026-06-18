package dsl

import "time"

type Trigger int

const (
	TriggerWhen      Trigger = iota
	TriggerEntering
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
}

func (*CDPAction) actionNode() {}

type HyprctlAction struct {
	Command string
}

func (*HyprctlAction) actionNode() {}

type PopupAction struct {
	Title   string
	Message string
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

type BudgetConfig struct {
	MaxPerHour  int
	MaxPerDay   int
	MaxPerWeek  int
}