# 11. 落とし穴・注意点

bob を実際に使って遭遇した問題と対策。

## 1. Factory の Create がリレーション経由で値を上書きする

### 問題

```go
f.NewTask(
    factory.TaskMods.WorkspaceID(wsID),
    factory.TaskMods.AssigneeID(null.From(memberID)),
).Create(ctx, bobDB)
```

Factory の `Create` メソッドは、未指定のリレーション（`o.r.Member == nil` 等）に対して
自動的に `WithNewMember()` を呼び、**新しい Member を DB に INSERT** した上で
`opt.WorkspaceID` と `opt.AssigneeID` を **上書き** する。

結果: 意図しない workspace_id の Member が作られ、**複合 FK 制約違反**になる。

### 対策

seed や手動データ投入では Factory を使わず、bob の直接 Insert を使う:

```go
dbgen.Tasks.Insert(&dbgen.TaskSetter{
    ID:          omit.From(tID),
    WorkspaceID: omit.From(wsID),
    ProjectID:   omit.From(pID),
    AssigneeID:  omitnull.From(assigneeID),
}).One(ctx, bobDB)
```

詳細: [Factory Plugin](./08-bob-usage-factory.md)

## 2. QueryHooks は queries plugin に効かない

### 問題

QueryHooks（SELECT/UPDATE/DELETE に自動 WHERE 注入）は、
queries plugin で生成されたクエリには **適用されない**。

bob のソーステンプレート（`gen/templates/queries/query/01_query.go.tpl`）で
Hooks フィールドが `nil` にされている。

### 対策

queries plugin の SQL には **手動で** `WHERE workspace_id = $1` を書く:

```sql
-- queries/reports.sql
SELECT ... FROM projects p
WHERE p.workspace_id = $1  -- ← 必ず手動で書く
```

詳細: [Queries Plugin](./06-bob-usage-queries-plugin.md)

## 3. QueryHooks は INSERT に効かない

### 問題

bob の QueryHooks は SELECT / UPDATE / DELETE のみ。INSERT には Hook がない。

### 対策

Service 層で INSERT 時に workspace_id を明示的にセットする:

```go
dbgen.Members.Insert(&dbgen.MemberSetter{
    WorkspaceID: omit.From(wsID),  // パスパラメータから取得した値
    // ...
}).One(ctx, exec)
```

DB 側でも **複合 FK** で cross-workspace 参照を防止:
```sql
UNIQUE (workspace_id, id),
FOREIGN KEY (workspace_id, project_id) REFERENCES projects(workspace_id, id)
```

詳細: [QueryHooks](./05-bob-usage-hooks.md)

## 4. omitnull の「未セット」と「NULL」の混同

### 問題

```go
type TaskSetter struct {
    AssigneeID omitnull.Val[uuid.UUID]
}
```

`omitnull.Val[T]` のゼロ値は **未セット (omit)** であり **NULL** ではない。

- 未セット → SQL に含まれない → DB デフォルト値（多くの場合 NULL だが、DEFAULT が設定されていれば違う）
- NULL → `SET assignee_id = NULL` が発行される

### 対策

NULL を明示的にセットする場合:

```go
var nullVal omitnull.Val[uuid.UUID]
nullVal.Null()
setter.AssigneeID = nullVal
```

詳細: [Nullable カラムの扱い](./07-bob-usage-nullable.md)

## 5. pgx と database/sql の間の変換が必要

### 問題

bob は `database/sql` の `*sql.DB` を前提とする。
pgx の native API（`pgx.Conn`, `pgxpool.Pool`）を直接渡せない。

### 対策

`stdlib.OpenDBFromPool()` で変換:

```go
pool, _ := pgxpool.New(ctx, dsn)
sqlDB := stdlib.OpenDBFromPool(pool)
bobDB := bob.NewDB(sqlDB)
```

詳細: [DB 接続](./09-bob-usage-executor.md)

## 6. bobgen の DSN と実行時の DSN の不一致

### 問題

bobgen.yaml の `psql.dsn` はコード生成時に使う DSN。
実行時の `DATABASE_URL` とポートが異なる場合がある
（特に mise.local.toml でポートを上書きしている場合）。

### 対策

`PSQL_DSN` 環境変数で上書きする（bob 公式サポート）:

```toml
# mise.toml
PSQL_DSN = "{{env.DATABASE_URL}}"
```

bobgen は内部的に koanf を使っており、環境変数 `PSQL_DSN` が `psql.dsn` を上書きする。

## 7. except で除外したカラムは Setter に含まれない

### 問題

```yaml
# bobgen.yaml
except:
  "*": [created_at, updated_at]
```

この設定で `created_at` と `updated_at` が **Setter から除外** される。
モデル構造体には含まれるが、INSERT/UPDATE で明示的にセットすることができない。

### 対策

これは意図的な設計。`created_at` は `DEFAULT NOW()`、`updated_at` は DB トリガーで管理:

```sql
CREATE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

## 8. Preload は to-many に使うべきではない

### 問題

Preload（LEFT JOIN）で to-many リレーションを取ると、カーテシアン積で行数が爆発する。

```go
// NG: TaskComments は to-many
dbgen.Preload.Task.TaskComments()  // ← JOIN で行数爆発
```

### 対策

to-many には ThenLoad（別クエリ）を使う:

```go
dbgen.SelectThenLoad.Task.TaskComments()  // ← 別クエリで安全
```

詳細: [リレーション](./04-bob-usage-relations.md)

## 9. bob.SkipHooks の存在に注意

```go
rows, err := dbgen.Members.Query().All(bob.SkipHooks(ctx), exec)
```

`bob.SkipHooks(ctx)` を渡すと **全 QueryHooks がバイパス** される。
テナント分離の Hook もスキップされるため、意図せず全テナントのデータが見えてしまう。

管理者操作やバッチ処理で意図的に使う場合のみ使用すること。

## 10. struct_tag_casing の影響

```yaml
struct_tag_casing: snake
tags: [json]
```

この設定により、生成されるモデルに `json:"workspace_id"` のような snake_case のタグが付く。
API レスポンスを camelCase にしたい場合は、DTO への変換レイヤーが必要
（本プロジェクトでは ogen が独自の型を生成するため、handler で変換している）。
