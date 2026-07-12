package v1

const (
	// GitLabCITemplateAnnotation is an annotation on a Codebase CR that specifies the ConfigMap
	// name to use as the GitLab CI template. When absent, the operator falls back to "gitlab-ci-default".
	GitLabCITemplateAnnotation = "app.edp.epam.com/gitlab-ci-template"

	// BranchCleanupStrategyAnnotation is an annotation on a Codebase CR that defines what the
	// operator does with CodebaseBranch resources whose branch no longer exists in git.
	// Supported values: "mark" (default) - only mark the branch as stale;
	// "auto" - delete the stale branch when it is not referenced by any CDPipeline/Stage.
	BranchCleanupStrategyAnnotation = "app.edp.epam.com/branch-cleanup-strategy"
)

const (
	// BranchCleanupStrategyMark marks stale branches for manual cleanup.
	BranchCleanupStrategyMark = "mark"

	// BranchCleanupStrategyAuto deletes stale branches that are not used by any CDPipeline/Stage.
	BranchCleanupStrategyAuto = "auto"
)
