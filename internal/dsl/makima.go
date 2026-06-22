package dsl

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/amirkhaki/makima/internal/log"
)

var ruleCounter int
var ruleCounterMu sync.Mutex

type MakimaFile struct {
	Categories map[string]*Category
	Rules      []*Rule
}

func LoadMakimaFile(path string) (*MakimaFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseMakimaFile(string(data))
}

func ParseMakimaFile(content string) (*MakimaFile, error) {
	file := &MakimaFile{
		Categories: make(map[string]*Category),
		Rules:      []*Rule{},
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "category ") {
			cat, err := parseCategoryLine(line)
			if err != nil {
				return nil, fmt.Errorf("category: %w", err)
			}
			file.Categories[cat.Name] = cat
		} else if strings.HasPrefix(line, "when ") || strings.HasPrefix(line, "entering ") {
			rule, err := parseRuleLine(line)
			if err != nil {
				return nil, fmt.Errorf("rule: %w", err)
			}
			file.Rules = append(file.Rules, rule)
		}
	}

	return file, scanner.Err()
}

func parseCategoryLine(line string) (*Category, error) {
	// category games: *.game.com, *.io
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid category format: %s", line)
	}

	name := strings.TrimSpace(strings.TrimPrefix(parts[0], "category"))
	patternsStr := strings.TrimSpace(parts[1])
	patterns := strings.Split(patternsStr, ",")

	for i := range patterns {
		patterns[i] = strings.TrimSpace(patterns[i])
	}

	return &Category{
		Name:     name,
		Patterns: patterns,
	}, nil
}

func parseRuleLine(line string) (*Rule, error) {
	rule := &Rule{
		Enabled: true,
		ID:      generateRuleID(),
	}

	// Parse trigger
	if strings.HasPrefix(line, "entering ") {
		rule.Trigger = TriggerEntering
		line = strings.TrimPrefix(line, "entering ")
	} else if strings.HasPrefix(line, "when ") {
		rule.Trigger = TriggerWhen
		line = strings.TrimPrefix(line, "when ")
		
		// Check if now starts with "entering"
		if strings.HasPrefix(line, "entering ") {
			rule.Trigger = TriggerEntering
			line = strings.TrimPrefix(line, "entering ")
		}
	} else {
		return nil, fmt.Errorf("expected when or entering, got: %s", line)
	}

	// Find "then" to split condition and actions
	thenIdx := strings.Index(line, " then ")
	if thenIdx == -1 {
		return nil, fmt.Errorf("missing 'then' in rule: %s", line)
	}

	conditionStr := strings.TrimSpace(line[:thenIdx])
	actionsStr := strings.TrimSpace(line[thenIdx+6:])

	// Parse condition
	condition, err := parseCondition(conditionStr)
	if err != nil {
		return nil, err
	}
	rule.Condition = condition

	// Parse actions
	actions, err := parseActions(actionsStr)
	if err != nil {
		return nil, err
	}
	rule.Actions = actions

	return rule, nil
}

func parseCondition(str string) (Condition, error) {
	// browser.category is games
	// browser.url matches *.game.com
	// app.mpv running
	// app.mpv running for 30m
	// window.class matches firefox

	str = strings.TrimSpace(str)

	if strings.HasPrefix(str, "browser.") {
		return parseBrowserCondition(str)
	} else if strings.HasPrefix(str, "app.") {
		return parseAppCondition(str)
	} else if strings.HasPrefix(str, "window.") {
		return parseWindowCondition(str)
	}

	return nil, fmt.Errorf("unknown condition: %s", str)
}

