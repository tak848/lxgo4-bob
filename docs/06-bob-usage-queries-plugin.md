# 6. Queries Plugin (sqlc 記法)

bob の queries プラグインは **sqlc と同様の記法** で、手書き SQL から型安全な Go コードを生成する。
集計クエリや複雑な JOIN など、bob のクエリビルダーでは表現しにくい SQL に使う。

## SQL ファイルの書き方

```sql
-- queries/reports.sql

-- GetProjectTaskStats
SELECT
    p.id, p.name,
    COUNT(t.id) AS total_tasks,
    COUNT(t.id) FILTER (WHERE t.status = 'done') AS done_tasks,
    COUNT(t.id) FILTER (WHERE t.status = 'in_progress') AS active_tasks
FROM projects p
LEFT JOIN tasks t ON t.project_id = p.id AND t.workspace_id = p.workspace_id
WHERE p.workspace_id = $1
GROUP BY p.id, p.name
ORDER BY p.name;
```

### ルール

- `-- 関数名` コメントで関数名を指定（1 ファイルに複数クエリ可）
- `$1`, `$2` 等で引数を指定（PostgreSQL のプレースホルダ記法）
- SELECT のカラム名が Go 構造体のフィールド名になる（snake_case → PascalCase）

## bobgen.yaml での設定

```yaml
psql:
  queries:
    - "./queries"
```

`./queries` ディレクトリ内の `.sql` ファイルを読み取る。

## 生成されるコード

### 関数

```go
// queries/reports.bob.go（生成コード）
func GetProjectTaskStats(WorkspaceID uuid.UUID) *GetProjectTaskStatsQuery {
    // ...
}
```

- 引数の型は SQL の `$1` に対応する型（DB スキーマから推論）
- 戻り値はクエリオブジェクト。`.All(ctx, exec)` / `.One(ctx, exec)` で実行

### 結果の構造体

```go
type GetProjectTaskStatsRow = struct {
    ID          uuid.UUID `db:"id"`
    Name        string    `db:"name"`
    TotalTasks  int64     `db:"total_tasks"`
    DoneTasks   int64     `db:"done_tasks"`
    ActiveTasks int64     `db:"active_tasks"`
}
```

- SELECT のカラム名がそのままフィールドになる
- `AS total_tasks` のエイリアスが使われる
- 型は DB のカラム型から推論（COUNT → int64 等）

## 呼び出し方

```go
// internal/service/report.go:25
rows, err := queries.GetProjectTaskStats(wsID).All(ctx, s.exec)
```

```go
// ダッシュボード（1行取得）
rows, err := queries.GetWorkspaceDashboard(wsID).All(ctx, s.exec)
if len(rows) == 0 {
    return &DashboardDTO{}, nil
}
dashboard := rows[0]
```

- `.All(ctx, exec)` → `[]XxxRow` を返す
- `.One(ctx, exec)` → `XxxRow` を返す（0 件なら `sql.ErrNoRows`）

## 3 つのクエリの詳細

### GetProjectTaskStats — プロジェクト別タスク統計

```sql
SELECT
    p.id, p.name,
    COUNT(t.id) AS total_tasks,
    COUNT(t.id) FILTER (WHERE t.status = 'done') AS done_tasks,
    COUNT(t.id) FILTER (WHERE t.status = 'in_progress') AS active_tasks
FROM projects p
LEFT JOIN tasks t ON t.project_id = p.id AND t.workspace_id = p.workspace_id
WHERE p.workspace_id = $1
GROUP BY p.id, p.name
ORDER BY p.name;
```

PostgreSQL の `FILTER (WHERE ...)` を使って条件別カウントを 1 クエリで取得。

### GetMemberTaskSummary — メンバー別タスクサマリ

```sql
SELECT
    m.id, m.name,
    COUNT(t.id) AS assigned_tasks,
    COUNT(t.id) FILTER (WHERE t.status = 'done') AS completed_tasks,
    COUNT(t.id) FILTER (WHERE t.due_date < CURRENT_DATE AND t.status != 'done') AS overdue_tasks
FROM members m
LEFT JOIN tasks t ON t.assignee_id = m.id AND t.workspace_id = m.workspace_id
WHERE m.workspace_id = $1
GROUP BY m.id, m.name
ORDER BY m.name;
```

