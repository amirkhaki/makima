package tracker

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

type TabInfo struct {
	ID     string
	URL    string
	Title  string
	Domain string
}

type ChromeTracker struct {
	events  chan Event
	state   *State
	browser *rod.Browser
}

func NewChromeTracker(state *State) *ChromeTracker {
	return &ChromeTracker{
		events: make(chan Event, 100),
		state:  state,
	}
}

func (t *ChromeTracker) Name() string {
	return "chrome"
}

func (t *ChromeTracker) Start(ctx context.Context) error {
	t.browser = rod.New().MustConnect()

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				t.browser.MustClose()
				return
			case <-ticker.C:
				tabs, err := t.getTabs()
				if err != nil {
					continue
				}
				for _, tab := range tabs {
					t.updateTab(tab)
					t.events <- Event{Type: "chrome", Data: tab}
				}
			}
		}
	}()

	return nil
}

func (t *ChromeTracker) Stop() error {
	if t.browser != nil {
		t.browser.MustClose()
	}
	return nil
}

func (t *ChromeTracker) Events() <-chan Event {
	return t.events
}

func (t *ChromeTracker) getTabs() ([]TabInfo, error) {
	pages := t.browser.MustPages()
	tabs := make([]TabInfo, 0, len(pages))

	for _, page := range pages {
		info, err := page.Info()
		if err != nil {
			continue
		}

		if info.Type != "page" {
			continue
		}

		tabs = append(tabs, TabInfo{
			ID:     string(info.TargetID),
			URL:    info.URL,
			Title:  info.Title,
			Domain: extractDomain(info.URL),
		})
	}

	return tabs, nil
}

func (t *ChromeTracker) updateTab(tab TabInfo) {
	t.state.UpdateBrowser(BrowserState{
		URL:      tab.URL,
		TabTitle: tab.Title,
		Domain:   tab.Domain,
	})
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
	pages := t.browser.MustPages()
	for _, page := range pages {
		info, err := page.Info()
		if err != nil {
			continue
		}
		if string(info.TargetID) == tabID {
			return page.Close()
		}
	}
	return nil
}

func (t *ChromeTracker) Navigate(url string) error {
	pages := t.browser.MustPages()
	if len(pages) > 0 {
		return pages[0].Navigate(url)
	}
	return nil
}

func (t *ChromeTracker) GetActiveTab() (*TabInfo, error) {
	pages := t.browser.MustPages()
	for _, page := range pages {
		info, err := page.Info()
		if err != nil {
			continue
		}
		if info.Type == "page" {
			tab := &TabInfo{
				ID:     string(info.TargetID),
				URL:    info.URL,
				Title:  info.Title,
				Domain: extractDomain(info.URL),
			}
			return tab, nil
		}
	}
	return nil, nil
}
