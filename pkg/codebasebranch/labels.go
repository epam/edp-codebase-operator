package codebasebranch

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	LabelCodebaseName = "app.edp.epam.com/codebaseName"
)

func AddCodebaseLabel(ctx context.Context, k8sClient client.Client, codebaseBranch client.Object, codebaseName string) error {
	codebaseNameLabel := LabelCodebaseName
	for k, v := range codebaseBranch.GetLabels() {
		if k == codebaseNameLabel && v == codebaseName {
			return nil // label exists, skip update
		}
	}

	newLabels := codebaseBranch.GetLabels()
	if newLabels == nil {
		newLabels = make(map[string]string)
	}
	newLabels[codebaseNameLabel] = codebaseName
	codebaseBranch.SetLabels(newLabels)
	return k8sClient.Update(ctx, codebaseBranch)
}
