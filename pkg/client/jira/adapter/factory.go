package adapter

import (
	gojira "github.com/andygrunwald/go-jira"

	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/dto"
)

type GoJiraAdapterFactory struct {
}

func (GoJiraAdapterFactory) New(jira dto.JiraServer) (jira.Client, error) {
	rl := log.WithValues("jira dto", jira)
	rl.V(2).Info("start new Jira client creation")
	client, err := initClient(jira)
	if err != nil {
		return nil, err
	}
	rl.Info("Jira client has been created")
	return &GoJiraAdapter{
		client: *client,
	}, nil
}

func initClient(jira dto.JiraServer) (*gojira.Client, error) {
	tp := gojira.BasicAuthTransport{
		Username: jira.User,
		Password: jira.Pwd,
	}
	client, err := gojira.NewClient(tp.Client(), jira.ApiUrl)
	if err != nil {
		return nil, err
	}
	return client, err
}
