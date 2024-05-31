package cdstagedeploy

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/cdstagedeploy/chain"
	"github.com/epam/edp-codebase-operator/v2/controllers/cdstagedeploy/chain/mocks"
)

func TestReconcileCDStageDeploy_Reconcile(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	tests := []struct {
		name         string
		request      reconcile.Request
		client       func(t *testing.T) client.Client
		chainFactory func(t *testing.T) chain.CDStageDeployChain
		want         reconcile.Result
		wantErr      require.ErrorAssertionFunc
	}{
		{
			name: "success reconciliation",
			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "deploy",
					Namespace: "default",
				},
			},
			client: func(t *testing.T) client.Client {
				dp := &codebaseApi.CDStageDeploy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "deploy",
						Namespace: "default",
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(dp).Build()
			},
			chainFactory: func(t *testing.T) chain.CDStageDeployChain {
				return func(cl client.Client) chain.CDStageDeployHandler {
					m := mocks.NewMockCDStageDeployHandler(t)

					m.On("ServeRequest", mock.Anything, mock.Anything).
						Return(nil)

					return m
				}
			},
			want: reconcile.Result{
				RequeueAfter: requestTimeout,
			},
			wantErr: require.NoError,
		},
		{
			name: "failed reconciliation",
			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "deploy",
					Namespace: "default",
				},
			},
			client: func(t *testing.T) client.Client {
				dp := &codebaseApi.CDStageDeploy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "deploy",
						Namespace: "default",
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(dp).Build()
			},
			chainFactory: func(t *testing.T) chain.CDStageDeployChain {
				return func(cl client.Client) chain.CDStageDeployHandler {
					m := mocks.NewMockCDStageDeployHandler(t)

					m.On("ServeRequest", mock.Anything, mock.Anything).
						Return(errors.New("failed to serve request"))

					return m
				}
			},
			want: reconcile.Result{},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to serve request")
			},
		},
		{
			name: "CDStageDeploy not found",
			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "deploy",
					Namespace: "default",
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			chainFactory: func(t *testing.T) chain.CDStageDeployChain {
				return func(cl client.Client) chain.CDStageDeployHandler {
					m := mocks.NewMockCDStageDeployHandler(t)

					m.On("ServeRequest", mock.Anything, mock.Anything).
						Return(nil)

					return m
				}
			},
			want:    reconcile.Result{},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReconcileCDStageDeploy(tt.client(t), logr.Discard(), tt.chainFactory(t))
			got, err := r.Reconcile(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.request)

			assert.Equal(t, tt.want, got)
			tt.wantErr(t, err)
		})
	}
}
