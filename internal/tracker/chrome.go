package tracker

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/amirkhaki/makima/internal/log"
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
	log.Info("chrome: reading port from %s", t.portFile)

	port, err := t.readPort()
	if err != nil {
		log.Error("chrome: %v", err)
		log.Info("chrome: running in passive mode (no browser control)")
		return nil
	}

	log.Info("chrome: found port %d", port)

	controlURL, err := launcher.ResolveURL(fmt.Sprintf("%d", port))
	if err != nil {
		log.Error("chrome: failed to resolve control URL: %v", err)
		return nil
	}

	log.Info("chrome: connecting to %s", controlURL)

	browser := rod.New().ControlURL(controlURL).NoDefaultDevice()
	if err := browser.Connect(); err != nil {
		log.Error("chrome: failed to connect: %v", err)
		return nil
	}
	t.browser = browser

	log.Info("chrome: connected, scanning initial tabs")

	// Get initial state of all tabs
	t.scanAllTabs()

	log.Info("chrome: listening for navigation events")

	// Subscribe to target info changed events (URL changes, title changes)
	go t.listenEvents(ctx)

	return nil
}

func (t *ChromeTracker) scanAllTabs() {
	if t.browser == nil {
		return
	}

	pages, err := t.browser.Pages()
	if err != nil {
		log.Error("chrome: failed to get pages: %v", err)
		return
	}

	log.Info("chrome: found %d pages", len(pages))

	for _, page := range pages {
		info, err := page.Info()
		if err != nil {
			continue
		}
		if info.Type == "page" {
			tab := TabInfo{
				ID:     string(info.TargetID),
				URL:    info.URL,
				Title:  info.Title,
				Domain: extractDomain(info.URL),
			}
			log.Event("chrome", "initial tab: %s (%s)", tab.Domain, tab.URL)
			t.updateTab(tab)
		}
	}
}

func (t *ChromeTracker) listenEvents(ctx context.Context) {
	if t.browser == nil {
		return
	}

	log.Event("chrome", "subscribed to Target.targetInfoChanged events")

	// Use EachEvent to listen for target info changes
	wait := t.browser.EachEvent(func(e *proto.TargetTargetInfoChanged) {
		if e.TargetInfo == nil {
			return
		}
		// Only track page targets (tabs)
		if e.TargetInfo.Type != "page" {
			return
		}

		tab := TabInfo{
			ID:     string(e.TargetInfo.TargetID),
			URL:    e.TargetInfo.URL,
			Title:  e.TargetInfo.Title,
			Domain: extractDomain(e.TargetInfo.URL),
		}

		log.Event("chrome", "navigation detected: %s (%s)", tab.Domain, tab.URL)

		t.updateTab(tab)

		t.events <- Event{
			Type: "chrome",
			Data: tab,
		}
	})

	// Wait for context cancellation
	<-ctx.Done()
	wait()
	log.Info("chrome: event listener stopped")
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
	log.Info("chrome: stopping")
	if t.browser != nil {
		t.browser.Close()
		t.browser = nil
	}
	return nil
}

func (t *ChromeTracker) Events() <-chan Event {
	return t.events
}

func (t *ChromeTracker) updateTab(tab TabInfo) {
	log.Debug("chrome", "state update: url=%s title=%s domain=%s", tab.URL, tab.Title, tab.Domain)
	
	// Get current state to calculate time on site
	current := t.state.GetBrowser()
	
	// If URL changed, reset time on site
	if current.URL != tab.URL {
		t.state.UpdateBrowser(BrowserState{
			URL:       tab.URL,
			TabTitle:  tab.Title,
			Domain:    tab.Domain,
			TimeOnSite: 0,
		})
	} else {
		// Same URL, increment time on site
		t.state.UpdateBrowser(BrowserState{
			URL:        tab.URL,
			TabTitle:   tab.Title,
			Domain:     tab.Domain,
			TimeOnSite: current.TimeOnSite + 1,
		})
	}
}

func extractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := parsed.Hostname()
	
	// Handle multi-part TLDs like .co.uk, .com.au
	// For simplicity, return the full hostname for now
	// A more sophisticated approach would use a public suffix list
	return host
}

func (t *ChromeTracker) CloseTab(tabID string) error {
	if t.browser == nil {
		return fmt.Errorf("browser not connected")
	}

	pages, err := t.browser.Pages()
	if err != nil {
		return err
	}
	for _, page := range pages {
		info, err := page.Info()
		if err != nil {
			continue
		}
		if string(info.TargetID) == tabID {
			log.Event("chrome", "closing tab: %s", tabID)
			return page.Close()
		}
	}
	return nil
}

func (t *ChromeTracker) Navigate(nurl string) error {
	if t.browser == nil {
		return fmt.Errorf("browser not connected")
	}

	pages, err := t.browser.Pages()
	if err != nil {
		return err
	}
	if len(pages) > 0 {
		log.Event("chrome", "navigating to: %s", nurl)
		return pages[0].Navigate(nurl)
	}
	return nil
}

func (t *ChromeTracker) GetActiveTab() (*TabInfo, error) {
	if t.browser == nil {
		return nil, fmt.Errorf("browser not connected")
	}

	pages, err := t.browser.Pages()
	if err != nil {
		return nil, err
	}
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

	pages, err := t.browser.Pages()
	if err != nil {
		return nil, err
	}
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
