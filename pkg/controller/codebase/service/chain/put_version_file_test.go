package chain

import (
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dchest/uniuri"
	edpV1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/helper"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/repository"
	mock2 "github.com/epmd-edp/codebase-operator/v2/pkg/controller/gitserver/mock"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	v1K8s "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
	"path/filepath"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
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

var path = getExecutableFilePath()

func TestVersionFileExists_VersionFileMustExist(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	h := PutVersionFile{
		next:      nil,
		clientSet: openshift.ClientSet{},
		cr: repository.SqlCodebaseRepository{
			DB: db,
		},
	}

	mock.ExpectPrepare(regexp.QuoteMeta(
		fmt.Sprintf(`select project_status from "%v".codebase where name = $1 ;`, fakeEdpName)))

	mock.ExpectQuery(regexp.QuoteMeta(
		fmt.Sprintf(`select project_status from "%v".codebase where name = $1 ;`, fakeEdpName))).
		WithArgs(fakeCodebaseName).
		WillReturnRows(sqlmock.NewRows([]string{"project_status"}).
			AddRow(util.ProjectVersionGoFilePushedStatus))

	e, err := h.versionFileExists(fakeCodebaseName, fakeEdpName)

	assert.NoError(t, err)
	assert.True(t, e)
}

func TestVersionFileExists_AnErrorOccursDueToInvalidInputParameter(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	h := PutVersionFile{
		next:      nil,
		clientSet: openshift.ClientSet{},
		cr: repository.SqlCodebaseRepository{
			DB: db,
		},
	}

	mock.ExpectPrepare(regexp.QuoteMeta(
		fmt.Sprintf(`select project_status from "%v".codebase where name = $1 ;`, fakeEdpName)))

	mock.ExpectQuery(regexp.QuoteMeta(
		fmt.Sprintf(`select project_status from "%v".codebase where name = $1 ;`, fakeInputParam))).
		WithArgs(fakeCodebaseName).
		WillReturnRows(sqlmock.NewRows([]string{"project_status"}).
			AddRow(util.ProjectVersionGoFilePushedStatus))

	e, err := h.versionFileExists(fakeCodebaseName, fakeEdpName)

	assert.Error(t, err)
	assert.False(t, e)
}

func TestCreateFile_FileMustBeCreated(t *testing.T) {
	path := fmt.Sprintf("%v/%v", path, uniuri.NewLen(5))
	err := createFile(path)
	defer clear(path)

	assert.NoError(t, err)
}

func TestCreateFile_MethodMustThrowAnException(t *testing.T) {
	err := createFile("")

	assert.Error(t, err)
}

func TestWriteFile_DataMustBeWritten(t *testing.T) {
	path := fmt.Sprintf("%v/%v", path, uniuri.NewLen(5))
	rerr := createFile(path)
	defer clear(path)
	werr := writeFile(path)

	assert.NoError(t, rerr)
	assert.NoError(t, werr)
}

func TestWriteFile_MethodMustThrowAnException(t *testing.T) {
	path := fmt.Sprintf("%v/%v", path, uniuri.NewLen(5))
	rerr := createFile(path)
	defer clear(path)
	werr := writeFile("")

	assert.NoError(t, rerr)
	assert.Error(t, werr)
}

func getExecutableFilePath() string {
	executableFilePath, err := os.Executable()
	if err != nil {
		println(err)
	}
	return filepath.Dir(executableFilePath)
}

func clear(path string) {
	if err := os.Remove(path); err != nil {
		os.Exit(1)
	}
}

func TestTryToPutVersionFileMethod_MustBeFinishedSuccessfully(t *testing.T) {
	//mock methods of git interface
	mGit := new(mock2.MockGit)
	mGit.On("CommitChanges", path, fmt.Sprintf("Add %v file", versionFileName)).Return(
		nil)
	mGit.On("PushChanges", fakePrivateKey, fakeUser, path).Return(
		nil)

	h := PutVersionFile{
		next: nil,
		clientSet: openshift.ClientSet{
			Client: initMockedClient(),
		},
		git: mGit,
	}

	c := &edpV1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeCodebaseName,
			Namespace: fakeNamespace,
		},
		Spec: edpV1alpha1.CodebaseSpec{
			Lang: goLang,
			Versioning: edpV1alpha1.Versioning{
				Type: edpV1alpha1.Default,
			},
		},
	}

	err := h.tryToPutVersionFile(c, path)
	defer clear(fmt.Sprintf("%v/VERSION", path))

	assert.NoError(t, err)
}

func initMockedClient() client.Client {
	gs := &edpV1alpha1.GitServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeGitServerName,
			Namespace: fakeNamespace,
		},
		Spec: edpV1alpha1.GitServerSpec{
			GitHost:                  "fake_host",
			GitUser:                  "fake_user",
			HttpsPort:                8080,
			SshPort:                  22,
			NameSshKeySecret:         "fake_secret_name",
			CreateCodeReviewPipeline: false,
		},
	}

	secret := &v1K8s.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake_secret_name",
			Namespace: "fake_namespace",
		},
		Data: map[string][]byte{
			util.PrivateSShKeyName: []byte(fakePrivateKey),
		},
	}

	cm := &v1K8s.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      helper.EDPConfigCM,
			Namespace: fakeNamespace,
		},
		Data: map[string]string{
			helper.EDPNameKey: fakeEdpName,
		},
	}

	objs := []runtime.Object{
		cm, gs, secret,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, gs)

	return fake.NewFakeClient(objs...)
}
