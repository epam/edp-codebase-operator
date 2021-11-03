package chain

import (
	"os"
	"strings"
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPutGerritReplication_ShouldFailWhenReloadGerritPlugin(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			Type:             util.Application,
			DeploymentScript: util.HelmChartDeploymentScriptType,
			Strategy:         v1alpha1.Create,
			Lang:             util.LanguageGo,
			DefaultBranch:    "fake-defaultBranch",
			GitUrlPath:       util.GetStringP(fakeName),
			Repository: &v1alpha1.Repository{
				Url: "https://example.com/repo",
			},
			GitServer: fakeName,
		},
		Status: v1alpha1.CodebaseStatus{
			Git: *util.GetStringP("pushed"),
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vcs-autouser-codebase-fake-name-temp",
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}

	gs := &v1alpha1.GitServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gerrit",
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.GitServerSpec{
			NameSshKeySecret: fakeName,
			GitHost:          fakeName,
			SshPort:          22,
			GitUser:          fakeName,
		},
	}
	cm := &coreV1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
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
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gerrit",
			Namespace: fakeNamespace,
		},
		Data: map[string]string{
			"replication.config": "stub-config",
		},
	}
	ssh := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gerrit-project-creator",
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			util.PrivateSShKeyName: []byte("fake"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, ssh, cm, s, cmg)
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, gs)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs, ssh, cm, s, cmg).Build()

	os.Setenv("ASSETS_DIR", "../../../../../build")

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

	pdc := PutGerritReplication{
		client: fakeCl,
	}

	err := pdc.ServeRequest(c)
	//TODO: mock sshclient and implement test that passes
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "failed to dial: dial tcp: lookup gerrit.fake_namespace") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
