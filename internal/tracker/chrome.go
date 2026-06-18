package tracker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type TabInfo struct {
	ID     string
	URL    string
	Title  string
	Domain string
}

type ChromeTracker struct {
	events chan Event
	state  *State
	client *http.Client
}

func NewChromeTracker(state *State) *ChromeTracker {
	return &ChromeTracker{
		events: make(chan Event, 100),
		state:  state,
		client: &http.Client{Timeout: 2 * time.Second},
	}
}

func (t *ChromeTracker) Name() string {
	return "chrome"
}

func (t *ChromeTracker) Start(ctx context.Context) error {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				tabs, err := t.getTabs()
				if err != nil {
					continue
				}
				if len(tabs) > 0 {
					t.updateTab(tabs[0])
					t.events <- Event{Type: "chrome", Data: tabs[0]}
				}
			}
		}
	}()
	return nil
}

func (t *ChromeTracker) Stop() error {
	return nil
}

func (t *ChromeTracker) Events() <-chan Event {
	return t.events
}

func (t *ChromeTracker) getTabs() ([]TabInfo, error) {
	resp, err := t.client.Get("http://localhost:9222/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rawTabs []map[string]interface{}
	if err := json.Unmarshal(body, &rawTabs); err != nil {
		return nil, err
	}

	return ParseTabInfo(rawTabs), nil
}

func (t *ChromeTracker) updateTab(tab TabInfo) {
	t.state.UpdateBrowser(BrowserState{
		URL:      tab.URL,
		TabTitle: tab.Title,
		Domain:   tab.Domain,
	})
}

func ParseTabInfo(rawTabs []map[string]interface{}) []TabInfo {
	tabs := make([]TabInfo, 0, len(rawTabs))
	for _, raw := range rawTabs {
		tabType, _ := raw["type"].(string)
		if tabType != "page" {
			continue
		}

		id, _ := raw["id"].(string)
		title, _ := raw["title"].(string)
		u, _ := raw["url"].(string)

		tabs = append(tabs, TabInfo{
			ID:     id,
			Title:  title,
			URL:    u,
			Domain: extractDomain(u),
		})
	}
	return tabs
}

func extractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := parsed.Hostname()
	parts := strings.Split(host, ".")
	if len(parts) > 2 {
		return strings.Join(parts[len(parts)-2:], ".")
	}
	return host
}

func (t *ChromeTracker) CloseTab(tabID string) error {
	resp, err := t.client.Get(fmt.Sprintf("http://localhost:9222/json/close/%s", tabID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
