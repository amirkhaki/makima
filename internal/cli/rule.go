package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
)

type RuleError struct {
	Message string
}

func (e *RuleError) Error() string {
	return e.Message
}

type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
	ID     int             `json:"id"`
}

type Response struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	mu     sync.Mutex
	id     int
	sockPath string
}

func NewClient(sockPath string) (*Client, error) {
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:     conn,
		reader:   bufio.NewReader(conn),
		sockPath: sockPath,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) send(method string, params any) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.id++

	var rawParams json.RawMessage
	if params != nil {
		var err error
		rawParams, err = json.Marshal(params)
		if err != nil {
			return nil, err
		}
	}

	req := Request{
		Method: method,
		Params: rawParams,
		ID:     c.id,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	data = append(data, '\n')
	if _, err := c.conn.Write(data); err != nil {
		return nil, err
	}

	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, &RuleError{Message: resp.Error}
	}

	return resp.Result, nil
}

type RuleInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Condition string `json:"condition"`
	Enabled   bool   `json:"enabled"`
}

type AddRuleParams struct {
	Name      string `json:"name"`
	Condition string `json:"condition"`
	Action    string `json:"action"`
}

type RemoveRuleParams struct {
	ID string `json:"id"`
}

type ToggleRuleParams struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

func (c *Client) RuleList() ([]RuleInfo, error) {
	result, err := c.send("rule.list", nil)
	if err != nil {
		return nil, err
	}

	var rules []RuleInfo
	if err := json.Unmarshal(result, &rules); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	return rules, nil
}

func (c *Client) RuleAdd(name, condition, action string) (string, error) {
	params := AddRuleParams{
		Name:      name,
		Condition: condition,
		Action:    action,
	}

	result, err := c.send("rule.add", params)
	if err != nil {
		return "", err
	}

	var id string
	if err := json.Unmarshal(result, &id); err != nil {
		return "", fmt.Errorf("invalid response: %w", err)
	}

	return id, nil
}

func (c *Client) RuleRemove(id string) error {
	params := RemoveRuleParams{ID: id}
	_, err := c.send("rule.remove", params)
	return err
}

func (c *Client) RuleEnable(id string) error {
	params := ToggleRuleParams{ID: id, Enabled: true}
	_, err := c.send("rule.enable", params)
	return err
}

func (c *Client) RuleDisable(id string) error {
	params := ToggleRuleParams{ID: id, Enabled: false}
	_, err := c.send("rule.disable", params)
	return err
}

func (c *Client) Status() (map[string]interface{}, error) {
	result, err := c.send("status", nil)
	if err != nil {
		return nil, err
	}

	var status map[string]interface{}
	if err := json.Unmarshal(result, &status); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	return status, nil
}
