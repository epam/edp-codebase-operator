package chain

import (
	"context"
	"fmt"
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	git "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type CloneGitProject struct {
	next   handler.CodebaseHandler
	client client.Client
	git    git.Git
}

func (h CloneGitProject) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase name", c.Name)
	rLog.Info("Start cloning project...")
	rLog.Info("codebase data", "spec", c.Spec)
	if err := h.setIntermediateSuccessFields(c, edpv1alpha1.AcceptCodebaseRegistration); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v Codebase status", c.Name)
	}

	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v", c.Namespace, c.Name)
	if err := util.CreateDirectory(wd); err != nil {
		setFailedFields(c, edpv1alpha1.ImportProject, err.Error())
		return err
	}

	gs, err := util.GetGitServer(h.client, c.Spec.GitServer, c.Namespace)
	if err != nil {
		setFailedFields(c, edpv1alpha1.ImportProject, err.Error())
		return err
	}

	secret, err := util.GetSecret(h.client, gs.NameSshKeySecret, c.Namespace)
	if err != nil {
		setFailedFields(c, edpv1alpha1.ImportProject, err.Error())
		return errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
	}

	td := fmt.Sprintf("%v/%v", wd, "templates")
	if err := util.CreateDirectory(td); err != nil {
		setFailedFields(c, edpv1alpha1.ImportProject, err.Error())
		return errors.Wrapf(err, "an error has occurred while creating folder %v", td)
	}

	gf := fmt.Sprintf("%v/%v", td, c.Name)
	log.Info("path to local Git folder", "path", gf)

	k := string(secret.Data[util.PrivateSShKeyName])
	u := gs.GitUser
	ru := fmt.Sprintf("%v:%v%v", gs.GitHost, gs.SshPort, *c.Spec.GitUrlPath)
	if !util.DoesDirectoryExist(gf) || util.IsDirectoryEmpty(gf) {
		if err := h.git.CloneRepositoryBySsh(k, u, ru, gf); err != nil {
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
