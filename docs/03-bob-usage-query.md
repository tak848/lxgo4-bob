# 3. 型安全フィルタ・ページネーション

bob の `where` プラグインと `sm`（select mods）パッケージによる型安全なクエリ構築。

## SelectWhere — 型安全な WHERE 条件

bobgen の `where` プラグインが生成する。テーブル・カラムごとにメソッドが用意される。

```go
// 生成される変数
var SelectWhere = Where[*dialect.SelectQuery]()

// 使い方
dbgen.SelectWhere.Tasks.Status.EQ(enums.TaskStatusDone)
dbgen.SelectWhere.Tasks.Priority.EQ(enums.TaskPriorityHigh)
dbgen.SelectWhere.Tasks.AssigneeID.EQ(someUUID)
dbgen.SelectWhere.Tasks.ProjectID.EQ(projectUUID)
```

### 実際のコード（task.go のフィルタ）

```go
// internal/service/task.go:34-46
var mods []bob.Mod[*dialect.SelectQuery]

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
```

### 型安全性のポイント

- `Status.EQ()` は `enums.TaskStatus` 型を要求 → 文字列を直接渡せない
- `AssigneeID.EQ()` は `uuid.UUID` 型を要求 → int を渡せない
- コンパイル時にフィルタ条件の型ミスが検出される

### 使える比較メソッド

```go
SelectWhere.Tasks.Status.EQ(v)   // = v
SelectWhere.Tasks.Status.NE(v)   // != v
SelectWhere.Tasks.Status.In(v1, v2)  // IN (v1, v2)
SelectWhere.Tasks.Status.NotIn(v1, v2)
// NULLABLE カラムの場合
SelectWhere.Tasks.AssigneeID.IsNull()
SelectWhere.Tasks.AssigneeID.IsNotNull()
```

## `bob.Mod[Q]` — クエリ修飾子

bob のクエリ修飾子は `bob.Mod[Q]` インターフェースを満たす。
WHERE 条件、ORDER BY、LIMIT/OFFSET はすべて同じ型で、クエリビルダーの引数に渡せる。

```go
// 型定義
type Mod[T any] interface {
    Apply(T)
}
```

## sm パッケージ — SELECT Mods

`github.com/stephenafamo/bob/dialect/psql/sm` パッケージ。

### LIMIT / OFFSET（ページネーション）

```go
// internal/service/task.go:52-54
mods = append(mods,
    sm.Limit(limit),     // LIMIT 20
    sm.Offset(offset),   // OFFSET 0
)
```

### ORDER BY

```go
// internal/service/task.go:55
sm.OrderBy(dbgen.Tasks.Columns.ID).Desc()  // ORDER BY tasks.id DESC
```

```go
// internal/service/comment.go:31
sm.OrderBy(dbgen.TaskComments.Columns.ID).Asc()  // ORDER BY task_comments.id ASC
```

- `dbgen.Tasks.Columns.ID` は `psql.Expression` 型。テーブル名が修飾される
- `.Desc()` / `.Asc()` でソート方向を指定

## クエリの組み立て

すべての Mod を可変長引数で `.Query()` に渡す:

```go
// internal/service/task.go:58
tasks, err := dbgen.Tasks.Query(mods...).All(ctx, exec)
```

生成される SQL:
```sql
SELECT tasks.id, tasks.workspace_id, ... FROM tasks
WHERE tasks.status = $1 AND tasks.priority = $2
ORDER BY tasks.id DESC
LIMIT 20 OFFSET 0
```

※ さらに QueryHooks が自動で `WHERE tasks.workspace_id = $N` を追加する（後述）

## 条件の動的組み立てパターン

```go
var mods []bob.Mod[*dialect.SelectQuery]

// 条件がある場合だけ追加
if filter.Status != nil {
    mods = append(mods, dbgen.SelectWhere.Tasks.Status.EQ(...))
}

// 常に追加
mods = append(mods, sm.Limit(20), sm.Offset(0))

// まとめて渡す
result, err := dbgen.Tasks.Query(mods...).All(ctx, exec)
```

これにより「条件がなければ WHERE なし、あれば AND で結合」が自然に実現される。
`if` 分岐で SQL 文字列を組み立てる必要がない。
