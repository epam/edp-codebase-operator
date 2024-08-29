package service

import (
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

type MockCodebasebranch struct {
	Client client.Client
	mock.Mock
}

func (m *MockCodebasebranch) AppendVersionToTheHistorySlice(cb *codebaseApi.CodebaseBranch) error {
	return m.Called(cb).Error(0)
}

func (m *MockCodebasebranch) ResetBranchBuildCounter(cb *codebaseApi.CodebaseBranch) error {
	return m.Called(cb).Error(0)
}

func (m *MockCodebasebranch) ResetBranchSuccessBuildCounter(cb *codebaseApi.CodebaseBranch) error {
	return m.Called(cb).Error(0)
}

func (m *MockCodebasebranch) updateStatus(cb *codebaseApi.CodebaseBranch) error {
	return m.Called(cb).Error(0)
}
