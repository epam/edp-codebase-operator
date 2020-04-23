package adapter

import (
	"github.com/andygrunwald/go-jira"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("gojira_adapter")

type GoJiraAdapter struct {
	client jira.Client
}

func (a GoJiraAdapter) Connected() (bool, error) {
	log.V(2).Info("start Connected method")
	user, _, err := a.client.User.GetSelf()
	if err != nil {
		return false, err
	}
	return user != nil, nil
}
