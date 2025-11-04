package codebase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/codebase"
)

func TestGetRepositoryCredentialsIfExists(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	tests := []struct {
		name     string
		codebase *codebaseApi.Codebase
		objects  []runtime.Object
		wantUser string
		wantPass string
		exists   bool
		wantErr  require.ErrorAssertionFunc
	}{
		{
			name: "credentials exist and are valid",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test-codebase",
					Namespace: "test-namespace",
				},
				Spec: codebaseApi.CodebaseSpec{
					CloneRepositoryCredentials: &codebaseApi.CloneRepositoryCredentials{
						SecretRef: corev1.LocalObjectReference{
							Name: "test-secret",
						},
					},
				},
			},
			objects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"username": []byte("test-user"),
						"password": []byte("test-pass"),
					},
				},
			},
			wantUser: "test-user",
			wantPass: "test-pass",
			exists:   true,
			wantErr:  require.NoError,
		},
		{
			name: "cloneRepositoryCredentials is nil - uses fallback secret name",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test-codebase",
					Namespace: "test-namespace",
				},
				Spec: codebaseApi.CodebaseSpec{
					CloneRepositoryCredentials: nil,
				},
			},
			objects:  []runtime.Object{},
			wantUser: "",
			wantPass: "",
			exists:   false,
			wantErr:  require.NoError,
		},
		{
			name: "secret name is empty - uses fallback secret name",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test-codebase",
					Namespace: "test-namespace",
				},
				Spec: codebaseApi.CodebaseSpec{
					CloneRepositoryCredentials: &codebaseApi.CloneRepositoryCredentials{
						SecretRef: corev1.LocalObjectReference{
							Name: "",
						},
					},
				},
			},
			objects:  []runtime.Object{},
			wantUser: "",
			wantPass: "",
			exists:   false,
			wantErr:  require.NoError,
		},
		{
			name: "secret does not exist (NotFound)",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test-codebase",
					Namespace: "test-namespace",
				},
				Spec: codebaseApi.CodebaseSpec{
					CloneRepositoryCredentials: &codebaseApi.CloneRepositoryCredentials{
						SecretRef: corev1.LocalObjectReference{
							Name: "non-existent-secret",
						},
					},
				},
			},
			objects:  []runtime.Object{},
			wantUser: "",
			wantPass: "",
			exists:   false,
			wantErr:  require.NoError,
		},
		{
			name: "username is missing in secret",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test-codebase",
					Namespace: "test-namespace",
				},
				Spec: codebaseApi.CodebaseSpec{
					CloneRepositoryCredentials: &codebaseApi.CloneRepositoryCredentials{
						SecretRef: corev1.LocalObjectReference{
							Name: "test-secret",
						},
					},
				},
			},
			objects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"password": []byte("test-pass"),
					},
				},
			},
			wantUser: "",
			wantPass: "",
			exists:   false,
			wantErr: func(t require.TestingT, err error, msgAndArgs ...any) {
				require.ErrorContains(t, err, "username key is not defined in secret")
			},
		},
		{
			name: "username is empty in secret",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test-codebase",
					Namespace: "test-namespace",
				},
				Spec: codebaseApi.CodebaseSpec{
					CloneRepositoryCredentials: &codebaseApi.CloneRepositoryCredentials{
						SecretRef: corev1.LocalObjectReference{
							Name: "test-secret",
						},
					},
				},
			},
			objects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"username": []byte(""),
						"password": []byte("test-pass"),
					},
				},
			},
			wantUser: "",
			wantPass: "",
			exists:   false,
			wantErr: func(t require.TestingT, err error, msgAndArgs ...any) {
				require.ErrorContains(t, err, "username key is not defined in secret")
			},
		},
		{
			name: "password is missing in secret",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test-codebase",
					Namespace: "test-namespace",
				},
				Spec: codebaseApi.CodebaseSpec{
					CloneRepositoryCredentials: &codebaseApi.CloneRepositoryCredentials{
						SecretRef: corev1.LocalObjectReference{
							Name: "test-secret",
						},
					},
				},
			},
			objects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"username": []byte("test-user"),
					},
				},
			},
			wantUser: "",
			wantPass: "",
			exists:   false,
			wantErr: func(t require.TestingT, err error, msgAndArgs ...any) {
				require.ErrorContains(t, err, "password key is not defined in secret")
			},
		},
		{
			name: "password is empty in secret",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "test-codebase",
					Namespace: "test-namespace",
				},
				Spec: codebaseApi.CodebaseSpec{
					CloneRepositoryCredentials: &codebaseApi.CloneRepositoryCredentials{
						SecretRef: corev1.LocalObjectReference{
							Name: "test-secret",
						},
					},
				},
			},
			objects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"username": []byte("test-user"),
						"password": []byte(""),
					},
				},
			},
			wantUser: "",
			wantPass: "",
			exists:   false,
			wantErr: func(t require.TestingT, err error, msgAndArgs ...any) {
				require.ErrorContains(t, err, "password key is not defined in secret")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build()

			gotUser, gotPass, gotExists, err := codebase.GetRepositoryCredentialsIfExists(
				context.Background(),
				tt.codebase,
				fakeClient,
			)

			tt.wantErr(t, err)
			assert.Equal(t, tt.wantUser, gotUser)
			assert.Equal(t, tt.wantPass, gotPass)
			assert.Equal(t, tt.exists, gotExists)
		})
	}
}
