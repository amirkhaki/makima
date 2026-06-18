package tracker

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type Event struct {
	Type string
	Data interface{}
}

type HyprlandTracker struct {
	events chan Event
	state  *State
}

func NewHyprlandTracker(state *State) *HyprlandTracker {
	return &HyprlandTracker{
		events: make(chan Event, 100),
		state:  state,
	}
}

func (t *HyprlandTracker) Name() string {
	return "hyprland"
}

func (t *HyprlandTracker) Start(ctx context.Context) error {
	socketPath := getHyprSocket()
	if socketPath == "" {
		return fmt.Errorf("HYPRLAND_INSTANCE_SIGNATURE not set")
	}

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to Hyprland socket: %w", err)
	}

	go func() {
		defer conn.Close()
		buf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn.SetReadDeadline(time.Now().Add(time.Second))
				n, err := conn.Read(buf)
				if err != nil {
					continue
				}
				lines := strings.Split(string(buf[:n]), "\n")
				for _, line := range lines {
					if line == "" {
						continue
					}
					state, err := ParseHyprlandEvent(line)
					if err == nil {
						t.state.UpdateHyprland(*state)
						t.events <- Event{Type: "hyprland", Data: state}
					}
				}
			}
		}
	}()

	return nil
}

func (t *HyprlandTracker) Stop() error {
	return nil
}

func (t *HyprlandTracker) Events() <-chan Event {
	return t.events
}

func ParseHyprlandEvent(event string) (*HyprlandState, error) {
	state := &HyprlandState{}

	if strings.HasPrefix(event, "workspace>>") {
		var ws int
		_, err := fmt.Sscanf(event, "workspace>>%d", &ws)
		if err != nil {
			return nil, err
		}
		state.ActiveWorkspace = ws
	} else if strings.HasPrefix(event, "focuswindow>>") {
		class := strings.TrimPrefix(event, "focuswindow>>")
		state.WindowClass = class
	}

	return state, nil
}

func getHyprSocket() string {
	sig := os.Getenv("HYPRLAND_INSTANCE_SIGNATURE")
	if sig == "" {
		return ""
	}
	return "/tmp/hypr/" + sig + "/.socket.sock"
}
