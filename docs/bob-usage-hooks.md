# 5. QueryHooks によるマルチテナント分離

bob の QueryHooks を使い、SELECT / UPDATE / DELETE に自動で `WHERE workspace_id = $1` を注入する。

## 仕組みの全体像

```
1. Service が ctx に workspace_id をセット
2. Service が dbgen.Xxx.Query() を呼ぶ
3. bob が QueryHooks を実行
4. Hook が ctx から workspace_id を取り出す
5. Hook がクエリに WHERE workspace_id = $1 を追加
6. SQL が実行される
```

Service 層のコードには `WHERE workspace_id = ?` が一切書かれない。

## Hook の実装

### workspace.go — 3 種類の Hook

```go
// internal/infra/hook/workspace.go

// SELECT 用 Hook
func WorkspaceSelectHook(tableName string) bob.Hook[*dialect.SelectQuery] {
    return func(ctx context.Context, exec bob.Executor, q *dialect.SelectQuery) (context.Context, error) {
        wsID, ok := infradb.WorkspaceIDFromContext(ctx)
        if !ok {
            return ctx, fmt.Errorf("workspace_id not found in context for table %s", tableName)
        }
        q.AppendWhere(psql.Quote(tableName, "workspace_id").EQ(psql.Arg(wsID)))
        return ctx, nil
    }
}

// UPDATE 用 Hook
func WorkspaceUpdateHook(tableName string) bob.Hook[*dialect.UpdateQuery] {
    return func(ctx context.Context, exec bob.Executor, q *dialect.UpdateQuery) (context.Context, error) {
        wsID, ok := infradb.WorkspaceIDFromContext(ctx)
        if !ok {
            return ctx, fmt.Errorf("workspace_id not found in context for table %s", tableName)
        }
        q.AppendWhere(psql.Quote(tableName, "workspace_id").EQ(psql.Arg(wsID)))
        return ctx, nil
    }
}

// DELETE 用 Hook
func WorkspaceDeleteHook(tableName string) bob.Hook[*dialect.DeleteQuery] {
    return func(ctx context.Context, exec bob.Executor, q *dialect.DeleteQuery) (context.Context, error) {
        wsID, ok := infradb.WorkspaceIDFromContext(ctx)
        if !ok {
            return ctx, fmt.Errorf("workspace_id not found in context for table %s", tableName)
        }
        q.AppendWhere(psql.Quote(tableName, "workspace_id").EQ(psql.Arg(wsID)))
        return ctx, nil
    }
}
```

### 重要な API

- `bob.Hook[Q]` は `func(ctx context.Context, exec bob.Executor, q Q) (context.Context, error)` 型
- `psql.Quote(tableName, "workspace_id")` でテーブル修飾されたカラム参照を生成（`"members"."workspace_id"` 等）
- `psql.Arg(wsID)` でプレースホルダ付きの値を生成
- `q.AppendWhere(...)` で WHERE 句に条件を追加

## Hook の登録

```go
// internal/infra/hook/register.go

func RegisterHooks() {
    // members
    dbgen.Members.SelectQueryHooks.AppendHooks(WorkspaceSelectHook("members"))
    dbgen.Members.UpdateQueryHooks.AppendHooks(WorkspaceUpdateHook("members"))
    dbgen.Members.DeleteQueryHooks.AppendHooks(WorkspaceDeleteHook("members"))

    // projects
    dbgen.Projects.SelectQueryHooks.AppendHooks(WorkspaceSelectHook("projects"))
    dbgen.Projects.UpdateQueryHooks.AppendHooks(WorkspaceUpdateHook("projects"))
    dbgen.Projects.DeleteQueryHooks.AppendHooks(WorkspaceDeleteHook("projects"))

    // tasks — 同様
    // task_comments — 同様
}
```

- `dbgen.Members.SelectQueryHooks` は生成されたテーブルオブジェクトのフィールド
- `.AppendHooks()` で Hook を追加。複数追加可能（全て順番に実行される）
- **アプリ起動時に 1 回だけ呼ぶ**（`main()` の冒頭）

### 対象テーブル

| テーブル | Select Hook | Update Hook | Delete Hook |
|---------|:-----------:|:-----------:|:-----------:|
| workspaces | - | - | - |
| members | o | o | o |
| projects | o | o | o |
| tasks | o | o | o |
| task_comments | o | o | o |

`workspaces` はテナントテーブル自体なので Hook を付けない。

## Context への workspace_id の注入

```go
// internal/infra/db/scope.go

type ctxKeyWorkspaceID struct{}

func WithWorkspaceID(ctx context.Context, id uuid.UUID) context.Context {
    return context.WithValue(ctx, ctxKeyWorkspaceID{}, id)
}

func WorkspaceIDFromContext(ctx context.Context) (uuid.UUID, bool) {
    id, ok := ctx.Value(ctxKeyWorkspaceID{}).(uuid.UUID)
    return id, ok
}

func WorkspaceScopedExec(ctx context.Context, exec bob.Executor, workspaceID uuid.UUID) (context.Context, bob.Executor) {
    return WithWorkspaceID(ctx, workspaceID), exec
}

func GlobalExec(exec bob.Executor) bob.Executor {
    return exec
}
```

### 使い分け

```go
// テナントスコープ（members, projects, tasks, task_comments）
ctx, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)
rows, err := dbgen.Members.Query().All(ctx, exec)
// → SELECT * FROM members WHERE members.workspace_id = $1

// グローバル（workspaces テーブル自体）
exec := db.GlobalExec(s.exec)
rows, err := dbgen.Workspaces.Query().All(ctx, exec)
// → SELECT * FROM workspaces（Hook なし）
```

## INSERT に Hook が効かない理由と対策

bob の QueryHooks は SELECT / UPDATE / DELETE のみ。INSERT には Hook がない。

### 対策: Service 層で workspace_id を明示的にセット

```go
// internal/service/member.go:56-62
m, err := dbgen.Members.Insert(&dbgen.MemberSetter{
    ID:          omit.From(id),
    WorkspaceID: omit.From(wsID),  // ← パスパラメータの wsID を明示セット
    Name:        omit.From(name),
    Email:       omit.From(email),
    Role:        omit.From(enums.MemberRole(role)),
}).One(ctx, exec)
```

- `wsID` は HTTP パスパラメータ `/workspaces/{wsId}/members` から取得
- Service 層が **必ず** workspace_id をセットする
- DB 側でも複合 FK で cross-workspace 参照を防止

## bob.SkipHooks()

```go
rows, err := dbgen.Members.Query().All(bob.SkipHooks(ctx), exec)
```

`bob.SkipHooks(ctx)` を渡すと Hook をバイパスできる。管理者操作やバッチ処理で使う想定。
本プロジェクトでは使用していないが、存在は把握しておくべき。

## QueryHooks vs ContextualMods

bob にはもう 1 つの方法 **ContextualMods** がある。

| | QueryHooks | ContextualMods |
|---|---|---|
| 適用タイミング | クエリ実行直前 | クエリビルド時 |
| 対象 | SELECT/UPDATE/DELETE | SELECT/INSERT/UPDATE/DELETE |
| バイパス | `bob.SkipHooks(ctx)` | Context を制御 |
| 適用範囲 | グローバル（テーブル単位） | Executor 単位 |

本プロジェクトではデモの簡潔さを優先して QueryHooks を採用。
ContextualMods は INSERT にも適用できるメリットがあるが、設定がやや複雑。
