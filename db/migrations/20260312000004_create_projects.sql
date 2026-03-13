-- migrate:up
CREATE TABLE projects (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status project_status NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, id)
);

CREATE INDEX idx_projects_workspace_id ON projects(workspace_id);

-- migrate:down
DROP TABLE IF EXISTS projects;
