package chain

import (
	"context"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	git "github.com/epmd-edp/codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

type CloneGitProject struct {
	next      handler.CodebaseHandler
	clientSet openshift.ClientSet
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

	gs, err := util.GetGitServer(h.clientSet.Client, c.Name, c.Spec.GitServer, c.Namespace)
	if err != nil {
		setFailedFields(c, edpv1alpha1.ImportProject, err.Error())
		return err
	}

	secret, err := util.GetSecret(*h.clientSet.CoreClient, gs.NameSshKeySecret, c.Namespace)
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
		if err := git.CloneRepositoryBySsh(k, u, ru, gf); err != nil {
			setFailedFields(c, edpv1alpha1.ImportProject, err.Error())
			return errors.Wrapf(err, "an error has occurred while cloning repository %v", ru)
		}
	}
	rLog.Info("end cloning project")
	return nextServeOrNil(h.next, c)
}

func (h CloneGitProject) getSecret(secretName, namespace string) (*v1.Secret, error) {
	log.Info("Start fetching Secret resource from k8s", "secret name", secretName, "namespace", namespace)
	secret, err := h.clientSet.CoreClient.
		Secrets(namespace).
		Get(secretName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) || k8serrors.IsForbidden(err) {
		return nil, err
	}
	log.Info("Secret has been fetched", "secret name", secretName, "namespace", namespace)
	return secret, nil
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
	}

	if err := h.clientSet.Client.Status().Update(context.TODO(), c); err != nil {
		if err := h.clientSet.Client.Update(context.TODO(), c); err != nil {
			return err
		}
	}
	return nil
}
