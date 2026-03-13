# 4. リレーション (Preload / ThenLoad)

bob のリレーション取得には **Preload**（LEFT JOIN）と **ThenLoad**（別クエリ）の 2 方式がある。
`loaders` プラグインと `joins` プラグインが生成する。

## Preload — LEFT JOIN による to-one 取得

**1 対 1 / 多 対 1** のリレーションに使う。JOIN で 1 回のクエリにまとめる。

```go
// internal/service/task.go:72-77
task, err := dbgen.Tasks.Query(
    dbgen.SelectWhere.Tasks.ID.EQ(id),
    dbgen.Preload.Task.Project(),   // LEFT JOIN projects ON ...
    dbgen.Preload.Task.Member(),    // LEFT JOIN members ON ...
    dbgen.SelectThenLoad.Task.TaskComments(),  // 別クエリ
).One(ctx, exec)
```

### 生成される SQL（Preload 部分）

```sql
SELECT
    tasks.id, tasks.workspace_id, tasks.project_id, ...,
    projects.id AS "Project.id", projects.name AS "Project.name", ...,
    members.id AS "Member.id", members.name AS "Member.name", ...
FROM tasks
LEFT JOIN projects ON tasks.project_id = projects.id AND tasks.workspace_id = projects.workspace_id
LEFT JOIN members ON tasks.assignee_id = members.id AND tasks.workspace_id = members.workspace_id
WHERE tasks.id = $1
```

### 結果の取り出し

```go
task.R.Project   // *dbgen.Project（JOIN で取得済み）
task.R.Member    // *dbgen.Member（JOIN で取得済み）
```

- `.R` フィールドにリレーション先のモデルが格納される
- FK が NULL の場合（assignee_id が NULL 等）、`.R.Member` は `nil` になる

### Preload のバリエーション

```go
// 生成される構造
var Preload = getPreloaders()

type preloaders struct {
    Member      memberPreloader
    Project     projectPreloader
    TaskComment taskCommentPreloader
    Task        taskPreloader
    Workspace   workspacePreloader
}
```

使い方:
```go
dbgen.Preload.Task.Project()    // Task → Project (to-one)
dbgen.Preload.Task.Member()     // Task → Member (to-one, nullable FK)
dbgen.Preload.Task.Workspace()  // Task → Workspace (to-one)
```

## ThenLoad — 別クエリによる to-many 取得

**1 対多** のリレーションに使う。親を取得した後、別の SELECT で子を取得する。

```go
dbgen.SelectThenLoad.Task.TaskComments()
```

### 動作の流れ

1. 親クエリ: `SELECT * FROM tasks WHERE id = $1`
2. 子クエリ: `SELECT * FROM task_comments WHERE task_id IN ($1) AND workspace_id = $2`

2 つの SQL が発行される。N+1 問題は発生しない（IN で一括取得）。

### 結果の取り出し

```go
task.R.TaskComments  // []*dbgen.TaskComment（別クエリで取得済み）
```

### ThenLoad のバリエーション

```go
var SelectThenLoad = getThenLoaders[*dialect.SelectQuery]()

// 使い方
dbgen.SelectThenLoad.Task.TaskComments()      // Task → TaskComments (to-many)
dbgen.SelectThenLoad.Project.Tasks()          // Project → Tasks (to-many)
dbgen.SelectThenLoad.Workspace.Members()      // Workspace → Members (to-many)
```

## Preload vs ThenLoad の使い分け

| 特性 | Preload | ThenLoad |
|------|---------|----------|
| SQL | LEFT JOIN（1クエリ） | 別 SELECT（2クエリ） |
| 適用場面 | to-one (belongs_to / has_one) | to-many (has_many) |
| パフォーマンス | 行数が増えない | 行数の爆発を防ぐ |
| 結果の格納先 | `model.R.Xxx` | `model.R.Xxxs` |

### なぜ to-many に ThenLoad を使うか

Preload（JOIN）で to-many を取ると、親 1 行 × 子 N 行 のカーテシアン積になる。
複数の to-many を JOIN すると行数が爆発する。

ThenLoad は `IN ($1, $2, ...)` で一括取得するので、常に最適な行数。

## 組み合わせの例

```go
// to-one は Preload、to-many は ThenLoad
task, err := dbgen.Tasks.Query(
    dbgen.SelectWhere.Tasks.ID.EQ(id),
    dbgen.Preload.Task.Project(),               // JOIN（to-one）
    dbgen.Preload.Task.Member(),                // JOIN（to-one）
    dbgen.SelectThenLoad.Task.TaskComments(),   // 別クエリ（to-many）
).One(ctx, exec)

// 結果
task.R.Project       // *Project（JOIN 取得）
task.R.Member        // *Member（JOIN 取得、NULL なら nil）
task.R.TaskComments  // []*TaskComment（別クエリ取得）
```

