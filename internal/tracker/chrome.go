package tracker

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

type TabInfo struct {
	ID     string
	URL    string
	Title  string
	Domain string
}

type ChromeTracker struct {
	events         chan Event
	state          *State
	browser        *rod.Browser
	portFile       string
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

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				if t.browser != nil {
					t.browser.MustClose()
				}
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
	if t.browser != nil {
		t.browser.MustClose()
	}
	return nil
}

func (t *ChromeTracker) Events() <-chan Event {
	return t.events
}

func (t *ChromeTracker) getTabs() ([]TabInfo, error) {
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
