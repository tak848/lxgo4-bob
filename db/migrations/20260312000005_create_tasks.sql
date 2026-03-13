-- migrate:up
CREATE TABLE tasks (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    project_id UUID NOT NULL,
    assignee_id UUID,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status task_status NOT NULL DEFAULT 'todo',
    priority task_priority NOT NULL DEFAULT 'medium',
    due_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, id),
    FOREIGN KEY (workspace_id, project_id) REFERENCES projects(workspace_id, id),
    FOREIGN KEY (workspace_id, assignee_id) REFERENCES members(workspace_id, id)
);

CREATE INDEX idx_tasks_workspace_id ON tasks(workspace_id);
CREATE INDEX idx_tasks_project ON tasks(workspace_id, project_id);
CREATE INDEX idx_tasks_assignee ON tasks(workspace_id, assignee_id);

-- migrate:down
DROP TABLE IF EXISTS tasks;