### GetWorkspaceDashboard — ワークスペース全体統計

```sql
SELECT
    (SELECT COUNT(*) FROM projects WHERE workspace_id = $1) AS project_count,
    (SELECT COUNT(*) FROM members WHERE workspace_id = $1) AS member_count,
    (SELECT COUNT(*) FROM tasks WHERE workspace_id = $1) AS total_tasks,
    (SELECT COUNT(*) FROM tasks WHERE workspace_id = $1 AND status = 'done') AS done_tasks,
    (SELECT COUNT(*) FROM tasks WHERE workspace_id = $1 AND due_date < CURRENT_DATE AND status != 'done') AS overdue_tasks;
```

サブクエリで複数テーブルの集計を 1 行で返す。

## 型推論の仕組み: なぜ型安全になるのか

queries plugin が手書き SQL から正しい Go 型を導出できる理由。

### 2 段構えの型推論

#### Step 1: pg_query_go で SQL を AST にパース

`github.com/pganalyze/pg_query_go`（PostgreSQL パーサの Go バインディング）を使い、
SQL を構文木（AST）に分解する。カラム参照、JOIN 条件、NULL 可能性をトラッキングする。

これは PostgreSQL 本体のパーサと同じコードを使っているため、
PostgreSQL が受け付ける SQL はすべてパースできる。

#### Step 2: PREPARE + pg_prepared_statements で PostgreSQL に型を聞く

```go
// bobgen-psql/driver/parser/args_cols.go:76（実際のコード）
p.conn.ExecContext(ctx, fmt.Sprintf("PREPARE %q AS %s", queryID, q))
```

SQL を `PREPARE` 文で**実際の DB に投げる**。クエリは実行せず、準備だけする。
その後 `pg_prepared_statements` システムカタログから `parameter_types`（引数の型）と
`result_types`（戻り値の型）を取得する。

```sql
-- bob が内部で実行するクエリ（簡略版）
SELECT
  prep.type AS arg_type,        -- 'parameter' or 'result'
  pg_type.typname AS column_type -- 'uuid', 'int8', 'text' 等
FROM
  pg_prepared_statements
  CROSS JOIN unnest(parameter_types::oid[]) ...
  LEFT JOIN pg_type ON pg_type.oid = prep.oid
WHERE
  name = $1;
```

つまり **PostgreSQL 自身が「このクエリの引数は uuid で、戻り値は int8 と text と...」と教えてくれる。**

### 具体例: 型推論の結果

```sql
SELECT
    p.id,                                            -- → uuid.UUID（スキーマから）
    p.name,                                          -- → string（スキーマから）
    COUNT(t.id) AS total_tasks,                      -- → int64（COUNT の戻り値型）
    COUNT(t.id) FILTER (WHERE t.status = 'done')     -- → int64（同上）
FROM projects p
LEFT JOIN tasks t ON ...
WHERE p.workspace_id = $1                            -- $1 → uuid.UUID（スキーマから）
```

PostgreSQL が `PREPARE` した時点で全カラムの OID（型ID）が確定するので、bob はそれを Go 型にマッピングする:

| PostgreSQL 型 | Go 型 |
|--------------|-------|
| `uuid` | `uuid.UUID` |
| `text` / `varchar` | `string` |
| `int8` (bigint) | `int64` |
| `int4` (integer) | `int32` |
| `bool` | `bool` |
| `timestamptz` | `time.Time` |
| `date` | `time.Time` |
| `numeric` | `decimal.Decimal`（設定次第） |
| ENUM | 生成された enum 型 |
| NULLABLE カラム | `null.Val[T]` |

### 複雑なクエリでも安全な理由

PostgreSQL が `PREPARE` を通せる SQL であれば、型は正確に推論される:

- **集約関数** (`COUNT`, `SUM`, `AVG`, `MAX`, `MIN`) → PostgreSQL が戻り値型を知っている
- **`FILTER (WHERE ...)`** → 集約関数の戻り値型は変わらない（PostgreSQL が解釈）
- **サブクエリ** → `PREPARE` で再帰的に解析される
- **`CASE WHEN ... THEN ... END`** → PostgreSQL が結果型を推論（全分岐の共通型）
- **Window 関数** (`ROW_NUMBER`, `RANK`, `LAG` 等) → PostgreSQL が型を知っている
- **CTE (`WITH`)** → `PREPARE` で型が確定
- **型キャスト** (`CAST(x AS int)`, `x::text`) → キャスト先の型が使われる

