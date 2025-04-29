package v1

const (
	// CdPipelineLabel is a label used to store the name of the CdPipeline in related resources.
	CdPipelineLabel = "app.edp.epam.com/cdpipeline"

	// CdStageLabel is a label used to store the name of the CDStage in related resources.
	CdStageLabel = "app.edp.epam.com/cdstage"

	// CdStageDeployLabel is a label used to store the name of the CDStageDeploy in related resources.
	CdStageDeployLabel = "app.edp.epam.com/cdstagedeploy"

	// CodebaseBranchLabel is a label used to store the name of the CodebaseBranch in related resources.
	CodebaseBranchLabel = "app.edp.epam.com/codebasebranch"

	// CodebaseLabel is a label used to store the name of the Codebase in related resources.
	CodebaseLabel = "app.edp.epam.com/codebase"

	// BranchHashLabel is a label used to store the hash of the branch name in related resources.
	// We can't use the branch name directly as a label value because it can contain special characters.
	// XXH64 is used to generate the hash.
	BranchHashLabel = "app.edp.epam.com/branch-hash"
)
