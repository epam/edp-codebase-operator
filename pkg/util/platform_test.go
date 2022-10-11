package util

import (
	"os"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	k8sMockClient "github.com/epam/edp-common/pkg/mock/controller-runtime/client"
	edpComponentApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1"
)

func TestGetGerritPort_ShouldFound(t *testing.T) {
	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "gerrit",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.GitServerSpec{
			SshPort: 22,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gs).Build()

	port, err := GetGerritPort(fakeCl, "stub-namespace")
	assert.Equal(t, *port, int32(22))
	assert.NoError(t, err)
}

func TestGetGerritPort_ShouldFailPortNotDefined(t *testing.T) {
	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "gerrit",
			Namespace: "stub-namespace",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gs).Build()

	port, err := GetGerritPort(fakeCl, "stub-namespace")
	assert.Nil(t, port)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "ssh port is zero or not defined in gerrit GitServer CR") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGetGerritPort_ShouldNotFound(t *testing.T) {
	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-gerrit",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.GitServerSpec{
			SshPort: 22,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gs).Build()

	port, err := GetGerritPort(fakeCl, "stub-namespace")
	assert.Nil(t, port)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "an error has occurred while getting gerrit Git Server CR") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGetGerritPort_ShouldFailToGetPort(t *testing.T) {
	mc := k8sMockClient.Client{}

	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-gerrit",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.GitServerSpec{
			SshPort: 22,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, gs)

	mockErr := errors.New("FATAL")

	mc.On("Get", types.NamespacedName{
		Namespace: "stub-namespace",
		Name:      "gerrit",
	}, &codebaseApi.GitServer{}).Return(mockErr)

	port, err := GetGerritPort(&mc, "stub-namespace")
	assert.Nil(t, port)
	assert.Error(t, err)

	if errors.Cause(err) != mockErr {
		t.Fatal("wrong error returned")
	}
}

func TestGetEdpComponent_ShouldPass(t *testing.T) {
	c := &edpComponentApi.EDPComponent{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: edpComponentApi.EDPComponentSpec{},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	edc, err := GetEdpComponent(fakeCl, "stub-name", "stub-namespace")
	assert.Equal(t, edc.Name, "stub-name")
	assert.NoError(t, err)
}

func TestGetEdpComponent_ShouldFail(t *testing.T) {
	mc := k8sMockClient.Client{}

	c := &edpComponentApi.EDPComponent{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: edpComponentApi.EDPComponentSpec{},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c)

	mockErr := errors.New("FATAL")

	mc.On("Get", types.NamespacedName{
		Namespace: "stub-namespace",
		Name:      "stub-name",
	}, &edpComponentApi.EDPComponent{}).Return(mockErr)

	edc, err := GetEdpComponent(&mc, "stub-name", "stub-namespace")
	assert.Error(t, err)
	assert.Nil(t, edc)

	if errors.Cause(err) != mockErr {
		t.Fatal("wrong error returned")
	}
}

func TestGetCodebase_ShouldPass(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseSpec{},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	cb, err := GetCodebase(fakeCl, "stub-name", "stub-namespace")
	assert.Equal(t, cb.Name, "stub-name")
	assert.NoError(t, err)
}

func TestGetCodebase_ShouldFail(t *testing.T) {
	mc := k8sMockClient.Client{}

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseSpec{},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c)

	mockErr := errors.New("FATAL")

	mc.On("Get", types.NamespacedName{
		Namespace: "stub-namespace",
		Name:      "stub-name",
	}, &codebaseApi.Codebase{}).Return(mockErr)

	cb, err := GetCodebase(&mc, "stub-name", "stub-namespace")
	assert.Error(t, err)
	assert.Nil(t, cb)

	if errors.Cause(err) != mockErr {
		t.Fatal("wrong error returned")
	}
}

func TestGetSecret_ShouldPass(t *testing.T) {
	secret := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, secret)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(secret).Build()

	gs, err := GetSecret(fakeCl, "stub-name", "stub-namespace")

	assert.Equal(t, gs.Name, "stub-name")
	assert.NoError(t, err)
}

func TestGetSecret_ShouldFail(t *testing.T) {
	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s).Build()

	gs, err := GetSecret(fakeCl, "non-existing-stub-name", "stub-namespace")

	assert.Nil(t, gs)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "secrets \"non-existing-stub-name\" not found") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGetVcsBasicAuthConfig_ShouldPass(t *testing.T) {
	secret := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, secret)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(secret).Build()

	u, p, err := GetVcsBasicAuthConfig(fakeCl, "stub-namespace", "stub-name")
	assert.Equal(t, u, "user")
	assert.Equal(t, p, "pass")
	assert.NoError(t, err)
}

