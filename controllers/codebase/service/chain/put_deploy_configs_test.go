package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	testify "github.com/stretchr/testify/mock"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	gitproviderv2 "github.com/epam/edp-codebase-operator/v2/pkg/git/v2"
	gitServerMocks "github.com/epam/edp-codebase-operator/v2/pkg/git/v2/mocks"
	"github.com/epam/edp-codebase-operator/v2/pkg/platform"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestPutDeployConfigs_ShouldPass(t *testing.T) {
	t.Setenv("WORKING_DIR", t.TempDir())

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
			GitUrlPath:       fakeName,
			Repository: &codebaseApi.Repository{
				Url: "repo",
			},
			GitServer: "gerrit",
		},
		Status: codebaseApi.CodebaseStatus{
			Git: *util.GetStringP("pushed"),
		},
	}
	s := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "repository-codebase-fake-name-temp",
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
			NameSshKeySecret: "gerrit-secret",
			GitHost:          fakeName,
			SshPort:          22,
			GitUser:          fakeName,
			GitProvider:      codebaseApi.GitProviderGerrit,
		},
	}
	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      platform.KrciConfigMap,
			Namespace: fakeNamespace,
		},
		Data: map[string]string{
			"dns_wildcard": "dns",
		},
	}
	ssh := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      gs.Spec.NameSshKeySecret,
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			util.PrivateSShKeyName: []byte("fake"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, ssh, cm, s)
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c, gs)

	fakeCl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(c, gs, ssh, cm, s).
		WithStatusSubresource(c, gs, ssh, cm, s).
		Build()

	t.Setenv(util.AssetsDirEnv, "../../../../build")

	wd := util.GetWorkDir(fakeName, fakeNamespace)

	mGit := gitServerMocks.NewMockGit(t)

	mGit.On("GetCurrentBranchName", testify.Anything, wd).Return("master", nil)
	mGit.On("Checkout", testify.Anything, wd, "fake-defaultBranch", false).Return(nil)
	mGit.On("Commit", testify.Anything, wd, fmt.Sprintf("Add deployment templates for %v", c.Name)).Return(nil)
	mGit.On("Push", testify.Anything, wd, gitproviderv2.RefSpecPushAllBranches).Return(nil)
	mGit.On("Clone", testify.Anything, testify.Anything, wd).Return(nil)

	pdc := NewPutDeployConfigs(
		fakeCl,
		func(cfg gitproviderv2.Config) gitproviderv2.Git {
			return mGit
		},
	)

	err := pdc.ServeRequest(context.Background(), c)
	assert.NoError(t, err)
}

func TestPutDeployConfigs_ShouldPassWithNonApplication(t *testing.T) {
	t.Setenv("WORKING_DIR", t.TempDir())

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Type: "Library",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	t.Setenv(util.AssetsDirEnv, "../../../../build")

	mGit := gitServerMocks.NewMockGit(t)

	pdc := NewPutDeployConfigs(
		fakeCl,
		func(config gitproviderv2.Config) gitproviderv2.Git {
			return mGit
		},
	)

	err := pdc.ServeRequest(context.Background(), c)
	assert.NoError(t, err)
}
