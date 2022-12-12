package cdstagedeploy

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
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
						Tag: jenkinsApi.Tag{
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
					scheme.AddKnownTypes(cdPipeApi.SchemeGroupVersion, env.s)
					scheme.AddKnownTypes(jenkinsApi.SchemeGroupVersion, env.jcdsd)

					return scheme
				},
				client: func(scheme *runtime.Scheme, env env) client.Client {
					return fake.NewClientBuilder().
						WithScheme(scheme).
						WithRuntimeObjects(env.cdsd, env.s, env.jcdsd, env.j, env.jl).
						Build()
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
						Tag: jenkinsApi.Tag{
							Codebase: "codebase",
							Tag:      "tag",
						},
					},
				},
			},
			configs: configs{
				scheme: func(env env) *runtime.Scheme {
					scheme := runtime.NewScheme()

					scheme.AddKnownTypes(cdPipeApi.SchemeGroupVersion, env.s)
					scheme.AddKnownTypes(codebaseApi.GroupVersion, env.cdsd)

					return scheme
				},
				client: func(scheme *runtime.Scheme, env env) client.Client {
					return fake.NewClientBuilder().
						WithScheme(scheme).
						WithRuntimeObjects(env.cdsd, env.s).
						Build()
				},
			},
			want: reconcile.Result{
				Requeue: false,
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)

				errText := "couldn't get NewCDStageDeploy cd stage jenkins deployment"

				require.Contains(t, err.Error(), errText)
			},
		},
		{
			name: "should fail to serve request with existing CR",
			args: args{
				request: reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      "NewCDStageDeploy",
						Namespace: "namespace",
					},
				},
			},
			env: env{
				jcdsd: &jenkinsApi.CDStageJenkinsDeployment{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "NewCDStageDeploy",
						Namespace: "namespace",
					},
				},
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
						Tag: jenkinsApi.Tag{
							Codebase: "codebase",
							Tag:      "tag",
						},
					},
				},
			},
			configs: configs{
				scheme: func(env env) *runtime.Scheme {
					scheme := runtime.NewScheme()

					scheme.AddKnownTypes(cdPipeApi.SchemeGroupVersion, env.s)
					scheme.AddKnownTypes(codebaseApi.GroupVersion, env.cdsd)
					scheme.AddKnownTypes(jenkinsApi.SchemeGroupVersion, env.jcdsd)

					return scheme
				},
				client: func(scheme *runtime.Scheme, env env) client.Client {
					return fake.NewClientBuilder().
						WithScheme(scheme).
						WithRuntimeObjects(env.cdsd, env.jcdsd, env.s).
						Build()
				},
			},
			want: reconcile.Result{
				Requeue:      false,
				RequeueAfter: 10 * time.Second,
			},
			wantFailureCount: 1,
			wantErr:          require.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scheme := tt.configs.scheme(tt.env)
			fakeClient := tt.configs.client(scheme, tt.env)

			r := &ReconcileCDStageDeploy{
				client: fakeClient,
				scheme: scheme,
				log:    logr.Discard(),
			}

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

func TestNewReconcileCDStageDeploy(t *testing.T) {
	t.Parallel()

	type args struct {
		c      client.Client
		scheme *runtime.Scheme
	}

	tests := []struct {
		name string
		args args
		want *ReconcileCDStageDeploy
	}{
		{
			name: "should complete successfully",
			args: args{
				c:      fake.NewClientBuilder().Build(),
				scheme: runtime.NewScheme(),
			},
			want: &ReconcileCDStageDeploy{},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			log := logr.Discard()

			tt.want.client = tt.args.c
			tt.want.scheme = tt.args.scheme
			tt.want.log = log

			got := NewReconcileCDStageDeploy(tt.args.c, tt.args.scheme, log)

			assert.Equal(t, tt.want, got)
		})
	}
}
