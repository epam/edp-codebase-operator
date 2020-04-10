package chain

import (
	"context"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	jenkinsv1alpha1 "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

type PutJenkinsFolder struct {
	next      handler.CodebaseHandler
	clientSet openshift.ClientSet
}

func (h PutJenkinsFolder) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase name", c.Name)
	rLog.Info("start creating jenkins folder...")
	if err := h.putJenkinsFolder(c); err != nil {
		setFailedFields(c, v1alpha1.PutJenkinsFolder, err.Error())
		return err
	}
	if err := h.updateFinishStatus(c); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v Codebase status", c.Name)
	}
	rLog.Info("end creating jenkins folder...")
	return nextServeOrNil(h.next, c)
}

func (h PutJenkinsFolder) putJenkinsFolder(c *v1alpha1.Codebase) error {
	jfn := fmt.Sprintf("%v-%v", c.Name, "codebase")
	jfr, err := h.getJenkinsFolder(jfn, c.Namespace)
	if err != nil {
		return err
	}

	if jfr != nil {
		log.Info("jenkins folder already exists in cluster", "name", jfn)
		return nil
	}

	jf := &jenkinsv1alpha1.JenkinsFolder{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1alpha1",
			Kind:       "JenkinsFolder",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      jfn,
			Namespace: c.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "v2.edp.epam.com/v1alpha1",
					Kind:               util.CodebaseKind,
					Name:               c.Name,
					UID:                c.UID,
					BlockOwnerDeletion: newTrue(),
				},
			},
		},
		Spec: jenkinsv1alpha1.JenkinsFolderSpec{
			JobName: &c.Spec.JobProvisioning,
		},
		Status: jenkinsv1alpha1.JenkinsFolderStatus{
			Available:       false,
			LastTimeUpdated: time.Time{},
			Status:          util.StatusInProgress,
		},
	}
	if err := h.clientSet.Client.Create(context.TODO(), jf); err != nil {
		return errors.Wrapf(err, "couldn't create jenkins folder %v", "name")
	}
	return nil
}

func (h PutJenkinsFolder) getJenkinsFolder(name, namespace string) (*jenkinsv1alpha1.JenkinsFolder, error) {
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	i := &jenkinsv1alpha1.JenkinsFolder{}
	if err := h.clientSet.Client.Get(context.TODO(), nsn, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get instance by owner %v", name)
	}
	return i, nil
}

func newTrue() *bool {
	b := true
	return &b
}

func (h PutJenkinsFolder) updateFinishStatus(c *v1alpha1.Codebase) error {
	c.Status = v1alpha1.CodebaseStatus{
		Status:          util.StatusFinished,
		Available:       true,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          v1alpha1.SetupDeploymentTemplates,
		Result:          v1alpha1.Success,
		Value:           "active",
		FailureCount:    0,
	}
	return h.updateStatus(c)
}

func (h PutJenkinsFolder) updateStatus(c *v1alpha1.Codebase) error {
	if err := h.clientSet.Client.Status().Update(context.TODO(), c); err != nil {
		if err := h.clientSet.Client.Update(context.TODO(), c); err != nil {
			return err
		}
	}
	return nil
}
