package cli

import (
	"encoding/json"
	"fmt"
)

type TodoItem struct {
	ID        string      `json:"id"`
	Text      string      `json:"text"`
	Completed bool        `json:"completed"`
	Children  []*TodoItem `json:"children,omitempty"`
	Progress  float64     `json:"progress"`
}

func (c *Client) TodoList() ([]TodoItem, error) {
	result, err := c.send("todo.list", nil)
	if err != nil {
		return nil, err
	}

	var items []TodoItem
	if err := json.Unmarshal(result, &items); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	return items, nil
}

func (c *Client) TodoAdd(text string, parentID *string) (string, error) {
	params := map[string]interface{}{
		"text": text,
	}
	if parentID != nil {
		params["parent_id"] = *parentID
	}

	result, err := c.send("todo.add", params)
	if err != nil {
		return "", err
	}

	var id string
	if err := json.Unmarshal(result, &id); err != nil {
		return "", fmt.Errorf("invalid response: %w", err)
	}

	return id, nil
}

func (c *Client) TodoDone(id string) error {
	_, err := c.send("todo.done", map[string]string{"id": id})
	return err
}

func (c *Client) TodoRemove(id string) error {
	_, err := c.send("todo.remove", map[string]string{"id": id})
	return err
}
