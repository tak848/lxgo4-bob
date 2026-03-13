package service

import (
	"context"
	"fmt"

	"github.com/aarondl/opt/omit"
	"github.com/google/uuid"
	"github.com/stephenafamo/bob"

	"github.com/tak848/lxgo4-bob/internal/handler"
	"github.com/tak848/lxgo4-bob/internal/infra/db"
	"github.com/tak848/lxgo4-bob/internal/infra/dbgen"
	enums "github.com/tak848/lxgo4-bob/internal/infra/dbgen/dbenums"
	"github.com/tak848/lxgo4-bob/queries"
)

type ProjectService struct {
	exec bob.Executor
}

var _ handler.ProjectService = (*ProjectService)(nil)

func NewProjectService(exec bob.Executor) *ProjectService {
	return &ProjectService{exec: exec}
}

// List はプロジェクト一覧 + queries plugin (GetProjectTaskStats) のタスク統計をマージして返す。
// bob のクエリビルダーで projects を取得し、queries plugin で集計クエリを実行し、
// Go 側で map を使ってマージする。
func (s *ProjectService) List(ctx context.Context, wsID uuid.UUID) ([]handler.ProjectWithStatsDTO, error) {
	ctx, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)

	// bob クエリビルダーでプロジェクト一覧取得（QueryHooks が workspace_id フィルタを自動注入）
	rows, err := dbgen.Projects.Query().All(ctx, exec)
	if err != nil {
		return nil, err
	}

	// queries plugin でプロジェクト別タスク統計を取得（手書き SQL、workspace_id は明示指定）
	stats, err := queries.GetProjectTaskStats(wsID).All(ctx, s.exec)
	if err != nil {
		return nil, fmt.Errorf("GetProjectTaskStats: %w", err)
	}

	// project_id → stats の map を作成してマージ
	statsMap := make(map[uuid.UUID]queries.GetProjectTaskStatsRow, len(stats))
	for _, st := range stats {
		statsMap[st.ID] = st
	}

	dtos := make([]handler.ProjectWithStatsDTO, len(rows))
	for i, r := range rows {
		dto := handler.ProjectWithStatsDTO{
			ProjectDTO: toProjectDTO(r),
		}
		if st, ok := statsMap[r.ID]; ok {
			dto.TotalTasks = st.TotalTasks
			dto.DoneTasks = st.DoneTasks
			dto.ActiveTasks = st.ActiveTasks
		}
		dtos[i] = dto
	}
	return dtos, nil
}

func (s *ProjectService) Get(ctx context.Context, wsID, id uuid.UUID) (*handler.ProjectDTO, error) {
	ctx, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)
	p, err := dbgen.FindProject(ctx, exec, id)
	if err != nil {
		return nil, wrapNotFound(err)
	}
	dto := toProjectDTO(p)
	return &dto, nil
}

func (s *ProjectService) Create(ctx context.Context, wsID uuid.UUID, name, description, status string) (*handler.ProjectDTO, error) {
	_, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("uuid.NewV7: %w", err)
	}
	p, err := dbgen.Projects.Insert(&dbgen.ProjectSetter{
		ID:          omit.From(id),
		WorkspaceID: omit.From(wsID),
		Name:        omit.From(name),
		Description: omit.From(description),
		Status:      omit.From(enums.ProjectStatus(status)),
	}).One(ctx, exec)
	if err != nil {
		return nil, err
	}
	dto := toProjectDTO(p)
	return &dto, nil
}

func (s *ProjectService) Update(ctx context.Context, wsID, id uuid.UUID, name, description, status string) (*handler.ProjectDTO, error) {
	ctx, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)
	p, err := dbgen.FindProject(ctx, exec, id)
	if err != nil {
		return nil, wrapNotFound(err)
	}
	if err := p.Update(ctx, exec, &dbgen.ProjectSetter{
		Name:        omit.From(name),
		Description: omit.From(description),
		Status:      omit.From(enums.ProjectStatus(status)),
	}); err != nil {
		return nil, err
	}
	dto := toProjectDTO(p)
	return &dto, nil
}

func (s *ProjectService) Delete(ctx context.Context, wsID, id uuid.UUID) error {
	ctx, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)
	p, err := dbgen.FindProject(ctx, exec, id)
	if err != nil {
		return wrapNotFound(err)
	}
	return p.Delete(ctx, exec)
}

func toProjectDTO(p *dbgen.Project) handler.ProjectDTO {
	return handler.ProjectDTO{
		ID:          p.ID,
		WorkspaceID: p.WorkspaceID,
		Name:        p.Name,
		Description: p.Description,
		Status:      string(p.Status),
	}
}
