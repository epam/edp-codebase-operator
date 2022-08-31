package mock

import (
	"github.com/andygrunwald/go-jira"
	"github.com/stretchr/testify/mock"
)

type MockClient struct {
	mock.Mock
}

func (m *MockClient) Connected() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

func (m *MockClient) GetProjectInfo(issueId string) (*jira.Project, error) {
	args := m.Called(issueId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jira.Project), args.Error(1)
}

func (m *MockClient) CreateFixVersionValue(projectId int, versionName string) error {
	args := m.Called(projectId, versionName)
	return args.Error(0)
}

func (m *MockClient) CreateComponentValue(projectId int, componentName string) error {
	args := m.Called(projectId, componentName)
	return args.Error(0)
}

func (m *MockClient) GetIssueMetadata(projectKey string) (*jira.CreateMetaInfo, error) {
	args := m.Called(projectKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jira.CreateMetaInfo), args.Error(1)
}

func (m *MockClient) GetIssueType(issueId string) (string, error) {
	args := m.Called(issueId)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.String(0), args.Error(1)
}

func (m *MockClient) ApplyTagsToIssue(issue string, tags map[string]interface{}) error {
	args := m.Called(issue, tags)
	return args.Error(0)
}

func (m *MockClient) CreateIssueLink(issueId, title, url string) error {
	args := m.Called(issueId, title, url)
	return args.Error(0)
}
