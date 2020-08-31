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
