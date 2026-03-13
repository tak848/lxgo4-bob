package handler

import (
	"context"

	"github.com/tak848/lxgo4-bob-playground/internal/oas"
)

func (h *Handler) ListProjects(ctx context.Context, params oas.ListProjectsParams) ([]oas.ProjectWithStats, error) {
	list, err := h.Projects.List(ctx, params.WsId)
	if err != nil {
		return nil, err
	}
	out := make([]oas.ProjectWithStats, len(list))
	for i, p := range list {
		out[i] = projectWithStatsToOAS(p)
	}
	return out, nil
}

func (h *Handler) CreateProject(ctx context.Context, req *oas.CreateProjectRequest, params oas.CreateProjectParams) (*oas.Project, error) {
	p, err := h.Projects.Create(ctx, params.WsId, req.Name, req.Description, string(req.Status))
	if err != nil {
		return nil, err
	}
	o := projectToOAS(*p)
	return &o, nil
}

func (h *Handler) GetProject(ctx context.Context, params oas.GetProjectParams) (*oas.Project, error) {
	p, err := h.Projects.Get(ctx, params.WsId, params.ID)
	if err != nil {
		return nil, err
	}
	o := projectToOAS(*p)
	return &o, nil
}

func (h *Handler) UpdateProject(ctx context.Context, req *oas.UpdateProjectRequest, params oas.UpdateProjectParams) (*oas.Project, error) {
	p, err := h.Projects.Update(ctx, params.WsId, params.ID, req.Name, req.Description, string(req.Status))
	if err != nil {
		return nil, err
	}
	o := projectToOAS(*p)
	return &o, nil
}

func (h *Handler) DeleteProject(ctx context.Context, params oas.DeleteProjectParams) error {
	return h.Projects.Delete(ctx, params.WsId, params.ID)
}

func projectToOAS(p ProjectDTO) oas.Project {
	return oas.Project{
		ID:          p.ID,
		WorkspaceID: p.WorkspaceID,
		Name:        p.Name,
		Description: p.Description,
		Status:      oas.ProjectStatus(p.Status),
	}
}

func projectWithStatsToOAS(p ProjectWithStatsDTO) oas.ProjectWithStats {
	return oas.ProjectWithStats{
		ID:          p.ID,
		WorkspaceID: p.WorkspaceID,
		Name:        p.Name,
		Description: p.Description,
		Status:      oas.ProjectStatus(p.Status),
		TotalTasks:  p.TotalTasks,
		DoneTasks:   p.DoneTasks,
		ActiveTasks: p.ActiveTasks,
	}
}
