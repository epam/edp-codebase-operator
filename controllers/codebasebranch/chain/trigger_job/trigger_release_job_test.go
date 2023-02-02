package trigger_job

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch/service"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestTriggerReleaseJob_ShouldPass(t *testing.T) {
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

	cbResp := &codebaseApi.CodebaseBranch{}
	err = fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "cb-stub-name",
			Namespace: "stub-namespace",
		},
		cbResp)
	// here we have values set with SetIntermediateSuccessFields()
	assert.NoError(t, err)
	assert.Equal(t, cbResp.Status.Value, "inactive")
	assert.Equal(t, cbResp.Status.Action, codebaseApi.ActionType("trigger_release_job"))
	// Our cb object should have values from SetFailedFields(), but values
	// will be set in Reconciler() using defer approach
	assert.Equal(t, cb.Status.Value, "failed")
	assert.Equal(t, cb.Status.DetailedMessage, "FATAL ERROR")
}

func TestTriggerReleaseJob_ShouldFailOnCodebaseNotFound(t *testing.T) {
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

	if !strings.Contains(err.Error(), "failed to get Codebase non-existing-stub-name: codebases.apps \"non-existing-stub-name\" not found") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	cbResp := &codebaseApi.CodebaseBranch{}
	err = fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "cb-stub-name",
			Namespace: "stub-namespace",
		},
		cbResp)
	// here we have values set with SetIntermediateSuccessFields()
	assert.NoError(t, err)
	assert.Equal(t, cbResp.Status.Value, "inactive")
	assert.Equal(t, cbResp.Status.Action, codebaseApi.ActionType("trigger_release_job"))
	// Our cb object should have values from SetFailedFields(), but values
	// will be set in Reconciler() using defer approach
	assert.Equal(t, cb.Status.Value, "failed")
	assert.Equal(t, cb.Status.DetailedMessage, "failed to get Codebase non-existing-stub-name: codebases.apps \"non-existing-stub-name\" not found")
}

func TestTriggerReleaseJob_ShouldFailOnJenkinsfolderNotFound(t *testing.T) {
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

	jf := &jenkinsApi.JenkinsFolder{}

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

func TestTriggerReleaseJob_ShouldFailOnSetIntermediateStatus(t *testing.T) {
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

	if !strings.Contains(err.Error(), `failed to update CodebaseBranchStatus status field "cb-stub-name"`) {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestTriggerReleaseJob_ShouldPassEDPVersioningAndHasNewVersion(t *testing.T) {
	var cbVersion string = "1.0.0-SNAPSHOT"

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "c-stub-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			Versioning: codebaseApi.Versioning{
				Type: "edp",
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
			Version:      &cbVersion,
		},
		Status: codebaseApi.CodebaseBranchStatus{
			VersionHistory: []string{"0.0.0-SNAPSHOT"},
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

	mockcbs := new(service.MockCodebasebranch)

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

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "c-stub-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			Versioning: codebaseApi.Versioning{
				Type: "edp",
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
			Version:      &cbVersion,
		},
		Status: codebaseApi.CodebaseBranchStatus{
			VersionHistory: []string{"0.0.0-SNAPSHOT"},
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

	mockcbs := new(service.MockCodebasebranch)

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

func TestTriggerReleaseJob_ShouldFailNoEDPVersion(t *testing.T) {
	codeBase := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "c-stub-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseSpec{
			Versioning: codebaseApi.Versioning{
				Type: "edp",
			},
		},
		Status: codebaseApi.CodebaseStatus{
			Available: true,
		},
	}

	codeBaseBranch := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cb-stub-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			BranchName:   "stub-name",
			CodebaseName: "c-stub-name",
		},
	}

	jenkinsFolder := &jenkinsApi.JenkinsFolder{
		TypeMeta: metaV1.TypeMeta{
			APIVersion: util.V2APIVersion,
			Kind:       util.JenkinsFolderKind,
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "c-stub-name-codebase",
			Namespace: codeBase.Namespace,
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
	scheme.AddKnownTypes(v1.SchemeGroupVersion, codeBase, codeBaseBranch, jenkinsFolder)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(codeBase, codeBaseBranch, jenkinsFolder).Build()
	ms := &service.MockCodebasebranch{}

	ms.On("TriggerReleaseJob", codeBaseBranch).Return(nil)

	job := TriggerReleaseJob{
		TriggerJob: TriggerJob{
			Client:  fakeCl,
			Service: ms,
		},
	}

	err := job.ServeRequest(codeBaseBranch)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "doesn't have version")
}
