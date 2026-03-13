-- migrate:up
CREATE TABLE members (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    role member_role NOT NULL DEFAULT 'viewer',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, id),
    UNIQUE (workspace_id, email)
);

CREATE INDEX idx_members_workspace_id ON members(workspace_id);

-- migrate:down
DROP TABLE IF EXISTS members;
