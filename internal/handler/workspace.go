package handler

import (
	"context"

	"github.com/tak848/lxgo4-bob-playground/internal/oas"
)

func (h *Handler) ListWorkspaces(ctx context.Context) ([]oas.Workspace, error) {
	list, err := h.Workspaces.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]oas.Workspace, len(list))
	for i, w := range list {
		out[i] = oas.Workspace{ID: w.ID, Name: w.Name}
	}
	return out, nil
}

func (h *Handler) CreateWorkspace(ctx context.Context, req *oas.CreateWorkspaceRequest) (*oas.Workspace, error) {
	w, err := h.Workspaces.Create(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	return &oas.Workspace{ID: w.ID, Name: w.Name}, nil
}

func (h *Handler) GetWorkspace(ctx context.Context, params oas.GetWorkspaceParams) (*oas.Workspace, error) {
	w, err := h.Workspaces.Get(ctx, params.WsId)
	if err != nil {
		return nil, err
	}
	return &oas.Workspace{ID: w.ID, Name: w.Name}, nil
}

func (h *Handler) UpdateWorkspace(ctx context.Context, req *oas.UpdateWorkspaceRequest, params oas.UpdateWorkspaceParams) (*oas.Workspace, error) {
	w, err := h.Workspaces.Update(ctx, params.WsId, req.Name)
	if err != nil {
		return nil, err
	}
	return &oas.Workspace{ID: w.ID, Name: w.Name}, nil
}

func (h *Handler) DeleteWorkspace(ctx context.Context, params oas.DeleteWorkspaceParams) error {
	return h.Workspaces.Delete(ctx, params.WsId)
}
