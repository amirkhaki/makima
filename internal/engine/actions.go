package engine

import (
	"fmt"
	"os/exec"
	"strings"

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
		tabs, err := a.chrome.GetTabs()
		if err != nil {
			return err
		}
		if len(tabs) > 0 {
			return a.chrome.CloseTab(tabs[0].ID)
		}
		return fmt.Errorf("no tabs to close")
	case "navigate":
		return a.chrome.Navigate("")
	default:
		return fmt.Errorf("unknown CDP command: %s", action.Command)
	}
}

func (a *ActionExecutor) executeHyprctl(action *dsl.HyprctlAction) error {
	parts := strings.Fields(action.Command)
	cmd := exec.Command("hyprctl", parts...)
	return cmd.Run()
}

func (a *ActionExecutor) executePopup(action *dsl.PopupAction) error {
	// Popup is handled by broadcasting to connected DMS plugin
	// The plugin shows a modal popup
	return nil
}

func (a *ActionExecutor) executeNotify(action *dsl.NotifyAction) error {
	// Use notify-send with full path for NixOS
	cmd := exec.Command("/nix/store/l8x85xcfsgi94hxxv868id2j8n5lg74p-libnotify-0.8.8/bin/notify-send", action.Summary, action.Body)
	return cmd.Run()
}

func (a *ActionExecutor) executeExec(action *dsl.ExecAction) error {
	cmd := exec.Command(action.Command, action.Args...)
	return cmd.Run()
}
