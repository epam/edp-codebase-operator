package controller

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/jiraserver"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, jiraserver.Add)
}
