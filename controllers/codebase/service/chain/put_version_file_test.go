package chain

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/dchest/uniuri"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1K8s "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/helper"
	mockGit "github.com/epam/edp-codebase-operator/v2/controllers/gitserver/mocks"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

const (
	fakeNamespace     = "fake_namespace"
	fakeEdpName       = "fake_edp_name"
	fakeGitServerName = "fake_git_server_name"
	fakeCodebaseName  = "fake_codebase_name"

	fakePrivateKey = "fake_private_key"
	fakeUser       = "fake_user"
)

func TestCreateFile_FileMustBeCreated(t *testing.T) {
	t.Parallel()

	p := path.Join(t.TempDir(), uniuri.NewLen(5))
	err := createFile(p)

	defer clear(p)

	assert.NoError(t, err)
}

func TestCreateFile_MethodMustThrowAnException(t *testing.T) {
	err := createFile("")

	assert.Error(t, err)
}

func TestWriteFile_DataMustBeWritten(t *testing.T) {
	p := path.Join(t.TempDir(), uniuri.NewLen(5))

	err := createFile(p)
	require.NoError(t, err)

	defer clear(p)

	err = writeFile(p)
	assert.NoError(t, err)
}

func TestWriteFile_MethodMustThrowAnException(t *testing.T) {
	p := path.Join(t.TempDir(), uniuri.NewLen(5))

	err := createFile(p)
	require.NoError(t, err)

	defer clear(p)

	err = writeFile("")
	assert.Error(t, err)
}

func clear(p string) {
	if err := os.Remove(p); err != nil {
		os.Exit(1)
	}
}

func TestTryToPutVersionFileMethod_MustBeFinishedSuccessfully(t *testing.T) {
	tmpDir := t.TempDir()
	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeGitServerName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.GitServerSpec{
			GitHost:          "fake_host",
			GitUser:          "fake_user",
			HttpsPort:        8080,
			SshPort:          22,
			NameSshKeySecret: "fake_secret_name",
		},
	}

	secret := &v1K8s.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake_secret_name",
			Namespace: "fake_namespace",
		},
		Data: map[string][]byte{
			util.PrivateSShKeyName: []byte(fakePrivateKey),
		},
	}

	cm := &v1K8s.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      helper.EDPConfigCM,
			Namespace: fakeNamespace,
		},
		Data: map[string]string{
			helper.EDPNameKey: fakeEdpName,
		},
	}

	testScheme := runtime.NewScheme()
	testScheme.AddKnownTypes(codebaseApi.GroupVersion, gs)
	testScheme.AddKnownTypes(v1K8s.SchemeGroupVersion, secret, cm)
	fakeCl := fake.NewClientBuilder().WithScheme(testScheme).WithRuntimeObjects(gs, secret, cm).Build()

	// mock methods of git interface
	mGit := new(mockGit.MockGit)
	mGit.On("CommitChanges", tmpDir, fmt.Sprintf("Add %v file", versionFileName)).Return(
		nil)
	mGit.On("PushChanges", fakePrivateKey, fakeUser, tmpDir, gs.Spec.SshPort).Return(
		nil)
	mGit.On("Checkout", util.GetPointerStringP(nil), util.GetPointerStringP(nil), tmpDir, "", true).Return(
		nil)
	mGit.On("GetCurrentBranchName", tmpDir).Return(
		"", nil)
	mGit.On("CheckPermissions", "https://github.com/epmd-edp/go-go-go.git", util.GetPointerStringP(nil), util.GetPointerStringP(nil)).Return(
		true)

	h := NewPutVersionFile(
		fakeCl,
		nil,
		mGit,
	)

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeCodebaseName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			GitServer: fakeGitServerName,
			Lang:      goLang,
			BuildTool: goLang,
			Framework: util.GetStringP(goLang),
			Versioning: codebaseApi.Versioning{
				Type: codebaseApi.Default,
			},
		},
	}

	err := h.tryToPutVersionFile(c, tmpDir)

	assert.NoError(t, err)
}
