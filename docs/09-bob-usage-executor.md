# 9. DB 接続・Executor パターン

## bob.Executor と bob.DB

bob の中心的な抽象は `bob.Executor` インターフェース:

```go
// bob パッケージで定義
type Executor interface {
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}
```

`*sql.DB` と `*sql.Tx` の両方がこのインターフェースを満たす。

`bob.DB` は `bob.NewDB(sqlDB)` で作成する構造体で、`bob.Executor` を満たしつつ
トランザクション開始のメソッドも持つ。

## DB 接続の初期化

```go
// internal/infra/db/conn.go
func NewDB(ctx context.Context, dsn string) (bob.DB, *pgxpool.Pool, error) {
    // pgx のコネクションプールを作成
    pool, err := pgxpool.New(ctx, dsn)
    if err != nil {
        return bob.DB{}, nil, fmt.Errorf("pgxpool.New: %w", err)
    }

    // 接続確認
    if err := pool.Ping(ctx); err != nil {
        pool.Close()
        return bob.DB{}, nil, fmt.Errorf("pool.Ping: %w", err)
    }

    // pgx pool → database/sql の *sql.DB に変換
    sqlDB := stdlib.OpenDBFromPool(pool)

    // bob.DB にラップ
    bobDB := bob.NewDB(sqlDB)

    return bobDB, pool, nil
}
```

### なぜ pgxpool → stdlib → bob.DB なのか

```
pgxpool.Pool (pgx native)
    ↓ stdlib.OpenDBFromPool()
*sql.DB (database/sql 互換)
    ↓ bob.NewDB()
bob.DB (bob.Executor 実装)
```

bob は `database/sql` の `*sql.DB` を前提とする。
pgx の native API（`pgx.Conn`）を直接使うことはできない。
`stdlib.OpenDBFromPool` で pgxpool のコネクションプールを `database/sql` 互換にする。

pgxpool を直接返すのは `defer pool.Close()` のため。

## Service への Executor 注入

```go
// cmd/server/main.go
bobDB, pool, err := infradb.NewDB(ctx, dsn)
defer pool.Close()

h := &handler.Handler{
    Workspaces: service.NewWorkspaceService(bobDB),
    Members:    service.NewMemberService(bobDB),
    // ...
}
```

```go
// internal/service/member.go
type MemberService struct {
    exec bob.Executor
}

func NewMemberService(exec bob.Executor) *MemberService {
    return &MemberService{exec: exec}
}
```

- 各 Service は `bob.Executor` を保持
- 生成された CRUD 関数に `exec` を渡す
- `bob.DB` は `bob.Executor` を満たすのでそのまま渡せる

## Scoped Executor パターン

```go
// internal/infra/db/scope.go

// テナントスコープ: context に workspace_id をセット
func WorkspaceScopedExec(ctx context.Context, exec bob.Executor, workspaceID uuid.UUID) (context.Context, bob.Executor) {
    return WithWorkspaceID(ctx, workspaceID), exec
}

// グローバル: そのまま返す
func GlobalExec(exec bob.Executor) bob.Executor {
    return exec
}
```

### 使い分け

```go
// テナントスコープ（QueryHooks が workspace_id を自動フィルタ）
ctx, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)
rows, err := dbgen.Members.Query().All(ctx, exec)

// グローバル（workspaces テーブル自体の操作）
exec := db.GlobalExec(s.exec)
rows, err := dbgen.Workspaces.Query().All(ctx, exec)
```

ポイント: `WorkspaceScopedExec` は **exec 自体は変えない**。
context に workspace_id を入れるだけ。
Hook が context から workspace_id を読み取る。

## トランザクション

bob.DB は `BeginTx` メソッドを持つ:

```go
tx, err := bobDB.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

// tx は bob.Executor を満たす
_, err = dbgen.Tasks.Insert(setter).One(ctx, tx)
if err != nil {
    return err
}

return tx.Commit()
```

本プロジェクトではシンプルさのためトランザクションは使っていないが、
Service メソッド内で複数テーブルを操作する場合はトランザクションを使うべき。
