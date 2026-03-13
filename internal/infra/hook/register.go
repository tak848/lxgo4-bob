package hook

import (
	"github.com/tak848/lxgo4-bob-playground/internal/infra/dbgen"
)

// RegisterHooks registers workspace-scoping query hooks on all tenant-scoped tables.
// This must be called once at application startup (typically in main).
func RegisterHooks() {
	// members
	dbgen.Members.SelectQueryHooks.AppendHooks(WorkspaceSelectHook("members"))
	dbgen.Members.UpdateQueryHooks.AppendHooks(WorkspaceUpdateHook("members"))
	dbgen.Members.DeleteQueryHooks.AppendHooks(WorkspaceDeleteHook("members"))

	// projects
	dbgen.Projects.SelectQueryHooks.AppendHooks(WorkspaceSelectHook("projects"))
	dbgen.Projects.UpdateQueryHooks.AppendHooks(WorkspaceUpdateHook("projects"))
	dbgen.Projects.DeleteQueryHooks.AppendHooks(WorkspaceDeleteHook("projects"))

	// tasks
	dbgen.Tasks.SelectQueryHooks.AppendHooks(WorkspaceSelectHook("tasks"))
	dbgen.Tasks.UpdateQueryHooks.AppendHooks(WorkspaceUpdateHook("tasks"))
	dbgen.Tasks.DeleteQueryHooks.AppendHooks(WorkspaceDeleteHook("tasks"))

	// task_comments
	dbgen.TaskComments.SelectQueryHooks.AppendHooks(WorkspaceSelectHook("task_comments"))
	dbgen.TaskComments.UpdateQueryHooks.AppendHooks(WorkspaceUpdateHook("task_comments"))
	dbgen.TaskComments.DeleteQueryHooks.AppendHooks(WorkspaceDeleteHook("task_comments"))
}
