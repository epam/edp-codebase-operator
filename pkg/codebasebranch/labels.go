package codebasebranch

import (
	"context"
	"fmt"

	"github.com/cespare/xxhash/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

const (
	// Deprecated: Use codebaseApi.CodebaseLabel instead.
	LabelCodebaseName = "app.edp.epam.com/codebaseName"
)

func AddCodebaseLabel(ctx context.Context, k8sClient client.Client, codebaseBranch *codebaseApi.CodebaseBranch, codebaseName string) error {
	labelsToSet := map[string]string{
		LabelCodebaseName:           codebaseName, // Set it for backward compatibility. TODO: remove and use only codebaseApi.CodebaseLabel.
		codebaseApi.CodebaseLabel:   codebaseName,
		codebaseApi.BranchHashLabel: MakeGitBranchHash(codebaseBranch.Spec.BranchName),
	}

	currentLabels := codebaseBranch.GetLabels()
	if currentLabels == nil {
		currentLabels = make(map[string]string)
	}

	needsUpdate := false

	for key, value := range labelsToSet {
		if currentLabels[key] != value {
			currentLabels[key] = value
			needsUpdate = true
		}
	}

	if !needsUpdate {
		return nil
	}

	codebaseBranch.SetLabels(currentLabels)

	err := k8sClient.Update(ctx, codebaseBranch)
	if err != nil {
		return fmt.Errorf("failed to update k8s resource labels: %w", err)
	}

	return nil
}

func MakeGitBranchHash(gitBranchName string) string {
	return fmt.Sprintf("%x", xxhash.Sum64([]byte(gitBranchName)))
}
