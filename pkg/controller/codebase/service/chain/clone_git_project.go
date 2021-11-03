package chain

import (
	"context"
	"fmt"
	"time"

	edpv1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	git "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CloneGitProject struct {
	next   handler.CodebaseHandler
	client client.Client
	git    git.Git
}

func (h CloneGitProject) ServeRequest(c *edpv1alpha1.Codebase) error {
	rLog := log.WithValues("codebase_name", c.Name)
	rLog.Info("Start cloning project...")
	rLog.Info("codebase data", "spec", c.Spec)
	if err := h.setIntermediateSuccessFields(c, edpv1alpha1.AcceptCodebaseRegistration); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v Codebase status", c.Name)
	}

	wd := util.GetWorkDir(c.Name, c.Namespace)
	log.Info("Setting path for local Git folder", "path", wd)
	if err := util.CreateDirectory(wd); err != nil {
		setFailedFields(c, edpv1alpha1.ImportProject, err.Error())
		return err
	}

	gs, err := util.GetGitServer(h.client, c.Spec.GitServer, c.Namespace)
	if err != nil {
		setFailedFields(c, edpv1alpha1.ImportProject, err.Error())
		return errors.Wrapf(err, "an error has occurred while getting %v GitServer", c.Spec.GitServer)
	}

	secret, err := util.GetSecret(h.client, gs.NameSshKeySecret, c.Namespace)
	if err != nil {
		setFailedFields(c, edpv1alpha1.ImportProject, err.Error())
		return errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
	}

	k := string(secret.Data[util.PrivateSShKeyName])
	u := gs.GitUser
	ru := fmt.Sprintf("%v:%v", gs.GitHost, *c.Spec.GitUrlPath)

	if !util.DoesDirectoryExist(wd) || util.IsDirectoryEmpty(wd) {
		if err := h.git.CloneRepositoryBySsh(k, u, ru, wd, gs.SshPort); err != nil {
			setFailedFields(c, edpv1alpha1.ImportProject, err.Error())
			return errors.Wrapf(err, "an error has occurred while cloning repository %v", ru)
		}
	}
	rLog.Info("end cloning project")
	return nextServeOrNil(h.next, c)
}

func (h CloneGitProject) setIntermediateSuccessFields(c *edpv1alpha1.Codebase, action edpv1alpha1.ActionType) error {
	c.Status = edpv1alpha1.CodebaseStatus{
		Status:          util.StatusInProgress,
		Available:       false,
		LastTimeUpdated: time.Now(),
		Action:          action,
		Result:          edpv1alpha1.Success,
		Username:        "system",
		Value:           "inactive",
		FailureCount:    c.Status.FailureCount,
		Git:             c.Status.Git,
	}

	if err := h.client.Status().Update(context.TODO(), c); err != nil {
		if err := h.client.Update(context.TODO(), c); err != nil {
			return err
		}
	}
	return nil
}
