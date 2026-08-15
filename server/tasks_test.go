package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"shelley.exe.dev/db"
)

func TestTaskHandlersCRUD(t *testing.T) {
	t.Parallel()
	h := NewTestHarness(t)
	ctx := context.Background()

	// Create
	createBody := `{"title":"Fix flaky test #working","cwd":"/tmp"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(createBody))
	w := httptest.NewRecorder()
	h.server.handleCreateTask(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create code = %d, body = %s", w.Code, w.Body.String())
	}
	var created TaskResponse
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.TaskID == "" {
		t.Fatal("expected task id")
	}
	if created.Handled {
		t.Error("new task should not be handled")
	}

	// List
	req = httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	w = httptest.NewRecorder()
	h.server.handleListTasks(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list code = %d", w.Code)
	}
	var list []TaskResponse
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}

	// Update (title + cwd)
	updateBody := `{"title":"Renamed task"}`
	req = httptest.NewRequest(http.MethodPatch, "/api/tasks/"+created.TaskID, bytes.NewBufferString(updateBody))
	w = httptest.NewRecorder()
	h.server.handleUpdateTask(w, req, created.TaskID)
	if w.Code != http.StatusOK {
		t.Fatalf("update code = %d, body = %s", w.Code, w.Body.String())
	}
	var updated TaskResponse
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Renamed task" {
		t.Errorf("updated title = %q", updated.Title)
	}

	// Mark handled via the conversation link
	if err := h.db.MarkTaskHandled(ctx, created.TaskID, "cABC123"); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/tasks/"+created.TaskID, nil)
	w = httptest.NewRecorder()
	h.server.handleGetTask(w, req, created.TaskID)
	if w.Code != http.StatusOK {
		t.Fatalf("get code = %d", w.Code)
	}
	var got TaskResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Handled {
		t.Error("task should be handled after link")
	}

	// Delete
	req = httptest.NewRequest(http.MethodDelete, "/api/tasks/"+created.TaskID, nil)
	w = httptest.NewRecorder()
	h.server.handleDeleteTask(w, req, created.TaskID)
	if w.Code != http.StatusOK {
		t.Fatalf("delete code = %d", w.Code)
	}
}

func TestTaskHandlersValidation(t *testing.T) {
	t.Parallel()
	h := NewTestHarness(t)

	// Empty title -> 400
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{"title":"  "}`))
	w := httptest.NewRecorder()
	h.server.handleCreateTask(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("empty title code = %d, want 400", w.Code)
	}

	// Relative cwd -> 400
	req = httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{"title":"T","cwd":"relative"}`))
	w = httptest.NewRecorder()
	h.server.handleCreateTask(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("relative cwd code = %d, want 400", w.Code)
	}
}

func TestTaskDirectoriesHandler(t *testing.T) {
	t.Parallel()
	h := NewTestHarness(t)
	ctx := context.Background()

	// Seed a conversation with a cwd.
	slug := "s"
	if _, err := h.db.CreateConversation(ctx, &slug, true, strPtr("/work/x"), nil, db.ConversationOptions{}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/directories", nil)
	w := httptest.NewRecorder()
	h.server.handleTaskDirectories(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("directories code = %d", w.Code)
	}
	var resp TaskDirectoriesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range resp.Cwds {
		if c == "/work/x" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected /work/x in cwds, got %v", resp.Cwds)
	}
}

func TestNewConversationMarksTaskHandled(t *testing.T) {
	t.Parallel()
	h := NewTestHarness(t)
	ctx := context.Background()

	task, err := h.db.CreateTask(ctx, "", "Task", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Send a new-conversation request with task_id.
	body := `{"message":"hello","task_id":"` + task.TaskID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/conversations/new", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.server.handleNewConversation(w, req)
	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("new conversation code = %d, body = %s", w.Code, w.Body.String())
	}

	got, err := h.db.GetTask(ctx, task.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Handled {
		t.Error("task should be handled after conversation created from it")
	}
}

func TestTaskRoutesRegistered(t *testing.T) {
	t.Parallel()
	h := NewTestHarness(t)
	mux := http.NewServeMux()
	h.server.RegisterRoutes(mux)

	// Create a task via the mux.
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{"title":"Route task"}`))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/tasks via mux code = %d, body = %s", w.Code, w.Body.String())
	}
	var created TaskResponse
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	// List via mux.
	req = httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/tasks via mux code = %d", w.Code)
	}

	// Directories via mux (must not be swallowed by /api/tasks/ subtree).
	req = httptest.NewRequest(http.MethodGet, "/api/tasks/directories", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/tasks/directories via mux code = %d, body = %s", w.Code, w.Body.String())
	}

	// Delete via mux.
	req = httptest.NewRequest(http.MethodDelete, "/api/tasks/"+created.TaskID, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("DELETE /api/tasks/{id} via mux code = %d", w.Code)
	}
}
