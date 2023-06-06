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
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	cHand "github.com/epam/edp-codebase-operator/v2/controllers/codebase/service/chain/handler"
	handlermocks "github.com/epam/edp-codebase-operator/v2/controllers/codebase/service/chain/handler/mocks"
	"github.com/epam/edp-codebase-operator/v2/pkg/objectmodifier"
)

func TestReconcileCodebase_Reconcile(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	defaultNs := "default"

	tests := []struct {
		name        string
		request     reconcile.Request
		objects     []runtime.Object
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
			objects: []runtime.Object{
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
					mock := handlermocks.NewCodebaseHandler(t)

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
			objects: []runtime.Object{
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
					mock := handlermocks.NewCodebaseHandler(t)

					mock.On("ServeRequest", testify.Anything, cr).Return(errors.New("some error"))

					return mock, nil
				}
			},
			want:    reconcile.Result{RequeueAfter: 10 * time.Second},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build()

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
		})
	}
}
