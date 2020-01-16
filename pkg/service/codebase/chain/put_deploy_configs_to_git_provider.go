package chain

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	git "github.com/epmd-edp/codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/service/codebase/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/service/codebase/template"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
)

type PutDeployConfigsToGitProvider struct {
	next      handler.CodebaseHandler
	clientSet openshift.ClientSet
}

func (h PutDeployConfigsToGitProvider) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase name", c.Name)
	rLog.Info("Start pushing configs...")

	if err := h.tryToPushConfigs(*c); err != nil {
		setFailedFields(*c, edpv1alpha1.SetupDeploymentTemplates, err.Error())
		return errors.Wrapf(err, "couldn't push deploy configs", "codebase name", c.Name)
	}

	rLog.Info("end pushing configs to remote git server")
	return nextServeOrNil(h.next, c)
}

func (h PutDeployConfigsToGitProvider) tryToPushConfigs(c v1alpha1.Codebase) error {
	gs, err := util.GetGitServer(h.clientSet.Client, c.Name, c.Spec.GitServer, c.Namespace)
	if err != nil {
		return err
	}

	secret, err := util.GetSecret(*h.clientSet.CoreClient, gs.NameSshKeySecret, c.Namespace)
	if err != nil {
		return errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
	}

	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v", c.Namespace, c.Name)
	td := fmt.Sprintf("%v/%v", wd, getTemplateFolderName(c.Spec.DeploymentScript))
	gf := fmt.Sprintf("%v/%v", td, c.Name)
	log.Info("path to local Git folder", "path", gf)

	d := fmt.Sprintf("%v/%v", td, c.Name)
	if err := template.PrepareTemplates(h.clientSet.CoreClient, c); err != nil {
		return err
	}

	if err := git.CommitChanges(d); err != nil {
		return err
	}

	k := string(secret.Data[util.PrivateSShKeyName])
	u := gs.GitUser
	dest := fmt.Sprintf("%v/%v", td, c.Name)
	if err := git.PushChanges(k, u, dest); err != nil {
		return errors.Wrapf(err, "an error has occurred while pushing changes for %v codebase", c.Name)
	}
	return nil
}
