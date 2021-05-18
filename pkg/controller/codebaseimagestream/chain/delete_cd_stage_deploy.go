package chain

import (
	"context"
	"fmt"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type DeleteCDStageDeploy struct {
	client client.Client
	log    logr.Logger
}

const (
	commaSplitSeparator = ","
	slashSplitSeparator = "/"
)

func (h DeleteCDStageDeploy) ServeRequest(is *codebaseApi.CodebaseImageStream) error {
	val := is.GetAnnotations()[util.LastDeletedEnvsAnnotationKey]
	if len(val) == 0 {
		h.log.Info("CodebaseImageStream doesn't contain %v annotation. skip deleting CdStageDeploy resources")
		return nil
	}
	log.Info("deleting CdStageDeploy resources", "value", val)

	envs := strings.Split(val, commaSplitSeparator)
	for _, env := range envs {
		tmp := strings.Split(env, slashSplitSeparator)
		stageDeployName := fmt.Sprintf("%v-%v", tmp[0], tmp[1])
		if err := h.deleteCdStageDeploy(stageDeployName, is.Namespace); err != nil {
			return errors.Wrapf(err, "unable to delete %v CdStageDeploy resource", stageDeployName)
		}
		if err := h.updateAnnotation(is, env); err != nil {
			return errors.Wrapf(err, "couldn't update %v CodebaseImageStream", is.Name)
		}
	}
	return nil
}

func (h DeleteCDStageDeploy) deleteCdStageDeploy(name, namespace string) error {
	err := h.client.Delete(context.TODO(), &codebaseApi.CDStageDeploy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			h.log.Info("unable to delete CdStageDeploy resource as it's absent in cluster",
				"name", name)
			return nil
		}
		return err
	}
	h.log.Info("CdStageDeploy has been deleted", "name", name)
	return nil
}

func (h DeleteCDStageDeploy) updateAnnotation(is *codebaseApi.CodebaseImageStream, deletedEnv string) error {
	envs := strings.Split(is.GetAnnotations()[util.LastDeletedEnvsAnnotationKey], commaSplitSeparator)
	envs = remove(envs, deletedEnv)
	setAnnotation(is.Annotations, envs)

	return h.client.Update(context.TODO(), is)
}

func setAnnotation(annotations map[string]string, envs []string) {
	if envs == nil {
		delete(annotations, util.LastDeletedEnvsAnnotationKey)
		return
	}
	annotations[util.LastDeletedEnvsAnnotationKey] = strings.Join(envs, commaSplitSeparator)
}

func remove(envs []string, val string) []string {
	for i, env := range envs {
		if env == val {
			if len(envs) == 1 {
				return nil
			}
			return append(envs[:i], envs[i+1])
		}
	}
	return envs
}
