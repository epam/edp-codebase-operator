package put_codebase_image_stream

import (
	"context"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	edpComponentV1alpha1 "github.com/epmd-edp/edp-component-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strings"
	"time"
)

type PutCodebaseImageStream struct {
	Next   handler.CodebaseBranchHandler
	Client client.Client
}

const dockerRegistryName = "docker-registry"

var log = logf.Log.WithName("put-codebase-image-stream-chain")

func (h PutCodebaseImageStream) ServeRequest(cb *v1alpha1.CodebaseBranch) error {
	rl := log.WithValues("namespace", cb.Namespace, "codebase branch", cb.Name)
	rl.Info("start PutCodebaseImageStream chain...")

	c, err := util.GetCodebase(h.Client, cb.Spec.CodebaseName, cb.Namespace)
	if err != nil {
		setFailedFields(cb, v1alpha1.PutCodebaseImageStream, err.Error())
		return err
	}

	ec, err := h.getDockerRegistryEdpComponent(cb.Namespace)
	if err != nil {
		err = errors.Wrapf(err, "couldn't get %v EDP component", dockerRegistryName)
		setFailedFields(cb, v1alpha1.PutCodebaseImageStream, err.Error())
		return err
	}

	cisName := createCodebaseImageStreamName(c.Name, cb.Spec.BranchName, string(c.Spec.Versioning.Type))
	imageName := fmt.Sprintf("%v/%v", ec.Spec.Url, cisName)
	if err := h.createCodebaseImageStreamIfNotExists(cisName, cb.Namespace, imageName); err != nil {
		setFailedFields(cb, v1alpha1.PutCodebaseImageStream, err.Error())
		return err
	}
	rl.Info("end PutCodebaseImageStream chain...")
	return nil
}

func createCodebaseImageStreamName(codebaseName, branchName, versioningType string) string {
	if versioningType == util.VersioningTypeEDP {
		return fmt.Sprintf("%v-edp-%v", codebaseName, processNameToK8sConvention(branchName))
	}
	return fmt.Sprintf("%v-%v", codebaseName, processNameToK8sConvention(branchName))
}

func processNameToK8sConvention(name string) string {
	return strings.Replace(name, "/", "-", -1)
}

func (h PutCodebaseImageStream) getDockerRegistryEdpComponent(namespace string) (*edpComponentV1alpha1.EDPComponent, error) {
	ec := &edpComponentV1alpha1.EDPComponent{}
	err := h.Client.Get(context.TODO(), types.NamespacedName{
		Name:      dockerRegistryName,
		Namespace: namespace,
	}, ec)
	if err != nil {
		return nil, err
	}
	return ec, nil
}

func (h PutCodebaseImageStream) createCodebaseImageStreamIfNotExists(name, namespace, imageName string) error {
	cis := &v1alpha1.CodebaseImageStream{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1alpha1",
			Kind:       "CodebaseImageStream",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.CodebaseImageStreamSpec{
			ImageName: imageName,
		},
	}

	if err := h.Client.Create(context.TODO(), cis); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			log.Info("codebase image stream already exists. skip creating...", "name", cis.Name)
			return nil
		}
		return err
	}
	log.Info("codebase image stream has been created", "name", name)
	return nil
}

func setFailedFields(cb *v1alpha1.CodebaseBranch, a v1alpha1.ActionType, message string) {
	cb.Status = v1alpha1.CodebaseBranchStatus{
		Status:          util.StatusFailed,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          a,
		Result:          edpv1alpha1.Error,
		DetailedMessage: message,
		Value:           "failed",
	}
}
