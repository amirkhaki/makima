package cli

import (
	"encoding/json"
)

func (c *Client) CategoryList() (map[string][]string, error) {
	result, err := c.send("category.list", nil)
	if err != nil {
		return nil, err
	}

	var categories map[string][]string
	if err := json.Unmarshal(result, &categories); err != nil {
		return nil, err
	}

	return categories, nil
}

func (c *Client) CategoryAdd(name string, patterns []string) error {
	_, err := c.send("category.add", map[string]interface{}{
		"name":     name,
		"patterns": patterns,
	})
	return err
}

func (c *Client) CategoryRemove(name string) error {
	_, err := c.send("category.remove", map[string]string{"name": name})
	return err
}
