package cli

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestRuleList(t *testing.T) {
	sockPath := filepath.Join(os.TempDir(), "makima-test-"+t.Name()+".sock")
	defer os.Remove(sockPath)

	// Start a mock server
	os.Remove(sockPath)
	l, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer l.Close()

	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		buf := make([]byte, 4096)
		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		var req Request
		if err := json.Unmarshal(buf[:n], &req); err != nil {
			return
		}

		rules := []RuleInfo{
			{ID: "1", Name: "test-rule", Enabled: true},
		}
		result, _ := json.Marshal(rules)

		resp := Response{
			ID:     req.ID,
			Result: result,
		}
		data, _ := json.Marshal(resp)
		data = append(data, '\n')
		conn.Write(data)
	}()

	client, err := NewClient(sockPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	rules, err := client.RuleList()
	if err != nil {
		t.Fatalf("RuleList failed: %v", err)
	}

	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}

	if rules[0].ID != "1" || rules[0].Name != "test-rule" || !rules[0].Enabled {
		t.Errorf("unexpected rule: %+v", rules[0])
	}
}
