# 12. omit/omitnull vs sql.Null 比較

bob は `bobgen.yaml` の `type_system` 設定で、Nullable 型の実装を切り替えられる。

```yaml
# bobgen.yaml
type_system: "github.com/aarondl/opt"  # デフォルト
# or
type_system: "database/sql"
```

本プロジェクトはデフォルト（`aarondl/opt`）を使用。この章ではもう一方の `database/sql` 方式との比較と、どちらを選ぶべきかを整理する。

## 根本的な問題: 3 状態の表現

DB の NULLABLE カラムを更新する際、アプリケーション側では **3 つの状態** を区別する必要がある:

1. **値をセット**: `UPDATE SET col = 'hello'`
2. **NULL をセット**: `UPDATE SET col = NULL`
3. **更新しない**: SET 句に含めない（現在の値を維持）

Go の標準的な型（`string`, `*string`, `sql.NullString`）では、この 3 状態を **1 つの型で** 表現するのが難しい。

### なぜ `sql.Null[T]` 単体では足りないか

```go
type sql.Null[T] struct {
    V     T
    Valid bool
}
```

| Valid | V | 意味 |
|-------|---|------|
| true | "hello" | 値あり |
| false | ゼロ値 | NULL |

2 状態しかない。「更新しない」を表現できない。
`sql.Null[T]` のゼロ値は `{V: ゼロ値, Valid: false}` = NULL と同じ。
**「明示的に NULL にした」と「何も指定していない」の区別がつかない。**

### なぜ `*string` 単体でも足りないか

```go
var p *string  // nil = NULL? or nil = 更新しない?
```

nil が「NULL」と「未指定」の両方に使われてしまう。

## Go における Nullable 表現の歴史と選択肢

bob の方式を理解するために、Go で「値がないかもしれない」を表現する方法を整理する。

### ゼロ値方式

```go
type User struct {
    Name string  // "" はゼロ値。空文字なのか未設定なのか区別できない
    Age  int     // 0 はゼロ値。0歳なのか未設定なのか区別できない
}
```

- Go の全ての型にゼロ値がある（`""`, `0`, `false`, `uuid.Nil` 等）
- DB の NULL とゼロ値の区別がつかない
- gorm はこの方式を基本とするため、ゼロ値問題（後述）が発生する

### ポインタ方式 (`*T`)

```go
type User struct {
    Name *string  // nil = NULL, &"hello" = 値あり
    Age  *int     // nil = NULL, &0 = 0（ゼロ値と NULL を区別可能）
}
```

- nil = NULL、非 nil = 値あり で 2 状態を表現
- ゼロ値と NULL の区別がつく（`*int` で 0 と nil は別）
- デメリット:
  - 値のアクセスに毎回 nil チェックとデリファレンスが必要
  - ポインタのため GC 負荷が微増
  - JSON で `omitempty` と組み合わせると「フィールドなし」と「null」の区別が曖昧

### 基底型方式 (`sql.NullString` 等)

Go 1.22 より前の標準ライブラリ。型ごとに専用の構造体が用意されていた。

```go
// Go 1.21 以前
type NullString struct {
    String string
    Valid  bool
}
type NullInt64 struct {
    Int64 int64
    Valid bool
}
type NullBool struct { ... }
type NullFloat64 struct { ... }
type NullTime struct { ... }
// ... 型ごとに個別定義
```

- `Valid=true` で値あり、`Valid=false` で NULL
- **ジェネリクスがない**: `uuid.UUID` や独自型に対応する `NullXxx` が存在しない
- sqlx や gorm はこの方式を長く使ってきた
- `NullString` のフィールド名は `String`、`NullInt64` は `Int64` — 一貫性がない

### ジェネリック方式 (`sql.Null[T]`)

Go 1.22 で追加。基底型方式をジェネリクスで統一。

```go
// Go 1.22+
type Null[T any] struct {
    V     T
    Valid bool
}

// 使い方
var id sql.Null[uuid.UUID]   // あらゆる型で使える
var name sql.Null[string]    // NullString の代替
var age sql.Null[int]        // NullInt64 の代替
```

