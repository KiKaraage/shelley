package claudetool

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"shelley.exe.dev/db"
)

// mockTaskDB implements TaskDB using an in-memory store.
type mockTaskDB struct {
	tasks    map[string]*db.TaskSummary
	handled  map[string][]string // taskID -> conversationIDs
	nextID   int
}

func newMockTaskDB() *mockTaskDB {
	return &mockTaskDB{
		tasks:   make(map[string]*db.TaskSummary),
		handled: make(map[string][]string),
	}
}

func (m *mockTaskDB) List(ctx context.Context) ([]*db.TaskSummary, error) {
	var out []*db.TaskSummary
	for _, t := range m.tasks {
		if len(m.handled[t.TaskID]) == 0 {
			out = append(out, t)
		}
	}
	return out, nil
}

func (m *mockTaskDB) Create(ctx context.Context, taskID, title string, cwd *string, tags []string) (*db.TaskSummary, error) {
	if taskID == "" {
		m.nextID++
		taskID = "t" + string(rune('0'+m.nextID))
	}
	t := &db.TaskSummary{TaskID: taskID, Title: title, Cwd: cwd, Tags: tags, CreatedAt: time.Now()}
	m.tasks[taskID] = t
	return t, nil
}

func (m *mockTaskDB) Update(ctx context.Context, taskID, title string, cwd *string, tags []string, conversationID string) (*db.TaskSummary, error) {
	t, ok := m.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	if title != "" {
		t.Title = title
	}
	if cwd != nil {
		t.Cwd = cwd
	}
	if tags != nil {
		t.Tags = tags
	}
	if conversationID != "" {
		m.handled[taskID] = append(m.handled[taskID], conversationID)
	}
	return t, nil
}

func (m *mockTaskDB) Delete(ctx context.Context, taskID string) error {
	delete(m.tasks, taskID)
	delete(m.handled, taskID)
	return nil
}

func TestTaskToolList(t *testing.T) {
	store := newMockTaskDB()
	store.Create(context.Background(), "", "Task A", nil, []string{"working"})
	store.Create(context.Background(), "", "Task B", nil, nil)
	// Mark A handled so it's excluded from list.
	store.handled["t1"] = []string{"c1"}

	tool := &TaskTool{DB: store}
	ts := tool.Tool()
	if ts.Name != "task" {
		t.Errorf("tool name = %q", ts.Name)
	}

	out := tool.run(context.Background(), taskInput{Action: "list"})
	text := out.LLMContent[0].Text
	if !strings.Contains(text, "Task B") {
		t.Errorf("list output missing Task B: %q", text)
	}
	if strings.Contains(text, "Task A") {
		t.Errorf("list output should exclude handled Task A: %q", text)
	}
}

func TestTaskToolCreate(t *testing.T) {
	store := newMockTaskDB()
	tool := &TaskTool{DB: store}

	out := tool.run(context.Background(), taskInput{Action: "create", Title: "New task #plan"})
	if out.Error != nil {
		t.Fatalf("create error: %s", out.Error)
	}
	if len(store.tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(store.tasks))
	}

	// Missing title -> error
	out = tool.run(context.Background(), taskInput{Action: "create"})
	if out.Error == nil {
		t.Error("expected error for missing title")
	}
}

func TestTaskToolUpdateHandle(t *testing.T) {
	store := newMockTaskDB()
	store.Create(context.Background(), "", "Task", nil, nil)
	taskID := "t1"

	tool := &TaskTool{DB: store}

	// Update with conversation_id marks handled.
	out := tool.run(context.Background(), taskInput{Action: "update", TaskID: taskID, Title: "Renamed", ConversationID: "cABC"})
	if out.Error != nil {
		t.Fatalf("update error: %s", out.Error)
	}
	if len(store.handled[taskID]) != 1 {
		t.Errorf("task should be handled, links = %v", store.handled[taskID])
	}

	// Missing task_id -> error
	out = tool.run(context.Background(), taskInput{Action: "update"})
	if out.Error == nil {
		t.Error("expected error for missing task_id")
	}
}

func TestTaskToolDelete(t *testing.T) {
	store := newMockTaskDB()
	store.Create(context.Background(), "", "Task", nil, nil)
	tool := &TaskTool{DB: store}

	out := tool.run(context.Background(), taskInput{Action: "delete", TaskID: "t1"})
	if out.Error != nil {
		t.Fatalf("delete error: %s", out.Error)
	}
	if len(store.tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(store.tasks))
	}

	// Unknown action
	out = tool.run(context.Background(), taskInput{Action: "bogus"})
	if out.Error == nil {
		t.Error("expected error for unknown action")
	}
}

