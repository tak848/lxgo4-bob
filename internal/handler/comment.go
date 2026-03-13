package handler

import (
	"context"

	"github.com/tak848/lxgo4-bob/internal/oas"
)

func (h *Handler) ListTaskComments(ctx context.Context, params oas.ListTaskCommentsParams) ([]oas.TaskComment, error) {
	list, err := h.Comments.List(ctx, params.WsId, params.TaskId)
	if err != nil {
		return nil, err
	}
	out := make([]oas.TaskComment, len(list))
	for i, c := range list {
		out[i] = commentToOAS(c)
	}
	return out, nil
}

func (h *Handler) CreateTaskComment(ctx context.Context, req *oas.CreateTaskCommentRequest, params oas.CreateTaskCommentParams) (*oas.TaskComment, error) {
	c, err := h.Comments.Create(ctx, params.WsId, params.TaskId, req.AuthorID, req.Body)
	if err != nil {
		return nil, err
	}
	o := commentToOAS(*c)
	return &o, nil
}

func (h *Handler) UpdateTaskComment(ctx context.Context, req *oas.UpdateTaskCommentRequest, params oas.UpdateTaskCommentParams) (*oas.TaskComment, error) {
	c, err := h.Comments.Update(ctx, params.WsId, params.TaskId, params.ID, req.Body)
	if err != nil {
		return nil, err
	}
	o := commentToOAS(*c)
	return &o, nil
}

func (h *Handler) DeleteTaskComment(ctx context.Context, params oas.DeleteTaskCommentParams) error {
	return h.Comments.Delete(ctx, params.WsId, params.TaskId, params.ID)
}

func commentToOAS(c TaskCommentDTO) oas.TaskComment {
	return oas.TaskComment{
		ID:          c.ID,
		WorkspaceID: c.WorkspaceID,
		TaskID:      c.TaskID,
		AuthorID:    c.AuthorID,
		Body:        c.Body,
	}
}
