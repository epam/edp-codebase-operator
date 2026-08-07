package codebase

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

// BranchCodebaseNameIndex indexes CodebaseBranch by the Codebase it belongs to, so that
// a deletion check reads the branches of one codebase and not of the whole namespace.
const BranchCodebaseNameIndex = "spec.codebaseName"

// IndexBranchByCodebaseName extracts the index value of BranchCodebaseNameIndex.
//
// It is exported for fake clients, which are not a client.FieldIndexer and take the
// extractor through fake.ClientBuilder.WithIndex; sharing it keeps the tested index
// identical to the registered one.
func IndexBranchByCodebaseName(o client.Object) []string {
	branch, ok := o.(*codebaseApi.CodebaseBranch)
	if !ok || branch.Spec.CodebaseName == "" {
		return nil
	}

	return []string{branch.Spec.CodebaseName}
}

// RegisterFieldIndexes must be called before the manager cache starts. A List selecting
// on an unregistered index fails outright, so a caller that skips this denies every
// deletion rather than under-reporting usage.
func RegisterFieldIndexes(ctx context.Context, indexer client.FieldIndexer) error {
	if err := indexer.IndexField(
		ctx, &codebaseApi.CodebaseBranch{}, BranchCodebaseNameIndex, IndexBranchByCodebaseName,
	); err != nil {
		return fmt.Errorf("failed to index CodebaseBranch by %s: %w", BranchCodebaseNameIndex, err)
	}

	return nil
}
