package chain

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/repository"
	mockGit "github.com/epam/edp-codebase-operator/v2/controllers/gitserver/mocks"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1"
)

func TestPutGitlabCiFile_ShouldPass(t *testing.T) {
	ctx := context.Background()

	dir, err := os.MkdirTemp("/tmp", "codebase")
	require.NoError(t, err, "unable to create temp directory for testing")

	defer func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	err = os.Setenv("WORKING_DIR", dir)
	require.NoError(t, err)

	err = os.Setenv("PLATFORM_TYPE", "kubernetes")
	require.NoError(t, err)

	ec := &edpCompApi.EDPComponent{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "kubernetes",
			Namespace: fakeNamespace,
		},
		Spec: edpCompApi.EDPComponentSpec{
			Type:    "",
			Url:     "https://kubernetes.default.svc",
			Icon:    "",
			Visible: false,
		},
	}

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Type:      util.Application,
			Framework: util.GetStringP("java11"),
			BuildTool: "Maven",
			GitServer: fakeName,
		},
		Status: codebaseApi.CodebaseStatus{
			Git: *util.GetStringP("pushed"),
		},
	}

	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
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
			"edp_name": "edp-name",
		},
	}
	ssh := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			util.PrivateSShKeyName: []byte("fake"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, ssh, cm)
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c, gs, ec)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs, ssh, cm, ec).Build()

	err = os.Setenv("ASSETS_DIR", "../../../../build")
	require.NoError(t, err)

	// it is expected that code is already landed before running this part of chain,
	// so let's create it
	wd := util.GetWorkDir(fakeName, fakeNamespace)

	err = util.CreateDirectory(wd)
	if err != nil {
		t.Error("Unable to create directory for testing")
	}

	mGit := new(mockGit.MockGit)
	mGit.On("CommitChanges", wd, "Add gitlab ci file").Return(nil)
	mGit.On("PushChanges", "fake", fakeName, wd, gs.Spec.SshPort).Return(nil)

	pg := NewPutGitlabCiFile(
		fakeCl,
		repository.NewK8SCodebaseRepository(fakeCl, c),
		mGit,
	)

	err = pg.ServeRequest(ctx, c)
	assert.NoError(t, err)
}

func TestParseTemplateMethod_ShouldFailToGetEdpComponent(t *testing.T) {
	ec := &edpCompApi.EDPComponent{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "fake-namespace",
		},
		Spec: edpCompApi.EDPComponentSpec{
			Type:    "",
			Url:     "",
			Icon:    "",
			Visible: false,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, ec)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ec).Build()

	ch := NewPutGitlabCiFile(
		fakeCl,
		nil,
		nil,
	)

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeCodebaseName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Framework: util.GetStringP("maven"),
			BuildTool: "maven",
			Lang:      goLang,
			Versioning: codebaseApi.Versioning{
				Type: codebaseApi.Default,
			},
		},
	}

	assert.Error(t, ch.parseTemplate(c))
}

func TestPushChangesMethod_ShouldBeExecutedSuccessfully(t *testing.T) {
	mGit := new(mockGit.MockGit)
	ch := NewPutGitlabCiFile(
		nil,
		nil,
		mGit,
	)

	mGit.On("CommitChanges", "path", "Add gitlab ci file").Return(nil)
	mGit.On("PushChanges", "pkey", "user", "path", int32(22)).Return(nil)

	assert.NoError(t, ch.pushChanges("path", "pkey", "user", "branch", 22))
}

func TestPutGitlabCiFile_gitlabCiFileExistsShouldReturnTrue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeCodebaseName,
			Namespace: fakeNamespace,
		},
		Status: codebaseApi.CodebaseStatus{
			Git: *util.GetStringP("gitlab ci"),
		},
	}
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	repo := repository.NewK8SCodebaseRepository(fakeCl, c)

	h := NewPutGitlabCiFile(
		nil,
		repo,
		nil,
	)

	got, err := h.gitlabCiFileExists(ctx, fakeCodebaseName, "edpName")

	assert.True(t, got)
	assert.Nil(t, err)
}

func TestPutGitlabCiFile_gitlabCiFileExistsShouldReturnFalse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeCodebaseName,
			Namespace: fakeNamespace,
		},
		Status: codebaseApi.CodebaseStatus{
			Git: *util.GetStringP(""),
		},
	}
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	repo := repository.NewK8SCodebaseRepository(fakeCl, c)

	h := NewPutGitlabCiFile(
		nil,
		repo,
		nil,
	)

	got, err := h.gitlabCiFileExists(ctx, fakeCodebaseName, "edpName")

	assert.False(t, got)
	assert.Nil(t, err)
}

func TestParseTemplate_ShouldPass(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	gitlabCIFile := path.Join(tempDir, "test.yaml")
	templatePath := "../../../../build/templates/gitlabci/kubernetes/java11-maven.tmpl"
	data := struct {
		CodebaseName   string
		Namespace      string
		VersioningType string
		ClusterUrl     string
	}{
		fakeName,
		fakeNamespace,
		string(codebaseApi.Default),
		"url",
	}

	err := parseTemplate(templatePath, gitlabCIFile, data)
	assert.NoError(t, err)
}