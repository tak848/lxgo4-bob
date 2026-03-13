# マルチテナント タスク管理システム — 実装プラン

## Context

layerx.go #4 イベント向けのデモアプリ。bob ORM の主要機能（CRUD、JOIN、型安全クエリビルド、queries plugin = sqlc記法、テナントフック）を PostgreSQL 上で実演する。
ogen でスキーマ駆動の REST API を生成し、Go 1.24+ の `go tool` を活用する。

## アーキテクチャ概要

```
OpenAPI spec (api/openapi.yaml)
    ↓ ogen generate
internal/oas/  (生成コード)
    ↓ implements Handler interface
internal/handler/  (ハンドラ実装)
    ↓ calls
internal/service/  (ビジネスロジック + bob クエリ構築)
    ↓ uses
internal/infra/db/  (DB接続, WorkspaceScoped/Global executor)
    ↓ hooks fire
internal/infra/dbgen/  (bob 生成モデル・where・joins・loaders)
internal/infra/dbgen/dbenums/
internal/infra/dbgen/factory/
queries/  (sqlc記法 SQL → bob queries plugin 生成)
```

layerone との差別化ポイント:
- テナントキーは `workspace_id`（organization_id / tenant_id は使わない）
- Repository パターンではなく **Service 層が直接 bob を操作**（薄いレイヤー構成）
- ScopedExecutor ではなく **WorkspaceScopedExec / GlobalExec** という命名
- Hook 登録は手動の `RegisterHooks()` 関数（custom-bobgen 不要）
- QueryHooks ベース（ContextualMod ではない）。デモ用の簡略化であり、本番では `bob.SkipHooks()` でバイパスされるリスクがある点はトレードオフとして認識

## テーブル設計（5テーブル）

| テーブル | パーティションキー | 主な関連 |
|---|---|---|
| workspaces | — (テナント自体) | — |
| members | workspace_id | workspaces |
| projects | workspace_id | workspaces |
| tasks | workspace_id | projects, members(assignee) |
| task_comments | workspace_id | tasks, members(author) |

ENUMs: `member_role` (owner/editor/viewer), `project_status` (active/archived), `task_status` (todo/in_progress/done), `task_priority` (low/medium/high/urgent)

全テーブルに `updated_at` 自動更新トリガーを設定。

**UUID 方針**: ADR に従い UUIDv7 をアプリ側で生成（`github.com/google/uuid` の `uuid.NewV7()`）。DB の `DEFAULT gen_random_uuid()` は設定しない。これにより時系列ソート可能な ID を実現。

**cross-workspace 参照防止**: テナントスコープのテーブルには `UNIQUE(workspace_id, id)` 制約を追加し、FK は複合FK `(workspace_id, xxx_id) REFERENCES xxx(workspace_id, id)` で定義。これにより DB レベルで workspace を跨いだ参照を防止。

**Insert 時の workspace_id 強制**: Service 層で path パラメータの workspace_id を setter に上書き固定する。リクエストbodyからは受け取らない。

## Hooks と queries plugin の関係（重要な知見）

bob の queries plugin で生成されるコードは `ExecQuery` の `Hooks` フィールドが **nil** のまま生成される（`gen/templates/queries/query/01_query.go.tpl` L86-91 で確認済み）。
よって **テナントフック（QueryHooks）は queries plugin 出力には効かない**。

対処方針:
- **CRUD（bob 標準モデル経由）**: フックが自動適用 → `workspace_id` フィルタ自動挿入
- **集計クエリ（queries plugin）**: SQL 自体に `WHERE workspace_id = $1` を明示的に書く（フックに頼らない）
- これは layerone が sqlc で採用しているのと同じ戦略

## ディレクトリ構成

pnpm workspace でモノレポ構成。ルートに Go バックエンド、`webapp/` にフロントエンド。

