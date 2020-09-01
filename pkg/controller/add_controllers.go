package controller

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/imagestreamtag"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/jirafixversion"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/jiraserver"
)

func init() {
	AddToManagerFuncs = append(AddToManagerFuncs, codebase.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, codebasebranch.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, gitserver.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, jirafixversion.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, jiraserver.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, imagestreamtag.Add)
}
