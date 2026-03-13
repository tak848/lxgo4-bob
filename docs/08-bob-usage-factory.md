# 8. Factory Plugin (テストデータ生成)

bob の factory プラグインは、テストデータ生成用のコードを自動生成する。

## 生成される場所

```
internal/infra/dbgen/factory/
├── bobfactory_main.bob.go       # Factory 本体
├── bobfactory_random.bob.go     # ランダム値生成
├── members.bob.go               # Member テンプレート
├── projects.bob.go              # Project テンプレート
├── tasks.bob.go                 # Task テンプレート
├── task_comments.bob.go         # TaskComment テンプレート
└── workspaces.bob.go            # Workspace テンプレート
```

## 基本的な使い方

```go
f := factory.New()

// Workspace を 1 件作成して DB に INSERT
ws, err := f.NewWorkspace(
    factory.WorkspaceMods.ID(wsID),
    factory.WorkspaceMods.Name("Engineering"),
).Create(ctx, bobDB)
```

- `f.NewXxx(mods...)` でテンプレートを作成
- `.Create(ctx, exec)` で DB に INSERT して結果を返す
- `.Build()` なら INSERT せずモデルだけ返す（テスト用）

## Factory の動作原理

### テンプレートとモディファイア

```go
// テンプレート構造体（生成コード）
type TaskTemplate struct {
    ID          func() uuid.UUID
    WorkspaceID func() uuid.UUID
    Title       func() string
    // ...
    r taskR  // リレーションテンプレート
}

// モディファイア
type taskMods struct{}
var TaskMods taskMods

func (taskMods) ID(v uuid.UUID) TaskMod {
    return taskModFunc(func(ctx context.Context, o *TaskTemplate) {
        o.ID = func() uuid.UUID { return v }
    })
}
```

- 各フィールドは `func() T` 型（遅延評価）
- `TaskMods.ID(v)` でフィールドをセット
- 未セットのフィールドは `ensureCreatableXxx` でランダム値が入る

### リレーションの自動生成

```go
// tasks.bob.go:Create メソッド（生成コード・要約）
func (o *TaskTemplate) Create(ctx context.Context, exec bob.Executor) (*models.Task, error) {
    opt := o.BuildSetter()
    ensureCreatableTask(opt)

    // Member リレーションがなければ自動生成
    if o.r.Member == nil {
        TaskMods.WithNewMember().Apply(ctx, o)
    }
    rel1, err = o.r.Member.o.Create(ctx, exec)
    opt.WorkspaceID = omit.From(rel1.WorkspaceID)  // ← 上書き！
    opt.AssigneeID = omitnull.From(rel1.ID)         // ← 上書き！

    // Workspace も同様に自動生成・上書き
    // Project も同様
    // ...

    m, err := models.Tasks.Insert(opt).One(ctx, exec)
    return m, err
}
```

## 重要な落とし穴

**Factory の `Create` メソッドは、リレーション経由で `workspace_id` 等を上書きする。**

つまり以下のコードは**期待通りに動かない**:

```go
// NG: AssigneeID と WorkspaceID を直接指定しても上書きされる
f.NewTask(
    factory.TaskMods.ID(tID),
    factory.TaskMods.WorkspaceID(wsID),      // ← 上書きされる
    factory.TaskMods.AssigneeID(null.From(memberID)), // ← 上書きされる
    factory.TaskMods.ProjectID(pID),
).Create(ctx, bobDB)
```

Factory の Create は:
1. `o.r.Member == nil` なので新しい Member を自動生成
2. 新 Member の ID と WorkspaceID で **上書き**
3. 結果: 意図しない Member が作られ、FK 制約違反になる可能性

### 正しい使い方: リレーションモディファイアを使う

```go
f.NewTask(
    factory.TaskMods.WithExistingMember(existingMemberTemplate),
    factory.TaskMods.WithExistingProject(existingProjectTemplate),
).Create(ctx, bobDB)
```

### 本プロジェクトでの対応: seed は bob 直接 Insert を使う

Factory のリレーション自動生成が複雑なため、seed では factory を使わず bob の直接 Insert を使用:

```go
// cmd/seed/main.go — Factory ではなく直接 Insert
dbgen.Tasks.Insert(&dbgen.TaskSetter{
    ID:          omit.From(tID),
    WorkspaceID: omit.From(wsID),
    ProjectID:   omit.From(pID),
    AssigneeID:  omitnull.From(assigneeID),
    Title:       omit.From("Task 1"),
}).One(ctx, bobDB)
```

## Factory が適している場面

- **テストコード**: 1 件だけ作って検証する場合（リレーション先も自動生成されて便利）
- **フィクスチャ**: リレーション構造を気にせず「それっぽいデータ」が欲しい場合

## Factory が適さない場面

- **Seed データ**: 既存のリレーション先を参照する場合（上書き問題）
- **マルチテナント**: workspace_id の制御が必要な場合
- **複合 FK**: Factory が FK の整合性を壊す可能性がある
