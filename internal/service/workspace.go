package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/aarondl/opt/omit"
	"github.com/google/uuid"
	"github.com/stephenafamo/bob"

	"github.com/tak848/lxgo4-bob-playground/internal/handler"
	"github.com/tak848/lxgo4-bob-playground/internal/infra/db"
	"github.com/tak848/lxgo4-bob-playground/internal/infra/dbgen"
)

type WorkspaceService struct {
	exec bob.Executor
}

var _ handler.WorkspaceService = (*WorkspaceService)(nil)

func NewWorkspaceService(exec bob.Executor) *WorkspaceService {
	return &WorkspaceService{exec: exec}
}

func (s *WorkspaceService) List(ctx context.Context) ([]handler.WorkspaceDTO, error) {
	exec := db.GlobalExec(s.exec)
	rows, err := dbgen.Workspaces.Query().All(ctx, exec)
	if err != nil {
		return nil, err
	}
	dtos := make([]handler.WorkspaceDTO, len(rows))
	for i, r := range rows {
		dtos[i] = toWorkspaceDTO(r)
	}
	return dtos, nil
}

func (s *WorkspaceService) Get(ctx context.Context, id uuid.UUID) (*handler.WorkspaceDTO, error) {
	exec := db.GlobalExec(s.exec)
	ws, err := dbgen.FindWorkspace(ctx, exec, id)
	if err != nil {
		return nil, wrapNotFound(err)
	}
	dto := toWorkspaceDTO(ws)
	return &dto, nil
}

func (s *WorkspaceService) Create(ctx context.Context, name string) (*handler.WorkspaceDTO, error) {
	exec := db.GlobalExec(s.exec)
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("uuid.NewV7: %w", err)
	}
	ws, err := dbgen.Workspaces.Insert(&dbgen.WorkspaceSetter{
		ID:   omit.From(id),
		Name: omit.From(name),
	}).One(ctx, exec)
	if err != nil {
		return nil, err
	}
	dto := toWorkspaceDTO(ws)
	return &dto, nil
}

func (s *WorkspaceService) Update(ctx context.Context, id uuid.UUID, name string) (*handler.WorkspaceDTO, error) {
	exec := db.GlobalExec(s.exec)
	ws, err := dbgen.FindWorkspace(ctx, exec, id)
	if err != nil {
		return nil, wrapNotFound(err)
	}
	if err := ws.Update(ctx, exec, &dbgen.WorkspaceSetter{
		Name: omit.From(name),
	}); err != nil {
		return nil, err
	}
	dto := toWorkspaceDTO(ws)
	return &dto, nil
}

func (s *WorkspaceService) Delete(ctx context.Context, id uuid.UUID) error {
	exec := db.GlobalExec(s.exec)
	ws, err := dbgen.FindWorkspace(ctx, exec, id)
	if err != nil {
		return wrapNotFound(err)
	}
	return ws.Delete(ctx, exec)
}

func toWorkspaceDTO(ws *dbgen.Workspace) handler.WorkspaceDTO {
	return handler.WorkspaceDTO{
		ID:   ws.ID,
		Name: ws.Name,
	}
}

func wrapNotFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return handler.ErrNotFound
	}
	return err
}
