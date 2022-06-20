package chain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type DropJenkinsFolders struct {
	next      handler.CodebaseHandler
	k8sClient client.Client
}

type ErrorBranchesExists string

func (e ErrorBranchesExists) Error() string {
	return string(e)
}

func (h DropJenkinsFolders) ServeRequest(c *codebaseApi.Codebase) error {
	rLog := log.WithValues("codebase_name", c.Name)
	rLog.Info("starting to delete related jenkins folders")

	var branchList codebaseApi.CodebaseBranchList
	if err := h.k8sClient.List(context.TODO(), &branchList, &client.ListOptions{
		Namespace: c.Namespace,
	}); err != nil {
		return errors.Wrap(err, "unable to list codebase branches")
	}

	for _, branch := range branchList.Items {
		for _, owner := range branch.OwnerReferences {
			if owner.Kind == c.Kind && owner.UID == c.UID {
				return ErrorBranchesExists("can not delete jenkins folder while any codebase branches exists")
			}
		}
	}

	selector, err := labels.Parse(fmt.Sprintf("%s=%s", util.CodebaseLabelKey, c.Name))
	if err != nil {
		return errors.Wrap(err, "couldn't parse label selector")
	}
	options := &client.ListOptions{
		LabelSelector: selector,
	}

	var jenkinsFolderList jenkinsApi.JenkinsFolderList
	if err := h.k8sClient.List(context.TODO(), &jenkinsFolderList, options); err != nil {
		return errors.Wrap(err, "unable to list jenkins folders")
	}

	for i, v := range jenkinsFolderList.Items {
		rLog.Info("trying to delete jenkins folder", "folder name", v.Name)
		if err := h.k8sClient.Delete(context.TODO(), &jenkinsFolderList.Items[i]); err != nil {
			return errors.Wrap(err, "unable to delete jenkins folder")
		}
	}

	rLog.Info("done deleting child jenkins folders")
	return nextServeOrNil(h.next, c)
}