- 任意の型に対応（`sql.Null[uuid.UUID]` も OK）
- フィールド名が統一（常に `.V` と `.Valid`）
- ただし前述の通り 2 状態しかない（NULL / 値）

### pgtype 方式（pgx 独自）

pgx は独自の Nullable 型を持つ。sqlc + pgx で使われる。

```go
// github.com/jackc/pgx/v5/pgtype
type Text struct {
    String string
    Valid  bool
}
type UUID struct {
    Bytes [16]byte
    Valid bool
}
type Int4 struct {
    Int32 int32
    Valid bool
}
```

- PostgreSQL 固有の型に最適化（`pgtype.Numeric`, `pgtype.Interval` 等）
- `sql.NullXxx` と似た構造だが、PostgreSQL 向けに拡張
- bob では pgtype を直接使わず、aarondl/opt または sql.Null で抽象化する

### aarondl/opt 方式（bob デフォルト）

前述の全ての問題を解決するために設計された型セット。

```go
null.Val[T]      // 2 状態: 値 / NULL（モデル用）
omit.Val[T]      // 2 状態: 値 / 未セット（Setter の NOT NULL 用）
omitnull.Val[T]  // 3 状態: 値 / NULL / 未セット（Setter の NULLABLE 用）
```

### 方式の比較まとめ

| 方式 | 型の例 | NULL 表現 | 未セット表現 | ゼロ値と NULL の区別 | 任意型対応 |
|------|-------|----------|------------|-------------------|----------|
| ゼロ値 | `string` | 不可 | 不可 | 不可 | - |
| ポインタ | `*string` | `nil` | `nil`（区別不可） | 可 | 可 |
| 基底型 | `sql.NullString` | `Valid=false` | 不可 | 可 | 不可（型ごとに定義必要） |
| ジェネリック | `sql.Null[T]` | `Valid=false` | 不可 | 可 | 可 |
| pgtype | `pgtype.Text` | `Valid=false` | 不可 | 可 | PostgreSQL 型のみ |
| aarondl/opt | `omitnull.Val[T]` | `.Null()` | ゼロ値 | 可 | 可 |
| ポインタ+Null | `*sql.Null[T]` | `&{Valid:false}` | `nil` | 可 | 可 |

## aarondl/opt の内部構造

bob デフォルトの型がどう実装されているか。ソースは `github.com/aarondl/opt`。

### `omit.Val[T]` の実体

```go
// omit/omit.go
type state int

const (
    StateUnset state = 0  // ゼロ値 = Unset（安全）
    StateSet   state = 1
)

type Val[T any] struct {
    value T      // 非公開フィールド（直接アクセス不可）
    state state  // 非公開フィールド
}
```

**ポイント**: `value` と `state` は非公開。直接代入できないため、必ずコンストラクタ関数を経由する。
ゼロ値 `Val[T]{}` は `state=0=StateUnset` → 安全に「何もしない」を表現。

### `null.Val[T]` の実体

```go
// null/null.go
const (
    StateNull state = 0  // ゼロ値 = Null
    StateSet  state = 1
)

type Val[T any] struct {
    value T
    state state
}
```

`omit.Val` と同じ構造だが、ゼロ値の意味が違う。`state=0=StateNull`。

### `omitnull.Val[T]` の実体

```go
// omitnull/omitnull.go
const (
    StateUnset state = 0  // ゼロ値 = Unset
    StateNull  state = 1
    StateSet   state = 2
)

type Val[T any] struct {
    value T
    state state
}
```

3 状態を `int` の 0/1/2 で表現。ゼロ値は `StateUnset`。

### 主要な関数・メソッド一覧

3 つの型で共通の API パターン（`omitnull.Val[T]` を例示、他も同様）:

#### コンストラクタ

```go
omitnull.From(val T) Val[T]          // 値をセット（StateSet）
omitnull.FromPtr(val *T) Val[T]      // nil → Null, 非nil → Set
omitnull.FromNull(val null.Val[T]) Val[T]   // null.Val → omitnull.Val（ロスレス変換）
omitnull.FromOmit(val omit.Val[T]) Val[T]   // omit.Val → omitnull.Val（ロスレス変換）
```

