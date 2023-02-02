package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"

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

	jf := &jenkinsApi.JenkinsFolder{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c, gs, jf)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs, jf).Build()

	pjf := NewPutJenkinsFolder(fakeCl)

	if err := pjf.ServeRequest(ctx, c); err != nil {
		t.Error("ServeRequest failed for PutJenkinsFolder")
	}

	gjf := &jenkinsApi.JenkinsFolder{}

	if err := fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "fake-name-codebase",
			Namespace: fakeNamespace,
		},
		gjf); err != nil {
		t.Error("failed to get JenkinsFolder")
	}

	assert.Equal(t, gjf.Spec.Job.Name, "job-provisions/job/ci/job/ci")
}

func TestPutJenkinsFolder_ShouldSkipWhenJenkinsFolderExists(t *testing.T) {
	ctx := context.Background()
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
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
	ctx := context.Background()
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			GitServer: fakeName,
		},
	}
	jf := &jenkinsApi.JenkinsFolder{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c, jf)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, jf).Build()

	pjf := NewPutJenkinsFolder(fakeCl)

	err := pjf.ServeRequest(ctx, c)
	assert.Error(t, err)

	assert.Contains(t, err.Error(), "failed to get fake-name Git Server CR")
}

func Test_getRepositoryPath(t *testing.T) {
	type args struct {
		codebaseName string
		strategy     string
		gitUrlPath   *string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{"Import strategy", args{"codebase-name", consts.ImportStrategy, util.GetStringP("url")}, "url"},
		{"Clone strategy", args{"codebase-name", string(codebaseApi.Clone), util.GetStringP("url")}, "/codebase-name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getRepositoryPath(tt.args.codebaseName, tt.args.strategy, tt.args.gitUrlPath); got != tt.want {
				t.Errorf("getRepositoryPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
