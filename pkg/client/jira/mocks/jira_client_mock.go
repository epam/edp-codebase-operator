// Code generated by mockery v2.28.0. DO NOT EDIT.

package mocks

import (
	context "context"

	go_jira "github.com/andygrunwald/go-jira"
	jira "github.com/epam/edp-codebase-operator/v2/pkg/client/jira"

	mock "github.com/stretchr/testify/mock"
)

// Client is an autogenerated mock type for the Client type
type Client struct {
	mock.Mock
}

// ApplyTagsToIssue provides a mock function with given fields: issue, tags
func (_m *Client) ApplyTagsToIssue(issue string, tags map[string]interface{}) error {
	ret := _m.Called(issue, tags)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, map[string]interface{}) error); ok {
		r0 = rf(issue, tags)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Connected provides a mock function with given fields:
func (_m *Client) Connected() (bool, error) {
	ret := _m.Called()

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func() (bool, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateComponentValue provides a mock function with given fields: ctx, projectId, componentName
func (_m *Client) CreateComponentValue(ctx context.Context, projectId int, componentName string) error {
	ret := _m.Called(ctx, projectId, componentName)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int, string) error); ok {
		r0 = rf(ctx, projectId, componentName)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateFixVersionValue provides a mock function with given fields: ctx, projectId, versionName
func (_m *Client) CreateFixVersionValue(ctx context.Context, projectId int, versionName string) error {
	ret := _m.Called(ctx, projectId, versionName)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int, string) error); ok {
		r0 = rf(ctx, projectId, versionName)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateIssueLink provides a mock function with given fields: issueId, title, url
func (_m *Client) CreateIssueLink(issueId string, title string, url string) error {
	ret := _m.Called(issueId, title, url)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string) error); ok {
		r0 = rf(issueId, title, url)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetIssue provides a mock function with given fields: ctx, issueId
func (_m *Client) GetIssue(ctx context.Context, issueId string) (*go_jira.Issue, error) {
	ret := _m.Called(ctx, issueId)

	var r0 *go_jira.Issue
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*go_jira.Issue, error)); ok {
		return rf(ctx, issueId)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *go_jira.Issue); ok {
		r0 = rf(ctx, issueId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*go_jira.Issue)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, issueId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetIssueTypeMeta provides a mock function with given fields: ctx, projectID, issueTypeID
func (_m *Client) GetIssueTypeMeta(ctx context.Context, projectID string, issueTypeID string) (map[string]jira.IssueTypeMeta, error) {
	ret := _m.Called(ctx, projectID, issueTypeID)

	var r0 map[string]jira.IssueTypeMeta
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (map[string]jira.IssueTypeMeta, error)); ok {
		return rf(ctx, projectID, issueTypeID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) map[string]jira.IssueTypeMeta); ok {
		r0 = rf(ctx, projectID, issueTypeID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]jira.IssueTypeMeta)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, projectID, issueTypeID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetProjectInfo provides a mock function with given fields: issueId
func (_m *Client) GetProjectInfo(issueId string) (*go_jira.Project, error) {
	ret := _m.Called(issueId)

	var r0 *go_jira.Project
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*go_jira.Project, error)); ok {
		return rf(issueId)
	}
	if rf, ok := ret.Get(0).(func(string) *go_jira.Project); ok {
		r0 = rf(issueId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*go_jira.Project)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(issueId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewClient interface {
	mock.TestingT
	Cleanup(func())
}

// NewClient creates a new instance of Client. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewClient(t mockConstructorTestingTNewClient) *Client {
	mock := &Client{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}