#### 値の取得

```go
v.Get() (T, bool)         // 値と存在フラグ。Set なら (値, true)、それ以外は (ゼロ値, false)
v.GetOr(fallback T) T     // Set なら値、それ以外は fallback
v.GetOrZero() T           // Set なら値、それ以外はゼロ値
v.MustGet() T             // Set なら値、それ以外は panic
```

#### 状態の取得

```go
v.IsValue() bool    // StateSet か（omit.Val では IsValue、null.Val でも IsValue）
v.IsNull() bool     // StateNull か（null.Val, omitnull.Val のみ）
v.IsUnset() bool    // StateUnset か（omit.Val, omitnull.Val のみ）
v.State() state     // 内部状態を直接取得（テスト用）
```

#### 状態の変更（ミュータブル）

```go
v.Set(val T)    // 値をセット（→ StateSet）
v.Null()        // NULL にする（→ StateNull）（null.Val, omitnull.Val のみ）
v.Unset()       // 未セットにする（→ StateUnset）（omit.Val, omitnull.Val のみ）
```

#### 変換・合成

```go
v.Or(other Val[T]) Val[T]     // v が未セット/NULL なら other を返す（優先度: Set > Null > Unset）
v.Map(fn func(T) T) Val[T]    // Set なら fn を適用、それ以外はそのまま
Map[A, B](v Val[A], fn func(A) B) Val[B]  // 型を変換する Map（トップレベル関数）

// omitnull 固有
v.GetNull() (null.Val[T], bool)     // omitnull → null に変換（Unset なら false）
v.GetOmit() (omit.Val[T], bool)     // omitnull → omit に変換（Null なら false）
v.MustGetNull() null.Val[T]         // Unset なら panic
v.MustGetOmit() omit.Val[T]         // Null なら panic
```

#### 実装しているインターフェース

| インターフェース | omit.Val | null.Val | omitnull.Val | 備考 |
|----------------|:--------:|:--------:|:------------:|------|
| `json.Marshaler` | o | o | o | Unset → omit（フィールドごと省略）、Null → `null`、Set → 値 |
| `json.Unmarshaler` | o | o | o | `null` → Null/エラー（omit は null 受け取り不可）、値 → Set |
| `encoding.TextMarshaler` | o | o | o | |
| `encoding.TextUnmarshaler` | o | o | o | |
| `encoding.BinaryMarshaler` | o | o | o | |
| `encoding.BinaryUnmarshaler` | o | o | o | |
| `sql.Scanner` | o | o | o | DB から読み取り。NULL → Null（omit は NULL をエラーにする） |
| `driver.Valuer` | o | o | o | DB に書き込み。Unset/Null → `nil`、Set → 値 |
| `fmt.Stringer` | - | - | - | 未実装（state のみ Stringer あり） |

**`omit.Val` の特殊挙動**: `Scan(nil)` と `UnmarshalJSON(null)` はエラーを返す。
NOT NULL カラムに対応するため、NULL を受け取ること自体がバグとみなされる。

### `sql.Null[T]` との構造比較

```go
// 標準ライブラリ
type sql.Null[T any] struct {
    V     T       // 公開フィールド（直接アクセス可能）
    Valid bool    // 公開フィールド
}

// aarondl/opt
type omitnull.Val[T any] struct {
    value T       // 非公開フィールド（直接アクセス不可）
    state state   // 非公開フィールド
}
```

| 観点 | `sql.Null[T]` | `omitnull.Val[T]` |
|------|--------------|-------------------|
| フィールドの可視性 | 公開 (`V`, `Valid`) | 非公開 (`value`, `state`) |
| 値の設定方法 | 直接代入 `v.V = x; v.Valid = true` | メソッド経由 `v.Set(x)` or `From(x)` |
| 不正状態の可能性 | あり（`Valid=true` で `V` がゼロ値） | なし（メソッドが整合性を保証） |
| JSON 対応 | 未実装（`{"V": ..., "Valid": ...}`） | 実装済み（値 or `null`） |
| sql.Scanner | 実装済み | 実装済み |
| driver.Valuer | 実装済み | 実装済み |
| 状態数 | 2（Valid / !Valid） | 3（Set / Null / Unset） |