func parseBrowserCondition(str string) (Condition, error) {
	// browser.category is games
	// browser.url matches *.game.com
	// browser.domain matches example.com
	// browser.tab.title matches "YouTube"
	// browser.time_on_site > 30m

	if strings.Contains(str, " is ") {
		parts := strings.SplitN(str, " is ", 2)
		field := strings.TrimPrefix(parts[0], "browser.")
		value := strings.TrimSpace(parts[1])

		if field == "category" {
			return &CategoryCondition{Category: value}, nil
		}
	} else if strings.Contains(str, " matches ") {
		parts := strings.SplitN(str, " matches ", 2)
		field := strings.TrimPrefix(parts[0], "browser.")
		pattern := strings.TrimSpace(parts[1])
		pattern = strings.Trim(pattern, "\"")

		if field == "url" {
			return &URLCondition{Pattern: pattern}, nil
		} else if field == "tab.title" || field == "tabtitle" {
			return &TabTitleCondition{Pattern: pattern}, nil
		} else if field == "domain" {
			return &DomainCondition{Pattern: pattern}, nil
		}
	} else if strings.Contains(str, " > ") || strings.Contains(str, " < ") ||
		strings.Contains(str, " >= ") || strings.Contains(str, " <= ") ||
		strings.Contains(str, " == ") {
		// Parse comparison operators
		parts := strings.SplitN(str, " ", 3)
		if len(parts) == 3 {
			field := strings.TrimPrefix(parts[0], "browser.")
			_ = parts[1] // operator
			value := parts[2]

			if field == "time_on_site" {
				dur, err := time.ParseDuration(value)
				if err != nil {
					return nil, err
				}
				return &TimeOnSiteCondition{Duration: dur}, nil
			}
		}
	}

	return nil, fmt.Errorf("unknown browser condition: %s", str)
}

func parseAppCondition(str string) (Condition, error) {
	// app.mpv running
	// app.mpv running for 30m

	parts := strings.Fields(str)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid app condition: %s", str)
	}

	name := strings.TrimPrefix(parts[0], "app.")

	return &AppCondition{Name: name}, nil
}

func parseWindowCondition(str string) (Condition, error) {
	// window.class matches firefox
	// window.class matches "Google Chrome"

	if strings.Contains(str, " matches ") {
		parts := strings.SplitN(str, " matches ", 2)
		field := strings.TrimPrefix(parts[0], "window.")
		pattern := strings.TrimSpace(parts[1])
		pattern = strings.Trim(pattern, "\"")

		if field == "class" {
			return &WindowClassCondition{Pattern: pattern}, nil
		}
	}

	return nil, fmt.Errorf("unknown window condition: %s", str)
}