```
.
├── pnpm-workspace.yaml             # packages: [webapp]
├── package.json                     # root（pnpm, scripts）
├── api/
│   └── openapi.yaml              # OpenAPI 3.0 spec（バックエンド・フロントエンド共有）
├── cmd/
│   ├── server/main.go            # HTTPサーバ起動
│   └── seed/main.go              # サンプルデータ投入
├── db/
│   └── migrations/               # dbmate マイグレーション（7本）
├── queries/
│   └── reports.sql               # sqlc記法の集計クエリ
├── internal/
│   ├── oas/                      # [生成] ogen サーバコード
│   ├── handler/                  # ogen Handler 実装
│   │   ├── handler.go            # 構造体定義 + NewError
│   │   ├── workspace.go
│   │   ├── member.go
│   │   ├── project.go
│   │   ├── task.go
│   │   ├── comment.go
│   │   └── report.go
│   ├── service/                  # ビジネスロジック（bob直接操作）
│   │   ├── workspace.go
│   │   ├── member.go
│   │   ├── project.go
│   │   ├── task.go
│   │   ├── comment.go
│   │   └── report.go            # queries plugin 生成関数を呼ぶ
│   └── infra/
│       ├── db/
│       │   ├── conn.go           # pgx 接続 + bob.NewDB
│       │   └── scope.go          # WorkspaceScopedExec / GlobalExec
│       ├── dbgen/                # [生成] bob モデル
│       │   ├── dbenums/          # [生成] enum型
│       │   └── factory/          # [生成] テストファクトリ
│       └── hook/
│           ├── workspace.go      # WorkspaceSelectHook / Update / Delete
│           └── register.go       # RegisterHooks()
├── webapp/                         # Next.js フロントエンド
│   ├── package.json
│   ├── next.config.ts
│   ├── tsconfig.json               # strict: ncr-orchestrator 参考
│   ├── eslint.config.mjs           # flat config, strict: ncr-orchestrator 参考
│   ├── .prettierrc.mjs
│   ├── tailwind.config.ts
│   ├── components.json             # shadcn/ui config
│   ├── src/
│   │   ├── app/                    # Next.js App Router
│   │   │   ├── layout.tsx
│   │   │   ├── page.tsx            # ワークスペース選択
│   │   │   └── workspaces/
│   │   │       └── [wsId]/
│   │   │           ├── layout.tsx  # サイドバー付きレイアウト
│   │   │           ├── page.tsx    # ダッシュボード（reports/dashboard API）
│   │   │           ├── projects/
│   │   │           ├── tasks/
│   │   │           ├── members/
│   │   │           └── reports/
│   │   ├── components/
│   │   │   ├── ui/                 # [生成] shadcn/ui コンポーネント
│   │   │   └── ...                 # アプリ固有コンポーネント
│   │   ├── lib/
│   │   │   ├── api/                # [生成] openapi-fetch クライアント
│   │   │   └── utils.ts            # tailwind-merge 等
│   │   └── .env.local              # NEXT_PUBLIC_API_URL
│   └── .env                        # デフォルト env
├── mise.toml                       # env定義（ポート、DSN）
├── compose.yaml
├── docs/
│   └── plan.md                     # 実装プランのコピー
├── docker/
│   ├── grafana/
│   │   └── datasource.yaml
│   └── promtail/
│       └── config.yaml
├── bobgen.yaml
├── Makefile
├── go.mod
└── go.sum
```

## 実装順序（チーム分担前提）

### Phase 0: インフラ・スキーマ基盤
1. `mise.toml` — ツールチェーン（go 1.26, node 22, pnpm 10）+ env 定義（ポート、DSN等）。`mise trust` で有効化。`mise.local.toml` は既存を維持（GH_TOKEN 等）
1b. `compose.yaml` — PostgreSQL 17 + Loki + Grafana + Promtail。全ポートを env で変更可能にし既存サービスと干渉しない（デフォルト: PG=15432, Grafana=13000, Loki=13100）
2. `Makefile` — up, down, migrate, bobgen, ogen, seed ターゲット
3. `go.mod` — tool ディレクティブ追加 + 依存取得
4. `db/migrations/` — 全マイグレーションファイル（7本）
5. `bobgen.yaml` — プラグイン設定
6. `queries/reports.sql` — 集計SQL（3クエリ）
7. **実行**: `make up && make migrate && make bobgen` で生成コード確認

### Phase 1: DB層・フック
8. `internal/infra/db/conn.go` — DB接続
9. `internal/infra/db/scope.go` — WorkspaceScopedExec / GlobalExec
10. `internal/infra/hook/workspace.go` — 3つのQueryHook実装
11. `internal/infra/hook/register.go` — フック登録

### Phase 2: OpenAPI・ogen
12. `api/openapi.yaml` — 全エンドポイント定義
13. **実行**: `make ogen` で `internal/oas/` 生成

