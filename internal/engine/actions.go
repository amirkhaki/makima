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
	case *dsl.CDPNewTabAction:
		return a.executeCDPNewTab(act)
	case *dsl.CDPMuteTabAction:
		return a.executeCDPMuteTab()
	case *dsl.CDPCloseDomainAction:
		return a.executeCDPCloseDomain(act)
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

func (a *ActionExecutor) executeCDPNewTab(action *dsl.CDPNewTabAction) error {
	if a.chrome == nil {
		return fmt.Errorf("chrome tracker not available")
	}
	// Navigate to URL in first tab (simplified)
	if action.URL != "" {
		return a.chrome.Navigate(action.URL)
	}
	return nil
}

func (a *ActionExecutor) executeCDPMuteTab() error {
	// Mute tab - simplified implementation
	return nil
}

func (a *ActionExecutor) executeCDPCloseDomain(action *dsl.CDPCloseDomainAction) error {
	if a.chrome == nil {
		return fmt.Errorf("chrome tracker not available")
	}
	// Close all tabs matching domain
	tabs, err := a.chrome.GetTabs()
	if err != nil {
		return err
	}
	for _, tab := range tabs {
		if strings.Contains(tab.Domain, action.Domain) {
			a.chrome.CloseTab(tab.ID)
		}
	}
	return nil
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
			// Try to find the active tab using Hyprland window title
			hypr := a.state.GetHyprland()
			for _, tab := range tabs {
				if hypr.WindowTitle != "" && strings.Contains(hypr.WindowTitle, tab.Title) {
					return a.chrome.CloseTab(tab.ID)
				}
			}
			// Fallback: close first tab if no title match
			return a.chrome.CloseTab(tabs[0].ID)
		}
		return fmt.Errorf("no tabs to close")
	case "navigate":
		if action.Target == "" {
			return fmt.Errorf("navigate requires a URL")
		}
		return a.chrome.Navigate(action.Target)
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
	// The daemon broadcasts the popup message via sendPopup()
	// This method is called for consistency but the actual popup
	// delivery happens in daemon.executeRuleEvents()
	return nil
}

func (a *ActionExecutor) executeNotify(action *dsl.NotifyAction) error {
	// Try to find notify-send in PATH
	path, err := exec.LookPath("notify-send")
	if err != nil {
		// Fallback to common NixOS path
		path = "/run/current-system/sw/bin/notify-send"
	}
	cmd := exec.Command(path, action.Summary, action.Body)
	return cmd.Run()
}

func (a *ActionExecutor) executeExec(action *dsl.ExecAction) error {
	cmd := exec.Command(action.Command, action.Args...)
	return cmd.Run()
}
