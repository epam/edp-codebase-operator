package integrationsecret

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileIntegrationSecret_Reconcile(t *testing.T) {
	t.Parallel()

	ns := "default"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), "success") {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
	}))

	defer server.Close()

	s := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name                 string
		secretName           string
		client               func(t *testing.T) client.Client
		wantRes              reconcile.Result
		wantErr              require.ErrorAssertionFunc
		wantConAnnotation    string
		wantConErrAnnotation string
	}{
		{
			name:       "success sonarqube",
			secretName: "sonarqube",
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: ns,
							Name:      "sonarqube",
							Labels: map[string]string{
								integrationSecretTypeLabel: "sonarqube",
							},
						},
						Data: map[string][]byte{
							"url":   []byte(server.URL + "/success"),
							"token": []byte("token"),
						},
					},
				).Build()
			},
			wantRes: reconcile.Result{
				RequeueAfter: successConnectionRequeueTime,
			},
			wantErr:           require.NoError,
			wantConAnnotation: "true",
		},
		{
			name:       "fail sonarqube",
			secretName: "sonarqube",
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: ns,
							Name:      "sonarqube",
							Labels: map[string]string{
								integrationSecretTypeLabel: "sonarqube",
							},
						},
						Data: map[string][]byte{
							"url":   []byte(server.URL + "/fail"),
							"token": []byte("token"),
						},
					},
				).Build()
			},
			wantRes: reconcile.Result{
				RequeueAfter: failConnectionRequeueTime,
			},
			wantErr:              require.NoError,
			wantConAnnotation:    "false",
			wantConErrAnnotation: "connection failed",
		},
		{
			name:       "success nexus with basic auth",
			secretName: "nexus",
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: ns,
							Name:      "nexus",
							Labels: map[string]string{
								integrationSecretTypeLabel: "nexus",
							},
						},
						Data: map[string][]byte{
							"url":      []byte(server.URL + "/success"),
							"username": []byte("username"),
							"password": []byte("password"),
						},
					},
				).Build()
			},
			wantRes: reconcile.Result{
				RequeueAfter: successConnectionRequeueTime,
			},
			wantErr:           require.NoError,
			wantConAnnotation: "true",
		},
		{
			name:       "success dependency-track",
			secretName: "dependency-track",
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: ns,
							Name:      "dependency-track",
							Labels: map[string]string{
								integrationSecretTypeLabel: "dependency-track",
							},
						},
						Data: map[string][]byte{
							"url":   []byte(server.URL + "/success"),
							"token": []byte("token"),
						},
					},
				).Build()
			},
			wantRes: reconcile.Result{
				RequeueAfter: successConnectionRequeueTime,
			},
			wantErr:           require.NoError,
			wantConAnnotation: "true",
		},
		{
			name:       "success defectdojo",
			secretName: "defectdojo",
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: ns,
							Name:      "defectdojo",
							Labels: map[string]string{
								integrationSecretTypeLabel: "defectdojo",
							},
						},
						Data: map[string][]byte{
							"url":   []byte(server.URL + "/success"),
							"token": []byte("token"),
						},
					},
				).Build()
			},
			wantRes: reconcile.Result{
				RequeueAfter: successConnectionRequeueTime,
			},
			wantErr:           require.NoError,
			wantConAnnotation: "true",
		},
		{
			name:       "success registry",
			secretName: "registry",
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: ns,
							Name:      "registry",
							Labels: map[string]string{
								integrationSecretTypeLabel: "registry",
							},
						},
						Type: corev1.SecretTypeDockerConfigJson,
						Data: map[string][]byte{
							".dockerconfigjson": []byte(`{"auths":{"` + server.URL + `/success":{"username":"user1", "password":"password1"}}}`),
						},
					},
				).Build()
			},
			wantRes: reconcile.Result{
				RequeueAfter: successConnectionRequeueTime,
			},
			wantErr:           require.NoError,
			wantConAnnotation: "true",
		},
		{
			name:       "registry with invalid credentials",
			secretName: "registry",
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: ns,
							Name:      "registry",
							Labels: map[string]string{
								integrationSecretTypeLabel: "registry",
							},
						},
						Type: corev1.SecretTypeDockerConfigJson,
						Data: map[string][]byte{
							".dockerconfigjson": []byte(`{"auths":{"` + server.URL + `/fail":{"username":"user1", "password":"password1"}}}`),
						},
					},
				).Build()
			},
			wantRes: reconcile.Result{
				RequeueAfter: failConnectionRequeueTime,
			},
			wantErr:           require.NoError,
			wantConAnnotation: "false",
		},
		{
			name:       "failed github registry",
			secretName: "registry",
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: ns,
							Name:      "registry",
							Labels: map[string]string{
								integrationSecretTypeLabel: "registry",
							},
						},
						Type: corev1.SecretTypeDockerConfigJson,
						Data: map[string][]byte{
							".dockerconfigjson": []byte(`{"auths":{"https://ghcr.io":{"username":"user1", "password":"password1"}}}`),
						},
					},
				).Build()
			},
			wantRes: reconcile.Result{
				RequeueAfter: failConnectionRequeueTime,
			},
			wantErr:           require.NoError,
			wantConAnnotation: "false",
		},
		{
			name:       "dockerhub with invalid credentials",
			secretName: "registry",
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: ns,
							Name:      "registry",
							Labels: map[string]string{
								integrationSecretTypeLabel: "registry",
							},
						},
						Type: corev1.SecretTypeDockerConfigJson,
						Data: map[string][]byte{
							".dockerconfigjson": []byte(`{"auths":{"https://index.docker.io/v1/":{"username":"user1", "password":"password1"}}}`),
						},
					},
				).Build()
			},
			wantRes: reconcile.Result{
				RequeueAfter: failConnectionRequeueTime,
			},
			wantErr:           require.NoError,
			wantConAnnotation: "false",
		},
		{
			name:       "registry without auth",
			secretName: "registry",
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: ns,
							Name:      "registry",
							Labels: map[string]string{
								integrationSecretTypeLabel: "registry",
							},
						},
						Type: corev1.SecretTypeDockerConfigJson,
						Data: map[string][]byte{
							".dockerconfigjson": []byte("{}"),
						},
					},
				).Build()
			},
			wantRes: reconcile.Result{
				RequeueAfter: failConnectionRequeueTime,
			},
			wantErr:              require.NoError,
			wantConAnnotation:    "false",
			wantConErrAnnotation: "no auths in .dockerconfigjson",
		},
		{
			name:       "registry with invalid config",
			secretName: "registry",
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: ns,
							Name:      "registry",
							Labels: map[string]string{
								integrationSecretTypeLabel: "registry",
							},
						},
						Type: corev1.SecretTypeDockerConfigJson,
						Data: map[string][]byte{
							".dockerconfigjson": []byte("not a json"),
						},
					},
				).Build()
			},
			wantRes: reconcile.Result{
				RequeueAfter: failConnectionRequeueTime,
			},
			wantErr:              require.NoError,
			wantConAnnotation:    "false",
			wantConErrAnnotation: "failed to unmarshal .dockerconfigjson",
		},
		{
			name:       "registry without .dockerconfigjson",
			secretName: "registry",
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: ns,
							Name:      "registry",
							Labels: map[string]string{
								integrationSecretTypeLabel: "registry",
							},
						},
						Type: corev1.SecretTypeDockerConfigJson,
						Data: map[string][]byte{},
					},
				).Build()
			},
			wantRes: reconcile.Result{
				RequeueAfter: failConnectionRequeueTime,
			},
			wantErr:              require.NoError,
			wantConAnnotation:    "false",
			wantConErrAnnotation: "no .dockerconfigjson key in secret",
		},
		{
			name:       "not reachable server",
			secretName: "integration-secret",
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: ns,
							Name:      "integration-secret",
						},
						Data: map[string][]byte{
							"url": []byte("http://not-reachable-server"),
						},
					},
				).Build()
			},
			wantRes: reconcile.Result{
				RequeueAfter: failConnectionRequeueTime,
			},
			wantErr:              require.NoError,
			wantConAnnotation:    "false",
			wantConErrAnnotation: "connection failed",
		},
		{
			name:       "secret not found",
			secretName: "not-exists",
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(s).Build()
			},
			wantRes:           reconcile.Result{},
			wantErr:           require.NoError,
			wantConAnnotation: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := tt.client(t)
			r := NewReconcileIntegrationSecret(cl)
			got, err := r.Reconcile(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: ns,
						Name:      tt.secretName,
					},
				},
			)

			tt.wantErr(t, err)
			require.Equal(t, tt.wantRes, got)

			if tt.wantConAnnotation == "" {
				return
			}

			s := &corev1.Secret{}
			require.NoError(t, cl.Get(context.Background(), client.ObjectKey{
				Namespace: ns,
				Name:      tt.secretName,
			}, s))

			require.Equal(t, tt.wantConAnnotation, s.GetAnnotations()[integrationSecretConnectionAnnotation])
			require.Contains(t, s.GetAnnotations()[integrationSecretErrorAnnotation], tt.wantConErrAnnotation)
		})
	}
}

func Test_hasIntegrationSecretLabelLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		Object client.Object
		want   bool
	}{
		{
			name: "has label",
			Object: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						integrationSecretLabel: "true",
					},
				},
			},
			want: true,
		},
		{
			name: "label is false",
			Object: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						integrationSecretLabel: "false",
					},
				},
			},
			want: false,
		},
		{
			name: "has not label",
			Object: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasIntegrationSecretLabelLabel(tt.Object)
			assert.Equal(t, tt.want, got)
		})
	}
}
