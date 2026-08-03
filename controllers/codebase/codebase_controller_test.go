package codebase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	testify "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	cHand "github.com/epam/edp-codebase-operator/v2/controllers/codebase/service/chain/handler"
	handlermocks "github.com/epam/edp-codebase-operator/v2/controllers/codebase/service/chain/handler/mocks"
	"github.com/epam/edp-codebase-operator/v2/pkg/objectmodifier"
	"github.com/epam/edp-codebase-operator/v2/pkg/util/gitpathlabel"
)

func TestReconcileCodebase_Reconcile(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	defaultNs := "default"

	tests := []struct {
		name        string
		request     reconcile.Request
		objects     []client.Object
		chainGetter func(t *testing.T) func(cr *codebaseApi.Codebase) (cHand.CodebaseHandler, error)
		want        reconcile.Result
		wantErr     require.ErrorAssertionFunc
	}{
		{
			name: "should reconcile successfully",
			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: defaultNs,
					Name:      "codebase",
				},
			},
			objects: []client.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "codebase",
						Namespace: defaultNs,
					},
					Spec: codebaseApi.CodebaseSpec{
						GitUrlPath: "/owner/repo",
						Strategy:   codebaseApi.Create,
					},
				},
			},
			chainGetter: func(t *testing.T) func(cr *codebaseApi.Codebase) (cHand.CodebaseHandler, error) {
				return func(cr *codebaseApi.Codebase) (cHand.CodebaseHandler, error) {
					mock := handlermocks.NewMockCodebaseHandler(t)

					mock.On("ServeRequest", testify.Anything, cr).Return(nil)

					return mock, nil
				}
			},
			want:    reconcile.Result{},
			wantErr: require.NoError,
		},
		{
			name: "chain failed",
			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: defaultNs,
					Name:      "codebase",
				},
			},
			objects: []client.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "codebase",
						Namespace: defaultNs,
					},
					Spec: codebaseApi.CodebaseSpec{
						GitUrlPath: "/owner/repo",
						Strategy:   codebaseApi.Create,
					},
				},
			},
			chainGetter: func(t *testing.T) func(cr *codebaseApi.Codebase) (cHand.CodebaseHandler, error) {
				return func(cr *codebaseApi.Codebase) (cHand.CodebaseHandler, error) {
					mock := handlermocks.NewMockCodebaseHandler(t)

					mock.On("ServeRequest", testify.Anything, cr).Return(errors.New("some error"))

					return mock, nil
				}
			},
			want:    reconcile.Result{RequeueAfter: 10 * time.Second},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithStatusSubresource(tt.objects...).
				Build()

			r := &ReconcileCodebase{
				client:      k8sClient,
				scheme:      scheme,
				log:         logr.Discard(),
				chainGetter: tt.chainGetter(t),
				modifier:    objectmodifier.NewCodebaseModifier(k8sClient),
			}

			got, err := r.Reconcile(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.request)
			tt.wantErr(t, err)
			require.Equal(t, tt.want, got)

			// initLabels runs before the chain, so the hash label must be
			// persisted regardless of chain success or failure.
			persisted := &codebaseApi.Codebase{}
			require.NoError(t, k8sClient.Get(context.Background(), tt.request.NamespacedName, persisted))
			require.Equal(t,
				gitpathlabel.Hash("/owner/repo"),
				persisted.GetLabels()[codebaseApi.GitUrlPathHashLabel],
			)
		})
	}
}

func TestReconcileCodebase_initLabels(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	const (
		typeLabel     = "app.edp.epam.com/codebaseType"
		cbName        = "app"
		cbNs          = "default"
		cbType        = "application"
		canonicalHash = "89f3172369ae09a6b91fa78494b89639"
	)

	tests := []struct {
		name       string
		codebase   *codebaseApi.Codebase
		wantLabels map[string]string
	}{
		{
			name: "stamps both labels on unlabeled codebase",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{Name: cbName, Namespace: cbNs},
				Spec: codebaseApi.CodebaseSpec{
					Type:       cbType,
					GitUrlPath: "/MyOrg/My-App",
				},
			},
			wantLabels: map[string]string{
				typeLabel:                       cbType,
				codebaseApi.GitUrlPathHashLabel: canonicalHash,
			},
		},
		{
			name: "converges stale hash label, preserves existing codebaseType",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      cbName,
					Namespace: cbNs,
					Labels: map[string]string{
						typeLabel:                       "custom-type",
						codebaseApi.GitUrlPathHashLabel: "00000000000000000000000000000000",
					},
				},
				Spec: codebaseApi.CodebaseSpec{
					Type:       cbType,
					GitUrlPath: "/myorg/my-app",
				},
			},
			wantLabels: map[string]string{
				typeLabel:                       "custom-type",
				codebaseApi.GitUrlPathHashLabel: canonicalHash,
			},
		},
		{
			// The operator-upgrade path: codebaseType was stamped by an older
			// operator, the hash label does not exist yet, and user/GitOps
			// labels are present. Only the hash may be added; nothing else
			// may be touched by the merge-patch.
			name: "backfills only the hash label on a pre-upgrade codebase",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      cbName,
					Namespace: cbNs,
					Labels: map[string]string{
						typeLabel:                      cbType,
						"argocd.argoproj.io/instance":  "krci",
						"user.example.com/cost-center": "team-a",
					},
				},
				Spec: codebaseApi.CodebaseSpec{
					Type:       cbType,
					GitUrlPath: "/myorg/my-app",
				},
			},
			wantLabels: map[string]string{
				typeLabel:                       cbType,
				codebaseApi.GitUrlPathHashLabel: canonicalHash,
				"argocd.argoproj.io/instance":   "krci",
				"user.example.com/cost-center":  "team-a",
			},
		},
		{
			name: "no-op when labels are already converged",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      cbName,
					Namespace: cbNs,
					Labels: map[string]string{
						typeLabel:                       cbType,
						codebaseApi.GitUrlPathHashLabel: gitpathlabel.Hash("/myorg/my-app"),
					},
				},
				Spec: codebaseApi.CodebaseSpec{
					Type:       cbType,
					GitUrlPath: "/myorg/my-app",
				},
			},
			wantLabels: map[string]string{
				typeLabel:                       cbType,
				codebaseApi.GitUrlPathHashLabel: canonicalHash,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.codebase).
				Build()

			r := &ReconcileCodebase{
				client: k8sClient,
				scheme: scheme,
				log:    logr.Discard(),
			}

			require.NoError(t, r.initLabels(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.codebase))

			persisted := &codebaseApi.Codebase{}
			require.NoError(t, k8sClient.Get(
				context.Background(),
				types.NamespacedName{Namespace: tt.codebase.Namespace, Name: tt.codebase.Name},
				persisted,
			))
			require.Equal(t, tt.wantLabels, persisted.GetLabels())
		})
	}
}
