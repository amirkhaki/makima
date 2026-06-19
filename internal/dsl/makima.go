package dsl

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/amirkhaki/makima/internal/log"
)

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
	rule := &Rule{Enabled: true}

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

	str = strings.TrimSpace(str)

	if strings.HasPrefix(str, "browser.") {
		return parseBrowserCondition(str)
	} else if strings.HasPrefix(str, "app.") {
		return parseAppCondition(str)
	}

	return nil, fmt.Errorf("unknown condition: %s", str)
}

func parseBrowserCondition(str string) (Condition, error) {
	// browser.category is games
	// browser.url matches *.game.com
	// browser.domain matches example.com

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

	for _, ch := range str {
		if ch == '"' {
			inQuote = !inQuote
			current.WriteRune(ch)
		} else if !inQuote && ch == ' ' {
			word := current.String()
			if word == "then" || word == "and" {
				if current.Len() > 0 {
					parts = append(parts, current.String())
				}
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		} else {
			current.WriteRune(ch)
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
	// popup "Take a break!" for 30s
	str = strings.TrimPrefix(str, "popup ")

	// Extract message in quotes
	msg, _ := extractQuoted(str)
	if msg == "" {
		return nil, fmt.Errorf("popup requires a message")
	}

	return &PopupAction{
		Title:   "Warning",
		Message: msg,
	}, nil
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
	// notify "Hello"
	str = strings.TrimPrefix(str, "notify ")
	msg, _ := extractQuoted(str)
	if msg == "" {
		msg = strings.TrimSpace(str)
	}

	return &NotifyAction{Summary: msg, Body: ""}, nil
}

func parseExecAction(str string) (Action, error) {
	// exec "echo hello"
	str = strings.TrimPrefix(str, "exec ")
	cmd, _ := extractQuoted(str)
	if cmd == "" {
		cmd = strings.TrimSpace(str)
	}

	return &ExecAction{Command: cmd, Args: []string{}}, nil
}

func parseCDPAction(str string) (Action, error) {
	// cdp close-tab
	str = strings.TrimPrefix(str, "cdp ")
	parts := strings.Fields(str)
	if len(parts) == 0 {
		return nil, fmt.Errorf("cdp requires a command")
	}

	return &CDPAction{Command: parts[0]}, nil
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
