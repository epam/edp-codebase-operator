package chain

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestCleaner_ServeRequest(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, coreV1.AddToScheme(scheme))
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	secretName := "test-secret"

	tests := []struct {
		name     string
		codebase *codebaseApi.Codebase
		prepare  func(t *testing.T, testDir string)
		wantErr  require.ErrorAssertionFunc
		objects  []client.Object
		verify   func(t *testing.T, cl client.Client, testDir string)
	}{
		{
			name: "successfully deletes secret and work directory when ClearSecretAfterUse is true",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      fakeName,
					Namespace: fakeNamespace,
				},
				Spec: codebaseApi.CodebaseSpec{
					CloneRepositoryCredentials: &codebaseApi.CloneRepositoryCredentials{
						SecretRef: coreV1.LocalObjectReference{
							Name: secretName,
						},
						ClearSecretAfterUse: true,
					},
				},
			},
			prepare: prepareWorkDir,
			wantErr: require.NoError,
			objects: []client.Object{
				&coreV1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      secretName,
						Namespace: fakeNamespace,
					},
				},
			},
			verify: func(t *testing.T, cl client.Client, testDir string) {
				// Secret should be deleted
				secret := &coreV1.Secret{}
				err := cl.Get(context.Background(), client.ObjectKey{Name: secretName, Namespace: fakeNamespace}, secret)
				assert.Error(t, err, "secret should be deleted")

				verifyWorkDirDeleted(t)
			},
		},
		{
			name: "successfully completes when secret not found but ClearSecretAfterUse is true",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      fakeName,
					Namespace: fakeNamespace,
				},
				Spec: codebaseApi.CodebaseSpec{
					CloneRepositoryCredentials: &codebaseApi.CloneRepositoryCredentials{
						SecretRef: coreV1.LocalObjectReference{
							Name: secretName,
						},
						ClearSecretAfterUse: true,
					},
				},
			},
			prepare: prepareWorkDir,
			wantErr: require.NoError,
			objects: []client.Object{},
			verify: func(t *testing.T, cl client.Client, testDir string) {
				verifyWorkDirDeleted(t)
			},
		},
		{
			name: "deletes fallback secret when CloneRepositoryCredentials is nil",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      fakeName,
					Namespace: fakeNamespace,
				},
				Spec: codebaseApi.CodebaseSpec{
					CloneRepositoryCredentials: nil,
				},
			},
			prepare: prepareWorkDir,
			wantErr: require.NoError,
			objects: []client.Object{
				&coreV1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						// Fallback secret should be deleted even when CloneRepositoryCredentials is nil
						Name:      "repository-codebase-fake-name-temp",
						Namespace: fakeNamespace,
					},
				},
			},
			verify: func(t *testing.T, cl client.Client, testDir string) {
				// Fallback secret should be deleted
				secret := &coreV1.Secret{}
				err := cl.Get(context.Background(), client.ObjectKey{
					Name:      "repository-codebase-fake-name-temp",
					Namespace: fakeNamespace,
				}, secret)
				assert.Error(t, err, "fallback secret should be deleted")

				verifyWorkDirDeleted(t)
			},
		},
		{
			name: "skips secret deletion when ClearSecretAfterUse is false",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      fakeName,
					Namespace: fakeNamespace,
				},
				Spec: codebaseApi.CodebaseSpec{
					CloneRepositoryCredentials: &codebaseApi.CloneRepositoryCredentials{
						SecretRef: coreV1.LocalObjectReference{
							Name: secretName,
						},
						ClearSecretAfterUse: false,
					},
				},
			},
			prepare: prepareWorkDir,
			wantErr: require.NoError,
			objects: []client.Object{
				&coreV1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      secretName,
						Namespace: fakeNamespace,
					},
				},
			},
			verify: func(t *testing.T, cl client.Client, testDir string) {
				// Secret should still exist
				secret := &coreV1.Secret{}
				err := cl.Get(context.Background(), client.ObjectKey{Name: secretName, Namespace: fakeNamespace}, secret)
				assert.NoError(t, err, "secret should still exist")

				verifyWorkDirDeleted(t)
			},
		},
		{
			name: "uses GetCloneRepositoryCredentialSecret to get secret name",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      fakeName,
					Namespace: fakeNamespace,
				},
				Spec: codebaseApi.CodebaseSpec{
					CloneRepositoryCredentials: &codebaseApi.CloneRepositoryCredentials{
						SecretRef: coreV1.LocalObjectReference{
							Name: "", // Empty name should trigger fallback
						},
						ClearSecretAfterUse: true,
					},
				},
			},
			prepare: prepareWorkDir,
			wantErr: require.NoError,
			objects: []client.Object{
				&coreV1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						// Fallback secret name generated by GetCloneRepositoryCredentialSecret
						Name:      "repository-codebase-fake-name-temp",
						Namespace: fakeNamespace,
					},
				},
			},
			verify: func(t *testing.T, cl client.Client, testDir string) {
				// Fallback secret should be deleted
				secret := &coreV1.Secret{}
				err := cl.Get(context.Background(), client.ObjectKey{
					Name:      "repository-codebase-fake-name-temp",
					Namespace: fakeNamespace,
				}, secret)
				assert.Error(t, err, "fallback secret should be deleted")

				verifyWorkDirDeleted(t)
			},
		},
		{
			name: "successfully completes when CloneRepositoryCredentials is nil and fallback secret not found",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      fakeName,
					Namespace: fakeNamespace,
				},
				Spec: codebaseApi.CodebaseSpec{
					CloneRepositoryCredentials: nil,
				},
			},
			prepare: prepareWorkDir,
			wantErr: require.NoError,
			objects: []client.Object{},
			verify: func(t *testing.T, cl client.Client, testDir string) {
				verifyWorkDirDeleted(t)
			},
		},
		{
			name: "deletes work directory even when work directory doesn't exist",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      fakeName,
					Namespace: fakeNamespace,
				},
				Spec: codebaseApi.CodebaseSpec{
					CloneRepositoryCredentials: nil,
				},
			},
			prepare: func(t *testing.T, testDir string) {
				t.Helper()
				t.Setenv(util.WorkDirEnv, testDir)
				// Don't create work directory - it doesn't exist
			},
			wantErr: require.NoError,
			objects: []client.Object{},
			verify: func(t *testing.T, cl client.Client, testDir string) {
				verifyWorkDirDeleted(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := t.TempDir()
			tt.prepare(t, testDir)

			fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build()
			cl := NewCleaner(fakeCl)

			err := cl.ServeRequest(context.Background(), tt.codebase)
			tt.wantErr(t, err)

			if tt.verify != nil {
				tt.verify(t, fakeCl, testDir)
			}
		})
	}
}

func prepareWorkDir(t *testing.T, testDir string) {
	t.Helper()
	t.Setenv(util.WorkDirEnv, testDir)

	// Create work directory structure
	workDir := util.GetWorkDir(fakeName, fakeNamespace)
	require.NoError(t, os.MkdirAll(workDir, 0755))
}

func verifyWorkDirDeleted(t *testing.T) {
	t.Helper()

	// Work directory should be deleted
	// Note: util.GetWorkDir() uses the WORKING_DIR env var set by prepareWorkDir
	workDir := util.GetWorkDir(fakeName, fakeNamespace)
	_, err := os.Stat(workDir)
	require.ErrorIs(t, err, os.ErrNotExist, "work directory should be deleted")
}
