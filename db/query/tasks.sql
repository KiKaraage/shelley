-- name: CreateTask :one
INSERT INTO tasks (task_id, title, cwd, tags)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE task_id = ?;

-- name: ListTasks :many
-- Active (unhandled) tasks, newest first. Handled is derived from the
-- task_conversations link table.
SELECT t.*
FROM tasks t
WHERE NOT EXISTS (SELECT 1 FROM task_conversations tc WHERE tc.task_id = t.task_id)
ORDER BY t.created_at DESC, t.task_id DESC;

-- name: ListAllTasks :many
-- Every task, newest first, with a handled flag and the first handled_at.
SELECT t.*, tc.handled_at
FROM tasks t
LEFT JOIN task_conversations tc
  ON tc.task_id = t.task_id
 AND tc.handled_at = (SELECT MIN(handled_at) FROM task_conversations WHERE task_id = t.task_id)
ORDER BY t.created_at DESC, t.task_id DESC;

-- name: ListTaskSummaries :many
-- Minimal agent payload for active (unhandled) tasks only.
SELECT task_id, title, cwd, tags, created_at
FROM tasks t
WHERE NOT EXISTS (SELECT 1 FROM task_conversations tc WHERE tc.task_id = t.task_id)
ORDER BY t.created_at DESC, t.task_id DESC;

-- name: UpdateTask :one
UPDATE tasks
SET title = ?,
    cwd = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE task_id = ?
RETURNING *;

-- name: UpdateTaskTags :one
-- Tagging is a metadata-only edit; deliberately does not bump updated_at.
UPDATE tasks
SET tags = ?
WHERE task_id = ?
RETURNING *;

-- name: MarkTaskHandled :exec
INSERT OR IGNORE INTO task_conversations (task_id, conversation_id)
VALUES (?, ?);

-- name: DeleteTask :exec
DELETE FROM tasks WHERE task_id = ?;

-- name: ListTaskConversations :many
SELECT task_id, conversation_id, handled_at
FROM task_conversations
WHERE task_id = ?
ORDER BY handled_at ASC;

-- name: ListDistinctCwds :many
-- Distinct non-empty cwds across all tasks and conversations, most recently
-- used first, for the task modal's "recent directories" dropdown.
SELECT cwd
FROM (
    SELECT cwd, MAX(updated_at) AS last_used FROM conversations WHERE cwd IS NOT NULL AND cwd <> '' GROUP BY cwd
    UNION
    SELECT cwd, MAX(updated_at) AS last_used FROM tasks WHERE cwd IS NOT NULL AND cwd <> '' GROUP BY cwd
)
ORDER BY last_used DESC;
