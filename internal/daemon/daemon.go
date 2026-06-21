package daemon

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"time"

	"github.com/amirkhaki/makima/internal/dsl"
	"github.com/amirkhaki/makima/internal/engine"
	"github.com/amirkhaki/makima/internal/log"
	"github.com/amirkhaki/makima/internal/tracker"
)

type Tracker interface {
	Name() string
	Start(ctx context.Context) error
	Stop() error
	Events() <-chan tracker.Event
}

type Daemon struct {
	state          *tracker.State
	sessionMgr     *engine.SessionManager
	actionExecutor *engine.ActionExecutor
	ruleEngine     *engine.Engine
	trackers       []Tracker
	chromeTracker  *tracker.ChromeTracker

	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

func NewDaemon(state *tracker.State, sessionMgr *engine.SessionManager, actionExecutor *engine.ActionExecutor, ruleEngine *engine.Engine, chrome *tracker.ChromeTracker) *Daemon {
	return &Daemon{
		state:          state,
		sessionMgr:     sessionMgr,
		actionExecutor: actionExecutor,
		ruleEngine:     ruleEngine,
		chromeTracker:  chrome,
		clients:        make(map[chan []byte]struct{}),
	}
}

func (d *Daemon) AddTracker(tracker Tracker) {
	d.trackers = append(d.trackers, tracker)
}

func (d *Daemon) RegisterClient(ch chan []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.clients[ch] = struct{}{}
}

func (d *Daemon) UnregisterClient(ch chan []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.clients, ch)
}

func (d *Daemon) Broadcast(msg []byte) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	clientCount := len(d.clients)
	log.Debug("daemon", "broadcasting to %d clients", clientCount)
	for ch := range d.clients {
		select {
		case ch <- msg:
			log.Debug("daemon", "sent to client channel")
		default:
			log.Debug("daemon", "client channel full, skipping")
		}
	}
}

func (d *Daemon) Run(ctx context.Context) error {
	for _, t := range d.trackers {
		if err := t.Start(ctx); err != nil {
			return err
		}
	}

	// Don't defer Stop - let trackers run until context is cancelled
	// The goroutines in trackers will exit when ctx.Done() fires

	events := d.eventChan(ctx)
	budgetTicker := time.NewTicker(5 * time.Second)
	defer budgetTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-events:
			if !ok {
				return nil
			}
			d.handleEvent(event)
		case <-budgetTicker.C:
			d.checkBudget()
		}
	}
}

func (d *Daemon) checkBudget() {
	budget := d.sessionMgr.GetBudget()
	if budget == nil {
		return
	}

	if budget.Expired() {
		log.Event("daemon", "budget expired (%d minutes), closing tab", budget.Minutes)
		if d.chromeTracker != nil {
			tabs, err := d.chromeTracker.GetTabs()
			if err == nil && len(tabs) > 0 {
				d.chromeTracker.CloseTab(tabs[0].ID)
			}
		}
		// Reset budget
		d.sessionMgr.SetBudget(0)
	}
}

func (d *Daemon) eventChan(ctx context.Context) <-chan tracker.Event {
	merged := make(chan tracker.Event)

	var cases []reflect.SelectCase
	for _, t := range d.trackers {
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(t.Events()),
		})
	}
	cases = append(cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(ctx.Done()),
	})

	go func() {
		defer close(merged)
		for {
			chosen, value, ok := reflect.Select(cases)
			if !ok {
				return
			}
			if chosen == len(cases)-1 {
				return
			}
			merged <- value.Interface().(tracker.Event)
		}
	}()

	return merged
}

func (d *Daemon) handleEvent(event tracker.Event) {
	log.Event("daemon", "event received: type=%s", event.Type)

	// When Hyprland detects browser window focus, poll CDP for active tab URL
	if event.Type == "window" {
		d.pollBrowserTab()
	}

	ruleEvents := d.ruleEngine.Evaluate()
	log.Event("daemon", "rule evaluation: %d rules matched", len(ruleEvents))

	for _, re := range ruleEvents {
		for _, action := range re.Actions {
			log.Event("daemon", "executing action: %T", action)
			result := d.actionExecutor.Execute(action)

			// Send popup to connected clients
			if popupAction, ok := action.(*dsl.PopupAction); ok {
				log.Event("daemon", "sending popup: %s", popupAction.Message)
				d.sendPopup(popupAction)
			}

			// Send result to connected clients
			if result != nil {
				log.Error("action failed: %v", result)
				d.sendError(result.Error())
			}
		}
	}
}

func (d *Daemon) pollBrowserTab() {
	hypr := d.state.GetHyprland()
	browser := d.state.GetBrowser()

	// Check if the focused window is a browser
	if hypr.WindowClass != "brave-browser" && hypr.WindowClass != "chrome" && hypr.WindowClass != "chromium" {
		return
	}

	// Query CDP for the active tab
	tabs, err := d.chromeTracker.GetTabs()
	if err != nil || len(tabs) == 0 {
		return
	}

	// Find the tab with the most recent URL (active tab)
	for _, tab := range tabs {
		if browser.URL != tab.URL {
			log.Event("daemon", "browser tab detected: %s (%s)", tab.Domain, tab.URL)
			d.state.UpdateBrowser(tracker.BrowserState{
				URL:      tab.URL,
				TabTitle: tab.Title,
				Domain:   tab.Domain,
			})
			// Trigger rule evaluation after state update
			d.evaluateAndExecute()
			return
		}
	}
}

func (d *Daemon) evaluateAndExecute() {
	ruleEvents := d.ruleEngine.Evaluate()
	for _, re := range ruleEvents {
		for _, action := range re.Actions {
			log.Event("daemon", "executing action: %T", action)
			result := d.actionExecutor.Execute(action)

			if popupAction, ok := action.(*dsl.PopupAction); ok {
				log.Event("daemon", "sending popup: %s", popupAction.Message)
				d.sendPopup(popupAction)
			}

			if result != nil {
				log.Error("action failed: %v", result)
				d.sendError(result.Error())
			}
		}
	}
}

func (d *Daemon) sendPopup(action *dsl.PopupAction) {
	params := map[string]interface{}{
		"title":   action.Title,
		"message": action.Message,
	}

	if len(action.Budget) > 0 {
		params["budget"] = map[string]interface{}{
			"options": action.Budget,
			"grace":   30,
		}
	}

	msg := map[string]interface{}{
		"method": "popup",
		"params": params,
	}
	data, _ := json.Marshal(msg)
	d.Broadcast(data)
}

func (d *Daemon) sendError(errMsg string) {
	msg := map[string]interface{}{
		"method": "error",
		"params": map[string]interface{}{
			"message": errMsg,
		},
	}
	data, _ := json.Marshal(msg)
	d.Broadcast(data)
}
