-- migrate:up
CREATE TYPE member_role AS ENUM ('owner', 'editor', 'viewer');
CREATE TYPE project_status AS ENUM ('active', 'archived');
CREATE TYPE task_status AS ENUM ('todo', 'in_progress', 'done');
CREATE TYPE task_priority AS ENUM ('low', 'medium', 'high', 'urgent');

-- migrate:down
DROP TYPE IF EXISTS task_priority;
DROP TYPE IF EXISTS task_status;
DROP TYPE IF EXISTS project_status;
DROP TYPE IF EXISTS member_role;
