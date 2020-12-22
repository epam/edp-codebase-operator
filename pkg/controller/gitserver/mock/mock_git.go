package mock

import "github.com/stretchr/testify/mock"

type MockGit struct {
	mock.Mock
}

func (m MockGit) CommitChanges(directory, commitMsg string) error {
	args := m.Called(directory, commitMsg)
	return args.Error(0)
}

func (m MockGit) PushChanges(key, user, directory string) error {
	args := m.Called(key, user, directory)
	return args.Error(0)
}

func (m MockGit) CheckPermissions(repo string, user string, pass string) (accessible bool) {
	panic("implement me")
}

func (m MockGit) CloneRepositoryBySsh(key, user, repoUrl, destination string) error {
	args := m.Called(key, user, repoUrl, destination)
	return args.Error(0)
}

func (m MockGit) CloneRepository(repo, user, pass, destination string) error { panic("implement me") }

func (m MockGit) CreateRemoteBranch(key, user, path, name string) error {
	args := m.Called(key, user, path, name)
	return args.Error(0)
}

func (m MockGit) CreateRemoteTag(key, user, path, branchName, name string) error {
	panic("implement me")
}

func (m MockGit) Fetch(key, user, path, branchName string) error { panic("implement me") }

func (m MockGit) Checkout(directory, branchName string) error {
	args := m.Called(directory, branchName)
	return args.Error(0)
}

func (m MockGit) CreateLocalBranch(path, name string) error { panic("implement me") }
