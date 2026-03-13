# 7. Nullable カラムの扱い (omit / omitnull / null)

bob は `aarondl/opt` パッケージの 3 つの型を使い分ける。
SQL の NULL と Go のゼロ値を正しく区別するための仕組み。

## 3 つの型の役割

### `null.Val[T]` — モデルの NULLABLE フィールド

```go
// 生成されたモデル
type Task struct {
    AssigneeID null.Val[uuid.UUID] `db:"assignee_id"`
    DueDate    null.Val[time.Time] `db:"due_date"`
}
```

- DB の NULLABLE カラムに対応
- 状態: **値あり** or **NULL**

```go
// 値の取り出し
if task.AssigneeID.IsValue() {
    id := task.AssigneeID.MustGet()  // uuid.UUID
}

// NULL チェック
if task.AssigneeID.IsNull() {
    // assignee_id IS NULL
}
```

### `omit.Val[T]` — Setter の NOT NULL フィールド

```go
// 生成された Setter
type TaskSetter struct {
    ID          omit.Val[uuid.UUID]        `db:"id,pk"`
    WorkspaceID omit.Val[uuid.UUID]        `db:"workspace_id"`
    Title       omit.Val[string]           `db:"title"`
}
```

- INSERT / UPDATE で使う
- 状態: **値あり** or **未セット (omit)**
- 未セットのフィールドは SQL に含まれない

```go
// 値のセット
setter.Title = omit.From("New Title")   // SET title = 'New Title'

// 未セット（ゼロ値のまま）
// → INSERT/UPDATE の SET 句に含まれない（DB デフォルト値が使われる）
```

### `omitnull.Val[T]` — Setter の NULLABLE フィールド

```go
// 生成された Setter
type TaskSetter struct {
    AssigneeID omitnull.Val[uuid.UUID] `db:"assignee_id"`
    DueDate    omitnull.Val[time.Time] `db:"due_date"`
}
```

- 3 つの状態を持つ: **値あり** / **NULL** / **未セット (omit)**

```go
// 値をセット → SET assignee_id = 'xxx'
setter.AssigneeID = omitnull.From(someUUID)

// NULL をセット → SET assignee_id = NULL
var nullVal omitnull.Val[uuid.UUID]
nullVal.Null()
setter.AssigneeID = nullVal

// 未セット（ゼロ値のまま）
// → SET 句に含まれない
```

## 状態遷移図

```
omit.Val[T]:
    ┌──────────┐     omit.From(v)     ┌──────────┐
    │  未セット  │ ──────────────────→ │   値あり   │
    │  (Omit)  │                      │  (Value)  │
    └──────────┘                      └──────────┘

omitnull.Val[T]:
    ┌──────────┐     omitnull.From(v)  ┌──────────┐
    │  未セット  │ ──────────────────→ │   値あり   │
    │  (Omit)  │                      │  (Value)  │
    └──────────┘                      └──────────┘
         │            .Null()          ┌──────────┐
         └────────────────────────→   │   NULL    │
                                      └──────────┘

null.Val[T]:
    ┌──────────┐    null.From(v)      ┌──────────┐
    │   NULL   │ ──────────────────→ │   値あり   │
    └──────────┘                      └──────────┘
```

## 実際の使用パターン

### INSERT: AssigneeID がある場合のみセット

```go
// internal/service/task.go:100-101
setter := &dbgen.TaskSetter{
    ID:          omit.From(id),
    WorkspaceID: omit.From(wsID),
    // ...
}
if input.AssigneeID != nil {
    setter.AssigneeID = omitnull.From(*input.AssigneeID)
}
// AssigneeID が nil なら、omitnull はゼロ値（未セット）のまま
// → INSERT 文に assignee_id が含まれない → DB デフォルト（NULL）
```

### UPDATE: AssigneeID を NULL に戻す

```go
// internal/service/task.go:127-133
if input.AssigneeID != nil {
    setter.AssigneeID = omitnull.From(*input.AssigneeID)
} else {
    var nullAssignee omitnull.Val[uuid.UUID]
    nullAssignee.Null()
    setter.AssigneeID = nullAssignee
    // → UPDATE SET assignee_id = NULL
}
```

### SELECT: null.Val から値を取り出す

```go
// internal/service/task.go:167-174
dto := handler.TaskDTO{...}
if t.AssigneeID.IsValue() {
    v := t.AssigneeID.MustGet()
    dto.AssigneeID = &v
}
if t.DueDate.IsValue() {
    v := t.DueDate.MustGet()
    dto.DueDate = &v
}
```

## まとめ

| 型 | 用途 | 状態 | コンテキスト |
|---|---|---|---|
| `null.Val[T]` | モデルのフィールド | 値 / NULL | SELECT の結果を表現 |
| `omit.Val[T]` | Setter の NOT NULL フィールド | 値 / 未セット | INSERT/UPDATE で使う |
| `omitnull.Val[T]` | Setter の NULLABLE フィールド | 値 / NULL / 未セット | INSERT/UPDATE で使う |

- `omit` = 「SQL に含めない」
- `null` = 「NULL をセットする」
- 2 つの概念は直交しており、`omitnull` は両方を兼ねる
