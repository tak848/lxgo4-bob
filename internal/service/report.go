package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/stephenafamo/bob"

	"github.com/tak848/lxgo4-bob-playground/internal/handler"
	"github.com/tak848/lxgo4-bob-playground/queries"
)

type ReportService struct {
	exec bob.Executor
}

var _ handler.ReportService = (*ReportService)(nil)

func NewReportService(exec bob.Executor) *ReportService {
	return &ReportService{exec: exec}
}

func (s *ReportService) ProjectStats(ctx context.Context, wsID uuid.UUID) ([]handler.ProjectTaskStatsDTO, error) {
	rows, err := queries.GetProjectTaskStats(wsID).All(ctx, s.exec)
	if err != nil {
		return nil, fmt.Errorf("GetProjectTaskStats: %w", err)
	}
	dtos := make([]handler.ProjectTaskStatsDTO, len(rows))
	for i, r := range rows {
		dtos[i] = handler.ProjectTaskStatsDTO{
			ID:          r.ID,
			Name:        r.Name,
			TotalTasks:  r.TotalTasks,
			DoneTasks:   r.DoneTasks,
			ActiveTasks: r.ActiveTasks,
		}
	}
	return dtos, nil
}

func (s *ReportService) MemberSummary(ctx context.Context, wsID uuid.UUID) ([]handler.MemberTaskSummaryDTO, error) {
	rows, err := queries.GetMemberTaskSummary(wsID).All(ctx, s.exec)
	if err != nil {
		return nil, fmt.Errorf("GetMemberTaskSummary: %w", err)
	}
	dtos := make([]handler.MemberTaskSummaryDTO, len(rows))
	for i, r := range rows {
		dtos[i] = handler.MemberTaskSummaryDTO{
			ID:             r.ID,
			Name:           r.Name,
			AssignedTasks:  r.AssignedTasks,
			CompletedTasks: r.CompletedTasks,
			OverdueTasks:   r.OverdueTasks,
		}
	}
	return dtos, nil
}

func (s *ReportService) Dashboard(ctx context.Context, wsID uuid.UUID) (*handler.DashboardDTO, error) {
	rows, err := queries.GetWorkspaceDashboard(wsID).All(ctx, s.exec)
	if err != nil {
		return nil, fmt.Errorf("GetWorkspaceDashboard: %w", err)
	}
	if len(rows) == 0 {
		return &handler.DashboardDTO{}, nil
	}
	return &handler.DashboardDTO{
		ProjectCount: rows[0].ProjectCount,
		MemberCount:  rows[0].MemberCount,
		TotalTasks:   rows[0].TotalTasks,
		DoneTasks:    rows[0].DoneTasks,
		OverdueTasks: rows[0].OverdueTasks,
	}, nil
}
