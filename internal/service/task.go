package service

import (
	"context"
	"fmt"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/google/uuid"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/tak848/lxgo4-bob/internal/handler"
	"github.com/tak848/lxgo4-bob/internal/infra/db"
	"github.com/tak848/lxgo4-bob/internal/infra/dbgen"
	enums "github.com/tak848/lxgo4-bob/internal/infra/dbgen/dbenums"
)

type TaskService struct {
	exec bob.Executor
}

var _ handler.TaskService = (*TaskService)(nil)

func NewTaskService(exec bob.Executor) *TaskService {
	return &TaskService{exec: exec}
}

// List は動的フィルタ + Preload を組み合わせた型安全クエリ構築の実例。
// bob.Mod[*dialect.SelectQuery] のスライスに WHERE 条件・Preload・ORDER BY・LIMIT/OFFSET を
// 動的に追加し、最後に .Query(mods...) でまとめて渡す。
// フィルタ条件は SelectWhere.Tasks.Status.EQ() 等の型安全メソッドを使い、
// 文字列ではなく enums.TaskStatus 型を要求するためコンパイル時に型ミスを検出できる。
func (s *TaskService) List(ctx context.Context, wsID uuid.UUID, filter handler.TaskFilter) ([]handler.TaskWithRelationsDTO, error) {
	ctx, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)

	// 動的にクエリ条件を組み立てる。条件がなければ WHERE なし、あれば AND で結合。
	var mods []bob.Mod[*dialect.SelectQuery]

	// 型安全フィルタ: SelectWhere.Tasks.Status.EQ() は enums.TaskStatus 型を要求する。
	// string を渡すとコンパイルエラーになるため、不正な値がランタイムに到達しない。
	if filter.Status != nil {
		mods = append(mods, dbgen.SelectWhere.Tasks.Status.EQ(enums.TaskStatus(*filter.Status)))
	}
	if filter.Priority != nil {
		mods = append(mods, dbgen.SelectWhere.Tasks.Priority.EQ(enums.TaskPriority(*filter.Priority)))
	}
	if filter.AssigneeID != nil {
		mods = append(mods, dbgen.SelectWhere.Tasks.AssigneeID.EQ(*filter.AssigneeID))
	}
	if filter.ProjectID != nil {
		mods = append(mods, dbgen.SelectWhere.Tasks.ProjectID.EQ(*filter.ProjectID))
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	// Preload（LEFT JOIN で to-one リレーション取得）をフィルタ条件と同じ mods に追加。
	// bob はこれらを 1 つの SELECT 文にまとめる:
	//   SELECT tasks.*, projects.*, members.*
	//   FROM tasks
	//   LEFT JOIN projects ON ...
	//   LEFT JOIN members ON ...
	//   WHERE tasks.status = $1 AND tasks.workspace_id = $2  ← QueryHook が追加
	//   ORDER BY tasks.id DESC LIMIT 20 OFFSET 0
	mods = append(mods,
		dbgen.Preload.Task.Project(),
		dbgen.Preload.Task.Member(),
		sm.Limit(limit),
		sm.Offset(filter.Offset),
		sm.OrderBy(dbgen.Tasks.Columns.ID).Desc(),
	)

	tasks, err := dbgen.Tasks.Query(mods...).All(ctx, exec)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	dtos := make([]handler.TaskWithRelationsDTO, len(tasks))
	for i, t := range tasks {
		dtos[i] = toTaskWithRelationsDTO(t)
	}
	return dtos, nil
}

// Get は Preload (to-one, LEFT JOIN) + ThenLoad (to-many, 別クエリ) の組み合わせ例。
// .R フィールドにリレーション先のモデルが格納される。
func (s *TaskService) Get(ctx context.Context, wsID, id uuid.UUID) (*handler.TaskDetailDTO, error) {
	ctx, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)
	task, err := dbgen.Tasks.Query(
		dbgen.SelectWhere.Tasks.ID.EQ(id),
		// Preload: LEFT JOIN で to-one リレーションを 1 クエリで取得
		dbgen.Preload.Task.Project(), // → task.R.Project に格納
		dbgen.Preload.Task.Member(),  // → task.R.Member に格納（nullable FK なので nil の場合あり）
		// ThenLoad: 別クエリで to-many リレーションを取得（N+1 なし、IN で一括）
		dbgen.SelectThenLoad.Task.TaskComments(), // → task.R.TaskComments に格納
	).One(ctx, exec)
	if err != nil {
		return nil, wrapNotFound(err)
	}
	dto := toTaskDetailDTO(task)
	return &dto, nil
}

