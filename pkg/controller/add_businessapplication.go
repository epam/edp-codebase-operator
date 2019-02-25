package controller

import (
	"business-app-handler-controller/pkg/controller/businessapplication"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, businessapplication.Add)
}