### Phase 3: Service 層
14. `internal/service/workspace.go` — CRUD (GlobalExec使用)
15. `internal/service/member.go` — CRUD
16. `internal/service/project.go` — CRUD + フィルタ
17. `internal/service/task.go` — CRUD + JOIN(Preload/ThenLoad) + フィルタ + ページネーション
18. `internal/service/comment.go` — CRUD
19. `internal/service/report.go` — queries plugin 呼び出し

### Phase 4: Handler 層
20. `internal/handler/handler.go` — 共通構造体・エラーハンドリング
21. `internal/handler/workspace.go` 〜 `report.go` — ogen 型変換 + service呼び出し

### Phase 5: サーバ起動・シード
22. `cmd/server/main.go` — wiring（CORS 設定含む: webapp のポートを許可）
23. `cmd/seed/main.go` — factory プラグインでサンプルデータ
24. Grafana ダッシュボード・datasource設定

### Phase 6: フロントエンド基盤
25. `pnpm-workspace.yaml` + ルート `package.json` — モノレポ設定
26. `webapp/` — `pnpm create next-app` ベースで初期化（App Router, TypeScript, Tailwind, ESLint）
27. `webapp/tsconfig.json` — strict 設定（ncr-orchestrator 参考: strict, noUnusedLocals, noUnusedParameters, noFallthroughCasesInSwitch）
28. `webapp/eslint.config.mjs` — flat config + @typescript-eslint/recommended-type-checked + prettier（ncr-orchestrator 参考）
29. `webapp/.prettierrc.mjs` — prettier-plugin-tailwindcss 含む
30. shadcn/ui 初期化（`pnpm dlx shadcn@latest init`）→ button, card, table, badge, dialog, input, select, dropdown-menu 等

### Phase 7: OpenAPI クライアント生成 + 画面実装
31. `openapi-typescript` で `api/openapi.yaml` → 型生成（`webapp/src/lib/api/schema.d.ts`）
32. `openapi-fetch` で型安全な API クライアント（`webapp/src/lib/api/client.ts`）
33. 画面実装:
    - `/` — ワークスペース一覧・選択
    - `/workspaces/[wsId]` — ダッシュボード（GetWorkspaceDashboard API）
    - `/workspaces/[wsId]/projects` — プロジェクト一覧 + CRUD
    - `/workspaces/[wsId]/tasks` — タスク一覧（フィルタ・ソート）+ CRUD
    - `/workspaces/[wsId]/members` — メンバー一覧 + CRUD
    - `/workspaces/[wsId]/reports` — プロジェクト統計・メンバーサマリ

### Phase 8: ドキュメント
34. `docs/plan.md` — 実装プランのコピーを配置

## チーム分担案

TeamCreate で `taskman` チームを作成し、以下のように Agent を spawn して分担:

| エージェント | 担当 | Phase | 依存 |
|---|---|---|---|
| **infra** | mise.toml, compose.yaml, Makefile, go.mod, migrations, bobgen.yaml, queries/reports.sql, DB層, フック | 0, 1 | なし（最初に実行） |
| **api** | openapi.yaml, ogen生成, handler層, CORS設定 | 2, 4 | infra 完了後 |
| **service** | service層全体, cmd/server, cmd/seed | 3, 5 | infra 完了後（api と並行可） |
| **frontend** | pnpm workspace, Next.js セットアップ, tsconfig/eslint/prettier 堅い設定, shadcn/ui, OpenAPI クライアント生成, 全画面実装 | 6, 7 | api 完了後（openapi.yaml 必要） |

依存関係:
```
infra → (api, service 並行) → server wiring
              api → frontend
```

**リード（自分）の役割**: TeamCreate → TaskCreate → 各エージェント spawn → 進捗監視 → server wiring 統合 → 検証

## 技術詳細

### go.mod tool ディレクティブ
```go
tool (
    github.com/stephenafamo/bob/gen/bobgen-psql
    github.com/amacneil/dbmate/v2
    github.com/ogen-go/ogen/cmd/ogen
)
```