## bob の 2 つの解決策

### 方式 A: `aarondl/opt`（デフォルト）

専用の 3 状態型を導入する。

| 型 | 使用場所 | 状態 |
|---|---|---|
| `null.Val[T]` | モデルのフィールド | 値 / NULL |
| `omit.Val[T]` | Setter の NOT NULL フィールド | 値 / 未セット |
| `omitnull.Val[T]` | Setter の NULLABLE フィールド | 値 / NULL / 未セット |

生成例:

```go
// モデル
type Task struct {
    AssigneeID null.Val[uuid.UUID]  // DB から読み出し: 値 or NULL
}

// Setter
type TaskSetter struct {
    Title      omit.Val[string]          // NOT NULL: 値 or 未セット
    AssigneeID omitnull.Val[uuid.UUID]   // NULLABLE: 値 or NULL or 未セット
}
```

使い方:

```go
// 値をセット
setter.AssigneeID = omitnull.From(someUUID)

// NULL をセット
var v omitnull.Val[uuid.UUID]
v.Null()
setter.AssigneeID = v

// 未セット（ゼロ値のまま何もしない）
// → SET 句に含まれない
```

### 方式 B: `database/sql`

標準ライブラリの型 + ポインタで 3 状態を表現する。

| 型 | 使用場所 | 状態 |
|---|---|---|
| `sql.Null[T]` | モデルの NULLABLE フィールド | 値 / NULL |
| `*T` | Setter の NOT NULL フィールド | 値 / 未セット (nil) |
| `*sql.Null[T]` | Setter の NULLABLE フィールド | 値 / NULL / 未セット (nil) |

生成例:

```go
// モデル
type Task struct {
    AssigneeID sql.Null[uuid.UUID]  // DB から読み出し: 値 or NULL
}

// Setter
type TaskSetter struct {
    Title      *string                   // NOT NULL: 値 or 未セット (nil)
    AssigneeID *sql.Null[uuid.UUID]      // NULLABLE: 値 or NULL or 未セット (nil)
}
```

使い方:

```go
// 値をセット
setter.AssigneeID = &sql.Null[uuid.UUID]{V: someUUID, Valid: true}

// NULL をセット
setter.AssigneeID = &sql.Null[uuid.UUID]{Valid: false}

// 未セット（nil のまま）
// → SET 句に含まれない
```

## 3 状態の表現比較

| 操作 | aarondl/opt | database/sql |
|------|-------------|-------------|
| 値セット | `omitnull.From(v)` | `&sql.Null[T]{V: v, Valid: true}` |
| NULL セット | `var v omitnull.Val[T]; v.Null()` | `&sql.Null[T]{Valid: false}` |
| 未セット | ゼロ値（何もしない） | `nil`（何もしない） |
| 値の読み出し | `v.MustGet()` | `v.V` |
| NULL 判定 | `v.IsNull()` | `!v.Valid` |
| 未セット判定 | `v.IsUnset()` | `v == nil`（ポインタ） |

## 何もセットしなかった場合の挙動

**Setter を作って何もフィールドをセットしなかった場合、各方式でどうなるか。**

これが最も重要な違いであり、最も事故が起きやすいポイント。

### aarondl/opt 方式

```go
setter := &dbgen.TaskSetter{}
// 全フィールドが omitnull.Val[T] / omit.Val[T] のゼロ値 = Unset

// INSERT の場合:
dbgen.Tasks.Insert(setter).One(ctx, exec)
// → INSERT INTO tasks DEFAULT VALUES
// → 全カラムに DB の DEFAULT 値が使われる
// → NOT NULL かつ DEFAULT なしのカラムは DB エラー

// UPDATE の場合:
task.Update(ctx, exec, setter)
// → UPDATE tasks SET WHERE id = $1
// → SET 句が空 → 何も更新しない（エラーにはならない）
```

**安全**: 何もしなければ何も起きない。意図しない上書きがない。

### database/sql 方式

