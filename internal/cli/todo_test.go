package cli

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestTodoList(t *testing.T) {
	sockPath := filepath.Join(os.TempDir(), "makima-test-"+t.Name()+".sock")
	defer os.Remove(sockPath)

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

		items := []TodoItem{
			{
				ID:        "1",
				Text:      "test todo",
				Completed: false,
				Progress:  0.5,
				Children: []*TodoItem{
					{ID: "2", Text: "child todo", Completed: true, Progress: 1.0},
				},
			},
		}
		result, _ := json.Marshal(items)

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

	items, err := client.TodoList()
	if err != nil {
		t.Fatalf("TodoList failed: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 todo item, got %d", len(items))
	}

	if items[0].ID != "1" || items[0].Text != "test todo" || items[0].Completed {
		t.Errorf("unexpected todo item: %+v", items[0])
	}

	if len(items[0].Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(items[0].Children))
	}

	if items[0].Children[0].ID != "2" || !items[0].Children[0].Completed {
		t.Errorf("unexpected child: %+v", items[0].Children[0])
	}
}

func TestTodoAdd(t *testing.T) {
	sockPath := filepath.Join(os.TempDir(), "makima-test-"+t.Name()+".sock")
	defer os.Remove(sockPath)

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

		result, _ := json.Marshal("new-id")
		resp := Response{ID: req.ID, Result: result}
		data, _ := json.Marshal(resp)
		data = append(data, '\n')
		conn.Write(data)
	}()

	client, err := NewClient(sockPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	id, err := client.TodoAdd("new todo", nil)
	if err != nil {
		t.Fatalf("TodoAdd failed: %v", err)
	}

	if id != "new-id" {
		t.Errorf("expected id 'new-id', got '%s'", id)
	}
}

func TestTodoDone(t *testing.T) {
	sockPath := filepath.Join(os.TempDir(), "makima-test-"+t.Name()+".sock")
	defer os.Remove(sockPath)

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

		resp := Response{ID: req.ID, Result: json.RawMessage("true")}
		data, _ := json.Marshal(resp)
		data = append(data, '\n')
		conn.Write(data)
	}()

	client, err := NewClient(sockPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if err := client.TodoDone("1"); err != nil {
		t.Fatalf("TodoDone failed: %v", err)
	}
}

func TestTodoRemove(t *testing.T) {
	sockPath := filepath.Join(os.TempDir(), "makima-test-"+t.Name()+".sock")
	defer os.Remove(sockPath)

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

		resp := Response{ID: req.ID, Result: json.RawMessage("true")}
		data, _ := json.Marshal(resp)
		data = append(data, '\n')
		conn.Write(data)
	}()

	client, err := NewClient(sockPath)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if err := client.TodoRemove("1"); err != nil {
		t.Fatalf("TodoRemove failed: %v", err)
	}
}