func parseActions(str string) ([]Action, error) {
	var actions []Action

	// Split by " then " or " and "
	parts := splitActions(str)

	for _, part := range parts {
		action, err := parseAction(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}

	return actions, nil
}

func splitActions(str string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false

	for i := 0; i < len(str); i++ {
		ch := str[i]

		if ch == '"' {
			inQuote = !inQuote
			current.WriteByte(ch)
		} else if !inQuote && ch == ' ' {
			// Check if next word is "then" or "and"
			remaining := str[i+1:]
			if strings.HasPrefix(remaining, "then ") || strings.HasPrefix(remaining, "and ") {
				if current.Len() > 0 {
					parts = append(parts, current.String())
				}
				current.Reset()
				// Skip past "then " or "and "
				if strings.HasPrefix(remaining, "then ") {
					i += 5
				} else {
					i += 4
				}
			} else {
				current.WriteByte(ch)
			}
		} else {
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

func parseAction(str string) (Action, error) {
	// popup "Take a break!" for 30s
	// hyprctl "dispatch workspace 2"
	// notify "Hello"
	// exec "echo hello"
	// cdp close-tab

	str = strings.TrimSpace(str)

	if strings.HasPrefix(str, "popup ") {
		return parsePopupAction(str)
	} else if strings.HasPrefix(str, "hyprctl ") {
		return parseHyprctlAction(str)
	} else if strings.HasPrefix(str, "notify ") {
		return parseNotifyAction(str)
	} else if strings.HasPrefix(str, "exec ") {
		return parseExecAction(str)
	} else if strings.HasPrefix(str, "cdp ") {
		return parseCDPAction(str)
	}

	return nil, fmt.Errorf("unknown action: %s", str)
}

func parsePopupAction(str string) (Action, error) {
	// popup "Take a break!" for 30s budget [5, 15, 30]
	str = strings.TrimPrefix(str, "popup ")

	// Extract message in quotes
	msg, remaining := extractQuoted(str)
	if msg == "" {
		return nil, fmt.Errorf("popup requires a message")
	}

	action := &PopupAction{
		Title:   "Warning",
		Message: msg,
	}

	// Check for budget option
	if strings.Contains(remaining, "budget") {
		budgetIdx := strings.Index(remaining, "budget")
		budgetStr := strings.TrimSpace(remaining[budgetIdx+6:])
		// Parse budget options like [5, 15, 30]
		if strings.HasPrefix(budgetStr, "[") && strings.HasSuffix(budgetStr, "]") {
			budgetStr = budgetStr[1 : len(budgetStr)-1]
			parts := strings.Split(budgetStr, ",")
			var options []int
			for _, p := range parts {
				p = strings.TrimSpace(p)
				var val int
				if _, err := fmt.Sscanf(p, "%d", &val); err == nil {
					options = append(options, val)
				}
			}
			if len(options) > 0 {
				action.Budget = options
			}
		}
	}

	return action, nil
}

func parseHyprctlAction(str string) (Action, error) {
	// hyprctl "dispatch workspace 2"
	str = strings.TrimPrefix(str, "hyprctl ")
	cmd, _ := extractQuoted(str)
	if cmd == "" {
		cmd = strings.TrimSpace(str)
	}

	return &HyprctlAction{Command: cmd}, nil
}

func parseNotifyAction(str string) (Action, error) {
	// notify "Hello" "World"
	str = strings.TrimPrefix(str, "notify ")
	msg, remaining := extractQuoted(str)
	if msg == "" {
		msg = strings.TrimSpace(str)
	}

	body := ""
	if remaining != "" {
		body, _ = extractQuoted(strings.TrimSpace(remaining))
	}

	return &NotifyAction{Summary: msg, Body: body}, nil
}

func parseExecAction(str string) (Action, error) {
	// exec "echo hello"
	str = strings.TrimPrefix(str, "exec ")
	cmd, _ := extractQuoted(str)
	if cmd == "" {
		cmd = strings.TrimSpace(str)
	}

	// Split command into parts
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil, fmt.Errorf("exec requires a command")
	}

	return &ExecAction{Command: parts[0], Args: parts[1:]}, nil
}

func parseCDPAction(str string) (Action, error) {
	// cdp close-tab
	// cdp navigate "https://example.com"
	// cdp new-tab "https://example.com"
	// cdp mute-tab
	// cdp close-domain example.com
	str = strings.TrimPrefix(str, "cdp ")
	parts := strings.Fields(str)
	if len(parts) == 0 {
		return nil, fmt.Errorf("cdp requires a command")
	}

	command := parts[0]
	target := ""
	if len(parts) > 1 {
		target = parts[1]
		// Remove quotes if present
		target = strings.Trim(target, "\"")
	}

	switch command {
	case "new-tab":
		return &CDPNewTabAction{URL: target}, nil
	case "mute-tab":
		return &CDPMuteTabAction{}, nil
	case "close-domain":
		return &CDPCloseDomainAction{Domain: target}, nil
	default:
		return &CDPAction{Command: command, Target: target}, nil
	}
}

func extractQuoted(str string) (string, string) {
	str = strings.TrimSpace(str)
	if !strings.HasPrefix(str, "\"") {
		return "", str
	}

	end := strings.Index(str[1:], "\"")
	if end == -1 {
		return str[1:], ""
	}

	return str[1 : end+1], str[end+2:]
}

func LoadConfigDir(dir string) (*MakimaFile, error) {
	file := &MakimaFile{
		Categories: make(map[string]*Category),
		Rules:      []*Rule{},
	}

	log.Info("config: loading from %s", dir)

	// Load categories.makima
	catPath := filepath.Join(dir, "categories.makima")
	if _, err := os.Stat(catPath); err == nil {
		log.Info("config: loading categories from %s", catPath)
		catFile, err := LoadMakimaFile(catPath)
		if err != nil {
			return nil, fmt.Errorf("categories: %w", err)
		}
		for k, v := range catFile.Categories {
			file.Categories[k] = v
			log.Info("config: category %s: %v", k, v.Patterns)
		}
	} else {
		log.Info("config: no categories.makima found")
	}

	// Load rules.makima
	rulesPath := filepath.Join(dir, "rules.makima")
	if _, err := os.Stat(rulesPath); err == nil {
		log.Info("config: loading rules from %s", rulesPath)
		rulesFile, err := LoadMakimaFile(rulesPath)
		if err != nil {
			return nil, fmt.Errorf("rules: %w", err)
		}
		for _, rule := range rulesFile.Rules {
			log.Info("config: rule loaded: condition=%T enabled=%v", rule.Condition, rule.Enabled)
		}
		file.Rules = append(file.Rules, rulesFile.Rules...)
	} else {
		log.Info("config: no rules.makima found")
	}

	return file, nil
}

func generateRuleID() string {
	ruleCounterMu.Lock()
	defer ruleCounterMu.Unlock()
	ruleCounter++
	return fmt.Sprintf("rule-%d", ruleCounter)
}
