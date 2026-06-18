package daemon

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestSocketIPC(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "test.sock")

	srv, err := NewSocketServer(sockPath)
	if err != nil {
		t.Fatalf("NewSocketServer: %v", err)
	}

	srv.SetHandler(func(req Request) Response {
		if req.Method == "ping" {
			return Response{
				ID:     req.ID,
				Result: "pong",
			}
		}
		return Response{
			ID:    req.ID,
			Error: "unknown method",
		}
	})

	go srv.Serve()
	defer srv.Close()

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	req := Request{
		Method: "ping",
		Params: nil,
		ID:     1,
	}
	data, _ := json.Marshal(req)
	data = append(data, '\n')

	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatal("no response received")
	}

	var resp Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.ID != 1 {
		t.Errorf("expected ID 1, got %d", resp.ID)
	}
	if resp.Result != "pong" {
		t.Errorf("expected result 'pong', got '%v'", resp.Result)
	}
	if resp.Error != "" {
		t.Errorf("expected no error, got '%s'", resp.Error)
	}

	_ = os.Remove(sockPath)
}
