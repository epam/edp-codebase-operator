package controller

import (
	"codebase-operator/pkg/controller/gitserver"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, gitserver.Add)
}
