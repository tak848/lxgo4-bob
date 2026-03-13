# lxgo4-bob

[layerx.go #4](https://layerx.connpass.com/event/383847/) 向けのデモアプリケーション。

[bob](https://github.com/stephenafamo/bob) ORM を使ったマルチテナント タスク管理システム。

## bob とは

Go の SQL toolkit / ORM。DB スキーマからコードを自動生成し、型安全なクエリビルダー・CRUD・リレーション取得・sqlc 記法のクエリ生成などを提供する。

## このリポジトリで試していること

- **コード生成**: bobgen-psql で DB スキーマから models / where / joins / loaders / factory / enums を生成
- **型安全 CRUD**: `omit.Val[T]` / `omitnull.Val[T]` による Insert/Update、`FindXxx` による PK 検索
- **型安全フィルタ**: `SelectWhere.Tasks.Status.EQ(enums.TaskStatusDone)` のようなコンパイル時型チェック
- **リレーション取得**: Preload (LEFT JOIN, to-one) と ThenLoad (別クエリ, to-many) の使い分け
- **QueryHooks**: SELECT/UPDATE/DELETE に `WHERE workspace_id = $1` を自動注入してマルチテナント分離
- **queries plugin** (sqlc 記法): 手書き SQL から型安全な Go コードを生成（集計・レポート系クエリ）
- **factory plugin**: テストデータ生成（ただし落とし穴あり、詳細は docs 参照）
- **ogen**: OpenAPI スキーマ駆動の REST API サーバ自動生成
- **フロントエンド**: Next.js + openapi-typescript + openapi-fetch で型安全 API クライアント

詳細なドキュメントは [`docs/bob-usage.md`](./docs/bob-usage.md) および webapp 上の `/docs` ページから閲覧可能。

## 技術スタック

| レイヤー | 技術 |
|---------|------|
| ORM | [bob](https://github.com/stephenafamo/bob) v0.42 |
| DB | PostgreSQL 17 |
| API | [ogen](https://github.com/ogen-go/ogen) (OpenAPI v3 → Go サーバ) |
| Migration | [dbmate](https://github.com/amacneil/dbmate) |
| Frontend | Next.js 15 (App Router) + React Compiler + TypeScript strict |
| UI | Tailwind CSS + [shadcn/ui](https://ui.shadcn.com/) |
| API Client | [openapi-typescript](https://github.com/openapi-ts/openapi-typescript) + [openapi-fetch](https://github.com/openapi-ts/openapi-typescript/tree/main/packages/openapi-fetch) |
| Logging | slog (JSON) → Promtail → Loki → Grafana |
| Toolchain | [mise](https://mise.jdx.dev/) (Go, Node.js, pnpm) |

## セットアップ

### 前提

- [mise](https://mise.jdx.dev/) がインストール済み
- Docker / Docker Compose が利用可能

### 起動

```bash
# ツールチェーンのインストール
mise trust && mise install

# PostgreSQL + Loki + Grafana + Promtail 起動
mise run up

# マイグレーション
mise run migrate

# サンプルデータ投入
mise run seed

# API サーバ起動 (デフォルト :8080)
mise run server

# フロントエンド起動 (デフォルト :3001)
pnpm install
mise run dev
```

利用可能なタスク一覧: `mise tasks`

### ポート一覧（デフォルト）

| サービス | ポート |
|---------|-------|
| API サーバ | 8080 |
| フロントエンド | 3001 |
| PostgreSQL | 5432 |
| Grafana | 3000 |
| Loki | 3100 |

ポートは `mise.toml` の `[env]` セクションで変更可能。
既存サービスと競合する場合は `mise.local.toml` で上書き:

```toml
# mise.local.toml (gitignore 済み)
[env]
POSTGRES_PORT = "15432"
APP_PORT = "18080"
WEBAPP_PORT = "13001"
DATABASE_URL = "postgres://postgres:password@localhost:15432/taskman?sslmode=disable"
PSQL_DSN = "postgres://postgres:password@localhost:15432/taskman?sslmode=disable"
NEXT_PUBLIC_API_URL = "http://localhost:18080"
```

### コード再生成

```bash
mise run bobgen         # bob コード生成（DB 接続が必要）
mise run ogen           # ogen コード生成
mise run generate:api   # OpenAPI → TypeScript 型生成
```

## ディレクトリ構成

```
.
├── api/openapi.yaml          # OpenAPI スキーマ（バックエンド・フロントエンド共通）
├── bobgen.yaml               # bobgen 設定
├── cmd/
│   ├── server/main.go        # API サーバエントリポイント
│   └── seed/main.go          # サンプルデータ投入
├── compose.yaml              # Docker Compose (PostgreSQL, Loki, Grafana, Promtail)
├── db/migrations/             # dbmate マイグレーション
├── docs/                      # bob 使い方ドキュメント (11 ファイル)
├── internal/
│   ├── handler/               # ogen Handler 実装 (DTO 変換)
│   ├── infra/
│   │   ├── db/                # DB 接続、Scoped Executor
│   │   ├── dbgen/             # bob 生成コード (models, where, joins, loaders, factory, enums)
│   │   └── hook/              # QueryHooks (マルチテナント自動フィルタ)
│   ├── oas/                   # ogen 生成コード
│   └── service/               # Service 層 (bob CRUD / JOIN / queries plugin 呼び出し)
├── queries/                   # sqlc 記法の手書き SQL + bob 生成コード
├── mise.toml                  # ツールチェーン + 環境変数
└── webapp/                    # Next.js フロントエンド
    └── src/
        ├── app/               # App Router ページ (6 画面 + docs ビューア)
        ├── components/ui/     # shadcn/ui コンポーネント
        └── lib/api/           # openapi-fetch クライアント
```

## 関連リンク

- [発表スライド: GoのDB アクセスにおける型安全と柔軟性の両立 ─ bob という選択肢](https://speakerdeck.com/tak848/gonodb-akusesuniokeru-xing-an-quan-to-rou-ruan-xing-noliang-li-bob-toiuxuan-ze-zhi)
- [layerx.go #4](https://layerx.connpass.com/event/383847/)
- [bob ORM](https://github.com/stephenafamo/bob)
- [bob ドキュメント](https://bob.stephenafamo.com/)
