# 2. CRUD 操作

bob の生成コードを使った基本的な CRUD 操作パターン。

## 全件取得 (SELECT *)

```go
// internal/service/workspace.go:30
rows, err := dbgen.Workspaces.Query().All(ctx, exec)
```

- `dbgen.Workspaces` はテーブル定義オブジェクト（`psql.NewTablex[*Workspace, WorkspaceSlice, *WorkspaceSetter]`）
- `.Query()` で SELECT クエリビルダーを返す
- `.All(ctx, exec)` で全件取得 → `[]*Workspace` を返す

## 1件取得 (FindXxx)

```go
// internal/service/workspace.go:43
ws, err := dbgen.FindWorkspace(ctx, exec, id)
```

- PK 検索のヘルパー関数。生成時にテーブルごとに `FindXxx` が作られる
- 見つからない場合は `sql.ErrNoRows` を返す

### Query + One で検索する場合

```go
// internal/service/comment.go:65-67
c, err := dbgen.TaskComments.Query(
    dbgen.SelectWhere.TaskComments.ID.EQ(id),
).One(ctx, exec)
```

- `.One(ctx, exec)` は 1 件だけ取得。0 件なら `sql.ErrNoRows`

## INSERT

```go
// internal/service/workspace.go:57-60
ws, err := dbgen.Workspaces.Insert(&dbgen.WorkspaceSetter{
    ID:   omit.From(id),
    Name: omit.From(name),
}).One(ctx, exec)
```

### Setter 構造体

```go
type WorkspaceSetter struct {
    ID   omit.Val[uuid.UUID] `db:"id,pk" json:"id"`
    Name omit.Val[string]    `db:"name" json:"name"`
}
```

- 全フィールドが `omit.Val[T]` 型
- `omit.From(v)` で値をセット → INSERT の SET 句に含まれる
- 未セット（ゼロ値）のフィールドは INSERT 文に含まれない → DB のデフォルト値が使われる
- `.One(ctx, exec)` で INSERT + RETURNING * の結果を返す

### NULLABLE カラムの INSERT

```go
// internal/service/task.go:100-101
if input.AssigneeID != nil {
    setter.AssigneeID = omitnull.From(*input.AssigneeID)
}
```

- NULLABLE カラムの Setter は `omitnull.Val[T]` 型
- `omitnull.From(v)` → 値をセット（NOT NULL として INSERT）
- 未セットなら INSERT 文に含まれない（DB デフォルト）
- 詳細は [Nullable カラムの扱い](./bob-usage-nullable.md) 参照

## UPDATE

```go
// internal/service/workspace.go:74-76
if err := ws.Update(ctx, exec, &dbgen.WorkspaceSetter{
    Name: omit.From(name),
}); err != nil {
    return nil, err
}
```

- まず `FindXxx` や `.Query().One()` でモデルを取得
- そのモデルの `.Update(ctx, exec, setter)` を呼ぶ
- Setter で指定したフィールドだけ UPDATE される
- UPDATE 後、モデルのフィールドが自動的に更新される（in-place）

### NULLABLE カラムを NULL に戻す

```go
// internal/service/task.go:130-132
var nullAssignee omitnull.Val[uuid.UUID]
nullAssignee.Null()
setter.AssigneeID = nullAssignee
```

- `omitnull.Val[T]` のゼロ値は「未セット (omit)」
- `.Null()` を呼ぶと「NULL をセットする」状態になる
- これにより `UPDATE SET assignee_id = NULL` が発行される

## DELETE

```go
// internal/service/workspace.go:89
return ws.Delete(ctx, exec)
```

- モデルを取得してから `.Delete(ctx, exec)` を呼ぶ
- PK で WHERE 条件が自動生成される

## まとめ: CRUD のパターン

| 操作 | コード | SQL |
|------|--------|-----|
| 全件取得 | `Table.Query().All(ctx, exec)` | `SELECT * FROM table` |
| PK検索 | `FindXxx(ctx, exec, pk)` | `SELECT * FROM table WHERE id = $1` |
| 条件検索 | `Table.Query(where...).One(ctx, exec)` | `SELECT * FROM table WHERE ...` |
| 挿入 | `Table.Insert(setter).One(ctx, exec)` | `INSERT INTO table (...) VALUES (...) RETURNING *` |
| 更新 | `model.Update(ctx, exec, setter)` | `UPDATE table SET ... WHERE id = $1` |
| 削除 | `model.Delete(ctx, exec)` | `DELETE FROM table WHERE id = $1` |
