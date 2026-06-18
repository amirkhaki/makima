package engine

import (
	"fmt"
	"os/exec"

	"github.com/amirkhaki/makima/internal/dsl"
	"github.com/amirkhaki/makima/internal/tracker"
)

type ActionExecutor struct {
	state  *tracker.State
	chrome *tracker.ChromeTracker
}

func NewActionExecutor(state *tracker.State, chrome *tracker.ChromeTracker) *ActionExecutor {
	return &ActionExecutor{
		state:  state,
		chrome: chrome,
	}
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
	if a.chrome == nil {
		return fmt.Errorf("chrome tracker not available")
	}

	switch action.Command {
	case "close-tab":
		browser := a.state.GetBrowser()
		return a.chrome.CloseTab(browser.Domain)
	default:
		return fmt.Errorf("unknown CDP command: %s", action.Command)
	}
}

func (a *ActionExecutor) executeHyprctl(action *dsl.HyprctlAction) error {
	cmd := exec.Command("hyprctl", "dispatch", action.Command)
	return cmd.Run()
}

func (a *ActionExecutor) executePopup(action *dsl.PopupAction) error {
	cmd := exec.Command("notify-send", action.Title, action.Message)
	return cmd.Run()
}

func (a *ActionExecutor) executeNotify(action *dsl.NotifyAction) error {
	cmd := exec.Command("notify-send", action.Summary, action.Body)
	return cmd.Run()
}

func (a *ActionExecutor) executeExec(action *dsl.ExecAction) error {
	cmd := exec.Command(action.Command, action.Args...)
	return cmd.Run()
}
