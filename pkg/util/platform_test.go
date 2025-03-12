package util

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
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

	assert.Contains(t, err.Error(), "failed to get gerrit Git Server CR")
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

	assert.Contains(t, err.Error(), "failed to find GitServer non-existing in k8s")
}

func TestGetUserSettings_ShouldPass(t *testing.T) {
	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "krci-config",
			Namespace: "stub-namespace",
		},
		Data: map[string]string{
			"dns_wildcard": "dns",
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cm).Build()

	model, err := GetUserSettings(fakeCl, "stub-namespace")
	assert.Equal(t, model.DnsWildcard, "dns")
	assert.NoError(t, err)
}

func TestGetUserSettings_ShouldFailOnFindConfigmap(t *testing.T) {
	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "krci-config",
			Namespace: "stub-namespace",
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, cm)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cm).Build()

	model, err := GetUserSettings(fakeCl, "another-namespace")
	assert.Error(t, err)
	assert.Nil(t, model)

	if !strings.Contains(err.Error(), "configmaps \"krci-config\" not found") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGetWatchNamespace_IsDefined(t *testing.T) {
	t.Setenv("WATCH_NAMESPACE", "namespace")

	ns, err := GetWatchNamespace()
	assert.Equal(t, ns, "namespace")
	assert.NoError(t, err)
}

func TestGetWatchNamespace_NotDefined(t *testing.T) {
	ns, err := GetWatchNamespace()
	assert.Equal(t, ns, "")
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "WATCH_NAMESPACE must be set") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGetDebugMode_IsDefined(t *testing.T) {
	t.Setenv("DEBUG_MODE", "true")

	d, err := GetDebugMode()
	assert.True(t, d)
	assert.NoError(t, err)
}

func TestGetDebugMode_NotDefined(t *testing.T) {
	d, err := GetDebugMode()
	assert.False(t, d)
	assert.Nil(t, err)
}

func TestGetDebugMode_ShouldFailOnConvertToBool(t *testing.T) {
	t.Setenv("DEBUG_MODE", "6")

	d, err := GetDebugMode()
	assert.False(t, d)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "strconv.ParseBool: parsing \"6\": invalid syntax") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
