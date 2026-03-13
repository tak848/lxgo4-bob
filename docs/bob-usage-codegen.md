# 1. コード生成 (bobgen)

## bobgen-psql とは

bob のコードジェネレータ。実際の DB スキーマに接続し、テーブル定義から Go コードを自動生成する。
sqlc のように SQL ファイルから生成するのではなく、**DB のスキーマそのものが唯一の真実 (Single Source of Truth)** となる。

## インストール方法

Go 1.24+ の `go tool` ディレクティブを使用:

```go
// go.mod
tool (
    github.com/stephenafamo/bob/gen/bobgen-psql
)
```

実行は `go tool bobgen-psql -c bobgen.yaml`。

## bobgen.yaml の設定

```yaml
# bobgen.yaml
struct_tag_casing: snake   # 構造体タグの命名規則
tags: [json]               # 追加する構造体タグ

plugins:
  models:                   # テーブルモデル生成（必須）
    destination: "internal/infra/dbgen"
    pkgname: "dbgen"
  enums:                    # PostgreSQL ENUM → Go 型
    destination: "internal/infra/dbgen/dbenums"
  factory:                  # テストデータ Factory
    destination: "internal/infra/dbgen/factory"
  where: {}                 # 型安全 WHERE 条件ヘルパー
  loaders: {}               # Preload / ThenLoad（リレーション取得）
  joins: {}                 # JOIN ヘルパー
  dberrors: {}              # DB エラーラッパー（unique 制約違反の判定等）

psql:
  dsn: "postgres://postgres:password@localhost:5432/taskman?sslmode=disable"
  uuid_pkg: google          # UUID パッケージ（google/uuid を使用）
  driver: "github.com/jackc/pgx/v5/stdlib"
  queries:                  # queries plugin（sqlc 記法）
    - "./queries"
  except:                   # 生成から除外するカラム
    schema_migrations: []   # schema_migrations テーブルを丸ごと除外
    "*": [created_at, updated_at]  # 全テーブルの created_at, updated_at を Setter から除外
```

### 重要な設定ポイント

#### `uuid_pkg: google`
デフォルトでは `[16]byte` が使われる。`google` を指定すると `github.com/google/uuid` の `uuid.UUID` 型になる。

#### `except: { "*": [created_at, updated_at] }`
`created_at` と `updated_at` は DB のデフォルト値とトリガーで管理するため、
Setter（Insert/Update で使う構造体）から除外する。モデル構造体には含まれたまま。

#### `driver: "github.com/jackc/pgx/v5/stdlib"`
pgx の `database/sql` 互換レイヤーを指定。bob は `database/sql` の `*sql.DB` 経由で接続するため。

#### DSN の環境変数上書き
bobgen は内部的に koanf を使っており、`PSQL_DSN` 環境変数で `psql.dsn` を上書きできる（公式仕様）。
mise.toml/mise.local.toml で設定すると便利。

## 生成されるファイル一覧

```
internal/infra/dbgen/
├── bob_counts.bob.go       # Count ヘルパー
├── bob_joins.bob.go        # SelectJoins, UpdateJoins, DeleteJoins
├── bob_loaders.bob.go      # Preload, SelectThenLoad
├── bob_types.bob_test.go   # 型テスト
├── bob_where.bob.go        # SelectWhere, UpdateWhere, DeleteWhere
├── dbenums/
│   └── enums.bob.go        # MemberRole, ProjectStatus, TaskStatus, TaskPriority
├── factory/
│   ├── bobfactory_main.bob.go
│   ├── bobfactory_random.bob.go
│   ├── members.bob.go
│   ├── projects.bob.go
│   ├── tasks.bob.go
│   ├── task_comments.bob.go
│   └── workspaces.bob.go
├── members.bob.go          # Member モデル + CRUD
├── projects.bob.go         # Project モデル + CRUD
├── tasks.bob.go            # Task モデル + CRUD
├── task_comments.bob.go    # TaskComment モデル + CRUD
└── workspaces.bob.go       # Workspace モデル + CRUD

queries/
├── reports.sql             # sqlc 記法の SQL（手書き）
├── reports.bob.go          # ↑から生成された Go コード
└── reports.bob.sql         # フォーマット済み SQL（embed 用）

dberrors/
├── bob_errors.bob.go       # 共通エラーヘルパー
├── members.bob.go          # members テーブルの制約エラー
├── projects.bob.go
├── tasks.bob.go
├── task_comments.bob.go
└── workspaces.bob.go
```

## 生成されるモデル構造体の例

```go
// Task は tasks テーブルに対応する構造体
type Task struct {
    ID          uuid.UUID           `db:"id,pk" json:"id"`
    WorkspaceID uuid.UUID           `db:"workspace_id" json:"workspace_id"`
    ProjectID   uuid.UUID           `db:"project_id" json:"project_id"`
    AssigneeID  null.Val[uuid.UUID] `db:"assignee_id" json:"assignee_id"`
    Title       string              `db:"title" json:"title"`
    Description string              `db:"description" json:"description"`
    Status      enums.TaskStatus    `db:"status" json:"status"`
    Priority    enums.TaskPriority  `db:"priority" json:"priority"`
    DueDate     null.Val[time.Time] `db:"due_date" json:"due_date"`

    R taskR `db:"-" json:"-"` // リレーション格納用
    C taskC `db:"-" json:"-"` // カウント格納用
}
```

- NOT NULL カラム → そのままの Go 型（`string`, `uuid.UUID` 等）
- NULLABLE カラム → `null.Val[T]`（aarondl/opt パッケージ）
- PostgreSQL ENUM → 生成された Go 型（`enums.TaskStatus` 等）
- `.R` フィールドにリレーション先のモデルが入る（Preload/ThenLoad で取得）

## 再生成のタイミング

マイグレーション（テーブル変更）後に `make bobgen` を実行する。
生成コードは `DO NOT EDIT` が付いているので手で編集してはいけない。
