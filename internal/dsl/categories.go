package dsl

import (
	"net/url"
	"os"
	"path/filepath"

	"github.com/amirkhaki/makima/internal/makima"
)

func (c *Category) Matches(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	host := u.Hostname()
	for _, pattern := range c.Patterns {
		if makima.MatchGlob(pattern, host) {
			return true
		}
	}
	return false
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

	file, err := ParseMakimaFile(string(data))
	if err != nil {
		return nil, err
	}
	return file.Categories, nil
}

func DefaultCategoryLoader() *CategoryLoader {
	home, err := os.UserHomeDir()
	if err != nil {
		return NewCategoryLoader("")
	}
	return NewCategoryLoader(filepath.Join(home, ".config", "makima", "categories.makima"))
}
