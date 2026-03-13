# bob ORM 使い方ドキュメント

本プロジェクト（マルチテナント タスク管理システム）で bob をどのように使っているかの詳細ドキュメント。
bob v0.42.1 + PostgreSQL 17 + pgx/v5 の組み合わせ。

## 目次

1. [コード生成 (bobgen)](./bob-usage-codegen.md)
2. [CRUD 操作](./bob-usage-crud.md)
3. [型安全フィルタ・ページネーション](./bob-usage-query.md)
4. [リレーション (Preload / ThenLoad)](./bob-usage-relations.md)
5. [QueryHooks によるマルチテナント分離](./bob-usage-hooks.md)
6. [Queries Plugin (sqlc 記法)](./bob-usage-queries-plugin.md)
7. [Nullable カラムの扱い (omit / omitnull / null)](./bob-usage-nullable.md)
8. [Factory Plugin (テストデータ生成)](./bob-usage-factory.md)
9. [DB 接続・Executor パターン](./bob-usage-executor.md)
10. [Enum 生成](./bob-usage-enums.md)
11. [落とし穴・注意点](./bob-usage-pitfalls.md)
12. [omit/omitnull vs sql.Null 比較](./bob-usage-typesystem-comparison.md)
