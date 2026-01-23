package put_codebase_image_stream

import (
	"context"
	"errors"
	"fmt"
	"strings"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/codebaseimagestream"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/platform"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutCodebaseImageStream struct {
	Next   handler.CodebaseBranchHandler
	Client client.Client
}

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

	if err := h.createCodebaseImageStreamIfNotExists(
		ctrl.LoggerInto(ctx, log),
		cb,
	); err != nil {
		setFailedFields(cb, codebaseApi.PutCodebaseImageStream, err.Error())

		return err
	}

	log.Info("End creating CodebaseImageStream")

	if err := handler.NextServeOrNil(ctx, h.Next, cb); err != nil {
		return fmt.Errorf("failed to process next handler in chain: %w", err)
	}

	return nil
}

// Deprecated: We don't need to make this conversion anymore in the future.
// TODO: Remove this function in the next releases.
func ProcessNameToK8sConvention(name string) string {
	r := strings.NewReplacer("/", "-", ".", "-")
	return r.Replace(name)
}

func (h PutCodebaseImageStream) getDockerRegistryUrl(ctx context.Context, namespace string) (string, error) {
	config, err := platform.GetKrciConfig(ctx, h.Client, namespace)
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}

	if config.KrciConfigContainerRegistryHost == "" {
		return "", fmt.Errorf(
			"%s is not set in %s config map",
			platform.KrciConfigContainerRegistryHost,
			platform.KrciConfigMap,
		)
	}

	if config.KrciConfigContainerRegistrySpace == "" {
		return "", fmt.Errorf(
			"%s is not set in %s config map",
			platform.KrciConfigContainerRegistrySpace,
			platform.KrciConfigMap,
		)
	}

	return fmt.Sprintf("%s/%s", config.KrciConfigContainerRegistryHost, config.KrciConfigContainerRegistrySpace), nil
}

func (h PutCodebaseImageStream) createCodebaseImageStreamIfNotExists(
	ctx context.Context,
	codebaseBranch *codebaseApi.CodebaseBranch,
) error {
	log := ctrl.LoggerFrom(ctx)

	cis, err := h.getCodebaseImageStream(ctx, codebaseBranch)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return fmt.Errorf("failed to get CodebaseImageStream: %w", err)
		}
	}

	if err == nil {
		log.Info("CodebaseImageStream already exists. Skip creating", "CodebaseImageStream", cis.Name)

		// For backward compatibility, we need to set branch label for the existing CodebaseImageStream.
		// TODO: remove this in the next releases.
		if v, ok := cis.GetLabels()[codebaseApi.CodebaseBranchLabel]; !ok || v != codebaseBranch.Name {
			patch := client.MergeFrom(cis.DeepCopy())

			if cis.Labels == nil {
				cis.Labels = make(map[string]string, 2)
			}

			cis.Labels[codebaseApi.CodebaseBranchLabel] = codebaseBranch.Name
			cis.Labels[codebaseApi.CodebaseLabel] = codebaseBranch.Spec.CodebaseName

			if err = h.Client.Patch(ctx, cis, patch); err != nil {
				return fmt.Errorf("failed to set branch label: %w", err)
			}
		}

		return nil
	}

	registryUrl, err := h.getDockerRegistryUrl(ctx, codebaseBranch.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get container registry url: %w", err)
	}

	cis = &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      codebaseBranch.Name,
			Namespace: codebaseBranch.Namespace,
			Labels: map[string]string{
				codebaseApi.CodebaseBranchLabel: codebaseBranch.Name,
				codebaseApi.CodebaseLabel:       codebaseBranch.Spec.CodebaseName,
			},
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase:  codebaseBranch.Spec.CodebaseName,
			ImageName: fmt.Sprintf("%v/%v", registryUrl, codebaseBranch.Spec.CodebaseName),
		},
	}

	if err = controllerutil.SetControllerReference(codebaseBranch, cis, h.Client.Scheme()); err != nil {
		return fmt.Errorf("failed to set controller reference for CodebaseImageStream: %w", err)
	}

	if err = h.Client.Create(ctx, cis); err != nil {
		return fmt.Errorf("failed to create CodebaseImageStream: %w", err)
	}

	return nil
}

func (h PutCodebaseImageStream) getCodebaseImageStream(
	ctx context.Context,
	codebaseBranch *codebaseApi.CodebaseBranch,
) (*codebaseApi.CodebaseImageStream, error) {
	cis, err := codebaseimagestream.GetCodebaseImageStreamByCodebaseBaseBranchName(
		ctx,
		h.Client,
		codebaseBranch.Name,
		codebaseBranch.Namespace,
	)
	if err == nil {
		return cis, nil
	}

	if !errors.Is(err, codebaseimagestream.ErrCodebaseImageStreamNotFound) {
		return nil, fmt.Errorf("failed to get CodebaseImageStream: %w", err)
	}

	// Get CodebaseImageStream by name old version for backward compatibility.
	// TODO: remove this in the next releases.
	cis = &codebaseApi.CodebaseImageStream{}

	err = h.Client.Get(ctx, types.NamespacedName{
		Namespace: codebaseBranch.Namespace,
		Name: fmt.Sprintf(
			"%v-%v",
			codebaseBranch.Spec.CodebaseName,
			ProcessNameToK8sConvention(codebaseBranch.Spec.BranchName),
		),
	}, cis)
	if err != nil {
		return nil, fmt.Errorf("failed to get CodebaseImageStream: %w", err)
	}

	return cis, nil
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
		VersionHistory:  cb.Status.VersionHistory,
		Build:           cb.Status.Build,
	}
}

func (h PutCodebaseImageStream) setIntermediateSuccessFields(
	cb *codebaseApi.CodebaseBranch,
	action codebaseApi.ActionType,
) error {
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
