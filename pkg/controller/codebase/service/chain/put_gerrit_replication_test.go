package chain

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestPutGerritReplication_ShouldFailWhenReloadGerritPlugin(t *testing.T) {
	//TODO: mock sshclient and implement test that passes
	t.Skip()

	ctx := context.Background()
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Type:             util.Application,
			DeploymentScript: util.HelmChartDeploymentScriptType,
			Strategy:         codebaseApi.Create,
			Lang:             util.LanguageGo,
			DefaultBranch:    "fake-defaultBranch",
			GitUrlPath:       util.GetStringP(fakeName),
			Repository: &codebaseApi.Repository{
				Url: "https://example.com/repo",
			},
			GitServer: fakeName,
		},
		Status: codebaseApi.CodebaseStatus{
			Git: *util.GetStringP("pushed"),
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "vcs-autouser-codebase-fake-name-temp",
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}

	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "gerrit",
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.GitServerSpec{
			NameSshKeySecret: fakeName,
			GitHost:          fakeName,
			SshPort:          22,
			GitUser:          fakeName,
		},
	}
	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "edp-config",
			Namespace: fakeNamespace,
		},
		Data: map[string]string{
			"vcs_integration_enabled":  "true",
			"perf_integration_enabled": "true",
			"dns_wildcard":             "dns",
			"edp_name":                 "edp-name",
			"edp_version":              "2.2.2",
			"vcs_group_name_url":       "https://gitlab.example.com/backup",
			"vcs_ssh_port":             "22",
			"vcs_tool_name":            "gitlab",
		},
	}
	cmg := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "gerrit",
			Namespace: fakeNamespace,
		},
		Data: map[string]string{
			"replication.config": "stub-config",
		},
	}

	pk, err := rsa.GenerateKey(rand.Reader, 128)
	require.NoError(t, err, "unable to generate test private key")

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(pk)
	privateKeyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privateKeyBytes,
		},
	)

	ssh := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "gerrit-project-creator",
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			util.PrivateSShKeyName: privateKeyPem,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, ssh, cm, s, cmg)
	scheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, c, gs)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs, ssh, cm, s, cmg).Build()

	err = os.Setenv("ASSETS_DIR", "../../../../../build")
	require.NoError(t, err)

	httpmock.Reset()
	httpmock.Activate()
	httpmock.RegisterResponder("GET", "https://gitlab.example.com/api/v4/projects/backup%252Ffake-name?private_token=pass&simple=true",
		httpmock.NewStringResponder(200, ""))

	jr := map[string]string{
		"access_token":    "access",
		"ssh_url_to_repo": "ssh://url",
	}
	httpmock.RegisterResponder("POST", "https://gitlab.example.com/oauth/token",
		httpmock.NewJsonResponderOrPanic(200, &jr))

	httpmock.RegisterResponder("GET", "https://gitlab.example.com/api/v4/projects/backup%252Ffake-name?simple=true",
		httpmock.NewJsonResponderOrPanic(200, &jr))

	pdc := NewPutGerritReplication(fakeCl)

	err = pdc.ServeRequest(ctx, c)

	assert.NoError(t, err)
}
