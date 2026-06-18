package daemon

import (
	"context"
	"reflect"

	"github.com/amirkhaki/makima/internal/engine"
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
}

func NewDaemon(state *tracker.State, sessionMgr *engine.SessionManager, actionExecutor *engine.ActionExecutor, ruleEngine *engine.Engine) *Daemon {
	return &Daemon{
		state:          state,
		sessionMgr:     sessionMgr,
		actionExecutor: actionExecutor,
		ruleEngine:     ruleEngine,
	}
}

func (d *Daemon) AddTracker(tracker Tracker) {
	d.trackers = append(d.trackers, tracker)
}

func (d *Daemon) Run(ctx context.Context) error {
	for _, t := range d.trackers {
		if err := t.Start(ctx); err != nil {
			return err
		}
	}

	defer func() {
		for _, t := range d.trackers {
			t.Stop()
		}
	}()

	events := d.eventChan(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-events:
			if !ok {
				return nil
			}
			d.handleEvent(event)
		}
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
	ruleEvents := d.ruleEngine.Evaluate()
	for _, re := range ruleEvents {
		for _, action := range re.Actions {
			d.actionExecutor.Execute(action)
		}
	}
}
