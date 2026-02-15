// nolint:dupl // Duplicate test setup is acceptable in tests for readability
package gitserver

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	routeApi "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/platform"
	"github.com/epam/edp-codebase-operator/v2/pkg/tektoncd"
)

func TestCreateEventListener_ServeRequest(t *testing.T) {
	scheme := runtime.NewScheme()

	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, networkingv1.AddToScheme(scheme))
	require.NoError(t, routeApi.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name      string
		gitServer *codebaseApi.GitServer
		k8sClient func(t *testing.T) client.Client
		prepare   func(t *testing.T)
		wantErr   require.ErrorAssertionFunc
		want      func(t *testing.T, k8sClient client.Client)
	}{
		{
			name: "skip creating event listener because tekton is disabled",
			gitServer: &codebaseApi.GitServer{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "test-git-server",
					Namespace: "default",
				},
				Spec: codebaseApi.GitServerSpec{
					TektonDisabled: true,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sClient client.Client) {
				el := tektoncd.NewEventListenerUnstructured()
				err := k8sClient.Get(context.Background(), client.ObjectKey{
					Namespace: "default",
					Name:      generateEventListenerName("test-git-server"),
				}, el)
				require.Error(t, err)
				require.True(t, k8sErrors.IsNotFound(err))

				i := &networkingv1.Ingress{}
				err = k8sClient.Get(context.Background(), client.ObjectKey{
					Namespace: "default",
					Name:      GenerateIngressName("test-git-server"),
				}, i)
				require.Error(t, err)
				require.True(t, k8sErrors.IsNotFound(err))
			},
		},
		{
			name: "skip creating event listener because tekton is disabled even with webhook URL set",
			gitServer: &codebaseApi.GitServer{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "test-git-server",
					Namespace: "default",
				},
				Spec: codebaseApi.GitServerSpec{
					TektonDisabled: true,
					WebhookUrl:     "https://external-webhook",
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sClient client.Client) {
				el := tektoncd.NewEventListenerUnstructured()
				err := k8sClient.Get(context.Background(), client.ObjectKey{
					Namespace: "default",
					Name:      generateEventListenerName("test-git-server"),
				}, el)
				require.Error(t, err)
				require.True(t, k8sErrors.IsNotFound(err))
			},
		},
		{
			name: "skip creating event listener because webhook URL is set",
			gitServer: &codebaseApi.GitServer{
				Spec: codebaseApi.GitServerSpec{
					WebhookUrl: "https://test-webhook-url",
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sClient client.Client) {
				el := tektoncd.NewEventListenerUnstructured()
				err := k8sClient.Get(context.Background(), client.ObjectKey{
					Namespace: "default",
					Name:      generateEventListenerName("test-git-server"),
				}, el)
				require.Error(t, err)
				require.True(t, k8sErrors.IsNotFound(err))

				i := &networkingv1.Ingress{}
				err = k8sClient.Get(context.Background(), client.ObjectKey{
					Namespace: "default",
					Name:      GenerateIngressName("test-git-server"),
				}, i)
				require.Error(t, err)
				require.True(t, k8sErrors.IsNotFound(err))
			},
		},
		{
			name: "create event listener success k8s",
			gitServer: &codebaseApi.GitServer{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "test-git-server",
					Namespace: "default",
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(&corev1.ConfigMap{
						ObjectMeta: controllerruntime.ObjectMeta{
							Namespace: "default",
							Name:      platform.KrciConfigMap,
						},
					}).
					Build()
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.K8S)
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sClient client.Client) {
				el := tektoncd.NewEventListenerUnstructured()
				require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
					Namespace: "default",
					Name:      generateEventListenerName("test-git-server"),
				}, el))

				i := &networkingv1.Ingress{}
				require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
					Namespace: "default",
					Name:      GenerateIngressName("test-git-server"),
				}, i))
			},
		},
		{
			name: "ingress already exists",
			gitServer: &codebaseApi.GitServer{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "test-git-server",
					Namespace: "default",
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&corev1.ConfigMap{
							ObjectMeta: controllerruntime.ObjectMeta{
								Namespace: "default",
								Name:      platform.KrciConfigMap,
							},
						},
						&networkingv1.Ingress{
							ObjectMeta: controllerruntime.ObjectMeta{
								Namespace: "default",
								Name:      GenerateIngressName("test-git-server"),
							},
						},
					).
					Build()
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.K8S)
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sClient client.Client) {
				el := tektoncd.NewEventListenerUnstructured()
				require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
					Namespace: "default",
					Name:      generateEventListenerName("test-git-server"),
				}, el))

				i := &networkingv1.Ingress{}
				require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
					Namespace: "default",
					Name:      GenerateIngressName("test-git-server"),
				}, i))
			},
		},
		{
			name: "event listener already exists",
			gitServer: &codebaseApi.GitServer{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "test-git-server",
					Namespace: "default",
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				el := tektoncd.NewEventListenerUnstructured()
				el.SetName("test-git-server")
				el.SetNamespace("default")

				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&corev1.ConfigMap{
							ObjectMeta: controllerruntime.ObjectMeta{
								Namespace: "default",
								Name:      platform.KrciConfigMap,
							},
						},
						el,
					).
					Build()
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.K8S)
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sClient client.Client) {
				el := tektoncd.NewEventListenerUnstructured()
				require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
					Namespace: "default",
					Name:      generateEventListenerName("test-git-server"),
				}, el))

				i := &networkingv1.Ingress{}
				require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
					Namespace: "default",
					Name:      GenerateIngressName("test-git-server"),
				}, i))
			},
		},
		{
			name: "create event listener success openshift",
			gitServer: &codebaseApi.GitServer{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "test-git-server",
					Namespace: "default",
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(&corev1.ConfigMap{
						ObjectMeta: controllerruntime.ObjectMeta{
							Namespace: "default",
							Name:      platform.KrciConfigMap,
						},
					}).
					Build()
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.Openshift)
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sClient client.Client) {
				el := tektoncd.NewEventListenerUnstructured()
				require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
					Namespace: "default",
					Name:      generateEventListenerName("test-git-server"),
				}, el))

				i := &routeApi.Route{}
				require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
					Namespace: "default",
					Name:      GenerateIngressName("test-git-server"),
				}, i))
			},
		},
		{
			name: "route already exists",
			gitServer: &codebaseApi.GitServer{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "test-git-server",
					Namespace: "default",
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&corev1.ConfigMap{
							ObjectMeta: controllerruntime.ObjectMeta{
								Namespace: "default",
								Name:      platform.KrciConfigMap,
							},
						},
						&routeApi.Route{
							ObjectMeta: controllerruntime.ObjectMeta{
								Namespace: "default",
								Name:      GenerateIngressName("test-git-server"),
							},
						},
					).
					Build()
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.Openshift)
			},
			wantErr: require.NoError,
			want: func(t *testing.T, k8sClient client.Client) {
				el := tektoncd.NewEventListenerUnstructured()
				require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
					Namespace: "default",
					Name:      generateEventListenerName("test-git-server"),
				}, el))

				i := &routeApi.Route{}
				require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
					Namespace: "default",
					Name:      GenerateIngressName("test-git-server"),
				}, i))
			},
		},
		{
			name: "config not found",
			gitServer: &codebaseApi.GitServer{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "test-git-server",
					Namespace: "default",
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects().
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get dnsWildcard")
			},
			want: func(t *testing.T, k8sClient client.Client) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sCl := tt.k8sClient(t)
			h := NewCreateEventListener(k8sCl)

			if tt.prepare != nil {
				tt.prepare(t)
			}

			tt.wantErr(
				t,
				h.ServeRequest(
					controllerruntime.LoggerInto(context.Background(), logr.Discard()),
					tt.gitServer,
				),
			)

			tt.want(t, k8sCl)
		})
	}
}
