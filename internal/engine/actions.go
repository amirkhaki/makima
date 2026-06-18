package engine

import (
	"fmt"
	"os/exec"

	"github.com/amirkhaki/makima/internal/dsl"
	"github.com/amirkhaki/makima/internal/tracker"
)

type ActionExecutor struct {
	state *tracker.State
}

func NewActionExecutor(state *tracker.State) *ActionExecutor {
	return &ActionExecutor{state: state}
}

func (a *ActionExecutor) Execute(action dsl.Action) error {
	switch act := action.(type) {
	case *dsl.CDPAction:
		return a.executeCDP(act)
	case *dsl.HyprctlAction:
		return a.executeHyprctl(act)
	case *dsl.PopupAction:
		return a.executePopup(act)
	case *dsl.NotifyAction:
		return a.executeNotify(act)
	case *dsl.ExecAction:
		return a.executeExec(act)
	default:
		return fmt.Errorf("unknown action type: %T", action)
	}
}

func (a *ActionExecutor) executeCDP(action *dsl.CDPAction) error {
	return nil
}

func (a *ActionExecutor) executeHyprctl(action *dsl.HyprctlAction) error {
	return nil
}

func (a *ActionExecutor) executePopup(action *dsl.PopupAction) error {
	return nil
}

func (a *ActionExecutor) executeNotify(action *dsl.NotifyAction) error {
	return nil
}

func (a *ActionExecutor) executeExec(action *dsl.ExecAction) error {
	cmd := exec.Command(action.Command, action.Args...)
	return cmd.Run()
}