### 限界: 破綻するケース

#### 1. 暗黙的 CROSS JOIN は禁止

```sql
-- NG: bob が明示的にエラーにする
SELECT * FROM users, orders WHERE users.id = orders.user_id;
-- → "multiple FROM tables are not supported, convert to a CROSS JOIN"

-- OK: 明示的な JOIN を使う
SELECT * FROM users CROSS JOIN orders WHERE users.id = orders.user_id;
```

bob の verify ステップで `FROM` に複数テーブルを並べる暗黙 CROSS JOIN を禁止している。

#### 2. Nullable 推論が不完全な場合

bob は AST を歩いて NULL 可能性を推論するが、完全ではない:

```sql
-- LEFT JOIN の右側のカラムは nullable（bob は正しく推論）
SELECT u.name, o.total
FROM users u LEFT JOIN orders o ON u.id = o.user_id;
-- → o.total は null.Val[T] になる ✓

-- COALESCE は non-nullable にするが...
SELECT COALESCE(o.total, 0) AS total FROM ...;
-- → bob がこれを non-nullable と判定するかは実装による
```

複雑な `CASE WHEN` + `COALESCE` の組み合わせで Nullable 判定が間違う可能性はある。
その場合、コメントアノテーションで上書きできる（sqlc と同様の仕組み）。

#### 3. bobgen 実行時と実行時の DB スキーマ不一致

`PREPARE` はコード生成時の DB スキーマで型を決定する。
マイグレーション後に `mise run bobgen` を忘れると、生成コードの型と実際の DB が乖離する。

#### 4. PREPARE に失敗する SQL

`PREPARE` 自体が失敗する SQL（構文エラー、存在しないテーブル参照等）は
bobgen の段階でエラーになり、コード生成されない。
つまり **コンパイルが通った時点で、SQL の型安全性は保証される。**

### sqlc との型推論方式の比較

| 観点 | sqlc | bob queries plugin |
|------|------|-------------------|
| パーサ | pg_query_go (同じ) | pg_query_go (同じ) |
| 型推論 | `PREPARE` + `pg_catalog` | `PREPARE` + `pg_prepared_statements` |
| DB 接続必要 | v2 から不要（pganalyze/pg_query_go でオフライン可） | **必要**（実 DB に PREPARE する） |
| アノテーション | `-- @name`, `-- @arg` 等 | `-- 関数名` コメント |
| 型上書き | `sqlc.narg()`, `sqlc.arg()` | コメントアノテーション |
| 生成結果 | 関数 + Row 構造体 | クエリオブジェクト + Row 構造体 |

bob は実 DB への接続が必須な分、型推論の精度は高い。sqlc v2 はオフライン解析が可能で CI フレンドリー。

## 重要: QueryHooks は queries plugin に効かない

**queries plugin で生成されたクエリには QueryHooks が適用されない。**

### 理由

queries plugin の生成コードは `dbgen.Tasks.Query()` を使わない。
独自のクエリオブジェクトを構築し、テーブルの `SelectQueryHooks` を経由しない。

bob のソースコード `gen/templates/queries/query/01_query.go.tpl` L86-91 を確認:
```go
Hooks: nil,  // ← Hooks が nil
```

### 対策: SQL に明示的に workspace_id 条件を書く

```sql
WHERE p.workspace_id = $1  -- ← 手動で書く
```

queries plugin を使う SQL には、**必ず** `WHERE workspace_id = $1` を含める。
Hook による自動注入に頼れないため、SQL 作成者が責任を持つ。

## queries plugin vs bob クエリビルダーの使い分け

| 用途 | 手段 |
|------|------|
| 単純な CRUD | bob クエリビルダー（`dbgen.Tasks.Query()` 等） |
| フィルタ・ページネーション | bob クエリビルダー + SelectWhere + sm |
| リレーション取得 | bob Preload / ThenLoad |
| 集計・GROUP BY | queries plugin（sqlc 記法の手書き SQL） |
| 複雑な JOIN / サブクエリ | queries plugin |
| Window 関数 | queries plugin |

基本は bob クエリビルダーで、クエリビルダーで表現しにくい場合に queries plugin を使う。
