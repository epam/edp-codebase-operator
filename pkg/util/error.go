package util

type CodebaseBranchReconcileError struct {
	Message string
}

func (e *CodebaseBranchReconcileError) Error() string {
	return e.Message
}

func NewCodebaseBranchReconcileError(msg string) *CodebaseBranchReconcileError {
	return &CodebaseBranchReconcileError{Message: msg}
}

type CDStageDeployHasNotBeenProcessedError struct {
	Message string
}

func (e *CDStageDeployHasNotBeenProcessedError) Error() string {
	return e.Message
}

type CDStageJenkinsDeploymentHasNotBeenProcessedError struct {
	Message string
}

func (e *CDStageJenkinsDeploymentHasNotBeenProcessedError) Error() string {
	return e.Message
}
