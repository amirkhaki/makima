package tracker

import (
	"context"
	"strconv"
	"sync"

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
		log.Error("hyprland: failed to get socket: %v", err)
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

	// Subscribe to events — this provides all state updates
	go func() {
		handler := &hyprlandHandler{tracker: t}
		t.eventCli.Subscribe(ctx, handler,
			event.EventWorkspace,
			event.EventActiveWindow,
		)
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
	// Get current state to merge with
	current := t.state.GetHyprland()

	// Get active workspace
	ws, err := t.requestCli.ActiveWorkspace()
	if err == nil {
		current.ActiveWorkspace = ws.Id
	}

	// Get active window
	window, err := t.requestCli.ActiveWindow()
	if err == nil {
		current.WindowClass = window.Class
		current.WindowTitle = window.Title
	}

	// Get workspace count
	workspaces, err := t.requestCli.Workspaces()
	if err == nil {
		current.WorkspaceCount = len(workspaces)
	}

	// Update merged state
	t.state.UpdateHyprland(current)
}

type hyprlandHandler struct {
	tracker *HyprlandTracker
	event.DefaultEventHandler
}

func (h *hyprlandHandler) Workspace(w event.WorkspaceName) {
	id, _ := strconv.Atoi(string(w))
	current := h.tracker.state.GetHyprland()
	current.ActiveWorkspace = id
	h.tracker.state.UpdateHyprland(current)
	h.tracker.events <- Event{
		Type: "workspace",
		Data: w,
	}
}

func (h *hyprlandHandler) ActiveWindow(w event.ActiveWindow) {
	current := h.tracker.state.GetHyprland()
	current.WindowClass = w.Name
	current.WindowTitle = w.Title
	h.tracker.state.UpdateHyprland(current)
	h.tracker.events <- Event{
		Type: "window",
		Data: w,
	}
}
