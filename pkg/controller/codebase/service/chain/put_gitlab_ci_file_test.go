package chain

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	edpV1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/repository"
	mockgit "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver/mock"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/epam/edp-component-operator/pkg/apis/v1/v1alpha1"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPutGitlabCiFile_ShouldPass(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer os.RemoveAll(dir)

	os.Setenv("WORKING_DIR", dir)
	os.Setenv("PLATFORM_TYPE", "kubernetes")

	ec := &v1alpha1.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernetes",
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.EDPComponentSpec{
			Type:    "",
			Url:     "https://kubernetes.default.svc",
			Icon:    "",
			Visible: false,
		},
	}

	c := &edpV1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: edpV1alpha1.CodebaseSpec{
			Type:      util.Application,
			Framework: util.GetStringP("java11"),
			BuildTool: "Maven",
			GitServer: fakeName,
		},
		Status: edpV1alpha1.CodebaseStatus{
			Git: *util.GetStringP("pushed"),
		},
	}

	gs := &edpV1alpha1.GitServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: edpV1alpha1.GitServerSpec{
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
			"edp_name": "edp-name",
		},
	}
	ssh := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Data: map[string][]byte{
			util.PrivateSShKeyName: []byte("fake"),
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, ssh, cm)
	scheme.AddKnownTypes(edpV1alpha1.SchemeGroupVersion, c, gs, ec)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs, ssh, cm, ec).Build()

	os.Setenv("ASSETS_DIR", "../../../../../build")

	// it is expected that code is already landed before running this part of chain,
	// so let's create it
	wd := util.GetWorkDir(fakeName, fakeNamespace)
	if err := util.CreateDirectory(wd); err != nil {
		t.Error("Unable to create directory for testing")
	}

	mGit := new(mockgit.MockGit)
	mGit.On("CommitChanges", wd, "Add gitlab ci file").Return(nil)
	mGit.On("PushChanges", "fake", fakeName, wd).Return(nil)

	pg := PutGitlabCiFile{
		client: fakeCl,
		git:    mGit,
		cr:     repository.NewK8SCodebaseRepository(fakeCl, c),
	}

	err = pg.ServeRequest(c)
	assert.NoError(t, err)
}

func TestParseTemplateMethod_ShouldFailToGetEdpComponent(t *testing.T) {
	ec := &v1alpha1.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "fake-namespace",
		},
		Spec: v1alpha1.EDPComponentSpec{
			Type:    "",
			Url:     "",
			Icon:    "",
			Visible: false,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, ec)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ec).Build()

	ch := PutGitlabCiFile{
		client: fakeCl,
	}

	c := &edpV1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeCodebaseName,
			Namespace: fakeNamespace,
		},
		Spec: edpV1alpha1.CodebaseSpec{
			Framework: util.GetStringP("maven"),
			BuildTool: "maven",
			Lang:      goLang,
			Versioning: edpV1alpha1.Versioning{
				Type: edpV1alpha1.Default,
			},
		},
	}

	assert.Error(t, ch.parseTemplate(c))
}

func TestPushChangesMethod_ShouldBeExecutedSuccessfully(t *testing.T) {

	mGit := new(mockgit.MockGit)
	ch := PutGitlabCiFile{
		git: mGit,
	}
	mGit.On("CommitChanges", "path", "Add gitlab ci file").Return(
		nil)
	mGit.On("PushChanges", "pkey", "user", "path").Return(
		nil)
	assert.NoError(t, ch.pushChanges("path", "pkey", "user", "branch"))
}

func TestPutGitlabCiFile_gitlabCiFileExistsShouldReturnTrue(t *testing.T) {
	c := &edpV1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeCodebaseName,
			Namespace: fakeNamespace,
		},
		Status: edpV1alpha1.CodebaseStatus{
			Git: *util.GetStringP("gitlab ci"),
		},
	}
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	repo := repository.NewK8SCodebaseRepository(fakeCl, c)

	h := PutGitlabCiFile{
		cr: repo,
	}
	got, err := h.gitlabCiFileExists(fakeCodebaseName, "edpName")
	assert.True(t, got)
	assert.Nil(t, err)
}

func TestPutGitlabCiFile_gitlabCiFileExistsShouldReturnFalse(t *testing.T) {
	c := &edpV1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeCodebaseName,
			Namespace: fakeNamespace,
		},
		Status: edpV1alpha1.CodebaseStatus{
			Git: *util.GetStringP(""),
		},
	}
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	repo := repository.NewK8SCodebaseRepository(fakeCl, c)

	h := PutGitlabCiFile{
		cr: repo,
	}
	got, err := h.gitlabCiFileExists(fakeCodebaseName, "edpName")
	assert.False(t, got)
	assert.Nil(t, err)
}

func TestPutGitlabCiFile_gitlabCiFileExistsShouldReturnError(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	h := PutGitlabCiFile{
		cr: repository.SqlCodebaseRepository{
			DB: db,
		},
	}
	got, err := h.gitlabCiFileExists(fakeCodebaseName, fakeEdpName)
	assert.False(t, got)
	if !strings.Contains(err.Error(), "couldn't get project_status value for fake_codebase_name codebase") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestParseTemplate_ShouldPass(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp/", "temp")
	if err != nil {
		t.Errorf("create tempDir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	data := struct {
		CodebaseName   string
		Namespace      string
		VersioningType string
		ClusterUrl     string
	}{
		fakeName,
		fakeNamespace,
		string(edpV1alpha1.Default),
		"url",
	}

	err = parseTemplate("../../../../../build/templates/gitlabci/kubernetes/java11-maven.tmpl", fmt.Sprintf("%v/test.yaml", tempDir), data)
	assert.NoError(t, err)
}
