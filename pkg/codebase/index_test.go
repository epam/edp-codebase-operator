package codebase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

type stubFieldIndexer struct {
	fields []string
	err    error
}

func (s *stubFieldIndexer) IndexField(_ context.Context, _ client.Object, field string, _ client.IndexerFunc) error {
	if s.err != nil {
		return s.err
	}

	s.fields = append(s.fields, field)

	return nil
}

func TestIndexBranchByCodebaseName(t *testing.T) {
	tests := []struct {
		name string
		obj  client.Object
		want []string
	}{
		{
			name: "indexes branch by its codebase",
			obj: &codebaseApi.CodebaseBranch{
				ObjectMeta: metav1.ObjectMeta{Name: "app-main", Namespace: "default"},
				Spec:       codebaseApi.CodebaseBranchSpec{CodebaseName: "app"},
			},
			want: []string{"app"},
		},
		{
			// An empty key would collect every branch with an unset codebaseName under one
			// index entry, so such branches are left out of the index entirely.
			name: "skips branch without a codebase",
			obj: &codebaseApi.CodebaseBranch{
				ObjectMeta: metav1.ObjectMeta{Name: "orphan", Namespace: "default"},
			},
			want: nil,
		},
		{
			name: "skips object of another kind",
			obj:  &codebaseApi.Codebase{ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"}},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IndexBranchByCodebaseName(tt.obj))
		})
	}
}

func TestRegisterFieldIndexes(t *testing.T) {
	indexer := &stubFieldIndexer{}

	require.NoError(t, RegisterFieldIndexes(context.Background(), indexer))
	assert.Equal(t, []string{BranchCodebaseNameIndex}, indexer.fields)
}

// A failure to register must stop the operator rather than leave the deletion check
// selecting on an index that does not exist.
func TestRegisterFieldIndexes_Error(t *testing.T) {
	indexer := &stubFieldIndexer{err: errors.New("cache already started")}

	err := RegisterFieldIndexes(context.Background(), indexer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to index CodebaseBranch by spec.codebaseName")
}
