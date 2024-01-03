package chain

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	routeApi "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/platform"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestPutWebHook_ServeRequest(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	defer httpmock.DeactivateAndReset()

	schema := runtime.NewScheme()

	require.NoError(t, codebaseApi.AddToScheme(schema))
	require.NoError(t, coreV1.AddToScheme(schema))
	require.NoError(t, networkingV1.AddToScheme(schema))
	require.NoError(t, routeApi.AddToScheme(schema))

	const namespace = "test-ns"

	gitURL := "test-owner/test-repo"
	fakeUrlRegexp := regexp.MustCompile(`.*`)

	tests := []struct {
		name        string
		codebase    *codebaseApi.Codebase
		k8sObjects  []client.Object
		prepare     func(t *testing.T)
		responder   func(t *testing.T)
		wantErr     require.ErrorAssertionFunc
		errContains string
	}{
		{
			name: "success gitlab",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace:       namespace,
					Name:            "test-codebase",
					ResourceVersion: "1",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:  "test-git-server",
					GitUrlPath: gitURL,
					CiTool:     util.CITekton,
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.K8S)
			},
			k8sObjects: []client.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace:       namespace,
						Name:            "test-codebase",
						ResourceVersion: "1",
					},
				},
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-git-server",
					},
					Spec: codebaseApi.GitServerSpec{
						GitHost:          "fake.gitlab.com",
						GitUser:          "git",
						HttpsPort:        443,
						NameSshKeySecret: "test-secret",
						GitProvider:      codebaseApi.GitProviderGitlab,
					},
				},
				&coreV1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-secret",
					},
					Data: map[string][]byte{
						util.GitServerSecretTokenField:         []byte("test-token"),
						util.GitServerSecretWebhookSecretField: []byte("test-webhook-secret"),
					},
				},
				fakeIngress(namespace, ingressName, "fake.gitlab.com"),
			},
			responder: func(t *testing.T) {
				POSTResponder := httpmock.NewStringResponder(http.StatusOK, "")
				httpmock.RegisterRegexpResponder(http.MethodPost, fakeUrlRegexp, POSTResponder)

				GETResponder := httpmock.NewStringResponder(http.StatusOK, "")
				httpmock.RegisterRegexpResponder(http.MethodGet, fakeUrlRegexp, GETResponder)
			},
			wantErr: require.NoError,
		},
		{
			name: "success gitlab with route",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace:       namespace,
					Name:            "test-codebase",
					ResourceVersion: "1",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:  "test-git-server",
					GitUrlPath: gitURL,
					CiTool:     util.CITekton,
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.Openshift)
			},
			k8sObjects: []client.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace:       namespace,
						Name:            "test-codebase",
						ResourceVersion: "1",
					},
				},
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-git-server",
					},
					Spec: codebaseApi.GitServerSpec{
						GitHost:          "fake.gitlab.com",
						GitUser:          "git",
						HttpsPort:        443,
						NameSshKeySecret: "test-secret",
						GitProvider:      codebaseApi.GitProviderGitlab,
					},
				},
				&coreV1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-secret",
					},
					Data: map[string][]byte{
						util.GitServerSecretTokenField:         []byte("test-token"),
						util.GitServerSecretWebhookSecretField: []byte("test-webhook-secret"),
					},
				},
				&routeApi.Route{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      ingressName,
					},
					Status: routeApi.RouteStatus{
						Ingress: []routeApi.RouteIngress{
							{
								Host: "fake.gitlab.com",
							},
						},
					},
				},
			},
			responder: func(t *testing.T) {
				POSTResponder := httpmock.NewStringResponder(http.StatusOK, "")
				httpmock.RegisterRegexpResponder(http.MethodPost, fakeUrlRegexp, POSTResponder)

				GETResponder := httpmock.NewStringResponder(http.StatusOK, "")
				httpmock.RegisterRegexpResponder(http.MethodGet, fakeUrlRegexp, GETResponder)
			},
			wantErr: require.NoError,
		},
		{
			name: "success github",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace:       namespace,
					Name:            "test-codebase",
					ResourceVersion: "1",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:  "test-git-server",
					GitUrlPath: gitURL,
					CiTool:     util.CITekton,
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.K8S)
			},
			k8sObjects: []client.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace:       namespace,
						Name:            "test-codebase",
						ResourceVersion: "1",
					},
				},
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-git-server",
					},
					Spec: codebaseApi.GitServerSpec{
						GitHost:          "fake.github.com",
						GitUser:          "git",
						HttpsPort:        443,
						NameSshKeySecret: "test-secret",
						GitProvider:      codebaseApi.GitProviderGithub,
					},
				},
				&coreV1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-secret",
					},
					Data: map[string][]byte{
						util.GitServerSecretTokenField:         []byte("test-token"),
						util.GitServerSecretWebhookSecretField: []byte("test-webhook-secret"),
					},
				},
				fakeIngress(namespace, ingressName, "fake.github.com"),
			},
			responder: func(t *testing.T) {
				POSTResponder := httpmock.NewStringResponder(http.StatusOK, "")
				httpmock.RegisterRegexpResponder(http.MethodPost, fakeUrlRegexp, POSTResponder)

				GETResponder := httpmock.NewStringResponder(http.StatusOK, "")
				httpmock.RegisterRegexpResponder(http.MethodGet, fakeUrlRegexp, GETResponder)
			},
			wantErr: require.NoError,
		},
		{
			name: "success use existing webhook",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace:       namespace,
					Name:            "test-codebase",
					ResourceVersion: "1",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:  "test-git-server",
					GitUrlPath: gitURL,
					CiTool:     util.CITekton,
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.K8S)
			},
			k8sObjects: []client.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace:       namespace,
						Name:            "test-codebase",
						ResourceVersion: "1",
					},
				},
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-git-server",
					},
					Spec: codebaseApi.GitServerSpec{
						GitHost:          "fake.gitlab.com",
						GitUser:          "git",
						HttpsPort:        443,
						NameSshKeySecret: "test-secret",
						GitProvider:      codebaseApi.GitProviderGitlab,
					},
				},
				&coreV1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-secret",
					},
					Data: map[string][]byte{
						util.GitServerSecretTokenField:         []byte("test-token"),
						util.GitServerSecretWebhookSecretField: []byte("test-webhook-secret"),
					},
				},
				fakeIngress(namespace, ingressName, "fake.gitlab.com"),
			},
			responder: func(t *testing.T) {
				getHookResponder, err := httpmock.NewJsonResponder(http.StatusNotFound, map[string]string{"message": "404 Not Found"})
				require.NoError(t, err)
				httpmock.RegisterRegexpResponder(http.MethodGet, regexp.MustCompile(`.*999$`), getHookResponder)

				getHooksResponder, err := httpmock.NewJsonResponder(http.StatusOK, []map[string]interface{}{{"id": 1, "url": "https://fake.gitlab.com"}})
				require.NoError(t, err)
				httpmock.RegisterRegexpResponder(http.MethodGet, regexp.MustCompile(`.*hooks$`), getHooksResponder)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - no webhook secret token",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace:       namespace,
					Name:            "test-codebase",
					ResourceVersion: "1",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:  "test-git-server",
					GitUrlPath: gitURL,
					CiTool:     util.CITekton,
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.K8S)
			},
			k8sObjects: []client.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace:       namespace,
						Name:            "test-codebase",
						ResourceVersion: "1",
					},
				},
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-git-server",
					},
					Spec: codebaseApi.GitServerSpec{
						GitHost:          "fake.gitlab.com",
						GitUser:          "git",
						HttpsPort:        443,
						NameSshKeySecret: "test-secret",
						GitProvider:      codebaseApi.GitProviderGitlab,
					},
				},
				&coreV1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-secret",
					},
					Data: map[string][]byte{
						util.GitServerSecretTokenField: []byte("test-token"),
					},
				},
				fakeIngress(namespace, ingressName, "fake.gitlab.com"),
			},
			responder: func(t *testing.T) {
				POSTResponder := httpmock.NewStringResponder(http.StatusOK, "")
				httpmock.RegisterRegexpResponder(http.MethodPost, fakeUrlRegexp, POSTResponder)

				GETResponder := httpmock.NewStringResponder(http.StatusOK, "")
				httpmock.RegisterRegexpResponder(http.MethodGet, fakeUrlRegexp, GETResponder)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - webhook already exists",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace:       namespace,
					Name:            "test-codebase",
					ResourceVersion: "1",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:  "test-git-server",
					GitUrlPath: gitURL,
					CiTool:     util.CITekton,
				},
				Status: codebaseApi.CodebaseStatus{
					WebHookID: 999,
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.K8S)
			},
			k8sObjects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-git-server",
					},
					Spec: codebaseApi.GitServerSpec{
						GitHost:          "fake.gitlab.com",
						GitUser:          "git",
						HttpsPort:        443,
						NameSshKeySecret: "test-secret",
						GitProvider:      codebaseApi.GitProviderGitlab,
					},
				},
				&coreV1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-secret",
					},
					Data: map[string][]byte{
						util.GitServerSecretTokenField: []byte("test-token"),
					},
				},
			},
			responder: func(t *testing.T) {
				responder, err := httpmock.NewJsonResponder(http.StatusOK, map[string]int{"id": 999})
				require.NoError(t, err)
				httpmock.RegisterRegexpResponder(http.MethodGet, fakeUrlRegexp, responder)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - webhookID exists but webhook does not",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace:       namespace,
					Name:            "test-codebase",
					ResourceVersion: "1",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:  "test-git-server",
					GitUrlPath: gitURL,
					CiTool:     util.CITekton,
				},
				Status: codebaseApi.CodebaseStatus{
					WebHookID: 999,
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.K8S)
			},
			k8sObjects: []client.Object{
				&codebaseApi.Codebase{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace:       namespace,
						Name:            "test-codebase",
						ResourceVersion: "1",
					},
				},
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-git-server",
					},
					Spec: codebaseApi.GitServerSpec{
						GitHost:          "fake.gitlab.com",
						GitUser:          "git",
						HttpsPort:        443,
						NameSshKeySecret: "test-secret",
						GitProvider:      codebaseApi.GitProviderGitlab,
					},
				},
				&coreV1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-secret",
					},
					Data: map[string][]byte{
						util.GitServerSecretTokenField:         []byte("test-token"),
						util.GitServerSecretWebhookSecretField: []byte("test-webhook-secret"),
					},
				},
				fakeIngress(namespace, ingressName, "fake.gitlab.com"),
			},
			responder: func(t *testing.T) {
				getHookResponder, err := httpmock.NewJsonResponder(http.StatusNotFound, map[string]string{"message": "404 Not Found"})
				require.NoError(t, err)
				httpmock.RegisterRegexpResponder(http.MethodGet, regexp.MustCompile(`.*999$`), getHookResponder)

				getHooksResponder, err := httpmock.NewJsonResponder(http.StatusOK, []map[string]interface{}{})
				require.NoError(t, err)
				httpmock.RegisterRegexpResponder(http.MethodGet, regexp.MustCompile(`.*hooks$`), getHooksResponder)

				postResponder := httpmock.NewStringResponder(http.StatusOK, "")
				httpmock.RegisterRegexpResponder(http.MethodPost, fakeUrlRegexp, postResponder)
			},
			wantErr: require.NoError,
		},
		{
			name: "skip creating webhook - unsupported git provider",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-codebase",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:  "test-git-server",
					GitUrlPath: gitURL,
					CiTool:     util.CITekton,
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.K8S)
			},
			k8sObjects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-git-server",
					},
					Spec: codebaseApi.GitServerSpec{
						GitHost:          "fake.github.com",
						GitUser:          "git",
						HttpsPort:        443,
						NameSshKeySecret: "test-secret",
						GitProvider:      codebaseApi.GitProviderGerrit,
					},
				},
			},
			responder: func(t *testing.T) {},
			wantErr:   require.NoError,
		},
		{
			name: "failed to get webhook",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace:       namespace,
					Name:            "test-codebase",
					ResourceVersion: "1",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:  "test-git-server",
					GitUrlPath: gitURL,
					CiTool:     util.CITekton,
				},
				Status: codebaseApi.CodebaseStatus{
					WebHookID: 999,
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.K8S)
			},
			k8sObjects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-git-server",
					},
					Spec: codebaseApi.GitServerSpec{
						GitHost:          "fake.gitlab.com",
						GitUser:          "git",
						HttpsPort:        443,
						NameSshKeySecret: "test-secret",
						GitProvider:      codebaseApi.GitProviderGitlab,
					},
				},
				&coreV1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-secret",
					},
					Data: map[string][]byte{
						util.GitServerSecretTokenField:         []byte("test-token"),
						util.GitServerSecretWebhookSecretField: []byte("test-webhook-secret"),
					},
				},
			},
			responder: func(t *testing.T) {
				getResponder, err := httpmock.NewJsonResponder(http.StatusBadRequest, map[string]string{"message": "400 Bad Request"})
				require.NoError(t, err)
				httpmock.RegisterRegexpResponder(http.MethodGet, fakeUrlRegexp, getResponder)
			},
			wantErr:     require.Error,
			errContains: "failed to get webhook",
		},
		{
			name: "failed to create webhook",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-codebase",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:  "test-git-server",
					GitUrlPath: gitURL,
					CiTool:     util.CITekton,
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.K8S)
			},
			k8sObjects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-git-server",
					},
					Spec: codebaseApi.GitServerSpec{
						GitHost:          "fake.gitlab.com",
						GitUser:          "git",
						HttpsPort:        443,
						NameSshKeySecret: "test-secret",
						GitProvider:      codebaseApi.GitProviderGitlab,
					},
				},
				&coreV1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-secret",
					},
					Data: map[string][]byte{
						util.GitServerSecretTokenField: []byte("test-token"),
					},
				},
				fakeIngress(namespace, ingressName, "fake.gitlab.com"),
			},
			responder: func(t *testing.T) {
				responder := httpmock.NewStringResponder(http.StatusInternalServerError, "")
				httpmock.RegisterRegexpResponder(http.MethodPost, fakeUrlRegexp, responder)
			},
			wantErr:     require.Error,
			errContains: "failed to create",
		},
		{
			name: "failed to get getWebHookUrl - no rules",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-codebase",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:  "test-git-server",
					GitUrlPath: gitURL,
					CiTool:     util.CITekton,
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.K8S)
			},
			k8sObjects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-git-server",
					},
					Spec: codebaseApi.GitServerSpec{
						GitHost:          "fake.gitlab.com",
						GitUser:          "git",
						HttpsPort:        443,
						NameSshKeySecret: "test-secret",
						GitProvider:      codebaseApi.GitProviderGitlab,
					},
				},
				&coreV1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-secret",
					},
					Data: map[string][]byte{
						util.GitServerSecretTokenField: []byte("test-token"),
					},
				},
				&networkingV1.Ingress{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      ingressName,
					},
				},
			},
			responder:   func(t *testing.T) {},
			wantErr:     require.Error,
			errContains: "doesn't have rules",
		},
		{
			name: "failed to get getWebHookUrl - no ingress",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-codebase",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:  "test-git-server",
					GitUrlPath: gitURL,
					CiTool:     util.CITekton,
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.K8S)
			},
			k8sObjects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-git-server",
					},
					Spec: codebaseApi.GitServerSpec{
						GitHost:          "fake.gitlab.com",
						GitUser:          "git",
						HttpsPort:        443,
						NameSshKeySecret: "test-secret",
						GitProvider:      codebaseApi.GitProviderGitlab,
					},
				},
				&coreV1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-secret",
					},
					Data: map[string][]byte{
						util.GitServerSecretTokenField: []byte("test-token"),
					},
				},
			},
			responder:   func(t *testing.T) {},
			wantErr:     require.Error,
			errContains: fmt.Sprintf("failed to get %s ingress", ingressName),
		},
		{
			name: "failed to get secret - no required field",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-codebase",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:  "test-git-server",
					GitUrlPath: gitURL,
					CiTool:     util.CITekton,
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.K8S)
			},
			k8sObjects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-git-server",
					},
					Spec: codebaseApi.GitServerSpec{
						GitHost:          "fake.gitlab.com",
						GitUser:          "git",
						HttpsPort:        443,
						NameSshKeySecret: "test-secret",
						GitProvider:      codebaseApi.GitProviderGitlab,
					},
				},
				&coreV1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-secret",
					},
				},
			},
			responder:   func(t *testing.T) {},
			wantErr:     require.Error,
			errContains: fmt.Sprintf("failed to get %s field", util.GitServerSecretTokenField),
		},
		{
			name: "failed to get secret",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-codebase",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:  "test-git-server",
					GitUrlPath: gitURL,
					CiTool:     util.CITekton,
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.K8S)
			},
			k8sObjects: []client.Object{
				&codebaseApi.GitServer{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: namespace,
						Name:      "test-git-server",
					},
					Spec: codebaseApi.GitServerSpec{
						GitHost:          "fake.gitlab.com",
						GitUser:          "git",
						HttpsPort:        443,
						NameSshKeySecret: "test-secret",
						GitProvider:      codebaseApi.GitProviderGitlab,
					},
				},
			},
			responder:   func(t *testing.T) {},
			wantErr:     require.Error,
			errContains: "failed to get test-secret",
		},
		{
			name: "git server not found",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-codebase",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer: "test-git-server",
					CiTool:    util.CITekton,
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.K8S)
			},
			k8sObjects:  []client.Object{},
			responder:   func(t *testing.T) {},
			wantErr:     require.Error,
			errContains: "failed to get git server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Reset()
			tt.responder(t)
			tt.prepare(t)

			k8sClient := fake.NewClientBuilder().WithScheme(schema).WithObjects(tt.k8sObjects...).Build()
			s := NewPutWebHook(k8sClient, restyClient)

			gotErr := s.ServeRequest(context.Background(), tt.codebase)
			tt.wantErr(t, gotErr)
			if tt.errContains != "" {
				assert.Contains(t, gotErr.Error(), tt.errContains)
			}
		})
	}
}

func fakeIngress(namespace, name, host string) *networkingV1.Ingress {
	return &networkingV1.Ingress{
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: networkingV1.IngressSpec{
			Rules: []networkingV1.IngressRule{
				{
					Host: host,
				},
			},
		},
	}
}
