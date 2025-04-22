package put_codebase_image_stream

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/codebaseimagestream"
	"github.com/epam/edp-codebase-operator/v2/pkg/platform"
)

func TestPutCodebaseImageStream_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		codebaseBranch *codebaseApi.CodebaseBranch
		objects        []client.Object
		wantErr        require.ErrorAssertionFunc
		want           func(t *testing.T, k8sCl client.Client)
	}{
		{
			name: "successfully put codebase image stream - get docker registry url from config map",
			codebaseBranch: &codebaseApi.CodebaseBranch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-branch",
					Namespace: "default",
				},
				Spec: codebaseApi.CodebaseBranchSpec{
					CodebaseName: "test-codebase",
					BranchName:   "test-branch-master",
				},
			},
			objects: []client.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-codebase",
						Namespace: "default",
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      platform.KrciConfigMap,
						Namespace: "default",
					},
					Data: map[string]string{
						platform.KrciConfigContainerRegistryHost:  "test-registry",
						platform.KrciConfigContainerRegistrySpace: "test-space",
					},
				},
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sCl client.Client) {
				cis, err := codebaseimagestream.GetCodebaseImageStreamByCodebaseBaseBranchName(context.Background(), k8sCl, "test-branch", "default")
				require.NoError(t, err)

				require.NotNil(t, metav1.GetControllerOf(cis))
				require.Contains(t, cis.Labels, codebaseApi.CodebaseBranchLabel)
				require.Equal(t, cis.Labels[codebaseApi.CodebaseBranchLabel], "test-branch")
			},
		},
		{
			name: "codebase image stream already exists",
			codebaseBranch: &codebaseApi.CodebaseBranch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-branch",
					Namespace: "default",
				},
				Spec: codebaseApi.CodebaseBranchSpec{
					CodebaseName: "test-codebase",
					BranchName:   "test-branch-master",
				},
			},
			objects: []client.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-codebase",
						Namespace: "default",
					},
				},
				&codebaseApi.CodebaseImageStream{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cis",
						Namespace: "default",
						Labels: map[string]string{
							codebaseApi.CodebaseBranchLabel: "test-branch",
						},
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      platform.KrciConfigMap,
						Namespace: "default",
					},
					Data: map[string]string{
						platform.KrciConfigContainerRegistryHost:  "test-registry",
						platform.KrciConfigContainerRegistrySpace: "test-space",
					},
				},
			},
			wantErr: require.NoError,
			want:    func(t *testing.T, k8sCl client.Client) {},
		},
		{
			name: "process with deprecated relation to CodebaseBranch",
			codebaseBranch: &codebaseApi.CodebaseBranch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-branch",
					Namespace: "default",
				},
				Spec: codebaseApi.CodebaseBranchSpec{
					CodebaseName: "test-codebase",
					BranchName:   "test-branch/master",
				},
			},
			objects: []client.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-codebase",
						Namespace: "default",
					},
				},
				&codebaseApi.CodebaseImageStream{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-codebase-test-branch-master",
						Namespace: "default",
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      platform.KrciConfigMap,
						Namespace: "default",
					},
					Data: map[string]string{
						platform.KrciConfigContainerRegistryHost:  "test-registry",
						platform.KrciConfigContainerRegistrySpace: "test-space",
					},
				},
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sCl client.Client) {
				cis, err := codebaseimagestream.GetCodebaseImageStreamByCodebaseBaseBranchName(context.Background(), k8sCl, "test-branch", "default")
				require.NoError(t, err)

				require.Contains(t, cis.Labels, codebaseApi.CodebaseBranchLabel)
				require.Equal(t, cis.Labels[codebaseApi.CodebaseBranchLabel], "test-branch")
			},
		},
		{
			name: "failed to get registry url",
			codebaseBranch: &codebaseApi.CodebaseBranch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-branch",
					Namespace: "default",
				},
				Spec: codebaseApi.CodebaseBranchSpec{
					CodebaseName: "test-codebase",
					BranchName:   "test-branch-master",
				},
			},
			objects: []client.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-codebase",
						Namespace: "default",
					},
				},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get container registry url")
			},
		},
		{
			name: "failed to get codebase",
			codebaseBranch: &codebaseApi.CodebaseBranch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-branch",
					Namespace: "default",
				},
				Spec: codebaseApi.CodebaseBranchSpec{
					CodebaseName: "test-codebase",
					BranchName:   "test-branch-master",
				},
			},
			objects: []client.Object{},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to fetch Codebase resource")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scheme := runtime.NewScheme()
			require.NoError(t, codebaseApi.AddToScheme(scheme))
			require.NoError(t, corev1.AddToScheme(scheme))

			h := PutCodebaseImageStream{
				Client: fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(append(tt.objects, tt.codebaseBranch)...).
					WithStatusSubresource(append(tt.objects, tt.codebaseBranch)...).
					Build(),
			}

			err := h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.codebaseBranch)
			tt.wantErr(t, err)

			if tt.want != nil {
				tt.want(t, h.Client)
			}
		})
	}
}
