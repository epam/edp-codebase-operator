package mock

import (
	"github.com/stretchr/testify/mock"
)

type MockGit struct {
	mock.Mock
}

func (m *MockGit) CommitChanges(directory, commitMsg string) error {
	args := m.Called(directory, commitMsg)
	return args.Error(0)
}

func (m *MockGit) PushChanges(key, user, directory string, pushArgs ...string) error {
	args := m.Called(key, user, directory)
	return args.Error(0)
}

func (m *MockGit) CheckPermissions(repo string, user, pass *string) (accessible bool) {
	args := m.Called(repo, user, pass)
	return args.Bool(0)
}

func (m *MockGit) CloneRepositoryBySsh(key, user, repoUrl, destination string, port int32) error {
	args := m.Called(key, user, repoUrl, destination, port)
	return args.Error(0)
}

func (m *MockGit) CloneRepository(repo string, user *string, pass *string, destination string) error {
	panic("implement me")
}

func (m *MockGit) CreateRemoteBranch(key, user, path, name string) error {
	args := m.Called(key, user, path, name)
	return args.Error(0)
}

func (m *MockGit) CreateRemoteTag(key, user, path, branchName, name string) error {
	panic("implement me")
}

func (m *MockGit) Fetch(key, user, path, branchName string) error { panic("implement me") }

func (m *MockGit) Checkout(user, pass *string, directory, branchName string, remote bool) error {
	args := m.Called(user, pass, directory, branchName, remote)
	return args.Error(0)
}

func (m *MockGit) GetCurrentBranchName(directory string) (string, error) {
	args := m.Called(directory)
	return args.String(), args.Error(1)
}

func (m *MockGit) Init(directory string) error { panic("implement me") }
