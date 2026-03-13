# 10. Enum 生成

bob の enums プラグインが PostgreSQL の `CREATE TYPE ... AS ENUM` から Go の型を自動生成する。

## PostgreSQL 側の定義

```sql
-- db/migrations/20260312000001_create_enums.sql
CREATE TYPE member_role AS ENUM ('owner', 'editor', 'viewer');
CREATE TYPE project_status AS ENUM ('active', 'archived');
CREATE TYPE task_status AS ENUM ('todo', 'in_progress', 'done');
CREATE TYPE task_priority AS ENUM ('low', 'medium', 'high', 'urgent');
```

## 生成されるコード

```go
// internal/infra/dbgen/dbenums/enums.bob.go

type MemberRole string

const (
    MemberRoleOwner  MemberRole = "owner"
    MemberRoleEditor MemberRole = "editor"
    MemberRoleViewer MemberRole = "viewer"
)

func AllMemberRole() []MemberRole {
    return []MemberRole{
        MemberRoleOwner,
        MemberRoleEditor,
        MemberRoleViewer,
    }
}

func (e MemberRole) Valid() bool {
    switch e {
    case MemberRoleOwner, MemberRoleEditor, MemberRoleViewer:
        return true
    default:
        return false
    }
}
```

### 自動実装されるインターフェース

各 Enum 型には以下が自動実装される:

- `String() string` — 文字列表現
- `Valid() bool` — 有効な値かチェック
- `MarshalText() / UnmarshalText()` — JSON シリアライゼーション
- `MarshalBinary() / UnmarshalBinary()` — バイナリシリアライゼーション
- `Value() (driver.Value, error)` — SQL ドライバの値変換
- `Scan(value any) error` — SQL ドライバからの読み取り（バリデーション付き）
- `All() []XxxType` — 全値の列挙

## モデルでの使われ方

```go
// 生成されたモデル
type Task struct {
    Status   enums.TaskStatus   `db:"status" json:"status"`
    Priority enums.TaskPriority `db:"priority" json:"priority"`
}
```

- NULLABLE でない ENUM カラムは Go の enum 型がそのまま使われる
- DB から読み取り時に `Scan` でバリデーションが走る → 不正な値はエラー

## Setter での使い方

```go
// Setter
type TaskSetter struct {
    Status   omit.Val[enums.TaskStatus]   `db:"status"`
    Priority omit.Val[enums.TaskPriority] `db:"priority"`
}
```

```go
// INSERT 時
setter.Status = omit.From(enums.TaskStatusTodo)
setter.Priority = omit.From(enums.TaskPriorityMedium)
```

## SelectWhere での使い方

```go
// 型安全フィルタ
dbgen.SelectWhere.Tasks.Status.EQ(enums.TaskStatusDone)
dbgen.SelectWhere.Tasks.Priority.EQ(enums.TaskPriorityHigh)
```

- `Status.EQ()` は `enums.TaskStatus` 型を要求する
- `"done"` のような生文字列を渡すとコンパイルエラーになる

## 文字列との変換

```go
// string → enum（外部入力のバリデーション等）
status := enums.TaskStatus(inputString)
if !status.Valid() {
    return errors.New("invalid status")
}

// enum → string
s := string(task.Status)
```

## Seed / Service での実際の使い方

```go
// internal/service/task.go:97
setter.Status = omit.From(enums.TaskStatus(input.Status))  // string → enum 変換

// cmd/seed/main.go
statuses := []enums.TaskStatus{
    enums.TaskStatusTodo,
    enums.TaskStatusInProgress,
    enums.TaskStatusDone,
}
```
