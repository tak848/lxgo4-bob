package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/tak848/lxgo4-bob/internal/oas"
)

type WorkspaceService interface {
	List(ctx context.Context) ([]WorkspaceDTO, error)
	Get(ctx context.Context, id uuid.UUID) (*WorkspaceDTO, error)
	Create(ctx context.Context, name string) (*WorkspaceDTO, error)
	Update(ctx context.Context, id uuid.UUID, name string) (*WorkspaceDTO, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type MemberService interface {
	List(ctx context.Context, wsID uuid.UUID) ([]MemberDTO, error)
	Get(ctx context.Context, wsID, id uuid.UUID) (*MemberDTO, error)
	Create(ctx context.Context, wsID uuid.UUID, name, email, role string) (*MemberDTO, error)
	Update(ctx context.Context, wsID, id uuid.UUID, name, email, role string) (*MemberDTO, error)
	Delete(ctx context.Context, wsID, id uuid.UUID) error
}

type ProjectService interface {
	List(ctx context.Context, wsID uuid.UUID) ([]ProjectWithStatsDTO, error)
	Get(ctx context.Context, wsID, id uuid.UUID) (*ProjectDTO, error)
	Create(ctx context.Context, wsID uuid.UUID, name, description, status string) (*ProjectDTO, error)
	Update(ctx context.Context, wsID, id uuid.UUID, name, description, status string) (*ProjectDTO, error)
	Delete(ctx context.Context, wsID, id uuid.UUID) error
}

type TaskFilter struct {
	Status     *string
	Priority   *string
	AssigneeID *uuid.UUID
	ProjectID  *uuid.UUID
	Limit      int
	Offset     int
}

type TaskService interface {
	List(ctx context.Context, wsID uuid.UUID, filter TaskFilter) ([]TaskWithRelationsDTO, error)
	Get(ctx context.Context, wsID, id uuid.UUID) (*TaskDetailDTO, error)
	Create(ctx context.Context, wsID uuid.UUID, input CreateTaskInput) (*TaskDTO, error)
	Update(ctx context.Context, wsID, id uuid.UUID, input UpdateTaskInput) (*TaskDTO, error)
	Delete(ctx context.Context, wsID, id uuid.UUID) error
}

type CreateTaskInput struct {
	ProjectID   uuid.UUID
	AssigneeID  *uuid.UUID
	Title       string
	Description string
	Status      string
	Priority    string
	DueDate     *time.Time
}

type UpdateTaskInput struct {
	ProjectID   uuid.UUID
	AssigneeID  *uuid.UUID
	Title       string
	Description string
	Status      string
	Priority    string
	DueDate     *time.Time
}

type CommentService interface {
	List(ctx context.Context, wsID, taskID uuid.UUID) ([]TaskCommentDTO, error)
	Create(ctx context.Context, wsID, taskID, authorID uuid.UUID, body string) (*TaskCommentDTO, error)
	Update(ctx context.Context, wsID, taskID, id uuid.UUID, body string) (*TaskCommentDTO, error)
	Delete(ctx context.Context, wsID, taskID, id uuid.UUID) error
}

type ReportService interface {
	ProjectStats(ctx context.Context, wsID uuid.UUID) ([]ProjectTaskStatsDTO, error)
	MemberSummary(ctx context.Context, wsID uuid.UUID) ([]MemberTaskSummaryDTO, error)
	Dashboard(ctx context.Context, wsID uuid.UUID) (*DashboardDTO, error)
}

// DTOs shared between handler and service layers.

type WorkspaceDTO struct {
	ID   uuid.UUID
	Name string
}

type MemberDTO struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Name        string
	Email       string
	Role        string
}

type ProjectDTO struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Name        string
	Description string
	Status      string
}

type ProjectWithStatsDTO struct {
	ProjectDTO
	TotalTasks  int64
	DoneTasks   int64
	ActiveTasks int64
}

type TaskDTO struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	AssigneeID  *uuid.UUID
	Title       string
	Description string
	Status      string
	Priority    string
	DueDate     *time.Time
}

type TaskWithRelationsDTO struct {
	TaskDTO
	ProjectName  string
	AssigneeName string
}

type TaskDetailDTO struct {
	TaskDTO
	Project  *ProjectDTO
	Assignee *MemberDTO
	Comments []TaskCommentDTO
}

type TaskCommentDTO struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	TaskID      uuid.UUID
	AuthorID    uuid.UUID
	Body        string
}

type ProjectTaskStatsDTO struct {
	ID          uuid.UUID
	Name        string
	TotalTasks  int64
	DoneTasks   int64
	ActiveTasks int64
}

type MemberTaskSummaryDTO struct {
	ID             uuid.UUID
	Name           string
	AssignedTasks  int64
	CompletedTasks int64
	OverdueTasks   int64
}

type DashboardDTO struct {
	ProjectCount int64
	MemberCount  int64
	TotalTasks   int64
	DoneTasks    int64
	OverdueTasks int64
}

// Sentinel errors used by service layer.
var (
	ErrNotFound = errors.New("not found")
)

type Handler struct {
	Workspaces WorkspaceService
	Members    MemberService
	Projects   ProjectService
	Tasks      TaskService
	Comments   CommentService
	Reports    ReportService
}

var _ oas.Handler = (*Handler)(nil)

func (h *Handler) NewError(ctx context.Context, err error) *oas.ErrorResponseStatusCode {
	code := http.StatusInternalServerError
	msg := "internal server error"
	switch {
	case errors.Is(err, ErrNotFound):
		code = http.StatusNotFound
		msg = "not found"
	case isConstraintViolation(err):
		code = http.StatusConflict
		msg = "operation conflicts with existing data"
	}
	if code == http.StatusInternalServerError {
		slog.ErrorContext(ctx, "unhandled error", "error", err)
	}
	return &oas.ErrorResponseStatusCode{
		StatusCode: code,
		Response: oas.ErrorResponse{
			Code:    code,
			Message: msg,
		},
	}
}

// isConstraintViolation は PostgreSQL の FK/unique 制約違反エラーを検出する。
// pgx の *pgconn.PgError から SQLSTATE を取り出して判定する。
func isConstraintViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	// 23503: foreign_key_violation, 23505: unique_violation
	return pgErr.Code == "23503" || pgErr.Code == "23505"
}
