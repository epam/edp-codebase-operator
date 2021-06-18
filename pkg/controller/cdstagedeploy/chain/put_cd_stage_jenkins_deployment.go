package chain

import (
	"context"
	"fmt"
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/platform"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PutCDStageJenkinsDeployment struct {
	client client.Client
	log    logr.Logger
}

const (
	cdPipelinePostfix = "-cd-pipeline"
	jenkinsKey        = "jenkinsName"
	cdStageDeployKey  = "cdStageDeployName"
)

func (h PutCDStageJenkinsDeployment) ServeRequest(stageDeploy *v1alpha1.CDStageDeploy) error {
	log := h.log.WithValues("name", stageDeploy.Name)
	log.Info("creating CDStageJenkinsDeployment.")

	jd, err := h.getCDStageJenkinsDeployment(stageDeploy.Name, stageDeploy.Namespace)
	if err != nil {
		return errors.Wrapf(err, "couldn't get %v cd stage jenkins deployment", stageDeploy.Name)
	}

	if jd != nil {
		h.log.Info("CDStageJenkinsDeployment already exists. skip creating.", "name", stageDeploy.Name)
		return &util.CDStageJenkinsDeploymentHasNotBeenProcessed{
			Message: fmt.Sprintf("%v has not been processed for previous version of application yet."+
				" Check status of %v CDStageJenkinsDeployment resource to get more information.",
				stageDeploy.Name, stageDeploy.Name),
		}
	}

	if err := h.create(stageDeploy); err != nil {
		return errors.Wrapf(err, "couldn't create %v cd stage jenkins deployment", stageDeploy.Name)
	}
	log.Info("creating CDStageJenkinsDeployment has been finished.")
	return nil
}

func (h PutCDStageJenkinsDeployment) getCDStageJenkinsDeployment(name, namespace string) (*jenkinsApi.CDStageJenkinsDeployment, error) {
	h.log.Info("getting cd stage jenkins deployment", "name", name)
	i := &jenkinsApi.CDStageJenkinsDeployment{}
	nn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	if err := h.client.Get(context.TODO(), nn, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return i, nil
}

func (h PutCDStageJenkinsDeployment) create(stageDeploy *v1alpha1.CDStageDeploy) error {
	log := h.log.WithValues("name", stageDeploy.Name)
	log.Info("cd stage jenkins deployment is not present in cluster. start creating...")

	labels, err := h.generateLabels(stageDeploy.Name, stageDeploy.Namespace)
	if err != nil {
		return errors.Wrap(err, "couldn't generate labels")
	}

	jdCommand := &jenkinsApi.CDStageJenkinsDeployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: util.V2APIVersion,
			Kind:       util.CDStageJenkinsDeploymentKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      stageDeploy.Name,
			Namespace: stageDeploy.Namespace,
			Labels:    labels,
		},
		Spec: jenkinsApi.CDStageJenkinsDeploymentSpec{
			Job: fmt.Sprintf("%v%v/job/%v", stageDeploy.Spec.Pipeline, cdPipelinePostfix, stageDeploy.Spec.Stage),
			Tag: jenkinsApi.Tag{
				Codebase: stageDeploy.Spec.Tag.Codebase,
				Tag:      stageDeploy.Spec.Tag.Tag,
			},
		},
	}
	if err := h.client.Create(context.TODO(), jdCommand); err != nil {
		return err
	}

	log.Info("cd stage jenkins deployment has been created.")
	return nil
}

func (h PutCDStageJenkinsDeployment) generateLabels(cdStageDeployName, ns string) (map[string]string, error) {
	ji, err := platform.GetFirstJenkinsInstance(h.client, ns)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		jenkinsKey:       ji.Name,
		cdStageDeployKey: cdStageDeployName,
	}, nil
}
