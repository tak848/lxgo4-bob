package handler

import (
	"context"

	"github.com/tak848/lxgo4-bob/internal/oas"
)

func (h *Handler) GetProjectStats(ctx context.Context, params oas.GetProjectStatsParams) ([]oas.ProjectTaskStats, error) {
	list, err := h.Reports.ProjectStats(ctx, params.WsId)
	if err != nil {
		return nil, err
	}
	out := make([]oas.ProjectTaskStats, len(list))
	for i, s := range list {
		out[i] = oas.ProjectTaskStats{
			ID:          s.ID,
			Name:        s.Name,
			TotalTasks:  s.TotalTasks,
			DoneTasks:   s.DoneTasks,
			ActiveTasks: s.ActiveTasks,
		}
	}
	return out, nil
}

func (h *Handler) GetMemberSummary(ctx context.Context, params oas.GetMemberSummaryParams) ([]oas.MemberTaskSummary, error) {
	list, err := h.Reports.MemberSummary(ctx, params.WsId)
	if err != nil {
		return nil, err
	}
	out := make([]oas.MemberTaskSummary, len(list))
	for i, s := range list {
		out[i] = oas.MemberTaskSummary{
			ID:             s.ID,
			Name:           s.Name,
			AssignedTasks:  s.AssignedTasks,
			CompletedTasks: s.CompletedTasks,
			OverdueTasks:   s.OverdueTasks,
		}
	}
	return out, nil
}

func (h *Handler) GetDashboard(ctx context.Context, params oas.GetDashboardParams) (*oas.DashboardData, error) {
	d, err := h.Reports.Dashboard(ctx, params.WsId)
	if err != nil {
		return nil, err
	}
	return &oas.DashboardData{
		ProjectCount: d.ProjectCount,
		MemberCount:  d.MemberCount,
		TotalTasks:   d.TotalTasks,
		DoneTasks:    d.DoneTasks,
		OverdueTasks: d.OverdueTasks,
	}, nil
}
