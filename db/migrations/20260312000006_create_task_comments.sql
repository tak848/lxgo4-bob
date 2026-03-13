-- migrate:up
CREATE TABLE task_comments (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    task_id UUID NOT NULL,
    author_id UUID NOT NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, id),
    FOREIGN KEY (workspace_id, task_id) REFERENCES tasks(workspace_id, id),
    FOREIGN KEY (workspace_id, author_id) REFERENCES members(workspace_id, id)
);

CREATE INDEX idx_task_comments_workspace_id ON task_comments(workspace_id);
CREATE INDEX idx_task_comments_task ON task_comments(workspace_id, task_id);

-- migrate:down
DROP TABLE IF EXISTS task_comments;
