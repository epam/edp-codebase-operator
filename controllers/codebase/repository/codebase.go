package repository

import "context"

// Codebase repository works with status of provisioning project into git.
type CodebaseRepository interface {
	SelectProjectStatusValue(ctx context.Context, codebase, edp string) (string, error)
	UpdateProjectStatusValue(ctx context.Context, gitStatus, codebase, edp string) error
}
