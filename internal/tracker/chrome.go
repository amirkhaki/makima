package tracker

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type TabInfo struct {
	ID     string
	URL    string
	Title  string
	Domain string
}

type ChromeTracker struct {
	events   chan Event
	state    *State
	browser  *rod.Browser
	portFile string
}

func NewChromeTracker(state *State) *ChromeTracker {
	return &ChromeTracker{
		events:   make(chan Event, 100),
		state:    state,
		portFile: getDefaultPortFile(),
	}
}

func NewChromeTrackerWithPortFile(state *State, portFile string) *ChromeTracker {
	return &ChromeTracker{
		events:   make(chan Event, 100),
		state:    state,
		portFile: portFile,
	}
}

func getDefaultPortFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home + "/.config/BraveSoftware/Brave-Browser/DevToolsActivePort"
}

func (t *ChromeTracker) Name() string {
	return "chrome"
}

func (t *ChromeTracker) Start(ctx context.Context) error {
	port, err := t.readPort()
	if err != nil {
		fmt.Printf("Chrome tracker: %v\n", err)
		fmt.Println("Chrome tracker: Running in passive mode (no browser control)")
		return nil
	}

	controlURL, err := launcher.ResolveURL(fmt.Sprintf("%d", port))
	if err != nil {
		fmt.Printf("Chrome tracker: failed to resolve control URL: %v\n", err)
		return nil
	}

	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		fmt.Printf("Chrome tracker: failed to connect: %v\n", err)
		return nil
	}
	t.browser = browser

	// Get initial state of all tabs
	t.scanAllTabs()

	// Subscribe to target info changed events (URL changes, title changes)
	go t.listenEvents(ctx)

	return nil
}

func (t *ChromeTracker) scanAllTabs() {
	if t.browser == nil {
		return
	}

	pages := t.browser.MustPages()
	for _, page := range pages {
		info, err := page.Info()
		if err != nil {
			continue
		}
		if info.Type == "page" {
			t.updateTab(TabInfo{
				ID:     string(info.TargetID),
				URL:    info.URL,
				Title:  info.Title,
				Domain: extractDomain(info.URL),
			})
		}
	}
}

func (t *ChromeTracker) listenEvents(ctx context.Context) {
	if t.browser == nil {
		return
	}

	// Use EachEvent to listen for target info changes
	wait := t.browser.EachEvent(func(e *proto.TargetTargetInfoChanged) {
		if e.TargetInfo == nil {
			return
		}
		// Only track page targets (tabs)
		if e.TargetInfo.Type != "page" {
			return
		}

		t.updateTab(TabInfo{
			ID:     string(e.TargetInfo.TargetID),
			URL:    e.TargetInfo.URL,
			Title:  e.TargetInfo.Title,
			Domain: extractDomain(e.TargetInfo.URL),
		})

		t.events <- Event{
			Type: "chrome",
			Data: TabInfo{
				ID:     string(e.TargetInfo.TargetID),
				URL:    e.TargetInfo.URL,
				Title:  e.TargetInfo.Title,
				Domain: extractDomain(e.TargetInfo.URL),
			},
		}
	})

	// Wait for context cancellation
	<-ctx.Done()
	wait()
}

func (t *ChromeTracker) readPort() (int, error) {
	if t.portFile == "" {
		return 0, fmt.Errorf("no port file configured")
	}

	file, err := os.Open(t.portFile)
	if err != nil {
		return 0, fmt.Errorf("failed to open %s: %w", t.portFile, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0, fmt.Errorf("empty port file")
	}

	port := strings.TrimSpace(scanner.Text())
	var portNum int
	if _, err := fmt.Sscanf(port, "%d", &portNum); err != nil {
		return 0, fmt.Errorf("invalid port: %s", port)
	}

	return portNum, nil
}

func (t *ChromeTracker) Stop() error {
	t.browser = nil
	return nil
}

func (t *ChromeTracker) Events() <-chan Event {
	return t.events
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
	if t.browser == nil {
		return fmt.Errorf("browser not connected")
	}

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

func (t *ChromeTracker) Navigate(nurl string) error {
	if t.browser == nil {
		return fmt.Errorf("browser not connected")
	}

	pages := t.browser.MustPages()
	if len(pages) > 0 {
		return pages[0].Navigate(nurl)
	}
	return nil
}

func (t *ChromeTracker) GetActiveTab() (*TabInfo, error) {
	if t.browser == nil {
		return nil, fmt.Errorf("browser not connected")
	}

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

func (t *ChromeTracker) GetTabs() ([]TabInfo, error) {
	if t.browser == nil {
		return nil, fmt.Errorf("browser not connected")
	}

	pages := t.browser.MustPages()
	tabs := make([]TabInfo, 0, len(pages))

	for _, page := range pages {
		info, err := page.Info()
		if err != nil {
			continue
		}
		if info.Type == "page" {
			tabs = append(tabs, TabInfo{
				ID:     string(info.TargetID),
				URL:    info.URL,
				Title:  info.Title,
				Domain: extractDomain(info.URL),
			})
		}
	}

	return tabs, nil
}
