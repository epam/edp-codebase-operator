package controller

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebaseimagestream"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/gittag"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/imagestreamtag"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/jiraissuemetadata"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/jiraserver"
)

func init() {
	AddToManagerFuncs = append(AddToManagerFuncs, codebase.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, codebasebranch.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, gitserver.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, jiraissuemetadata.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, jiraserver.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, imagestreamtag.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, gittag.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, codebaseimagestream.Add)
}