```go
setter := &dbgen.TaskSetter{}
// 全フィールドが *T / *sql.Null[T] の nil = Unset

// INSERT / UPDATE の挙動は aarondl/opt と同じ
// nil = SET 句に含まれない
```

**安全**: こちらもポインタの nil = 未セットなので同等。

### gorm 方式（参考: bob を使わない場合）

```go
type Task struct {
    Title    string
    Priority string
    Age      int
}

task := Task{}
db.Create(&task)
// → INSERT INTO tasks (title, priority, age) VALUES ('', '', 0)
// → 全フィールドにゼロ値が入る！空文字と 0 が DB に入ってしまう

db.Model(&existing).Updates(Task{Priority: ""})
// → UPDATE は何もしない（gorm がゼロ値を無視）
// → "" に更新したいのに更新されない！
```

**危険**: ゼロ値が「値」として扱われる場合と「未セット」として無視される場合があり、予測が困難。

### 具体的な事故パターン

```go
// aarondl/opt: 事故が起きない
setter := &dbgen.TaskSetter{
    Title: omit.From("Fix bug"),
    // Priority を指定し忘れた
}
task.Update(ctx, exec, setter)
// → UPDATE tasks SET title = 'Fix bug' WHERE id = $1
// → Priority は変更されない（意図通り）

// gorm: 事故が起きる
db.Model(&task).Updates(Task{
    Title: "Fix bug",
    // Priority を指定し忘れた → ゼロ値 ""
})
// → UPDATE tasks SET title = 'Fix bug', priority = '' WHERE id = $1
// → Priority が空文字で上書きされてしまう！（gorm の設定次第）
```

bob の omit/omitnull 方式と database/sql ポインタ方式は、この問題を **型レベルで完全に解決** している。
フィールドを指定しなければ絶対に SQL に含まれない。明示的に `omit.From()` や `&値` をしない限り更新されない。

## 詳細比較

### 1. 型の明示性

**aarondl/opt の方が良い。**

```go
// aarondl/opt: 型名から意図が明確
AssigneeID omitnull.Val[uuid.UUID]  // "nullable かつ省略可能"

// database/sql: ポインタの入れ子で意図が読みにくい
AssigneeID *sql.Null[uuid.UUID]     // "nullable のポインタ...?"
```

`omitnull` という名前自体が「omit（省略）と null の両方を表現する」と語っている。
`*sql.Null[T]` は「ポインタの nil = 省略」というコンベンションを知らないと意図が分からない。

### 2. ゼロ値の安全性

**aarondl/opt の方が安全。**

```go
// aarondl/opt: ゼロ値 = 未セット（安全）
var setter TaskSetter
// setter.AssigneeID は Unset → SET 句に含まれない → 現在値を維持

// database/sql: ゼロ値 = nil（安全）
var setter TaskSetter
// setter.AssigneeID は nil → SET 句に含まれない → 現在値を維持
```

ゼロ値の安全性は同等。ただし、以下のケースで差が出る:

```go
// aarondl/opt: コンパイル時に型チェックが効く
setter.AssigneeID = someUUID  // コンパイルエラー！omitnull.Val[T] に UUID は直接入らない
setter.AssigneeID = omitnull.From(someUUID)  // OK

// database/sql: ポインタ操作で事故りやすい
setter.AssigneeID = &someUUID  // コンパイルエラー！*sql.Null[T] に *UUID は入らない
// ↑ この間違いは検出されるが、以下は見逃す:
nullVal := sql.Null[uuid.UUID]{}  // Valid=false = NULL
setter.AssigneeID = &nullVal       // 意図せず NULL をセットしてしまう？
```

### 3. 標準ライブラリとの親和性

**database/sql の方が良い。**

- `sql.Null[T]` は Go 1.22 で追加された標準型
- 他のライブラリやフレームワークとの相互運用が容易
- 外部依存パッケージが不要
- チームメンバーが既に知っている可能性が高い

aarondl/opt は:
- bob 専用の外部パッケージ（`github.com/aarondl/opt`）
- bob を使わないコードとの境界で変換が必要
- 学習コストがある

