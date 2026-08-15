-- Tasks: global standalone work items with a title, optional target
-- directory (cwd), and tags derived from #hashtags in the title.
-- Handled is DERIVED from the task_conversations link table, not stored.
CREATE TABLE tasks (
    task_id    TEXT PRIMARY KEY,
    title      TEXT NOT NULL,
    cwd        TEXT,
    tags       TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- One-way, many-to-many: a task is "handled" when it has >= 1 row here.
-- conversation_id deliberately has NO FK so deleting a conversation leaves
-- an orphaned row as a permanent "was handled" tombstone (the task never
-- silently reverts to idle).
CREATE TABLE task_conversations (
    task_id         TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    conversation_id TEXT NOT NULL,
    handled_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (task_id, conversation_id)
);

CREATE INDEX idx_tasks_created_at ON tasks(created_at DESC);
