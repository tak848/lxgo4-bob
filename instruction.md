`CLAUDE.local.md`` を見てもらいつつ、add-dirした参考実装をもとに、実際にpostgresql使ってbobのアプリ作って。
    そこそこのjoinもありつつ、bobの生成boilerもsqlcもどちらも使う。openapiからogenでスキーマ自動生成。認証認可は不要。bobがメインなので。
  go 1.24で出たgo toolを使うこと。migrationはdbmateかgoose。postgresはusernameがpostgres, passwordはpassword。全部ベタ打ちでよい。go tool使う。agent
  teamと適宜分担。とりあえずスキーマ駆動は守ること。そしてbobのCRUD・joinや、複雑な型安全クエリビルドをできるようにしつつ、そしてsqlc記法てきなやつもサンプルでいれてもらいつつ、アプリケーションの題材は超一般的な者とすること。ただ、tenantid的なhookとか見越して、何かpartition的なキーが作れると良いのかもしれない。ログは雑にgrafanaで見られ
