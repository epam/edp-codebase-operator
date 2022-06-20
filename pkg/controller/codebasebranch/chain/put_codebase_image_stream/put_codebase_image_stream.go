package put_codebase_image_stream

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	edpComponentApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutCodebaseImageStream struct {
	Next   handler.CodebaseBranchHandler
	Client client.Client
}

const dockerRegistryName = "docker-registry"

var log = ctrl.Log.WithName("put-codebase-image-stream-chain")

func (h PutCodebaseImageStream) ServeRequest(cb *codebaseApi.CodebaseBranch) error {
	rl := log.WithValues("namespace", cb.Namespace, "codebase branch", cb.Name)
	rl.Info("start PutCodebaseImageStream chain...")

	if err := h.setIntermediateSuccessFields(cb, codebaseApi.PutCodebaseImageStream); err != nil {
		return err
	}

	c, err := util.GetCodebase(h.Client, cb.Spec.CodebaseName, cb.Namespace)
	if err != nil {
		setFailedFields(cb, codebaseApi.PutCodebaseImageStream, err.Error())
		return err
	}

	ec, err := h.getDockerRegistryEdpComponent(cb.Namespace)
	if err != nil {
		err = errors.Wrapf(err, "couldn't get %v EDP component", dockerRegistryName)
		setFailedFields(cb, codebaseApi.PutCodebaseImageStream, err.Error())
		return err
	}

	cisName := fmt.Sprintf("%v-%v", c.Name, processNameToK8sConvention(cb.Spec.BranchName))
	imageName := fmt.Sprintf("%v/%v/%v", ec.Spec.Url, cb.Namespace, cb.Spec.CodebaseName)
	if err := h.createCodebaseImageStreamIfNotExists(cisName, imageName, cb.Spec.CodebaseName, cb.Namespace); err != nil {
		setFailedFields(cb, codebaseApi.PutCodebaseImageStream, err.Error())
		return err
	}
	rl.Info("end PutCodebaseImageStream chain...")
	return handler.NextServeOrNil(h.Next, cb)
}

func processNameToK8sConvention(name string) string {
	r := strings.NewReplacer("/", "-", ".", "-")
	return r.Replace(name)
}

func (h PutCodebaseImageStream) getDockerRegistryEdpComponent(namespace string) (*edpComponentApi.EDPComponent, error) {
	ec := &edpComponentApi.EDPComponent{}
	err := h.Client.Get(context.TODO(), types.NamespacedName{
		Name:      dockerRegistryName,
		Namespace: namespace,
	}, ec)
	if err != nil {
		return nil, err
	}
	return ec, nil
}

func (h PutCodebaseImageStream) createCodebaseImageStreamIfNotExists(name, imageName, codebaseName, namespace string) error {
	cis := &codebaseApi.CodebaseImageStream{
		TypeMeta: metaV1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1",
			Kind:       "CodebaseImageStream",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase:  codebaseName,
			ImageName: imageName,
		},
	}

	if err := h.Client.Create(context.TODO(), cis); err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			log.Info("codebase image stream already exists. skip creating...", "name", cis.Name)
			return nil
		}
		return err
	}
	log.Info("codebase image stream has been created", "name", name)
	return nil
}

func setFailedFields(cb *codebaseApi.CodebaseBranch, a codebaseApi.ActionType, message string) {
	cb.Status = codebaseApi.CodebaseBranchStatus{
		Status:          util.StatusFailed,
		LastTimeUpdated: metaV1.Now(),
		Username:        "system",
		Action:          a,
		Result:          codebaseApi.Error,
		DetailedMessage: message,
		Value:           "failed",
	}
}

func (h PutCodebaseImageStream) setIntermediateSuccessFields(cb *codebaseApi.CodebaseBranch, action codebaseApi.ActionType) error {
	cb.Status = codebaseApi.CodebaseBranchStatus{
		Status:              model.StatusInit,
		LastTimeUpdated:     metaV1.Now(),
		Action:              action,
		Result:              codebaseApi.Success,
		Username:            "system",
		Value:               "inactive",
		VersionHistory:      cb.Status.VersionHistory,
		LastSuccessfulBuild: cb.Status.LastSuccessfulBuild,
		Build:               cb.Status.Build,
	}

	if err := h.Client.Status().Update(context.TODO(), cb); err != nil {
		if err := h.Client.Update(context.TODO(), cb); err != nil {
			return err
		}
	}
	return nil
}
