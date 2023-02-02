package trigger_job

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/service"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestDeletionJob_ShouldPass(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "c-stub-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			Versioning: codebaseApi.Versioning{
				Type: "default",
			},
		},
		Status: codebaseApi.CodebaseStatus{
			Available: true,
		},
	}

	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cb-stub-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			BranchName:   "stub-name",
			CodebaseName: "c-stub-name",
		},
	}

	jf := &jenkinsApi.JenkinsFolder{
		TypeMeta: metaV1.TypeMeta{
			APIVersion: util.V2APIVersion,
			Kind:       util.JenkinsFolderKind,
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "c-stub-name-codebase",
			Namespace: c.Namespace,
		},
		Spec: jenkinsApi.JenkinsFolderSpec{
			Job: &jenkinsApi.Job{
				Name:   "job-provisions/job/ci/job/name",
				Config: "jenkins-config",
			},
		},
		Status: jenkinsApi.JenkinsFolderStatus{
			Available: true,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c, cb, jf)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb, jf).Build()
	ms := new(service.MockCodebasebranch)

	ms.On("TriggerDeletionJob", cb).Return(nil)

	trj := TriggerDeletionJob{
		TriggerJob: TriggerJob{
			Client:  fakeCl,
			Service: ms,
		},
	}

	err := trj.ServeRequest(cb)
	assert.NoError(t, err)
}
