package handler

import (
	"context"

	"github.com/tak848/lxgo4-bob-playground/internal/oas"
)

func (h *Handler) ListTasks(ctx context.Context, params oas.ListTasksParams) ([]oas.TaskWithRelations, error) {
	filter := TaskFilter{
		Limit:  params.Limit.Or(20),
		Offset: params.Offset.Or(0),
	}
	if v, ok := params.Status.Get(); ok {
		s := string(v)
		filter.Status = &s
	}
	if v, ok := params.Priority.Get(); ok {
		s := string(v)
		filter.Priority = &s
	}
	if v, ok := params.AssigneeID.Get(); ok {
		filter.AssigneeID = &v
	}
	if v, ok := params.ProjectID.Get(); ok {
		filter.ProjectID = &v
	}

	list, err := h.Tasks.List(ctx, params.WsId, filter)
	if err != nil {
		return nil, err
	}
	out := make([]oas.TaskWithRelations, len(list))
	for i, t := range list {
		out[i] = taskWithRelationsToOAS(t)
	}
	return out, nil
}

func (h *Handler) CreateTask(ctx context.Context, req *oas.CreateTaskRequest, params oas.CreateTaskParams) (*oas.Task, error) {
	input := CreateTaskInput{
		ProjectID:   req.ProjectID,
		Title:       req.Title,
		Description: req.Description,
		Status:      string(req.Status),
		Priority:    string(req.Priority),
	}
	if v, ok := req.AssigneeID.Get(); ok {
		input.AssigneeID = &v
	}
	if v, ok := req.DueDate.Get(); ok {
		input.DueDate = &v
	}

	t, err := h.Tasks.Create(ctx, params.WsId, input)
	if err != nil {
		return nil, err
	}
	o := taskToOAS(*t)
	return &o, nil
}

func (h *Handler) GetTask(ctx context.Context, params oas.GetTaskParams) (*oas.TaskDetail, error) {
	t, err := h.Tasks.Get(ctx, params.WsId, params.ID)
	if err != nil {
		return nil, err
	}
	o := taskDetailToOAS(*t)
	return &o, nil
}

func (h *Handler) UpdateTask(ctx context.Context, req *oas.UpdateTaskRequest, params oas.UpdateTaskParams) (*oas.Task, error) {
	input := UpdateTaskInput{
		ProjectID:   req.ProjectID,
		Title:       req.Title,
		Description: req.Description,
		Status:      string(req.Status),
		Priority:    string(req.Priority),
	}
	if v, ok := req.AssigneeID.Get(); ok {
		input.AssigneeID = &v
	}
	if v, ok := req.DueDate.Get(); ok {
		input.DueDate = &v
	}

	t, err := h.Tasks.Update(ctx, params.WsId, params.ID, input)
	if err != nil {
		return nil, err
	}
	o := taskToOAS(*t)
	return &o, nil
}

func (h *Handler) DeleteTask(ctx context.Context, params oas.DeleteTaskParams) error {
	return h.Tasks.Delete(ctx, params.WsId, params.ID)
}

func taskToOAS(t TaskDTO) oas.Task {
	o := oas.Task{
		ID:          t.ID,
		WorkspaceID: t.WorkspaceID,
		ProjectID:   t.ProjectID,
		Title:       t.Title,
		Description: t.Description,
		Status:      oas.TaskStatus(t.Status),
		Priority:    oas.TaskPriority(t.Priority),
	}
	if t.AssigneeID != nil {
		o.AssigneeID = oas.NewOptUUID(*t.AssigneeID)
	}
	if t.DueDate != nil {
		o.DueDate = oas.NewOptDate(*t.DueDate)
	}
	return o
}

func taskWithRelationsToOAS(t TaskWithRelationsDTO) oas.TaskWithRelations {
	o := oas.TaskWithRelations{
		ID:          t.ID,
		WorkspaceID: t.WorkspaceID,
		ProjectID:   t.ProjectID,
		Title:       t.Title,
		Description: t.Description,
		Status:      oas.TaskStatus(t.Status),
		Priority:    oas.TaskPriority(t.Priority),
	}
	if t.AssigneeID != nil {
		o.AssigneeID = oas.NewOptUUID(*t.AssigneeID)
	}
	if t.DueDate != nil {
		o.DueDate = oas.NewOptDate(*t.DueDate)
	}
	if t.ProjectName != "" {
		o.ProjectName = oas.NewOptString(t.ProjectName)
	}
	if t.AssigneeName != "" {
		o.AssigneeName = oas.NewOptString(t.AssigneeName)
	}
	return o
}

func taskDetailToOAS(t TaskDetailDTO) oas.TaskDetail {
	o := oas.TaskDetail{
		ID:          t.ID,
		WorkspaceID: t.WorkspaceID,
		ProjectID:   t.ProjectID,
		Title:       t.Title,
		Description: t.Description,
		Status:      oas.TaskStatus(t.Status),
		Priority:    oas.TaskPriority(t.Priority),
	}
	if t.AssigneeID != nil {
		o.AssigneeID = oas.NewOptUUID(*t.AssigneeID)
	}
	if t.DueDate != nil {
		o.DueDate = oas.NewOptDate(*t.DueDate)
	}
	if t.Project != nil {
		o.Project = projectToOAS(*t.Project)
	}
	if t.Assignee != nil {
		o.Assignee = oas.NewOptMember(memberToOAS(*t.Assignee))
	}
	comments := make([]oas.TaskComment, len(t.Comments))
	for i, c := range t.Comments {
		comments[i] = commentToOAS(c)
	}
	o.Comments = comments
	return o
}
