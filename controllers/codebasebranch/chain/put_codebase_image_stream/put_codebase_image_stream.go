package put_codebase_image_stream

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/platform"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutCodebaseImageStream struct {
	Next   handler.CodebaseBranchHandler
	Client client.Client
}

const (
	EdpConfigContainerRegistryHost  = "container_registry_host"
	EdpConfigContainerRegistrySpace = "container_registry_space"
)

func (h PutCodebaseImageStream) ServeRequest(ctx context.Context, cb *codebaseApi.CodebaseBranch) error {
	log := ctrl.LoggerFrom(ctx).WithName("put-codebase-image-stream")

	log.Info("Start creating CodebaseImageStream")

	if err := h.setIntermediateSuccessFields(cb, codebaseApi.PutCodebaseImageStream); err != nil {
		return err
	}

	c := &codebaseApi.Codebase{}
	if err := h.Client.Get(ctx, types.NamespacedName{
		Namespace: cb.Namespace,
		Name:      cb.Spec.CodebaseName,
	}, c); err != nil {
		setFailedFields(cb, codebaseApi.PutCodebaseImageStream, err.Error())

		return fmt.Errorf("failed to fetch Codebase resource: %w", err)
	}

	registryUrl, err := h.getDockerRegistryUrl(ctx, cb.Namespace)
	if err != nil {
		err = fmt.Errorf("failed to get container registry url: %w", err)
		setFailedFields(cb, codebaseApi.PutCodebaseImageStream, err.Error())

		return err
	}

	cisName := fmt.Sprintf("%v-%v", c.Name, ProcessNameToK8sConvention(cb.Spec.BranchName))
	imageName := fmt.Sprintf("%v/%v", registryUrl, cb.Spec.CodebaseName)

	if err = h.createCodebaseImageStreamIfNotExists(
		ctrl.LoggerInto(ctx, log),
		cisName,
		imageName,
		cb.Spec.CodebaseName,
		cb.Namespace,
		cb,
	); err != nil {
		setFailedFields(cb, codebaseApi.PutCodebaseImageStream, err.Error())

		return err
	}

	log.Info("End creating CodebaseImageStream")

	err = handler.NextServeOrNil(ctx, h.Next, cb)
	if err != nil {
		return fmt.Errorf("failed to process next handler in chain: %w", err)
	}

	return nil
}

func ProcessNameToK8sConvention(name string) string {
	r := strings.NewReplacer("/", "-", ".", "-")
	return r.Replace(name)
}

func (h PutCodebaseImageStream) getDockerRegistryUrl(ctx context.Context, namespace string) (string, error) {
	config := &corev1.ConfigMap{}
	if err := h.Client.Get(ctx, types.NamespacedName{
		Name:      platform.EdpConfigMap,
		Namespace: namespace,
	}, config); err != nil {
		return "", fmt.Errorf("failed to get %s config map: %w", platform.EdpConfigMap, err)
	}

	if _, ok := config.Data[EdpConfigContainerRegistryHost]; !ok {
		return "", fmt.Errorf("%s is not set in %s config map", EdpConfigContainerRegistryHost, platform.EdpConfigMap)
	}

	if _, ok := config.Data[EdpConfigContainerRegistrySpace]; !ok {
		return "", fmt.Errorf("%s is not set in %s config map", EdpConfigContainerRegistrySpace, platform.EdpConfigMap)
	}

	return fmt.Sprintf("%s/%s", config.Data[EdpConfigContainerRegistryHost], config.Data[EdpConfigContainerRegistrySpace]), nil
}

func (h PutCodebaseImageStream) createCodebaseImageStreamIfNotExists(
	ctx context.Context,
	name, imageName, codebaseName, namespace string,
	codebaseBranch *codebaseApi.CodebaseBranch,
) error {
	log := ctrl.LoggerFrom(ctx)

	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase:  codebaseName,
			ImageName: imageName,
		},
	}

	if err := controllerutil.SetControllerReference(codebaseBranch, cis, h.Client.Scheme()); err != nil {
		return fmt.Errorf("failed to set controller reference for CodebaseImageStream: %w", err)
	}

	if err := h.Client.Create(ctx, cis); err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			log.Info("CodebaseImageStream already exists. Skip creating", "CodebaseImageStream", cis.Name)

			// For backward compatibility, we need to update the controller reference for the existing CodebaseImageStream.
			// We can remove this in the next releases.
			existingCIS := &codebaseApi.CodebaseImageStream{}
			if err = h.Client.Get(ctx, types.NamespacedName{
				Namespace: namespace,
				Name:      name,
			}, existingCIS); err != nil {
				return fmt.Errorf("failed to get CodebaseImageStream: %w", err)
			}

			if metaV1.GetControllerOf(existingCIS) == nil {
				if err = controllerutil.SetControllerReference(codebaseBranch, existingCIS, h.Client.Scheme()); err != nil {
					return fmt.Errorf("failed to set controller reference for CodebaseImageStream: %w", err)
				}
			}

			if err = h.Client.Update(ctx, existingCIS); err != nil {
				return fmt.Errorf("failed to update CodebaseImageStream controller reference: %w", err)
			}

			return nil
		}

		return fmt.Errorf("failed to create CodebaseImageStream %s: %w", name, err)
	}

	log.Info("CodebaseImageStream has been created", "CodebaseImageStream", name)

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
		Git:             cb.Status.Git,
	}
}

func (h PutCodebaseImageStream) setIntermediateSuccessFields(cb *codebaseApi.CodebaseBranch, action codebaseApi.ActionType) error {
	ctx := context.Background()
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
		Git:                 cb.Status.Git,
	}

	err := h.Client.Status().Update(ctx, cb)
	if err != nil {
		return fmt.Errorf("failed to update CodebaseBranch status field %q: %w", cb.Name, err)
	}

	err = h.Client.Update(ctx, cb)
	if err != nil {
		return fmt.Errorf("failed to update CodebaseBranch resource %q: %w", cb.Name, err)
	}

	return nil
}