## モデルの .R フィールドと .C フィールド

bob の生成モデルには `.R`（Relations）と `.C`（Counts）の 2 つの特殊フィールドがある。

### .R — リレーション格納フィールド

各モデルに FK ベースで自動生成される。Preload / ThenLoad の結果がここに入る。

```go
// 生成されたモデル（tasks.bob.go）
type Task struct {
    ID          uuid.UUID       `db:"id,pk" json:"id"`
    WorkspaceID uuid.UUID       `db:"workspace_id" json:"workspace_id"`
    // ... 他のカラム

    R taskR `db:"-" json:"-"`  // ← リレーション
    C taskC `db:"-" json:"-"`  // ← カウント
}

// taskR の構造体（FK から自動生成）
type taskR struct {
    TaskComments TaskCommentSlice `json:"TaskComments"` // to-many: task_comments → tasks
    Member       *Member          `json:"Member"`       // to-one: tasks.assignee_id → members.id
    Workspace    *Workspace       `json:"Workspace"`    // to-one: tasks.workspace_id → workspaces.id
    Project      *Project         `json:"Project"`      // to-one: tasks.project_id → projects.id
}
```

全テーブルの `.R` 構造:

| モデル | .R フィールド | 型 | 方向 |
|--------|-------------|---|------|
| Task | `.R.Project` | `*Project` | to-one (FK: project_id) |
| Task | `.R.Member` | `*Member` | to-one (FK: assignee_id, nullable) |
| Task | `.R.Workspace` | `*Workspace` | to-one (FK: workspace_id) |
| Task | `.R.TaskComments` | `TaskCommentSlice` | to-many |
| Project | `.R.Workspace` | `*Workspace` | to-one |
| Project | `.R.Tasks` | `TaskSlice` | to-many |
| Member | `.R.Workspace` | `*Workspace` | to-one |
| Member | `.R.Tasks` | `TaskSlice` | to-many (assignee) |
| Member | `.R.TaskComments` | `TaskCommentSlice` | to-many (author) |
| TaskComment | `.R.Task` | `*Task` | to-one |
| TaskComment | `.R.Member` | `*Member` | to-one (author) |
| TaskComment | `.R.Workspace` | `*Workspace` | to-one |
| Workspace | `.R.Members` | `MemberSlice` | to-many |
| Workspace | `.R.Projects` | `ProjectSlice` | to-many |
| Workspace | `.R.Tasks` | `TaskSlice` | to-many |
| Workspace | `.R.TaskComments` | `TaskCommentSlice` | to-many |

**重要**: `.R` は Preload / ThenLoad しない限り**空（nil / 空スライス）**のまま。
自動的には埋まらない。明示的にロードする必要がある。

### .R の実際の使い方（本プロジェクト）

```go
// List: Preload で JOIN → .R から名前を取り出す
func toTaskWithRelationsDTO(t *dbgen.Task) handler.TaskWithRelationsDTO {
    dto := handler.TaskWithRelationsDTO{TaskDTO: toTaskDTO(t)}
    if t.R.Project != nil {
        dto.ProjectName = t.R.Project.Name  // ← .R.Project を使用
    }
    if t.R.Member != nil {
        dto.AssigneeName = t.R.Member.Name  // ← .R.Member を使用
    }
    return dto
}

// Get: Preload + ThenLoad → .R からフルオブジェクトを取り出す
func toTaskDetailDTO(t *dbgen.Task) handler.TaskDetailDTO {
    dto := handler.TaskDetailDTO{TaskDTO: toTaskDTO(t)}
    if t.R.Project != nil {
        p := toProjectDTO(t.R.Project)
        dto.Project = &p                    // ← .R.Project をそのまま DTO に
    }
    if t.R.Member != nil {
        m := toMemberDTO(t.R.Member)
        dto.Assignee = &m                   // ← .R.Member をそのまま DTO に
    }
    // ThenLoad で取得した to-many
    for _, c := range t.R.TaskComments {    // ← .R.TaskComments を使用
        dto.Comments = append(dto.Comments, toCommentDTO(c))
    }
    return dto
}
```

### .C — カウント格納フィールド

リレーション先のレコード数だけを取得したい場合に使う。全件をロードするより効率的。

```go
// 生成された構造体
type taskC struct {
    TaskComments *int64 `json:"TaskComments"`
}

type memberC struct {
    TaskComments *int64 `json:"TaskComments"`
    Tasks        *int64 `json:"Tasks"`
}
```

