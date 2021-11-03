package chain

import (
	"context"
	"strings"
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	jenkinsv1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPutJenkinsFolder_ShouldCreateJenkinsfolder(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			BuildTool:       "Maven",
			GitServer:       fakeName,
			JobProvisioning: util.GetStringP("ci"),
			Strategy:        v1alpha1.Clone,
			Repository: &v1alpha1.Repository{
				Url: "https://example.com",
			},
		},
	}

	gs := &v1alpha1.GitServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.GitServerSpec{
			NameSshKeySecret: fakeName,
			GitHost:          fakeName,
			SshPort:          22,
			GitUser:          fakeName,
		},
	}

	jf := &jenkinsv1alpha1.JenkinsFolder{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, gs, jf)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, gs, jf).Build()

	pjf := PutJenkinsFolder{
		client: fakeCl,
	}

	if err := pjf.ServeRequest(c); err != nil {
		t.Error("ServeRequest failed for PutJenkinsFolder")
	}
	gjf := &jenkinsv1alpha1.JenkinsFolder{}
	if err := fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "fake-name-codebase",
			Namespace: fakeNamespace,
		},
		gjf); err != nil {
		t.Error("Unable to get JenkinsFolder")
	}
	assert.Equal(t, gjf.Spec.Job.Name, "job-provisions/job/ci/job/ci")
}

func TestPutJenkinsFolder_ShouldSkipWhenJenkinsfolderExists(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
	}

	jf := &jenkinsv1alpha1.JenkinsFolder{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name-codebase",
			Namespace: fakeNamespace,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, jf)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, jf).Build()

	pjf := PutJenkinsFolder{
		client: fakeCl,
	}

	if err := pjf.ServeRequest(c); err != nil {
		t.Error("ServeRequest failed for PutJenkinsFolder")
	}
}

func TestPutJenkinsFolder_ShouldFailWhenGetJenkinsfolder(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	pjf := PutJenkinsFolder{
		client: fakeCl,
	}

	if err := pjf.ServeRequest(c); err == nil {
		t.Error("ServeRequest must fail because kind JenkinsFolder is not registered")
	}
}

func TestPutJenkinsFolder_ShouldFailWhenGetGitServer(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			GitServer: fakeName,
		},
	}
	jf := &jenkinsv1alpha1.JenkinsFolder{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, jf)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, jf).Build()

	pjf := PutJenkinsFolder{
		client: fakeCl,
	}

	err := pjf.ServeRequest(c)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "an error has occurred while getting fake-name Git Server CR") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
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
		{"Clone strategy", args{"codebase-name", string(v1alpha1.Clone), util.GetStringP("url")}, "/codebase-name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getRepositoryPath(tt.args.codebaseName, tt.args.strategy, tt.args.gitUrlPath); got != tt.want {
				t.Errorf("getRepositoryPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
