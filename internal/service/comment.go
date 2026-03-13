package service

import (
	"context"
	"fmt"

	"github.com/aarondl/opt/omit"
	"github.com/google/uuid"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/tak848/lxgo4-bob-playground/internal/handler"
	"github.com/tak848/lxgo4-bob-playground/internal/infra/db"
	"github.com/tak848/lxgo4-bob-playground/internal/infra/dbgen"
)

type CommentService struct {
	exec bob.Executor
}

var _ handler.CommentService = (*CommentService)(nil)

func NewCommentService(exec bob.Executor) *CommentService {
	return &CommentService{exec: exec}
}

func (s *CommentService) List(ctx context.Context, wsID, taskID uuid.UUID) ([]handler.TaskCommentDTO, error) {
	ctx, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)
	rows, err := dbgen.TaskComments.Query(
		dbgen.SelectWhere.TaskComments.TaskID.EQ(taskID),
		sm.OrderBy(dbgen.TaskComments.Columns.ID).Asc(),
	).All(ctx, exec)
	if err != nil {
		return nil, err
	}
	dtos := make([]handler.TaskCommentDTO, len(rows))
	for i, r := range rows {
		dtos[i] = toCommentDTO(r)
	}
	return dtos, nil
}

func (s *CommentService) Create(ctx context.Context, wsID, taskID, authorID uuid.UUID, body string) (*handler.TaskCommentDTO, error) {
	_, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("uuid.NewV7: %w", err)
	}
	c, err := dbgen.TaskComments.Insert(&dbgen.TaskCommentSetter{
		ID:          omit.From(id),
		WorkspaceID: omit.From(wsID),
		TaskID:      omit.From(taskID),
		AuthorID:    omit.From(authorID),
		Body:        omit.From(body),
	}).One(ctx, exec)
	if err != nil {
		return nil, err
	}
	dto := toCommentDTO(c)
	return &dto, nil
}

func (s *CommentService) Update(ctx context.Context, wsID, taskID, id uuid.UUID, body string) (*handler.TaskCommentDTO, error) {
	ctx, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)
	c, err := dbgen.TaskComments.Query(
		dbgen.SelectWhere.TaskComments.ID.EQ(id),
		dbgen.SelectWhere.TaskComments.TaskID.EQ(taskID),
	).One(ctx, exec)
	if err != nil {
		return nil, wrapNotFound(err)
	}
	if err := c.Update(ctx, exec, &dbgen.TaskCommentSetter{
		Body: omit.From(body),
	}); err != nil {
		return nil, err
	}
	dto := toCommentDTO(c)
	return &dto, nil
}

func (s *CommentService) Delete(ctx context.Context, wsID, taskID, id uuid.UUID) error {
	ctx, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)
	c, err := dbgen.TaskComments.Query(
		dbgen.SelectWhere.TaskComments.ID.EQ(id),
		dbgen.SelectWhere.TaskComments.TaskID.EQ(taskID),
	).One(ctx, exec)
	if err != nil {
		return wrapNotFound(err)
	}
	return c.Delete(ctx, exec)
}

func toCommentDTO(c *dbgen.TaskComment) handler.TaskCommentDTO {
	return handler.TaskCommentDTO{
		ID:          c.ID,
		WorkspaceID: c.WorkspaceID,
		TaskID:      c.TaskID,
		AuthorID:    c.AuthorID,
		Body:        c.Body,
	}
}
