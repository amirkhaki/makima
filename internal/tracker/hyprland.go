package tracker

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/amirkhaki/makima/internal/log"
	"github.com/thiagokokada/hyprland-go"
	"github.com/thiagokokada/hyprland-go/event"
	"github.com/thiagokokada/hyprland-go/helpers"
)

type Event struct {
	Type string
	Data interface{}
}

type HyprlandTracker struct {
	events     chan Event
	state      *State
	requestCli *hyprland.RequestClient
	eventCli   *event.EventClient
	socket     string
	mu         sync.Mutex
	running    bool
}

func NewHyprlandTracker(state *State) *HyprlandTracker {
	socket, err := helpers.GetSocket(helpers.RequestSocket)
	if err != nil {
		fmt.Printf("Hyprland tracker: failed to get socket: %v\n", err)
		return &HyprlandTracker{
			events: make(chan Event, 100),
			state:  state,
		}
	}
	cli := hyprland.NewClient(socket)
	return &HyprlandTracker{
		events:     make(chan Event, 100),
		state:      state,
		requestCli: cli,
		socket:     socket,
	}
}

func (t *HyprlandTracker) Name() string {
	return "hyprland"
}

func (t *HyprlandTracker) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return nil
	}

	cli, err := event.NewClient(t.socket)
	if err != nil {
		log.Error("hyprland: failed to create event client: %v", err)
		return err
	}
	t.eventCli = cli

	// Subscribe to events in a goroutine
	go func() {
		handler := &hyprlandHandler{tracker: t}
		t.eventCli.Subscribe(ctx, handler,
			event.EventWorkspace,
			event.EventActiveWindow,
		)
	}()

	// Poll current state periodically
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				t.updateState()
			}
		}
	}()

	t.running = true
	return nil
}

func (t *HyprlandTracker) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil
	}

	if t.eventCli != nil {
		t.eventCli.Close()
	}
	t.running = false
	return nil
}

func (t *HyprlandTracker) Events() <-chan Event {
	return t.events
}

func (t *HyprlandTracker) updateState() {
	// Get active workspace
	ws, err := t.requestCli.ActiveWorkspace()
	if err == nil {
		t.state.UpdateHyprland(HyprlandState{
			ActiveWorkspace: ws.Id,
		})
	}

	// Get active window
	window, err := t.requestCli.ActiveWindow()
	if err == nil {
		t.state.UpdateHyprland(HyprlandState{
			WindowClass: window.Class,
			WindowTitle: window.Title,
		})
	}
}

type hyprlandHandler struct {
	tracker *HyprlandTracker
	event.DefaultEventHandler
}

func (h *hyprlandHandler) Workspace(w event.WorkspaceName) {
	id, _ := strconv.Atoi(string(w))
	h.tracker.state.UpdateHyprland(HyprlandState{
		ActiveWorkspace: id,
	})
	h.tracker.events <- Event{
		Type: "workspace",
		Data: w,
	}
}

func (h *hyprlandHandler) ActiveWindow(w event.ActiveWindow) {
	h.tracker.state.UpdateHyprland(HyprlandState{
		WindowClass: w.Name,
		WindowTitle: w.Title,
	})
	h.tracker.events <- Event{
		Type: "window",
		Data: w,
	}
}
