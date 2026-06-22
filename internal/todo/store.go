package todo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Todo struct {
	ID        string   `json:"id"`
	Text      string   `json:"text"`
	Completed bool     `json:"completed"`
	ParentID  string   `json:"parent_id,omitempty"`
	Children  []*Todo  `json:"children,omitempty"`
	Progress  float64  `json:"progress"`
	CreatedAt time.Time `json:"created_at"`
}

type Store struct {
	filePath string
	todos    []*Todo
	mu       sync.RWMutex
}

func NewStore(filePath string) (*Store, error) {
	s := &Store{filePath: filePath}
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &s.todos)
}

func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.todos, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}

func (s *Store) Add(text, parentID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.generateID()
	todo := &Todo{
		ID:        id,
		Text:      text,
		ParentID:  parentID,
		CreatedAt: time.Now(),
	}

	if parentID == "" {
		s.todos = append(s.todos, todo)
	} else {
		parent := s.findTodo(parentID)
		if parent == nil {
			return "", fmt.Errorf("parent todo %s not found", parentID)
		}
		todo.ParentID = parentID
		parent.Children = append(parent.Children, todo)
		s.updateProgress(parent)
	}

	if err := s.save(); err != nil {
		return "", err
	}
	return id, nil
}

func (s *Store) Complete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo := s.findTodo(id)
	if todo == nil {
		return fmt.Errorf("todo %s not found", id)
	}

	s.completeTodo(todo)

	if todo.ParentID != "" {
		if parent := s.findTodo(todo.ParentID); parent != nil {
			s.updateProgress(parent)
			if parent.Progress == 1.0 {
				parent.Completed = true
			}
		}
	}

	return s.save()
}

func (s *Store) completeTodo(todo *Todo) {
	todo.Completed = true
	for _, child := range todo.Children {
		s.completeTodo(child)
	}
	s.updateTodoProgress(todo)
}

func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.removeTodoFromSlice(&s.todos, id) {
		return s.save()
	}
	return fmt.Errorf("todo %s not found", id)
}

func (s *Store) removeTodoFromSlice(todos *[]*Todo, id string) bool {
	for i, todo := range *todos {
		if todo.ID == id {
			*todos = append((*todos)[:i], (*todos)[i+1:]...)
			return true
		}
		if s.removeTodoFromSlice(&todo.Children, id) {
			return true
		}
	}
	return false
}

func (s *Store) List() []*Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.todos
}

func (s *Store) findTodo(id string) *Todo {
	return s.findTodoInSlice(s.todos, id)
}

func (s *Store) findTodoInSlice(todos []*Todo, id string) *Todo {
	for _, todo := range todos {
		if todo.ID == id {
			return todo
		}
		if found := s.findTodoInSlice(todo.Children, id); found != nil {
			return found
		}
	}
	return nil
}

func (s *Store) updateProgress(todo *Todo) {
	if len(todo.Children) == 0 {
		if todo.Completed {
			todo.Progress = 1.0
		} else {
			todo.Progress = 0.0
		}
		return
	}

	completed := 0
	for _, child := range todo.Children {
		s.updateTodoProgress(child)
		if child.Completed {
			completed++
		}
	}
	todo.Progress = float64(completed) / float64(len(todo.Children))

	if todo.Progress == 1.0 {
		todo.Completed = true
	} else {
		todo.Completed = false
	}
}

func (s *Store) updateTodoProgress(todo *Todo) {
	if len(todo.Children) == 0 {
		if todo.Completed {
			todo.Progress = 1.0
		} else {
			todo.Progress = 0.0
		}
		return
	}

	completed := 0
	for _, child := range todo.Children {
		if child.Completed {
			completed++
		}
	}
	todo.Progress = float64(completed) / float64(len(todo.Children))

	if todo.Progress == 1.0 {
		todo.Completed = true
	} else {
		todo.Completed = false
	}
}

func (s *Store) generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func (s *Store) TreeString() string {
	var sb strings.Builder
	for _, todo := range s.todos {
		s.writeTree(&sb, todo, 0)
	}
	return sb.String()
}

func (s *Store) writeTree(sb *strings.Builder, todo *Todo, depth int) {
	prefix := strings.Repeat("  ", depth)
	status := "[ ]"
	if todo.Completed {
		status = "[x]"
	}
	sb.WriteString(fmt.Sprintf("%s%s %s (%.0f%%)\n", prefix, status, todo.Text, todo.Progress*100))
	for _, child := range todo.Children {
		s.writeTree(sb, child, depth+1)
	}
}
