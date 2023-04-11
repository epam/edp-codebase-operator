// Code generated by mockery v2.23.1. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// GitProjectProvider is an autogenerated mock type for the GitProjectProvider type
type GitProjectProvider struct {
	mock.Mock
}

// CreateProject provides a mock function with given fields: ctx, gitlabURL, token, fullPath
func (_m *GitProjectProvider) CreateProject(ctx context.Context, gitlabURL string, token string, fullPath string) error {
	ret := _m.Called(ctx, gitlabURL, token, fullPath)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string) error); ok {
		r0 = rf(ctx, gitlabURL, token, fullPath)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ProjectExists provides a mock function with given fields: ctx, gitlabURL, token, projectID
func (_m *GitProjectProvider) ProjectExists(ctx context.Context, gitlabURL string, token string, projectID string) (bool, error) {
	ret := _m.Called(ctx, gitlabURL, token, projectID)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string) (bool, error)); ok {
		return rf(ctx, gitlabURL, token, projectID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string) bool); ok {
		r0 = rf(ctx, gitlabURL, token, projectID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, string) error); ok {
		r1 = rf(ctx, gitlabURL, token, projectID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewGitProjectProvider interface {
	mock.TestingT
	Cleanup(func())
}

// NewGitProjectProvider creates a new instance of GitProjectProvider. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewGitProjectProvider(t mockConstructorTestingTNewGitProjectProvider) *GitProjectProvider {
	mock := &GitProjectProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}