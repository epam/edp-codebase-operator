package putcdstagejenkinsdeployment

import (
	"context"
	"fmt"
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	v1alphaJenkins "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/platform"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

type PutCDStageJenkinsDeployment struct {
	Client client.Client
}

const (
	cdPipelinePostfix = "-cd-pipeline"
	jenkinsKey        = "jenkinsName"
	cdStageDeployKey  = "cdStageDeployName"
)

var log = logf.Log.WithName("put-cd-stage-jenkins-deployment-controller")

func (h PutCDStageJenkinsDeployment) ServeRequest(stageDeploy *v1alpha1.CDStageDeploy) error {
	vLog := log.WithValues("name", stageDeploy.Name)
	vLog.Info("creating/updating CDStageJenkinsDeployment.")

	jd, err := h.getCDStageJenkinsDeployment(stageDeploy.Name, stageDeploy.Namespace)
	if err != nil {
		return errors.Wrapf(err, "couldn't get %v cd stage jenkins deployment", stageDeploy.Name)
	}

	if jd == nil {
		if err := h.create(stageDeploy); err != nil {
			return errors.Wrapf(err, "couldn't create %v cd stage jenkins deployment", stageDeploy.Name)
		}
		return nil
	}

	if err := h.update(jd, stageDeploy.Spec.Tags); err != nil {
		return errors.Wrapf(err, "couldn't update %v cd stage jenkins deployment", stageDeploy.Name)
	}

	vLog.Info("creating/updating CDStageJenkinsDeployment has been finished.")
	return nil
}

func (h PutCDStageJenkinsDeployment) getCDStageJenkinsDeployment(name, namespace string) (*v1alphaJenkins.CDStageJenkinsDeployment, error) {
	log.Info("getting cd stage jenkins deployment", "name", name)
	i := &v1alphaJenkins.CDStageJenkinsDeployment{}
	nn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	if err := h.Client.Get(context.TODO(), nn, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return i, nil
}

func (h PutCDStageJenkinsDeployment) create(stageDeploy *v1alpha1.CDStageDeploy) error {
	name := fmt.Sprintf("%v-%v", stageDeploy.Spec.Pipeline, stageDeploy.Spec.Stage)
	vLog := log.WithValues("name", name)
	vLog.Info("cd stage jenkins deployment is not present in cluster. start creating...")

	labels, err := h.generateLabels(stageDeploy.Name, stageDeploy.Namespace)
	if err != nil {
		return errors.Wrap(err, "couldn't generate labels")
	}

	jdCommand := &v1alphaJenkins.CDStageJenkinsDeployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: util.V2APIVersion,
			Kind:       util.CDStageJenkinsDeploymentKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: stageDeploy.Namespace,
			Labels:    labels,
		},
		Spec: v1alphaJenkins.CDStageJenkinsDeploymentSpec{
			Job:  fmt.Sprintf("%v%v/job/%v", stageDeploy.Spec.Pipeline, cdPipelinePostfix, stageDeploy.Spec.Stage),
			Tags: stageDeploy.Spec.Tags,
		},
	}
	if err := h.Client.Create(context.TODO(), jdCommand); err != nil {
		return err
	}

	vLog.Info("cd stage jenkins deployment has been created.")
	return nil
}

func (h PutCDStageJenkinsDeployment) generateLabels(cdStageDeployName, ns string) (map[string]string, error) {
	ji, err := platform.GetFirstJenkinsInstance(h.Client, ns)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		jenkinsKey:       ji.Name,
		cdStageDeployKey: cdStageDeployName,
	}, nil
}

func (h PutCDStageJenkinsDeployment) update(jenkinsDeployment *v1alphaJenkins.CDStageJenkinsDeployment, tags []v1alphaJenkins.Tag) error {
	vLog := log.WithValues("name", jenkinsDeployment.Name)
	vLog.Info("cd stage jenkins deployment is present in cluster. start updating...")
	jenkinsDeployment.Spec.Tags = tags
	if err := h.Client.Update(context.TODO(), jenkinsDeployment); err != nil {
		return err
	}
	vLog.Info("cd stage jenkins deployment has been updated.")
	return nil
}
