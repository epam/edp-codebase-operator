package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestPutJenkinsFolder_ShouldCreateJenkinsFolder(t *testing.T) {
	ctx := context.Background()
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			BuildTool:       "Maven",
			GitServer:       fakeName,
			JobProvisioning: util.GetStringP("ci"),
			Strategy:        codebaseApi.Clone,
			Repository: &codebaseApi.Repository{
				Url: "https://example.com",
			},
			CiTool:     util.CIJenkins,
			GitUrlPath: pointer.String("/owner/repo"),
		},
	}

	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.GitServerSpec{
			NameSshKeySecret: fakeName,
			GitHost:          "github.com",
			SshPort:          22,
			GitUser:          "git",
		},
	}

	jf := &jenkinsApi.JenkinsFolder{}

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, jenkinsApi.AddToScheme(scheme))

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs, jf).Build()

	pjf := NewPutJenkinsFolder(fakeCl)

	err := pjf.ServeRequest(ctx, c)
	require.NoError(t, err)

	gjf := &jenkinsApi.JenkinsFolder{}

	err = fakeCl.Get(context.Background(),
		types.NamespacedName{
			Name:      "fake-name-codebase",
			Namespace: fakeNamespace,
		},
		gjf)
	require.NoError(t, err)

	assert.Equal(t, gjf.Spec.Job.Name, "job-provisions/job/ci/job/ci")
}

func TestPutJenkinsFolder_ShouldSkipWhenJenkinsFolderExists(t *testing.T) {
	ctx := context.Background()
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			CiTool: util.CIJenkins,
		},
	}

	jf := &jenkinsApi.JenkinsFolder{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "fake-name-codebase",
			Namespace: fakeNamespace,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c, jf)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, jf).Build()

	pjf := NewPutJenkinsFolder(fakeCl)

	if err := pjf.ServeRequest(ctx, c); err != nil {
		t.Error("ServeRequest failed for PutJenkinsFolder")
	}
}

func TestPutJenkinsFolder_ShouldFailWhenGetJenkinsFolder(t *testing.T) {
	ctx := context.Background()
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			CiTool: util.CIJenkins,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	pjf := NewPutJenkinsFolder(fakeCl)

	if err := pjf.ServeRequest(ctx, c); err == nil {
		t.Error("ServeRequest must fail because kind JenkinsFolder is not registered")
	}
}

func TestPutJenkinsFolder_ShouldFailWhenGetGitServer(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			GitServer: fakeName,
			CiTool:    util.CIJenkins,
		},
	}
	jf := &jenkinsApi.JenkinsFolder{}

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, jenkinsApi.AddToScheme(scheme))

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, jf).Build()

	pjf := NewPutJenkinsFolder(fakeCl)

	err := pjf.ServeRequest(context.Background(), c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get GitServer")
}
