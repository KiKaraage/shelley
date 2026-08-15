package db

import (
	"context"
	"testing"
)

func TestTaskCRUD(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Create
	task, err := db.CreateTask(ctx, "", "Fix flaky test #working", nil, []string{"working"})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if task.TaskID == "" {
		t.Fatal("expected generated task id")
	}
	if task.Title != "Fix flaky test #working" {
		t.Errorf("title = %q", task.Title)
	}
	if len(task.Tags) != 1 || task.Tags[0] != "working" {
		t.Errorf("tags = %v", task.Tags)
	}
	if task.Handled {
		t.Error("new task should not be handled")
	}

	// Get
	got, err := db.GetTask(ctx, task.TaskID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.TaskID != task.TaskID {
		t.Errorf("GetTask id = %q", got.TaskID)
	}

	// Update title + cwd
	cwd := "/tmp"
	updated, err := db.UpdateTask(ctx, task.TaskID, "Renamed", &cwd)
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if updated.Title != "Renamed" {
		t.Errorf("updated title = %q", updated.Title)
	}
	if updated.Cwd == nil || *updated.Cwd != cwd {
		t.Errorf("updated cwd = %v", updated.Cwd)
	}

	// Update tags (metadata-only)
	reTagged, err := db.UpdateTaskTags(ctx, task.TaskID, []string{"done"})
	if err != nil {
		t.Fatalf("UpdateTaskTags: %v", err)
	}
	if len(reTagged.Tags) != 1 || reTagged.Tags[0] != "done" {
		t.Errorf("reTagged tags = %v", reTagged.Tags)
	}

	// List (unhandled)
	list, err := db.ListTasks(ctx)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListTasks len = %d, want 1", len(list))
	}

	// Mark handled
	if err := db.MarkTaskHandled(ctx, task.TaskID, "cABC123"); err != nil {
		t.Fatalf("MarkTaskHandled: %v", err)
	}
	handled, err := db.GetTask(ctx, task.TaskID)
	if err != nil {
		t.Fatalf("GetTask after handle: %v", err)
	}
	if !handled.Handled {
		t.Error("task should be handled after link")
	}
	if handled.HandledAt == nil {
		t.Error("handled_at should be set")
	}

	// List after handle: task should no longer be in active summaries
	summaries, err := db.ListTaskSummaries(ctx)
	if err != nil {
		t.Fatalf("ListTaskSummaries: %v", err)
	}
	if len(summaries) != 0 {
		t.Errorf("ListTaskSummaries len = %d, want 0", len(summaries))
	}

	// ListAllTasks still includes it with handled=true
	all, err := db.ListTasks(ctx)
	if err != nil {
		t.Fatalf("ListTasks after handle: %v", err)
	}
	if len(all) != 1 || !all[0].Handled {
		t.Errorf("ListTasks after handle = %+v", all)
	}

	// Delete
	if err := db.DeleteTask(ctx, task.TaskID); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
	if _, err := db.GetTask(ctx, task.TaskID); err == nil {
		t.Error("GetTask should fail after delete")
	}
}

func TestTaskConversationLinkTombstone(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	task, err := db.CreateTask(ctx, "", "T", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.MarkTaskHandled(ctx, task.TaskID, "cGONE"); err != nil {
		t.Fatal(err)
	}
	// Deleting the task removes its links via cascade.
	if err := db.DeleteTask(ctx, task.TaskID); err != nil {
		t.Fatal(err)
	}
	links, err := db.ListTaskConversations(ctx, task.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 0 {
		t.Errorf("links after task delete = %d, want 0", len(links))
	}
}

func TestListDistinctCwds(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// No cwds initially.
	cwds, err := db.ListDistinctCwds(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(cwds) != 0 {
		t.Errorf("initial cwds = %v", cwds)
	}

	// Create a conversation with a cwd.
	slug := "s"
	conv, err := db.CreateConversation(ctx, &slug, true, strPtr("/work/a"), nil, ConversationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_ = conv

	// Create tasks with cwds.
	if _, err := db.CreateTask(ctx, "", "T1", strPtr("/work/b"), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateTask(ctx, "", "T2", strPtr("/work/a"), nil); err != nil {
		t.Fatal(err)
	}

	cwds, err = db.ListDistinctCwds(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(cwds) != 2 {
		t.Errorf("cwds = %v, want 2", cwds)
	}
	seen := map[string]bool{}
	for _, c := range cwds {
		seen[c] = true
	}
	if !seen["/work/a"] || !seen["/work/b"] {
		t.Errorf("missing cwds: %v", cwds)
	}
}

func strPtr(s string) *string { return &s }