func (s *TaskService) Create(ctx context.Context, wsID uuid.UUID, input handler.CreateTaskInput) (*handler.TaskDTO, error) {
	_, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("uuid.NewV7: %w", err)
	}
	setter := &dbgen.TaskSetter{
		ID:          omit.From(id),
		WorkspaceID: omit.From(wsID),
		ProjectID:   omit.From(input.ProjectID),
		Title:       omit.From(input.Title),
		Description: omit.From(input.Description),
		Status:      omit.From(enums.TaskStatus(input.Status)),
		Priority:    omit.From(enums.TaskPriority(input.Priority)),
	}
	if input.AssigneeID != nil {
		setter.AssigneeID = omitnull.From(*input.AssigneeID)
	}
	if input.DueDate != nil {
		setter.DueDate = omitnull.From(*input.DueDate)
	}
	task, err := dbgen.Tasks.Insert(setter).One(ctx, exec)
	if err != nil {
		return nil, err
	}
	dto := toTaskDTO(task)
	return &dto, nil
}

func (s *TaskService) Update(ctx context.Context, wsID, id uuid.UUID, input handler.UpdateTaskInput) (*handler.TaskDTO, error) {
	ctx, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)
	task, err := dbgen.FindTask(ctx, exec, id)
	if err != nil {
		return nil, wrapNotFound(err)
	}
	setter := &dbgen.TaskSetter{
		ProjectID:   omit.From(input.ProjectID),
		Title:       omit.From(input.Title),
		Description: omit.From(input.Description),
		Status:      omit.From(enums.TaskStatus(input.Status)),
		Priority:    omit.From(enums.TaskPriority(input.Priority)),
	}
	if input.AssigneeID != nil {
		setter.AssigneeID = omitnull.From(*input.AssigneeID)
	} else {
		var nullAssignee omitnull.Val[uuid.UUID]
		nullAssignee.Null()
		setter.AssigneeID = nullAssignee
	}
	if input.DueDate != nil {
		setter.DueDate = omitnull.From(*input.DueDate)
	} else {
		var nullDue omitnull.Val[time.Time]
		nullDue.Null()
		setter.DueDate = nullDue
	}
	if err := task.Update(ctx, exec, setter); err != nil {
		return nil, err
	}
	dto := toTaskDTO(task)
	return &dto, nil
}

func (s *TaskService) Delete(ctx context.Context, wsID, id uuid.UUID) error {
	ctx, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)
	task, err := dbgen.FindTask(ctx, exec, id)
	if err != nil {
		return wrapNotFound(err)
	}
	return task.Delete(ctx, exec)
}

func toTaskDTO(t *dbgen.Task) handler.TaskDTO {
	dto := handler.TaskDTO{
		ID:          t.ID,
		WorkspaceID: t.WorkspaceID,
		ProjectID:   t.ProjectID,
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),
		Priority:    string(t.Priority),
	}
	if t.AssigneeID.IsValue() {
		v := t.AssigneeID.MustGet()
		dto.AssigneeID = &v
	}
	if t.DueDate.IsValue() {
		v := t.DueDate.MustGet()
		dto.DueDate = &v
	}
	return dto
}

// toTaskWithRelationsDTO は Preload で取得した .R を使って
// project_name / assignee_name を埋める。List 用。
func toTaskWithRelationsDTO(t *dbgen.Task) handler.TaskWithRelationsDTO {
	dto := handler.TaskWithRelationsDTO{
		TaskDTO: toTaskDTO(t),
	}
	// .R.Project は Preload.Task.Project() で LEFT JOIN 取得済み
	if t.R.Project != nil {
		dto.ProjectName = t.R.Project.Name
	}
	// .R.Member は Preload.Task.Member() で LEFT JOIN 取得済み
	// assignee_id が NULL の場合は .R.Member が nil
	if t.R.Member != nil {
		dto.AssigneeName = t.R.Member.Name
	}
	return dto
}

// toTaskDetailDTO は Preload + ThenLoad で取得した .R を使って
// project / assignee / comments を埋める。Get (詳細) 用。
func toTaskDetailDTO(t *dbgen.Task) handler.TaskDetailDTO {
	dto := handler.TaskDetailDTO{
		TaskDTO: toTaskDTO(t),
	}
	// .R.Project は Preload で LEFT JOIN 取得済み
	if t.R.Project != nil {
		p := toProjectDTO(t.R.Project)
		dto.Project = &p
	}
	// .R.Member は Preload で LEFT JOIN 取得済み（nullable FK）
	if t.R.Member != nil {
		m := toMemberDTO(t.R.Member)
		dto.Assignee = &m
	}
	// .R.TaskComments は ThenLoad で別クエリ取得済み
	dto.Comments = make([]handler.TaskCommentDTO, len(t.R.TaskComments))
	for i, c := range t.R.TaskComments {
		dto.Comments[i] = handler.TaskCommentDTO{
			ID:          c.ID,
			WorkspaceID: c.WorkspaceID,
			TaskID:      c.TaskID,
			AuthorID:    c.AuthorID,
			Body:        c.Body,
		}
	}
	return dto
}
