package trigger_job

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/service"
	mockservice "github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch/service"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	jenkinsv1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestTriggerReleaseJob_ShouldPass(t *testing.T) {
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
	ms := new(mockservice.MockCodebasebranch)

	ms.On("TriggerReleaseJob", cb).Return(nil)

	trj := TriggerReleaseJob{
		TriggerJob: TriggerJob{
			Client:  fakeCl,
			Service: ms,
		},
	}

	err := trj.ServeRequest(cb)
	assert.NoError(t, err)
}

func TestTriggerReleaseJob_ShouldFailWhenTriggerJobReturnsErr(t *testing.T) {
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
	ms := new(mockservice.MockCodebasebranch)

	ms.On("TriggerReleaseJob", cb).Return(errors.New("FATAL ERROR"))

	trj := TriggerReleaseJob{
		TriggerJob: TriggerJob{
			Client:  fakeCl,
			Service: ms,
		},
	}

	err := trj.ServeRequest(cb)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "FATAL ERROR") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	cbResp := &v1alpha1.CodebaseBranch{}
	err = fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "cb-stub-name",
			Namespace: "stub-namespace",
		},
		cbResp)
	// here we have values set with SetIntermediateSuccessFields()
	assert.NoError(t, err)
	assert.Equal(t, cbResp.Status.Value, "inactive")
	assert.Equal(t, cbResp.Status.Action, v1alpha1.ActionType("trigger_release_job"))
	// Our cb object should have values from SetFailedFields(), but values
	// will be set in Reconciler() using defer approach
	assert.Equal(t, cb.Status.Value, "failed")
	assert.Equal(t, cb.Status.DetailedMessage, "FATAL ERROR")
}

func TestTriggerReleaseJob_ShouldFailOnCodebaseNotFound(t *testing.T) {
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
			CodebaseName: "non-existing-stub-name",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c, cb)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb).Build()

	trj := TriggerReleaseJob{
		TriggerJob: TriggerJob{
			Client: fakeCl,
			Service: &service.CodebaseBranchServiceProvider{
				Client: fakeCl,
			},
		},
	}

	err := trj.ServeRequest(cb)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "Unable to get Codebase non-existing-stub-name: codebases.apps \"non-existing-stub-name\" not found") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	cbResp := &v1alpha1.CodebaseBranch{}
	err = fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "cb-stub-name",
			Namespace: "stub-namespace",
		},
		cbResp)
	// here we have values set with SetIntermediateSuccessFields()
	assert.NoError(t, err)
	assert.Equal(t, cbResp.Status.Value, "inactive")
	assert.Equal(t, cbResp.Status.Action, v1alpha1.ActionType("trigger_release_job"))
	// Our cb object should have values from SetFailedFields(), but values
	// will be set in Reconciler() using defer approach
	assert.Equal(t, cb.Status.Value, "failed")
	assert.Equal(t, cb.Status.DetailedMessage, "Unable to get Codebase non-existing-stub-name: codebases.apps \"non-existing-stub-name\" not found")
}

func TestTriggerReleaseJob_ShouldFailOnJenkinsfolderNotFound(t *testing.T) {
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

	jf := &jenkinsv1alpha1.JenkinsFolder{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c, cb, jf)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb, jf).Build()

	trj := TriggerReleaseJob{
		TriggerJob: TriggerJob{
			Client: fakeCl,
			Service: &service.CodebaseBranchServiceProvider{
				Client: fakeCl,
			},
		},
	}

	err := trj.ServeRequest(cb)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "c-stub-name codebase and/or jenkinsfolder c-stub-name-codebase are/is unavailable") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestTriggerReleaseJob_ShouldFailOnSetintermediatestatus(t *testing.T) {

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

	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	trj := TriggerReleaseJob{
		TriggerJob: TriggerJob{
			Client: fakeCl,
			Service: &service.CodebaseBranchServiceProvider{
				Client: fakeCl,
			},
		},
	}

	err := trj.ServeRequest(cb)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "SetIntermediateSuccessFields failed for cb-stub-name branch") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

}

func TestTriggerReleaseJob_ShouldPassEDPVersioningAndHasNewVersion(t *testing.T) {
	var cbVersion string = "1.0.0-SNAPSHOT"

	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "c-stub-name",
			Namespace: "stub-namespace",
		},
		Spec: v1alpha1.CodebaseSpec{
			Versioning: v1alpha1.Versioning{
				Type: "edp",
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
			Version:      &cbVersion,
		},
		Status: v1alpha1.CodebaseBranchStatus{
			VersionHistory: []string{"0.0.0-SNAPSHOT"},
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

	mockcbs := new(mockservice.MockCodebasebranch)

	mockcbs.On("TriggerReleaseJob", cb).Return(nil)
	mockcbs.On("ResetBranchBuildCounter", cb).Return(nil)
	mockcbs.On("ResetBranchSuccessBuildCounter", cb).Return(nil)
	mockcbs.On("AppendVersionToTheHistorySlice", cb).Return(nil)

	trj := TriggerReleaseJob{
		TriggerJob: TriggerJob{
			Client:  fakeCl,
			Service: mockcbs,
		},
	}

	err := trj.ServeRequest(cb)
	assert.NoError(t, err)
}

func TestTriggerReleaseJob_ShouldFailEDPVersioningAndHasNewVersion(t *testing.T) {
	var cbVersion string = "1.0.0-SNAPSHOT"

	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "c-stub-name",
			Namespace: "stub-namespace",
		},
		Spec: v1alpha1.CodebaseSpec{
			Versioning: v1alpha1.Versioning{
				Type: "edp",
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
			Version:      &cbVersion,
		},
		Status: v1alpha1.CodebaseBranchStatus{
			VersionHistory: []string{"0.0.0-SNAPSHOT"},
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

	mockcbs := new(mockservice.MockCodebasebranch)

	mockcbs.On("TriggerReleaseJob", cb).Return(nil)
	mockcbs.On("ResetBranchBuildCounter", cb).Return(nil)
	mockcbs.On("ResetBranchSuccessBuildCounter", cb).Return(errors.New("FATAL ERROR"))
	mockcbs.On("AppendVersionToTheHistorySlice", cb).Return(nil)

	trj := TriggerReleaseJob{
		TriggerJob: TriggerJob{
			Client:  fakeCl,
			Service: mockcbs,
		},
	}

	err := trj.ServeRequest(cb)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "FATAL ERROR") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
