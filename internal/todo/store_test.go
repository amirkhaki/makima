package todo

import (
	"os"
	"testing"
)

func TestTodoStore(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "todo-test-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	id1, err := store.Add("Read book", "")
	if err != nil {
		t.Fatalf("failed to add todo: %v", err)
	}
	if id1 == "" {
		t.Fatal("expected non-empty ID")
	}

	id2, err := store.Add("Chapter 1", id1)
	if err != nil {
		t.Fatalf("failed to add child: %v", err)
	}
	id3, err := store.Add("Chapter 2", id1)
	if err != nil {
		t.Fatalf("failed to add child: %v", err)
	}

	todos := store.List()
	if len(todos) != 1 {
		t.Fatalf("expected 1 root todo, got %d", len(todos))
	}
	if todos[0].Text != "Read book" {
		t.Errorf("expected text 'Read book', got '%s'", todos[0].Text)
	}
	if len(todos[0].Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(todos[0].Children))
	}

	if err := store.Complete(id2); err != nil {
		t.Fatalf("failed to complete child: %v", err)
	}

	todos = store.List()
	if len(todos) != 1 {
		t.Fatalf("expected 1 root todo, got %d", len(todos))
	}
	if todos[0].Completed {
		t.Error("parent should not be completed yet")
	}
	if todos[0].Progress != 0.5 {
		t.Errorf("expected progress 0.5, got %f", todos[0].Progress)
	}

	if err := store.Complete(id3); err != nil {
		t.Fatalf("failed to complete child: %v", err)
	}

	todos = store.List()
	if !todos[0].Completed {
		t.Error("parent should be completed after all children done")
	}
	if todos[0].Progress != 1.0 {
		t.Errorf("expected progress 1.0, got %f", todos[0].Progress)
	}
}
