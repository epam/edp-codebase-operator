// Code generated by mockery v2.23.1. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// Git is an autogenerated mock type for the Git type
type Git struct {
	mock.Mock
}

// AddRemoteLink provides a mock function with given fields: repoPath, remoteUrl
func (_m *Git) AddRemoteLink(repoPath string, remoteUrl string) error {
	ret := _m.Called(repoPath, remoteUrl)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(repoPath, remoteUrl)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CheckPermissions provides a mock function with given fields: repo, user, pass
func (_m *Git) CheckPermissions(repo string, user *string, pass *string) bool {
	ret := _m.Called(repo, user, pass)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string, *string, *string) bool); ok {
		r0 = rf(repo, user, pass)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Checkout provides a mock function with given fields: user, pass, directory, branchName, remote
func (_m *Git) Checkout(user *string, pass *string, directory string, branchName string, remote bool) error {
	ret := _m.Called(user, pass, directory, branchName, remote)

	var r0 error
	if rf, ok := ret.Get(0).(func(*string, *string, string, string, bool) error); ok {
		r0 = rf(user, pass, directory, branchName, remote)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CheckoutRemoteBranchBySSH provides a mock function with given fields: key, user, gitPath, remoteBranchName
func (_m *Git) CheckoutRemoteBranchBySSH(key string, user string, gitPath string, remoteBranchName string) error {
	ret := _m.Called(key, user, gitPath, remoteBranchName)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string, string) error); ok {
		r0 = rf(key, user, gitPath, remoteBranchName)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CloneRepository provides a mock function with given fields: repo, user, pass, destination
func (_m *Git) CloneRepository(repo string, user *string, pass *string, destination string) error {
	ret := _m.Called(repo, user, pass, destination)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, *string, *string, string) error); ok {
		r0 = rf(repo, user, pass, destination)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CloneRepositoryBySsh provides a mock function with given fields: key, user, repoUrl, destination, port
func (_m *Git) CloneRepositoryBySsh(key string, user string, repoUrl string, destination string, port int32) error {
	ret := _m.Called(key, user, repoUrl, destination, port)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string, string, int32) error); ok {
		r0 = rf(key, user, repoUrl, destination, port)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CommitChanges provides a mock function with given fields: directory, commitMsg
func (_m *Git) CommitChanges(directory string, commitMsg string) error {
	ret := _m.Called(directory, commitMsg)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(directory, commitMsg)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CommitExists provides a mock function with given fields: directory, hash
func (_m *Git) CommitExists(directory string, hash string) (bool, error) {
	ret := _m.Called(directory, hash)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string) (bool, error)); ok {
		return rf(directory, hash)
	}
	if rf, ok := ret.Get(0).(func(string, string) bool); ok {
		r0 = rf(directory, hash)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(directory, hash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateChildBranch provides a mock function with given fields: directory, currentBranch, newBranch
func (_m *Git) CreateChildBranch(directory string, currentBranch string, newBranch string) error {
	ret := _m.Called(directory, currentBranch, newBranch)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string) error); ok {
		r0 = rf(directory, currentBranch, newBranch)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateRemoteBranch provides a mock function with given fields: key, user, path, name, fromcommit, port
func (_m *Git) CreateRemoteBranch(key string, user string, path string, name string, fromcommit string, port int32) error {
	ret := _m.Called(key, user, path, name, fromcommit, port)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string, string, string, int32) error); ok {
		r0 = rf(key, user, path, name, fromcommit, port)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateRemoteTag provides a mock function with given fields: key, user, path, branchName, name
func (_m *Git) CreateRemoteTag(key string, user string, path string, branchName string, name string) error {
	ret := _m.Called(key, user, path, branchName, name)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string, string, string) error); ok {
		r0 = rf(key, user, path, branchName, name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Fetch provides a mock function with given fields: key, user, path, branchName
func (_m *Git) Fetch(key string, user string, path string, branchName string) error {
	ret := _m.Called(key, user, path, branchName)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string, string) error); ok {
		r0 = rf(key, user, path, branchName)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetCurrentBranchName provides a mock function with given fields: directory
func (_m *Git) GetCurrentBranchName(directory string) (string, error) {
	ret := _m.Called(directory)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (string, error)); ok {
		return rf(directory)
	}
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(directory)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(directory)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Init provides a mock function with given fields: directory
func (_m *Git) Init(directory string) error {
	ret := _m.Called(directory)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(directory)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// PushChanges provides a mock function with given fields: key, user, directory, port, pushParams
func (_m *Git) PushChanges(key string, user string, directory string, port int32, pushParams ...string) error {
	_va := make([]interface{}, len(pushParams))
	for _i := range pushParams {
		_va[_i] = pushParams[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, key, user, directory, port)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string, int32, ...string) error); ok {
		r0 = rf(key, user, directory, port, pushParams...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RemoveBranch provides a mock function with given fields: directory, branchName
func (_m *Git) RemoveBranch(directory string, branchName string) error {
	ret := _m.Called(directory, branchName)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(directory, branchName)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RenameBranch provides a mock function with given fields: directory, currentName, newName
func (_m *Git) RenameBranch(directory string, currentName string, newName string) error {
	ret := _m.Called(directory, currentName, newName)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string) error); ok {
		r0 = rf(directory, currentName, newName)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewGit interface {
	mock.TestingT
	Cleanup(func())
}

// NewGit creates a new instance of Git. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewGit(t mockConstructorTestingTNewGit) *Git {
	mock := &Git{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