### bobgen.yaml 主要設定
```yaml
struct_tag_casing: snake
tags: [json]
plugins:
  models:
    destination: "internal/infra/dbgen"
    pkgname: "dbgen"
  enums:
    destination: "internal/infra/dbgen/dbenums"
  factory:
    destination: "internal/infra/dbgen/factory"
  where: {}
  loaders: {}
  joins: {}
  dberrors: {}
  dbinfo:
    disabled: true
psql:
  dsn: "postgres://postgres:password@localhost:15432/taskman?sslmode=disable"
  # 公式仕様: bobgen は koanf + env.Provider で PSQL_ プレフィックスの環境変数をサポート
  # PSQL_DSN 環境変数で psql.dsn を上書き可能（公式ドキュメント: code-generation/psql.md）
  # mise.toml で PSQL_DSN を定義すれば OK
  uuid_pkg: google
  driver: "github.com/jackc/pgx/v5/stdlib"
  queries:
    - "./queries"
  except:
    "schema_migrations": {}
    "*":
      columns: [created_at, updated_at]
```

### queries/reports.sql（sqlc記法の実演）
```sql
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

-- GetMemberTaskSummary
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

-- GetWorkspaceDashboard
SELECT
    (SELECT COUNT(*) FROM projects WHERE workspace_id = $1) AS project_count,
    (SELECT COUNT(*) FROM members WHERE workspace_id = $1) AS member_count,
    (SELECT COUNT(*) FROM tasks WHERE workspace_id = $1) AS total_tasks,
    (SELECT COUNT(*) FROM tasks WHERE workspace_id = $1 AND status = 'done') AS done_tasks,
    (SELECT COUNT(*) FROM tasks WHERE workspace_id = $1 AND due_date < CURRENT_DATE AND status != 'done') AS overdue_tasks;
```

### WorkspaceSelectHook の仕組み（概念）

**注意**: 以下は設計意図を示す擬似コード。実装時に bob の実際の Hook 型シグネチャ（`bob.Hook[T] = func(context.Context, bob.Executor, T) (context.Context, error)` 等）を確認して正確に合わせること。また JOIN/Preload 時に列名が曖昧にならないよう、テーブル名を明示（`psql.Quote("members", "workspace_id")`）する。

```go
// 概念的な hook 設計（実装時に正確なシグネチャに合わせる）
// SELECT クエリに WHERE <table>.workspace_id = ? を自動注入
// executor から scope 情報を取得し、ModeGlobal なら skip
```

### task service での JOIN 実演（Preload + ThenLoad）（概念）

**注意**: 以下は概念的な記法。bob の生成コードは `Preload.<SingularTable>.<Rel>()` / `SelectThenLoad.<SingularTable>.<Rel>()` 形式で生成される。正確な API 名は bobgen 実行後の生成コードを参照すること。bob バージョンは固定して使う。

```go
// 概念: task を project, assignee と共に取得し、comments は別クエリで取得
// Preload → LEFT JOIN（to-one）、ThenLoad → 別クエリ（to-many）
```

### フロントエンド技術スタック

**フレームワーク**: Next.js (App Router) + TypeScript
**UIライブラリ**: Tailwind CSS + shadcn/ui (new-york スタイル)
**API クライアント**: openapi-typescript + openapi-fetch（OpenAPI spec から型安全クライアント自動生成）
**パッケージマネージャ**: pnpm（workspace でモノレポ）
**ポート**: `${WEBAPP_PORT:-13001}`（next.config.ts で `--port` 指定）

**厳格な TypeScript 設定**（ncr-orchestrator 参考）:
```json
{
  "compilerOptions": {
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "isolatedModules": true
  }
}
```

**厳格な ESLint 設定**（ncr-orchestrator 参考、flat config）:
- `@typescript-eslint/recommended-type-checked` + `stylistic-type-checked`
- `noExplicitAny` with `fixToUnknown: true`
- `consistentTypeImports`（type import 分離）
- prettier 統合（prettier-plugin-tailwindcss）

**OpenAPI クライアント生成**:
```bash
# package.json script
pnpm exec openapi-typescript ../api/openapi.yaml -o src/lib/api/schema.d.ts
```
→ `openapi-fetch` の `createClient<paths>()` で型安全 API 呼び出し

