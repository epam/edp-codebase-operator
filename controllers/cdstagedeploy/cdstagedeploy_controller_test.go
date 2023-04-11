package cdstagedeploy

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	testify "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/cdstagedeploy/chain"
	"github.com/epam/edp-codebase-operator/v2/controllers/cdstagedeploy/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/controllers/cdstagedeploy/chain/handler/mocks"
)

func TestReconcileCDStageDeploy_Reconcile(t *testing.T) {
	t.Parallel()

	type args struct {
		request reconcile.Request
	}

	type env struct {
		j     *jenkinsApi.Jenkins
		jl    *jenkinsApi.JenkinsList
		jcdsd *jenkinsApi.CDStageJenkinsDeployment
		s     *cdPipeApi.Stage
		cdsd  *codebaseApi.CDStageDeploy
	}

	type configs struct {
		scheme func(env env) *runtime.Scheme
		client func(scheme *runtime.Scheme, env env) client.Client
		chain  func(t *testing.T) chain.CDStageDeployChain
	}

	tests := []struct {
		name                    string
		args                    args
		env                     env
		configs                 configs
		want                    reconcile.Result
		wantErr                 require.ErrorAssertionFunc
		shouldFindCDStageDeploy bool
		wantFailureCount        int64
	}{
		{
			name: "should reconcile",
			args: args{
				request: reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      "NewCDStageDeploy",
						Namespace: "namespace",
					},
				},
			},
			env: env{
				j: &jenkinsApi.Jenkins{},
				jl: &jenkinsApi.JenkinsList{
					Items: []jenkinsApi.Jenkins{
						{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "jenkins",
								Namespace: "namespace",
							},
						},
					},
				},
				jcdsd: &jenkinsApi.CDStageJenkinsDeployment{},
				s: &cdPipeApi.Stage{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "pipeline-stage",
						Namespace: "namespace",
					},
				},
				cdsd: &codebaseApi.CDStageDeploy{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "NewCDStageDeploy",
						Namespace: "namespace",
					},
					Spec: codebaseApi.CDStageDeploySpec{
						Pipeline: "pipeline",
						Stage:    "stage",
						Tag: codebaseApi.CodebaseTag{
							Codebase: "codebase",
							Tag:      "tag",
						},
					},
				},
			},
			configs: configs{
				scheme: func(env env) *runtime.Scheme {
					scheme := runtime.NewScheme()

					scheme.AddKnownTypes(metaV1.SchemeGroupVersion, env.jl, env.j)
					scheme.AddKnownTypes(codebaseApi.GroupVersion, env.cdsd)
					scheme.AddKnownTypes(cdPipeApi.GroupVersion, env.s)
					scheme.AddKnownTypes(jenkinsApi.SchemeGroupVersion, env.jcdsd)

					return scheme
				},
				client: func(scheme *runtime.Scheme, env env) client.Client {
					return fake.NewClientBuilder().
						WithScheme(scheme).
						WithRuntimeObjects(env.cdsd, env.s, env.jcdsd, env.j, env.jl).
						Build()
				},
				chain: func(t *testing.T) chain.CDStageDeployChain {
					mock := mocks.NewCDStageDeployHandler(t)

					mock.
						On("ServeRequest", testify.Anything, testify.Anything).
						Return(nil)

					return func(ctx context.Context, cl client.Client, deploy *codebaseApi.CDStageDeploy) (handler.CDStageDeployHandler, error) {
						return mock, nil
					}
				},
			},
			want: reconcile.Result{
				Requeue: false,
			},
			wantErr:                 require.NoError,
			shouldFindCDStageDeploy: false,
		},
		{
			name: "should fail to get CDStageDeploy",
			args: args{
				request: reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      "NewCDStageDeploy",
						Namespace: "namespace",
					},
				},
			},
			env: env{},
			configs: configs{
				scheme: func(env env) *runtime.Scheme {
					return runtime.NewScheme()
				},
				client: func(scheme *runtime.Scheme, _ env) client.Client {
					return fake.NewClientBuilder().WithScheme(scheme).Build()
				},
				chain: func(t *testing.T) chain.CDStageDeployChain {
					return func(ctx context.Context, cl client.Client, deploy *codebaseApi.CDStageDeploy) (handler.CDStageDeployHandler, error) {
						return mocks.NewCDStageDeployHandler(t), nil
					}
				},
			},
			want: reconcile.Result{
				Requeue: false,
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)

				errText := "no kind is registered for the type v1.CDStageDeploy in scheme"

				require.Contains(t, err.Error(), errText)
			},
		},
		{
			name: "should pass with no CDStageDeploy found",
			args: args{
				request: reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      "NewCDStageDeploy",
						Namespace: "namespace",
					},
				},
			},
			env: env{
				cdsd: &codebaseApi.CDStageDeploy{},
			},
			configs: configs{
				scheme: func(env env) *runtime.Scheme {
					scheme := runtime.NewScheme()

					scheme.AddKnownTypes(codebaseApi.GroupVersion, env.cdsd)

					return scheme
				},
				client: func(scheme *runtime.Scheme, env env) client.Client {
					return fake.NewClientBuilder().
						WithScheme(scheme).
						WithRuntimeObjects(env.cdsd).
						Build()
				},
				chain: func(t *testing.T) chain.CDStageDeployChain {
					return func(ctx context.Context, cl client.Client, deploy *codebaseApi.CDStageDeploy) (handler.CDStageDeployHandler, error) {
						return mocks.NewCDStageDeployHandler(t), nil
					}
				},
			},
			want: reconcile.Result{
				Requeue: false,
			},
			wantErr:                 require.NoError,
			shouldFindCDStageDeploy: false,
		},
		{
			name: "should fail set owner ref",
			args: args{
				request: reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      "NewCDStageDeploy",
						Namespace: "namespace",
					},
				},
			},
			env: env{
				cdsd: &codebaseApi.CDStageDeploy{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "NewCDStageDeploy",
						Namespace: "namespace",
						DeletionTimestamp: &metaV1.Time{
							Time: metaV1.Now().Time,
						},
					},
				},
			},
			configs: configs{
				scheme: func(env env) *runtime.Scheme {
					scheme := runtime.NewScheme()

					scheme.AddKnownTypes(codebaseApi.GroupVersion, env.cdsd)

					return scheme
				},
				client: func(scheme *runtime.Scheme, env env) client.Client {
					return fake.NewClientBuilder().
						WithScheme(scheme).
						WithRuntimeObjects(env.cdsd).
						Build()
				},
				chain: func(t *testing.T) chain.CDStageDeployChain {
					return func(ctx context.Context, cl client.Client, deploy *codebaseApi.CDStageDeploy) (handler.CDStageDeployHandler, error) {
						return mocks.NewCDStageDeployHandler(t), nil
					}
				},
			},
			want: reconcile.Result{
				Requeue: false,
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)

				errText := "no kind is registered for the type v1.Stage in scheme"

				require.Contains(t, err.Error(), errText)
			},
		},
		{
			name: "should fail to serve request",
			args: args{
				request: reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      "NewCDStageDeploy",
						Namespace: "namespace",
					},
				},
			},
			env: env{
				s: &cdPipeApi.Stage{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "pipeline-stage",
						Namespace: "namespace",
					},
				},
				cdsd: &codebaseApi.CDStageDeploy{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "NewCDStageDeploy",
						Namespace: "namespace",
					},
					Spec: codebaseApi.CDStageDeploySpec{
						Pipeline: "pipeline",
						Stage:    "stage",
						Tag: codebaseApi.CodebaseTag{
							Codebase: "codebase",
							Tag:      "tag",
						},
					},
				},
			},
			configs: configs{
				scheme: func(env env) *runtime.Scheme {
					scheme := runtime.NewScheme()

					scheme.AddKnownTypes(cdPipeApi.GroupVersion, env.s)
					scheme.AddKnownTypes(codebaseApi.GroupVersion, env.cdsd)

					return scheme
				},
				client: func(scheme *runtime.Scheme, env env) client.Client {
					return fake.NewClientBuilder().
						WithScheme(scheme).
						WithRuntimeObjects(env.cdsd, env.s).
						Build()
				},
				chain: func(t *testing.T) chain.CDStageDeployChain {
					mock := mocks.NewCDStageDeployHandler(t)

					mock.
						On("ServeRequest", testify.Anything, testify.Anything).
						Return(errors.New("chainFactory failed"))

					return func(ctx context.Context, cl client.Client, deploy *codebaseApi.CDStageDeploy) (handler.CDStageDeployHandler, error) {
						return mock, nil
					}
				},
			},
			want: reconcile.Result{
				Requeue: false,
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)

				require.Contains(t, err.Error(), "chainFactory failed")
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scheme := tt.configs.scheme(tt.env)
			fakeClient := tt.configs.client(scheme, tt.env)

			r := NewReconcileCDStageDeploy(
				fakeClient,
				scheme,
				logr.Discard(),
				tt.configs.chain(t),
			)

			ctx := context.Background()

			got, err := r.Reconcile(ctx, tt.args.request)

			tt.wantErr(t, err)

			assert.Equal(t, tt.want, got)

			if tt.shouldFindCDStageDeploy {
				cdsdResp := &codebaseApi.CDStageDeploy{}

				err = fakeClient.Get(ctx,
					types.NamespacedName{
						Name:      tt.args.request.Name,
						Namespace: tt.args.request.Namespace,
					},
					cdsdResp)
				assert.NoError(t, err)

				assert.Equal(t, cdsdResp.Status.FailureCount, tt.wantFailureCount)
			}
		})
	}
}
