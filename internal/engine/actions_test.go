package engine

import (
	"os/exec"
	"testing"

	"github.com/amirkhaki/makima/internal/dsl"
	"github.com/amirkhaki/makima/internal/tracker"
)

func TestActionExecutor(t *testing.T) {
	state := tracker.NewState()
	actionExec := NewActionExecutor(state, nil)

	tests := []struct {
		name    string
		action  dsl.Action
		wantErr bool
	}{
		{
			name:    "CDP close-tab without chrome",
			action:  &dsl.CDPAction{Command: "close-tab"},
			wantErr: true,
		},
		{
			name:    "hyprctl dispatch",
			action:  &dsl.HyprctlAction{Command: "workspace 2"},
			wantErr: false,
		},
		{
			name:    "exec",
			action:  &dsl.ExecAction{Command: "echo", Args: []string{"hello"}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := actionExec.Execute(tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestActionExecutorNotify(t *testing.T) {
	if _, err := exec.LookPath("notify-send"); err != nil {
		t.Skip("notify-send not available")
	}

	state := tracker.NewState()
	actionExec := NewActionExecutor(state, nil)

	tests := []struct {
		name    string
		action  dsl.Action
		wantErr bool
	}{
		{
			name:    "popup",
			action:  &dsl.PopupAction{Title: "Warning", Message: "Time is up"},
			wantErr: false,
		},
		{
			name:    "notify",
			action:  &dsl.NotifyAction{Summary: "Makima", Body: "Test notification"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := actionExec.Execute(tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
