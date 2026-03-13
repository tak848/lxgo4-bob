package hook

import (
	"context"
	"fmt"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"

	infradb "github.com/tak848/lxgo4-bob/internal/infra/db"
)

// WorkspaceSelectHook returns a QueryHook that appends
// WHERE <table>.workspace_id = ? to SELECT queries.
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

// WorkspaceUpdateHook returns a QueryHook that appends
// WHERE <table>.workspace_id = ? to UPDATE queries.
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

// WorkspaceDeleteHook returns a QueryHook that appends
// WHERE <table>.workspace_id = ? to DELETE queries.
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
