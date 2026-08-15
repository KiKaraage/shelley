package claudetool

import (
	"context"
	"fmt"
	"strings"
	"time"

	"shelley.exe.dev/db"
	"shelley.exe.dev/llm"
)

// TaskDB is the database interface for task operations.
// This is implemented by the db package.
type TaskDB interface {
	// List returns the minimal payload for active (unhandled) tasks.
	List(ctx context.Context) ([]*db.TaskSummary, error)
	// Create creates a task and returns its summary.
	Create(ctx context.Context, taskID, title string, cwd *string, tags []string) (*db.TaskSummary, error)
	// Update updates a task's title/cwd/tags. If conversationID is non-empty,
	// the task is also marked handled (linked to that conversation).
	Update(ctx context.Context, taskID, title string, cwd *string, tags []string, conversationID string) (*db.TaskSummary, error)
	// Delete removes a task.
	Delete(ctx context.Context, taskID string) error
}

// TaskTool provides agent access to tasks: list, create, update, delete.
type TaskTool struct {
	DB TaskDB
}

const taskName = "task"

// taskDescription describes the task tool for the model.
const taskDescription = `Manage work-item tasks.

Tasks are lightweight work items that live on the Shelley homepage. Each task
has a title, an optional target directory (cwd), and tags derived from
#hashtags in the title. A task is "handled" once a conversation is created
from it.

Use the "action" field to select an operation:
- action: "list" — List active (unhandled) tasks. Returns one task per line,
  tab-separated: task_id, title, cwd, tags, created_at. No other parameters.
- action: "create" — Create a task. Requires "title"; "cwd" and "tags" are
  optional. Echoes the created task.
- action: "update" — Update a task. Requires "task_id"; optionally update
  "title", "cwd", "tags". If "conversation_id" is provided, the task is also
  marked handled (linked to that conversation). Echoes the updated task.
- action: "delete" — Delete a task. Requires "task_id". Echoes the deleted id.`

// taskInputSchema is the JSON schema for the task tool.
const taskInputSchema = `{
  "type": "object",
  "required": ["action"],
  "properties": {
    "action": {
      "type": "string",
      "description": "The task operation to perform",
      "enum": ["list", "create", "update", "delete"]
    },
    "task_id": {
      "type": "string",
      "description": "The task ID (required for update and delete)"
    },
    "title": {
      "type": "string",
      "description": "The task title (required for create; optional for update). Tags are derived from #hashtags in the title."
    },
    "cwd": {
      "type": "string",
      "description": "Optional target directory (absolute path)"
    },
    "tags": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Optional tags (create/update). If omitted on update, existing tags are kept."
    },
    "conversation_id": {
      "type": "string",
      "description": "When provided on update, marks the task handled by linking it to this conversation"
    }
  }
}`

type taskInput struct {
	Action         string   `json:"action"`
	TaskID         string   `json:"task_id"`
	Title          string   `json:"title"`
	Cwd            string   `json:"cwd"`
	Tags           []string `json:"tags"`
	ConversationID string   `json:"conversation_id"`
}

// Tool returns an llm.Tool for task management.
func (t *TaskTool) Tool() *llm.Tool {
	return &llm.Tool{
		Name:        taskName,
		Description: taskDescription,
		InputSchema: llm.MustSchema(taskInputSchema),
		Run:         llm.RunJSON(t.run),
	}
}

func (t *TaskTool) run(ctx context.Context, req taskInput) llm.ToolOut {
	switch req.Action {
	case "list":
		return t.runList(ctx)
	case "create":
		return t.runCreate(ctx, req)
	case "update":
		return t.runUpdate(ctx, req)
	case "delete":
		return t.runDelete(ctx, req)
	default:
		return llm.ErrorfToolOut("unknown task action %q; expected list, create, update, or delete", req.Action)
	}
}

func (t *TaskTool) runList(ctx context.Context) llm.ToolOut {
	tasks, err := t.DB.List(ctx)
	if err != nil {
		return llm.ErrorfToolOut("failed to list tasks: %w", err)
	}
	if len(tasks) == 0 {
		return llm.ToolOut{LLMContent: llm.TextContent("No active tasks.")}
	}
	var b strings.Builder
	for _, task := range tasks {
		cwd := ""
		if task.Cwd != nil {
			cwd = *task.Cwd
		}
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\n",
			task.TaskID, task.Title, cwd, strings.Join(task.Tags, ","), task.CreatedAt.Format(time.RFC3339))
	}
	return llm.ToolOut{LLMContent: llm.TextContent(strings.TrimSuffix(b.String(), "\n"))}
}

func (t *TaskTool) runCreate(ctx context.Context, req taskInput) llm.ToolOut {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return llm.ErrorfToolOut("title is required")
	}
	var cwd *string
	if req.Cwd != "" {
		cwd = &req.Cwd
	}
	task, err := t.DB.Create(ctx, "", title, cwd, req.Tags)
	if err != nil {
		return llm.ErrorfToolOut("failed to create task: %w", err)
	}
	return llm.ToolOut{LLMContent: llm.TextContent(formatTask(task))}
}

func (t *TaskTool) runUpdate(ctx context.Context, req taskInput) llm.ToolOut {
	if req.TaskID == "" {
		return llm.ErrorfToolOut("task_id is required for update")
	}
	title := strings.TrimSpace(req.Title)
	var cwd *string
	if req.Cwd != "" {
		cwd = &req.Cwd
	}
	var tags []string
	if req.Tags != nil {
		tags = req.Tags
	}
	task, err := t.DB.Update(ctx, req.TaskID, title, cwd, tags, req.ConversationID)
	if err != nil {
		return llm.ErrorfToolOut("failed to update task: %w", err)
	}
	return llm.ToolOut{LLMContent: llm.TextContent(formatTask(task))}
}

func (t *TaskTool) runDelete(ctx context.Context, req taskInput) llm.ToolOut {
	if req.TaskID == "" {
		return llm.ErrorfToolOut("task_id is required for delete")
	}
	if err := t.DB.Delete(ctx, req.TaskID); err != nil {
		return llm.ErrorfToolOut("failed to delete task: %w", err)
	}
	return llm.ToolOut{LLMContent: llm.TextContent(fmt.Sprintf("Deleted task %s", req.TaskID))}
}

func formatTask(t *db.TaskSummary) string {
	cwd := ""
	if t.Cwd != nil {
		cwd = *t.Cwd
	}
	return fmt.Sprintf("task_id=%s title=%q cwd=%s tags=[%s] created_at=%s",
		t.TaskID, t.Title, cwd, strings.Join(t.Tags, ","), t.CreatedAt.Format(time.RFC3339))
}