func TestGetVcsBasicAuthConfig_ShouldFail(t *testing.T) {
	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, s)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s).Build()

	u, p, err := GetVcsBasicAuthConfig(fakeCl, "stub-namespace", "non-existing-stub-name")
	assert.Equal(t, u, "")
	assert.Equal(t, p, "")
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "secrets \"non-existing-stub-name\" not found") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGetVcsBasicAuthConfig_ShouldFailIfUsernameOrPasswordIsNotDefined(t *testing.T) {
	secret := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-name",
			Namespace: "stub-namespace",
		},
		Data: map[string][]byte{
			"username": []byte("user"),
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, secret)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(secret).Build()

	u, p, err := GetVcsBasicAuthConfig(fakeCl, "stub-namespace", "stub-name")
	assert.Equal(t, u, "")
	assert.Equal(t, p, "")
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "username/password keys are not defined in Secret stub-name") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGetGitServer_ShouldPass(t *testing.T) {
	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "gerrit",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.GitServerSpec{
			SshPort: 22,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gs).Build()

	ggs, err := GetGitServer(fakeCl, "gerrit", "stub-namespace")
	assert.Equal(t, ggs.Name, "gerrit")
	assert.NoError(t, err)
}

func TestGetGitServer_ShouldFailIfNotFound(t *testing.T) {
	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "gerrit",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.GitServerSpec{
			SshPort: 22,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gs).Build()

	ggs, err := GetGitServer(fakeCl, "non-existing", "stub-namespace")
	assert.Nil(t, ggs)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "GitServer non-existing doesn't exist in k8s") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGetUserSettings_ShouldPass(t *testing.T) {
	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "edp-config",
			Namespace: "stub-namespace",
		},
		Data: map[string]string{
			"vcs_integration_enabled":  "true",
			"perf_integration_enabled": "true",
			"dns_wildcard":             "dns",
			"edp_name":                 "edp-name",
			"edp_version":              "2.2.2",
			"vcs_group_name_url":       "edp",
			"vcs_ssh_port":             "22",
			"vcs_tool_name":            "stub",
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cm).Build()

	model, err := GetUserSettings(fakeCl, "stub-namespace")
	assert.Equal(t, model.EdpName, "edp-name")
	assert.Equal(t, model.EdpVersion, "2.2.2")
	assert.True(t, model.PerfIntegrationEnabled)
	assert.True(t, model.VcsIntegrationEnabled)
	assert.NoError(t, err)
}

func TestGetUserSettings_ShouldFailToConvertBoolVcsintegration(t *testing.T) {
	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "edp-config",
			Namespace: "stub-namespace",
		},
		Data: map[string]string{
			"vcs_integration_enabled": "5",
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cm).Build()

	model, err := GetUserSettings(fakeCl, "stub-namespace")
	assert.Error(t, err)
	assert.Nil(t, model)

	if !strings.Contains(err.Error(), "strconv.ParseBool: parsing \"5\": invalid syntax") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGetUserSettings_ShouldFailToConvertBoolPerfintegration(t *testing.T) {
	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "edp-config",
			Namespace: "stub-namespace",
		},
		Data: map[string]string{
			"vcs_integration_enabled": "true",
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cm).Build()

	model, err := GetUserSettings(fakeCl, "stub-namespace")
	assert.Error(t, err)
	assert.Nil(t, model)

	if !strings.Contains(err.Error(), "strconv.ParseBool: parsing \"\": invalid syntax") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGetUserSettings_ShouldFailOnFindConfigmap(t *testing.T) {
	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "edp-config",
			Namespace: "stub-namespace",
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cm).Build()

	model, err := GetUserSettings(fakeCl, "another-namespace")
	assert.Error(t, err)
	assert.Nil(t, model)

	if !strings.Contains(err.Error(), "configmaps \"edp-config\" not found") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGetWatchNamespace_IsDefined(t *testing.T) {
	err := os.Setenv("WATCH_NAMESPACE", "namespace")
	require.NoError(t, err)

	ns, err := GetWatchNamespace()
	assert.Equal(t, ns, "namespace")
	assert.NoError(t, err)
}

func TestGetWatchNamespace_NotDefined(t *testing.T) {
	err := os.Unsetenv("WATCH_NAMESPACE")
	require.NoError(t, err)

	ns, err := GetWatchNamespace()
	assert.Equal(t, ns, "")
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "WATCH_NAMESPACE must be set") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGetDebugMode_IsDefined(t *testing.T) {
	err := os.Setenv("DEBUG_MODE", "true")
	require.NoError(t, err)

	d, err := GetDebugMode()
	assert.True(t, d)
	assert.NoError(t, err)
}

func TestGetDebugMode_NotDefined(t *testing.T) {
	err := os.Unsetenv("DEBUG_MODE")
	require.NoError(t, err)

	d, err := GetDebugMode()
	assert.False(t, d)
	assert.Nil(t, err)
}

func TestGetDebugMode_ShouldFailOnConvertToBool(t *testing.T) {
	err := os.Setenv("DEBUG_MODE", "6")
	require.NoError(t, err)

	d, err := GetDebugMode()
	assert.False(t, d)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "strconv.ParseBool: parsing \"6\": invalid syntax") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
