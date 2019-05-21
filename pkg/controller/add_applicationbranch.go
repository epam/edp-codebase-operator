package controller

import (
	"codebase-operator/pkg/controller/applicationbranch"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, applicationbranch.Add)
}