**画面構成**:
| パス | 内容 | 使用 API |
|---|---|---|
| `/` | ワークスペース一覧・新規作成 | GET/POST /workspaces |
| `/workspaces/[wsId]` | ダッシュボード | GET reports/dashboard |
| `/workspaces/[wsId]/projects` | プロジェクト一覧・CRUD | GET/POST/PUT/DELETE projects |
| `/workspaces/[wsId]/tasks` | タスク一覧（フィルタ・ソート）・CRUD | GET/POST/PUT/DELETE tasks |
| `/workspaces/[wsId]/members` | メンバー一覧・CRUD | GET/POST/PUT/DELETE members |
| `/workspaces/[wsId]/reports` | プロジェクト統計・メンバーサマリ | GET reports/* |

### compose.yaml + mise.toml による環境分離

全ポートを環境変数で変更可能にし、既存サービスと干渉しない設計。

**mise.toml**（リポジトリにコミット）— ツールチェーン + デフォルト env:
```toml
[tools]
go = "1.26.1"
node = "24.14.0"
pnpm = "10.32.1"

[env]
POSTGRES_PORT = "15432"
POSTGRES_USER = "postgres"
POSTGRES_PASSWORD = "password"
POSTGRES_DB = "taskman"
GRAFANA_PORT = "13000"
LOKI_PORT = "13100"
APP_PORT = "18080"
WEBAPP_PORT = "13001"
NEXT_PUBLIC_API_URL = "http://localhost:{{env.APP_PORT}}"
DATABASE_URL = "postgres://{{env.POSTGRES_USER}}:{{env.POSTGRES_PASSWORD}}@localhost:{{env.POSTGRES_PORT}}/{{env.POSTGRES_DB}}?sslmode=disable"
PSQL_DSN = "{{env.DATABASE_URL}}"  # bobgen が koanf 経由で psql.dsn を上書き
```

**mise.local.toml**（.gitignore済み、個人PC向け上書き）— 既存の GH_TOKEN 等に加え、ポート競合時の上書き等:
```toml
[env]
# 既にこのPCでは GH_TOKEN, MISE_GITHUB_TOKEN, DEVIN_API_KEY が定義済み
# ポート競合時はここで上書き:
# POSTGRES_PORT = "25432"
```

**compose.yaml** で `${POSTGRES_PORT:-15432}` 等の env 展開を使用:
- `postgres:17` — port `${POSTGRES_PORT:-15432}`:5432
- `grafana/loki:3.4.3` — port `${LOKI_PORT:-13100}`:3100
- `grafana/grafana:12.0.0` — port `${GRAFANA_PORT:-13000}`:3000, anonymous auth
- `grafana/promtail:3.4.3` — Docker logs → Loki

デフォルトで本業の 5432/3000/3100 と干渉しないポート（15432/13000/13100）を使用。
bobgen.yaml の DSN も `DATABASE_URL` 環境変数から取る or mise.toml 経由で注入。

## 検証手順

1. `docker compose up -d` → PostgreSQL(15432) + Grafana(13000) + Loki(13100) 起動（ポート干渉なし）
2. `make migrate` → テーブル作成
3. `make bobgen` → モデル・クエリ生成、コンパイル確認
4. `make ogen` → API サーバコード生成
5. `go test ./queries/...` → 生成クエリのテスト実行（bob 公式推奨）
6. `go build ./...` → 全体コンパイル
7. `go run ./cmd/seed/` → サンプルデータ投入
8. `go run ./cmd/server/` → サーバ起動 (`:${APP_PORT:-18080}`)
9. curl でエンドポイント確認（ポートは `${APP_PORT:-18080}`）:
   - `POST /workspaces` → ワークスペース作成
   - `POST /workspaces/{wsId}/members` → メンバー追加
   - `POST /workspaces/{wsId}/projects` → プロジェクト作成
   - `POST /workspaces/{wsId}/tasks` → タスク作成（assignee, project 紐づけ）
   - `GET /workspaces/{wsId}/tasks?status=todo&priority=high` → フィルタ付き一覧
   - `GET /workspaces/{wsId}/reports/project-stats` → 集計レポート（queries plugin）
   - `GET /workspaces/{wsId}/reports/dashboard` → ダッシュボード
10. テナント分離確認: WS-A のデータが WS-B から見えないこと
11. Grafana (`localhost:${GRAFANA_PORT:-13000}`) でアプリログ確認
12. `cd webapp && pnpm install && pnpm dev` → フロントエンド起動 (`:${WEBAPP_PORT:-13001}`)
13. `pnpm run generate:api` → OpenAPI クライアント型生成
14. ブラウザで `localhost:${WEBAPP_PORT:-13001}` → ワークスペース一覧 → タスク操作 → ダッシュボード確認
15. `pnpm run lint && pnpm run type-check` → TypeScript/ESLint エラーなし確認
