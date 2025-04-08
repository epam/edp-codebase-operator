package v1

const (
	// CdPipelineLabel is a label that is used to store the name of the CD pipeline in the related resources.
	CdPipelineLabel = "app.edp.epam.com/cdpipeline"

	// CdStageLabel is a label that is used to store the name of the CD stage in the related resources.
	CdStageLabel = "app.edp.epam.com/cdstage"

	// CdStageDeployLabel is a label that is used to store the name of the CD stage deploy in the related resources.
	CdStageDeployLabel = "app.edp.epam.com/cdstagedeploy"

	// CodebaseImageStreamCodebaseBranchLabel is a label that is used to store CodebaseBranch name of the CodebaseImageStreamCodebaseImageStream.
	CodebaseImageStreamCodebaseBranchLabel = "app.edp.epam.com/cbis-codebasebranch"

	// CodebaseImageStreamCodebaseLabel is a label that is used to store Codebase name of the CodebaseImageStreamCodebaseImageStream.
	CodebaseImageStreamCodebaseLabel = "app.edp.epam.com/cbis-codebase"
)