### 4. JSON シリアライゼーション

**aarondl/opt の方が良い。**

```go
// aarondl/opt: null.Val[T] は JSON null を正しく扱う
// {"assignee_id": null} → null.Val[T]{IsNull: true}
// {"assignee_id": "xxx"} → null.Val[T]{Val: "xxx"}
// フィールドなし → omitnull ならデコード時に Unset のまま

// database/sql: sql.Null[T] の JSON 対応は自前実装が必要
// sql.Null[T] は database/sql.Scanner と driver.Valuer だけ実装
// json.Marshaler / json.Unmarshaler は実装していない
// → {"assignee_id": {"V": "xxx", "Valid": true}} になってしまう
```

API レスポンスで `sql.Null[T]` を直接 JSON に出すと `{"V": "...", "Valid": true}` という構造になる。
`null.Val[T]` は `MarshalJSON` / `UnmarshalJSON` を実装済みで、`null` や値がそのまま出力される。

### 5. NULL セットの記述量

**database/sql の方がやや簡潔。**

```go
// aarondl/opt: NULL セットが冗長
var nullVal omitnull.Val[uuid.UUID]
nullVal.Null()
setter.AssigneeID = nullVal
// 3 行必要

// database/sql: 1 行で書ける
setter.AssigneeID = &sql.Null[uuid.UUID]{}
```

### 6. gorm / sqlx / sqlc との比較

| ORM/ツール | Nullable 型 | Optional (Setter) 型 | 3 状態対応 |
|-----------|------------|---------------------|-----------|
| gorm | `*T` / `sql.NullXxx` | 構造体のゼロ値判定 | 不完全（ゼロ値と未セットの区別不可） |
| sqlx | `sql.NullXxx` / `*T` | N/A（手書き SQL） | N/A |
| sqlc | `sql.NullXxx` / `pgtype.T` | N/A（手書き SQL） | N/A |
| ent | `*T` | `Optional()` メソッド | フィールドごとの Set/Clear メソッド |
| bob (opt) | `null.Val[T]` | `omitnull.Val[T]` | 完全（型レベルで 3 状態） |
| bob (sql) | `sql.Null[T]` | `*sql.Null[T]` | 完全（ポインタ + Valid で 3 状態） |

gorm の問題点:
```go
// gorm: int のカラムを 0 にしたい場合
db.Model(&user).Update("age", 0)
// → gorm はゼロ値を「未セット」と判断して UPDATE しない場合がある
// → Select("age") で明示指定するか、*int を使う必要がある
```

bob はどちらの type_system でも **型レベルで 3 状態を表現** しているため、gorm のようなゼロ値問題は発生しない。

## 結論: どちらを選ぶか

### `aarondl/opt`（デフォルト）を選ぶべき場合

- bob 中心のプロジェクト（他の ORM と混在しない）
- API サーバで JSON レスポンスに nullable フィールドがある
- 型の意図を明示的にしたい（`omitnull` という名前の自己文書化効果）
- チーム全体が bob を使い込む前提

### `database/sql` を選ぶべき場合

- 既存プロジェクトへの bob 導入（`sql.Null[T]` が既に使われている）
- 外部ライブラリとの相互運用が多い
- 外部依存を減らしたい
- チームメンバーの学習コストを最小化したい

### 本プロジェクトの選択理由

デフォルト（`aarondl/opt`）を採用。理由:

1. bob のデモアプリなので、bob の推奨方式を使う
2. ogen が独自の型を生成するため、handler 層で変換が必要 → どちらの方式でも変換コストは同じ
3. `omitnull.Val[T]` の型名が自己文書化的で、bob の設計思想を理解しやすい
4. JSON シリアライゼーションで `sql.Null[T]` の `{"V": ..., "Valid": ...}` 問題を避けられる

## bobgen.yaml での切り替え方

```yaml
# aarondl/opt（デフォルト、省略可）
type_system: "github.com/aarondl/opt"

# database/sql に切り替える場合
type_system: "database/sql"
```

設定変更後に `mise run bobgen` で再生成すると、全モデル・Setter の型が切り替わる。
