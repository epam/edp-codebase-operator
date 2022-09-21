package chain

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dchest/uniuri"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1K8s "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/helper"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/repository"
	mockGit "github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver/mock"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	perfApi "github.com/epam/edp-perf-operator/v2/pkg/apis/edp/v1alpha1"
)

const (
	fakeNamespace     = "fake_namespace"
	fakeEdpName       = "fake_edp_name"
	fakeGitServerName = "fake_git_server_name"
	fakeCodebaseName  = "fake_codebase_name"

	fakePrivateKey = "fake_private_key"
	fakeUser       = "fake_user"
	fakeInputParam = "fake_input_param"
)

var path = "/tmp"

func init() {
	utilRuntime.Must(perfApi.AddToScheme(scheme.Scheme))
	utilRuntime.Must(codebaseApi.AddToScheme(scheme.Scheme))
}

func TestVersionFileExists_VersionFileMustExist(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)

	defer func() {
		err = db.Close()
		require.NoError(t, err)
	}()

	h := NewPutVersionFile(
		nil,
		repository.SqlCodebaseRepository{
			DB: db,
		},
		nil,
	)

	dbMock.ExpectPrepare(regexp.QuoteMeta(
		fmt.Sprintf(`select project_status from "%v".codebase where name = $1 ;`, fakeEdpName)))

	dbMock.ExpectQuery(regexp.QuoteMeta(
		fmt.Sprintf(`select project_status from "%v".codebase where name = $1 ;`, fakeEdpName))).
		WithArgs(fakeCodebaseName).
		WillReturnRows(sqlmock.NewRows([]string{"project_status"}).
			AddRow(util.ProjectVersionGoFilePushedStatus))

	dbMock.ExpectClose()

	e, err := h.versionFileExists(fakeCodebaseName, fakeEdpName)
	assert.NoError(t, err)
	assert.True(t, e)
}

func TestVersionFileExists_AnErrorOccursDueToInvalidInputParameter(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)

	defer func() {
		err = db.Close()
		require.NoError(t, err)
	}()

	h := NewPutVersionFile(
		nil,
		repository.SqlCodebaseRepository{
			DB: db,
		},
		nil,
	)

	dbMock.ExpectPrepare(
		regexp.QuoteMeta(fmt.Sprintf(`select project_status from "%v".codebase where name = $1 ;`, fakeEdpName)),
	).WillReturnError(assert.AnError)

	dbMock.ExpectClose()

	e, err := h.versionFileExists(fakeCodebaseName, fakeEdpName)
	assert.ErrorIs(t, err, assert.AnError)
	assert.False(t, e)
}

func TestCreateFile_FileMustBeCreated(t *testing.T) {
	p := fmt.Sprintf("%v/%v", path, uniuri.NewLen(5))
	err := createFile(p)
	defer clear(p)

	assert.NoError(t, err)
}

func TestCreateFile_MethodMustThrowAnException(t *testing.T) {
	err := createFile("")

	assert.Error(t, err)
}

func TestWriteFile_DataMustBeWritten(t *testing.T) {
	p := fmt.Sprintf("%v/%v", path, uniuri.NewLen(5))

	err := createFile(p)
	require.NoError(t, err)

	defer clear(p)

	err = writeFile(p)
	assert.NoError(t, err)
}

func TestWriteFile_MethodMustThrowAnException(t *testing.T) {
	p := fmt.Sprintf("%v/%v", path, uniuri.NewLen(5))

	err := createFile(p)
	require.NoError(t, err)

	defer clear(p)

	err = writeFile("")
	assert.Error(t, err)
}

func clear(path string) {
	if err := os.Remove(path); err != nil {
		os.Exit(1)
	}
}

func TestTryToPutVersionFileMethod_MustBeFinishedSuccessfully(t *testing.T) {
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
	testScheme.AddKnownTypes(codebaseApi.SchemeGroupVersion, gs)
	testScheme.AddKnownTypes(v1K8s.SchemeGroupVersion, secret, cm)
	fakeCl := fake.NewClientBuilder().WithScheme(testScheme).WithRuntimeObjects(gs, secret, cm).Build()

	// mock methods of git interface
	mGit := new(mockGit.MockGit)
	mGit.On("CommitChanges", path, fmt.Sprintf("Add %v file", versionFileName)).Return(
		nil)
	mGit.On("PushChanges", fakePrivateKey, fakeUser, path).Return(
		nil)
	mGit.On("Checkout", util.GetPointerStringP(nil), util.GetPointerStringP(nil), path, "", true).Return(
		nil)
	mGit.On("GetCurrentBranchName", path).Return(
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

	err := h.tryToPutVersionFile(c, path)
	defer clear(fmt.Sprintf("%v/VERSION", path))

	assert.NoError(t, err)
}
