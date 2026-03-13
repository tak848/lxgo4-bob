package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/stephenafamo/bob"
)

type ctxKeyWorkspaceID struct{}

// WithWorkspaceID sets the workspace ID in the context.
func WithWorkspaceID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxKeyWorkspaceID{}, id)
}

// WorkspaceIDFromContext retrieves the workspace ID from the context.
// Returns uuid.Nil and false if not set.
func WorkspaceIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ctxKeyWorkspaceID{}).(uuid.UUID)
	return id, ok
}

// WorkspaceScopedExec returns an executor with workspace context set.
// Hooks will read the workspace ID from context to filter queries.
func WorkspaceScopedExec(ctx context.Context, exec bob.Executor, workspaceID uuid.UUID) (context.Context, bob.Executor) {
	return WithWorkspaceID(ctx, workspaceID), exec
}

// GlobalExec returns the executor as-is (no workspace scoping).
// Used for workspace-level CRUD where tenant filtering is not needed.
func GlobalExec(exec bob.Executor) bob.Executor {
	return exec
}
