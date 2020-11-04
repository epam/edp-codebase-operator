package repository

// Codebase repository works with status of provisioning project into git
type CodebaseRepository interface {
	SelectProjectStatusValue(codebase, edp string) (*string, error)
	UpdateProjectStatusValue(gitStatus, codebase, edp string) error
}
