# bob ORM 使い方ドキュメント

本プロジェクト（マルチテナント タスク管理システム）で bob をどのように使っているかの詳細ドキュメント。
bob v0.42.1 + PostgreSQL 17 + pgx/v5 の組み合わせ。

## 目次

1. [コード生成 (bobgen)](./01-bob-usage-codegen.md)
2. [CRUD 操作](./02-bob-usage-crud.md)
3. [型安全フィルタ・ページネーション](./03-bob-usage-query.md)
4. [リレーション (Preload / ThenLoad)](./04-bob-usage-relations.md)
5. [QueryHooks によるマルチテナント分離](./05-bob-usage-hooks.md)
6. [Queries Plugin (sqlc 記法)](./06-bob-usage-queries-plugin.md)
7. [Nullable カラムの扱い (omit / omitnull / null)](./07-bob-usage-nullable.md)
8. [Factory Plugin (テストデータ生成)](./08-bob-usage-factory.md)
9. [DB 接続・Executor パターン](./09-bob-usage-executor.md)
10. [Enum 生成](./10-bob-usage-enums.md)
11. [落とし穴・注意点](./11-bob-usage-pitfalls.md)
12. [omit/omitnull vs sql.Null 比較](./12-bob-usage-typesystem-comparison.md)
