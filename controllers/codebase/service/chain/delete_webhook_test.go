package chain

import (
	"context"
	"net/http"
	"regexp"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/platform"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestDeleteWebHook_ServeRequest(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	defer httpmock.DeactivateAndReset()

	schema := runtime.NewScheme()
	err := codebaseApi.AddToScheme(schema)
	require.NoError(t, err)
	err = coreV1.AddToScheme(schema)
	require.NoError(t, err)

	const namespace = "test-ns"

	gitURL := "test-owner/test-repo"
	fakeUrlRegexp := regexp.MustCompile(`.*`)

	tests := []struct {
		name       string
		codebase   *codebaseApi.Codebase
		k8sObjects []client.Object
		responder  func(t *testing.T)
		hasError   bool
	}{
		{
			name: "success gitlab",
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
				Status: codebaseApi.CodebaseStatus{
					WebHookID: 1,
				},
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
				responder := httpmock.NewStringResponder(http.StatusOK, "")
				httpmock.RegisterRegexpResponder(http.MethodDelete, fakeUrlRegexp, responder)
			},
		},
		{
			name: "success github",
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
				Status: codebaseApi.CodebaseStatus{
					WebHookID: 1,
				},
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
						GitProvider:      codebaseApi.GitProviderGithub,
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
				responder := httpmock.NewStringResponder(http.StatusOK, "")
				httpmock.RegisterRegexpResponder(http.MethodDelete, fakeUrlRegexp, responder)
			},
		},
		{
			name: "success with empty webhook id",
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
			k8sObjects: []client.Object{},
			responder:  func(t *testing.T) {},
		},
		{
			name: "fail to delete webhook",
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
				Status: codebaseApi.CodebaseStatus{
					WebHookID: 1,
				},
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
				responder := httpmock.NewStringResponder(http.StatusBadRequest, "")
				httpmock.RegisterRegexpResponder(http.MethodDelete, fakeUrlRegexp, responder)
			},
			hasError: true,
		},
		{
			name: "fail to get git provider",
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
				Status: codebaseApi.CodebaseStatus{
					WebHookID: 1,
				},
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
						GitProvider:      codebaseApi.GitProviderGerrit,
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
			responder: func(t *testing.T) {},
			hasError:  true,
		},
		{
			name: "project not found",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-codebase",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer: "test-git-server",
					CiTool:    util.CITekton,
				},
				Status: codebaseApi.CodebaseStatus{
					WebHookID: 1,
				},
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
			responder: func(t *testing.T) {},
			hasError:  true,
		},
		{
			name: "fail to get secret token field",
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
				Status: codebaseApi.CodebaseStatus{
					WebHookID: 1,
				},
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
					Data: map[string][]byte{},
				},
			},
			responder: func(t *testing.T) {},
			hasError:  true,
		},
		{
			name: "fail to get secret",
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
				Status: codebaseApi.CodebaseStatus{
					WebHookID: 1,
				},
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
			responder: func(t *testing.T) {},
			hasError:  true,
		},
		{
			name: "fail to get git server",
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
				Status: codebaseApi.CodebaseStatus{
					WebHookID: 1,
				},
			},
			k8sObjects: []client.Object{},
			responder:  func(t *testing.T) {},
			hasError:   true,
		},
		{
			name: "skip if ci tool is not tekton",
			codebase: &codebaseApi.Codebase{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-codebase",
				},
				Spec: codebaseApi.CodebaseSpec{
					GitServer:  "test-git-server",
					GitUrlPath: gitURL,
					CiTool:     util.CIJenkins,
				},
				Status: codebaseApi.CodebaseStatus{
					WebHookID: 1,
				},
			},
			responder: func(t *testing.T) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Reset()
			tt.responder(t)

			k8sClient := fake.NewClientBuilder().WithScheme(schema).WithObjects(tt.k8sObjects...).Build()

			logger := platform.NewLoggerMock()
			loggerSink, ok := logger.GetSink().(*platform.LoggerMock)
			require.True(t, ok)

			s := NewDeleteWebHook(k8sClient, restyClient, logger)

			assert.NoError(t, s.ServeRequest(ctrl.LoggerInto(context.Background(), logger), tt.codebase))
			assert.Equalf(t, tt.hasError, loggerSink.LastError() != nil, "expected error: %v, got: %v", tt.hasError, loggerSink.LastError())
		})
	}
}
