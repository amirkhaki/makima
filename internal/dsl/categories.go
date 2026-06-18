package dsl

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func (c *Category) Matches(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	host := u.Hostname()
	for _, pattern := range c.Patterns {
		if matchGlob(pattern, host) {
			return true
		}
	}
	return false
}

func matchGlob(pattern, host string) bool {
	if pattern == "" {
		return host == ""
	}
	if pattern == "*" {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:]
		return host == suffix[1:] || strings.HasSuffix(host, suffix)
	}
	return host == pattern
}

type CategoryLoader struct {
	path string
}

func NewCategoryLoader(path string) *CategoryLoader {
	return &CategoryLoader{path: path}
}

func (cl *CategoryLoader) Load() (map[string]*Category, error) {
	data, err := os.ReadFile(cl.path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*Category), nil
		}
		return nil, err
	}

	parser := NewParser(string(data))
	return parser.ParseCategories()
}

func DefaultCategoryLoader() *CategoryLoader {
	home, err := os.UserHomeDir()
	if err != nil {
		return NewCategoryLoader("")
	}
	return NewCategoryLoader(filepath.Join(home, ".config", "makima", "categories.makima"))
}
