package engine

import (
	"testing"

	"github.com/makima/makima/internal/dsl"
	"github.com/makima/makima/internal/tracker"
)

func TestActionExecutor(t *testing.T) {
	state := tracker.NewState()
	exec := NewActionExecutor(state)

	tests := []struct {
		name    string
		action  dsl.Action
		wantErr bool
	}{
		{
			name:    "CDP close-tab",
			action:  &dsl.CDPAction{Command: "close-tab"},
			wantErr: false,
		},
		{
			name:    "CDP navigate",
			action:  &dsl.CDPAction{Command: "navigate https://example.com"},
			wantErr: false,
		},
		{
			name:    "hyprctl dispatch",
			action:  &dsl.HyprctlAction{Command: "dispatch workspace 2"},
			wantErr: false,
		},
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
		{
			name:    "exec",
			action:  &dsl.ExecAction{Command: "echo", Args: []string{"hello"}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := exec.Execute(tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
