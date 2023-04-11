package chain

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/cdstagedeploy/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/argocd"
)

// argoApplicationImagePatch is a struct that represents ArgoCD Application image patch.
type argoApplicationImagePatch struct {
	Spec struct {
		Source struct {
			Helm struct {
				Parameters []struct {
					Name        string `json:"name,omitempty"`
					Value       string `json:"value,omitempty"`
					ForceString bool   `json:"forceString,omitempty"`
				} `json:"parameters,omitempty"`
			} `json:"helm,omitempty"`
			TargetRevision string `json:"targetRevision,omitempty"`
		} `json:"source,omitempty"`
	} `json:"spec,omitempty"`
}

// UpdateArgoApplicationTag is chain element that updates ArgoCD Application image tag.
type UpdateArgoApplicationTag struct {
	client client.Client
	next   handler.CDStageDeployHandler
}

// ServeRequest serves request to update ArgoCD Application image tag.
func (h *UpdateArgoApplicationTag) ServeRequest(ctx context.Context, stageDeploy *codebaseApi.CDStageDeploy) error {
	log := ctrl.LoggerFrom(ctx).WithValues("imageTag", stageDeploy.Spec.Tag.Tag)
	log.Info("Updating ArgoCD Application image tag")

	argoApplication, err := h.getArgoApplicationByCDStageDeploy(ctx, stageDeploy)
	if err != nil {
		return fmt.Errorf("failed to get %v ArgoCD Application: %w", stageDeploy.Name, err)
	}

	if err := h.updateArgoApplicationImageTag(ctx, argoApplication, stageDeploy.Spec.Tag.Tag); err != nil {
		return err
	}

	log.Info("Updating ArgoCD Application image tag has been finished")

	return nextServeOrNil(ctx, h.next, stageDeploy)
}

// getArgoApplicationByCDStageDeploy returns an ArgoCD Application object found by the provided CDStageDeploy.
func (h *UpdateArgoApplicationTag) getArgoApplicationByCDStageDeploy(ctx context.Context, deploy *codebaseApi.CDStageDeploy) (*unstructured.Unstructured, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Getting Argo Application by CDStageDeploy")

	apps := argocd.NewArgoCDApplicationList()

	labelSelector := labels.SelectorFromSet(map[string]string{
		"app.edp.epam.com/app-name": deploy.Spec.Tag.Codebase,
		"app.edp.epam.com/pipeline": deploy.Spec.Pipeline,
		"app.edp.epam.com/stage":    deploy.Spec.Stage,
	})

	log.Info("Getting Argo Application with the provided labels", "labels", labelSelector.String())

	if err := h.client.List(
		ctx,
		apps,
		&client.ListOptions{
			Namespace:     deploy.Namespace,
			LabelSelector: labelSelector,
		},
	); err != nil {
		return nil, fmt.Errorf("failed to get Argo Application with the provided labels %s: %w", labelSelector, err)
	}

	if len(apps.Items) == 0 {
		return nil, fmt.Errorf("failed to find Argo Application with the provided labels %s", labelSelector)
	}

	if len(apps.Items) > 1 {
		return nil, fmt.Errorf(
			"found multiple Argo Application with the provided labels %s: %w",
			labelSelector, ErrMultipleArgoApplicationsFound,
		)
	}

	return &apps.Items[0], nil
}

// updateArgoApplicationImageTag updates ArgoCD Application image tag.
func (h *UpdateArgoApplicationTag) updateArgoApplicationImageTag(ctx context.Context, application *unstructured.Unstructured, imageTag string) error {
	var applicationPatch argoApplicationImagePatch
	if err := mapstructure.Decode(application.Object, &applicationPatch); err != nil {
		return fmt.Errorf("failed to decode ArgoCD Application spec: %w", err)
	}

	applicationPatch.Spec.Source.TargetRevision = imageTag

	for i := range applicationPatch.Spec.Source.Helm.Parameters {
		if applicationPatch.Spec.Source.Helm.Parameters[i].Name == "image.tag" {
			applicationPatch.Spec.Source.Helm.Parameters[i].Value = imageTag
		}
	}

	rawPatch, _ := json.Marshal(applicationPatch)

	if err := h.client.Patch(ctx, application, client.RawPatch(types.MergePatchType, rawPatch)); err != nil {
		return fmt.Errorf("failed to patch ArgoCD Application: %w", err)
	}

	return nil
}
