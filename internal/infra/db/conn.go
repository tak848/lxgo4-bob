package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/stephenafamo/bob"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// NewDB は pgx stdlib ドライバで bob.DB を返す。
// bob.DB は bob.Executor を満たし、Close() で内部の *sql.DB も閉じる。
func NewDB(ctx context.Context, dsn string) (bob.DB, error) {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return bob.DB{}, fmt.Errorf("sql.Open: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return bob.DB{}, fmt.Errorf("ping: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)

	return bob.NewDB(sqlDB), nil
}
