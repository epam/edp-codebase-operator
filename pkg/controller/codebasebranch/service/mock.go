package service

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MockCodebasebranch struct {
	Client client.Client
	mock.Mock
}

func (m *MockCodebasebranch) TriggerDeletionJob(cb *v1alpha1.CodebaseBranch) error {
	return m.Called(cb).Error(0)
}
func (m *MockCodebasebranch) TriggerReleaseJob(cb *v1alpha1.CodebaseBranch) error {
	return m.Called(cb).Error(0)
}

func (m *MockCodebasebranch) convertCodebaseBranchSpecToParams(cb *v1alpha1.CodebaseBranch) (map[string]string, error) {
	var a map[string]string

	return a, m.Called(cb).Error(0)
}

func (m *MockCodebasebranch) AppendVersionToTheHistorySlice(cb *v1alpha1.CodebaseBranch) error {
	return m.Called(cb).Error(0)
}

func (m *MockCodebasebranch) ResetBranchBuildCounter(cb *v1alpha1.CodebaseBranch) error {
	return m.Called(cb).Error(0)
}

func (m *MockCodebasebranch) ResetBranchSuccessBuildCounter(cb *v1alpha1.CodebaseBranch) error {
	return m.Called(cb).Error(0)
}

func (m *MockCodebasebranch) updateStatus(cb *v1alpha1.CodebaseBranch) error {
	return m.Called(cb).Error(0)
}
