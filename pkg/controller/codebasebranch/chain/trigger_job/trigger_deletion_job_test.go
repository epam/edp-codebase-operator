package trigger_job

import (
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/service"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	jenkinsv1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDeletionJob_ShouldPass(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "c-stub-name",
			Namespace: "stub-namespace",
		},
		Spec: v1alpha1.CodebaseSpec{
			Versioning: v1alpha1.Versioning{
				Type: "default",
			},
		},
		Status: v1alpha1.CodebaseStatus{
			Available: true,
		},
	}

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cb-stub-name",
			Namespace: "stub-namespace",
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			BranchName:   "stub-name",
			CodebaseName: "c-stub-name",
		},
	}

	jf := &jenkinsv1alpha1.JenkinsFolder{
		TypeMeta: metav1.TypeMeta{
			APIVersion: util.V2APIVersion,
			Kind:       util.JenkinsFolderKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "c-stub-name-codebase",
			Namespace: c.Namespace,
		},
		Spec: jenkinsv1alpha1.JenkinsFolderSpec{
			Job: &jenkinsv1alpha1.Job{
				Name:   "job-provisions/job/ci/job/name",
				Config: "jenkins-config",
			},
		},
		Status: jenkinsv1alpha1.JenkinsFolderStatus{
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
