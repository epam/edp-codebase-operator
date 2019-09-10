package controller

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, codebasebranch.Add)
}
