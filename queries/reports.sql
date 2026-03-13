-- GetProjectTaskStats
SELECT
    p.id, p.name,
    COUNT(t.id) AS total_tasks,
    COUNT(t.id) FILTER (WHERE t.status = 'done') AS done_tasks,
    COUNT(t.id) FILTER (WHERE t.status = 'in_progress') AS active_tasks
FROM projects p
LEFT JOIN tasks t ON t.project_id = p.id AND t.workspace_id = p.workspace_id
WHERE p.workspace_id = $1
GROUP BY p.id, p.name
ORDER BY p.name;

-- GetMemberTaskSummary
SELECT
    m.id, m.name,
    COUNT(t.id) AS assigned_tasks,
    COUNT(t.id) FILTER (WHERE t.status = 'done') AS completed_tasks,
    COUNT(t.id) FILTER (WHERE t.due_date < CURRENT_DATE AND t.status != 'done') AS overdue_tasks
FROM members m
LEFT JOIN tasks t ON t.assignee_id = m.id AND t.workspace_id = m.workspace_id
WHERE m.workspace_id = $1
GROUP BY m.id, m.name
ORDER BY m.name;

-- GetWorkspaceDashboard
SELECT
    (SELECT COUNT(*) FROM projects WHERE workspace_id = $1) AS project_count,
    (SELECT COUNT(*) FROM members WHERE workspace_id = $1) AS member_count,
    (SELECT COUNT(*) FROM tasks WHERE workspace_id = $1) AS total_tasks,
    (SELECT COUNT(*) FROM tasks WHERE workspace_id = $1 AND status = 'done') AS done_tasks,
    (SELECT COUNT(*) FROM tasks WHERE workspace_id = $1 AND due_date < CURRENT_DATE AND status != 'done') AS overdue_tasks;
