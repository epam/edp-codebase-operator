package util

type CodebaseBranchReconcileError struct {
	Message string
}

func (e *CodebaseBranchReconcileError) Error() string {
	return e.Message
}

func NewCodebaseBranchReconcileError(msg string) error {
	return &CodebaseBranchReconcileError{Message: msg}
}

type CDStageDeployHasNotBeenProcessed struct {
	Message string
}

func (e *CDStageDeployHasNotBeenProcessed) Error() string {
	return e.Message
}

type CDStageJenkinsDeploymentHasNotBeenProcessed struct {
	Message string
}

func (e *CDStageJenkinsDeploymentHasNotBeenProcessed) Error() string {
	return e.Message
}
