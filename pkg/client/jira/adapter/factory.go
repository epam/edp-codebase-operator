package adapter

import (
	"fmt"

	goJira "github.com/andygrunwald/go-jira"

	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/dto"
)

type GoJiraAdapterFactory struct{}

func (GoJiraAdapterFactory) New(js dto.JiraServer) (jira.Client, error) {
	rl := log.WithValues("jira dto", js)
	rl.V(2).Info("start new Jira client creation")

	client, err := initClient(js)
	if err != nil {
		return nil, err
	}

	rl.Info("Jira client has been created")

	return &GoJiraAdapter{
		client: *client,
	}, nil
}

func initClient(js dto.JiraServer) (*goJira.Client, error) {
	tp := goJira.BasicAuthTransport{
		Username: js.User,
		Password: js.Pwd,
	}

	client, err := goJira.NewClient(tp.Client(), js.ApiUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create new jira client: %w", err)
	}

	return client, nil
}