### PreloadCount — LEFT JOIN + COUNT

```go
// PreloadCount: JOIN で COUNT を取得
tasks, err := dbgen.Tasks.Query(
    dbgen.PreloadCount.Task.TaskComments(),  // LEFT JOIN + COUNT → task.C.TaskComments
).All(ctx, exec)

for _, t := range tasks {
    if t.C.TaskComments != nil {
        fmt.Printf("Task %s has %d comments\n", t.Title, *t.C.TaskComments)
    }
}
```

生成される SQL:
```sql
SELECT tasks.*, COUNT(task_comments.id) AS "C.TaskComments"
FROM tasks
LEFT JOIN task_comments ON ...
GROUP BY tasks.id
```

### CountThenLoad — 別クエリで COUNT

```go
// CountThenLoad: 別クエリで COUNT を取得
members, err := dbgen.Members.Query(
    dbgen.SelectCountThenLoad.Member.Tasks(),  // 別クエリで COUNT → member.C.Tasks
).All(ctx, exec)

for _, m := range members {
    if m.C.Tasks != nil {
        fmt.Printf("Member %s has %d tasks\n", m.Name, *m.C.Tasks)
    }
}
```

### .C の使い分け

| 方式 | API | SQL | 用途 |
|------|-----|-----|------|
| PreloadCount | `PreloadCount.Task.TaskComments()` | LEFT JOIN + COUNT (1クエリ) | 一覧で件数表示 |
| CountThenLoad | `SelectCountThenLoad.Member.Tasks()` | 別クエリ COUNT (2クエリ) | to-many の件数だけ欲しい場合 |

本プロジェクトではプロジェクトのタスク数取得に queries plugin の `GetProjectTaskStats` を使っているが、
単純な件数取得なら `.C` + PreloadCount の方がシンプル。

## 動的フィルタ + Preload の組み合わせ

本プロジェクトの `List` は、フィルタ条件と Preload を **同じ `[]bob.Mod` スライス** に動的に追加する。
bob がこれを 1 つの SELECT 文にまとめてくれる。

```go
var mods []bob.Mod[*dialect.SelectQuery]

// 動的フィルタ（条件がある場合だけ追加）
if filter.Status != nil {
    mods = append(mods, dbgen.SelectWhere.Tasks.Status.EQ(enums.TaskStatus(*filter.Status)))
}
if filter.ProjectID != nil {
    mods = append(mods, dbgen.SelectWhere.Tasks.ProjectID.EQ(*filter.ProjectID))
}

// Preload も同じ mods に追加
mods = append(mods,
    dbgen.Preload.Task.Project(),   // LEFT JOIN projects
    dbgen.Preload.Task.Member(),    // LEFT JOIN members
    sm.Limit(20), sm.Offset(0),
    sm.OrderBy(dbgen.Tasks.Columns.ID).Desc(),
)

// 全部まとめて渡す → bob が 1 つの SQL にまとめる
tasks, err := dbgen.Tasks.Query(mods...).All(ctx, exec)
```

生成される SQL（フィルタ + Preload + QueryHook が合成される）:
```sql
SELECT tasks.*, projects.*, members.*
FROM tasks
LEFT JOIN projects ON tasks.project_id = projects.id AND tasks.workspace_id = projects.workspace_id
LEFT JOIN members ON tasks.assignee_id = members.id AND tasks.workspace_id = members.workspace_id
WHERE tasks.status = $1              -- ← 動的フィルタ
  AND tasks.project_id = $2          -- ← 動的フィルタ
  AND tasks.workspace_id = $3        -- ← QueryHook が自動注入
ORDER BY tasks.id DESC
LIMIT 20 OFFSET 0
```

フィルタ条件・Preload・ORDER BY・LIMIT/OFFSET・QueryHook がすべて `bob.Mod[Q]` という共通の型で表現されるため、
`append` で自由に組み合わせられる。文字列で SQL を組み立てる必要がない。

## 注意: QueryHooks との連携

Preload で JOIN されるテーブルにも QueryHooks が適用される。
たとえば Task の SelectHook が `WHERE tasks.workspace_id = $1` を追加するのと同時に、
Preload で JOIN された Member や Project にも各テーブルの Hook が効く
（ただし JOIN 条件に workspace_id が含まれているため、実質的にはフィルタ済み）。

ThenLoad の子クエリにも SelectHook が適用される。
子テーブル（task_comments）の SelectHook が `WHERE task_comments.workspace_id = $1` を追加する。
