package chain

import (
	"context"
	"fmt"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/chain/handler"
)

type ProcessNewVersion struct {
	Next   handler.CodebaseBranchHandler
	Client client.Client
}

func (h ProcessNewVersion) ServeRequest(ctx context.Context, codebaseBranch *codebaseApi.CodebaseBranch) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start processing new version.")

	codebase := &codebaseApi.Codebase{}
	if err := h.Client.Get(ctx, client.ObjectKey{
		Namespace: codebaseBranch.Namespace,
		Name:      codebaseBranch.Spec.CodebaseName,
	}, codebase); err != nil {
		return fmt.Errorf("failed to get Codebase: %w", err)
	}

	if err := h.processNewVersion(ctx, codebaseBranch, codebase); err != nil {
		return fmt.Errorf("failed to process new version for %s branch: %w", codebaseBranch.Name, err)
	}

	log.Info("End processing new version.")

	err := handler.NextServeOrNil(ctx, h.Next, codebaseBranch)
	if err != nil {
		return fmt.Errorf("failed to serve next chain element: %w", err)
	}

	return nil
}

func (h ProcessNewVersion) processNewVersion(ctx context.Context, codebaseBranch *codebaseApi.CodebaseBranch, codebase *codebaseApi.Codebase) error {
	log := ctrl.LoggerFrom(ctx)

	if codebase.Spec.Versioning.Type != codebaseApi.VersioningTypeEDP {
		log.Info("Versioning type is not EDP. Skip processing new version.")

		return nil
	}

	hasVersion, err := HasNewVersion(codebaseBranch)
	if err != nil {
		return fmt.Errorf("failed to check if branch %s has new version: %w", codebaseBranch.Name, err)
	}

	if !hasVersion {
		log.Info("CodebaseBranch doesn't have new version. Skip processing new version.")

		return nil
	}

	codebaseBranch.Status.Build = ptr.To("0")
	codebaseBranch.Status.LastSuccessfulBuild = nil
	codebaseBranch.Status.VersionHistory = append(codebaseBranch.Status.VersionHistory, *codebaseBranch.Spec.Version)

	if err = h.Client.Status().Update(ctx, codebaseBranch); err != nil {
		return fmt.Errorf("failed to update CodebaseBranch status: %w", err)
	}

	log.Info("CodebaseBranch status build and version history have been updated.")

	return nil
}
