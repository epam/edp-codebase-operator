package update_perf_data_sources

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	perfApi "github.com/epam/edp-perf-operator/v2/pkg/apis/edp/v1alpha1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
)

const (
	fakeName      = "fake-name"
	fakeNamespace = "fake-namespace"
)

func TestUpdatePerfDataSources_ShouldSkipUpdating(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
	}

	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			OwnerReferences: []metaV1.OwnerReference{{
				Kind: "Codebase",
				Name: fakeName,
			}},
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: fakeName,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c, cb)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cb).Build()

	assert.NoError(t, UpdatePerfDataSources{Client: fakeCl}.ServeRequest(cb))
}

func TestUpdatePerfDataSources_DsShouldBeUpdated(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Perf: &codebaseApi.Perf{
				Name:        fakeName,
				DataSources: []string{"Jenkins", "Sonar"},
			},
		},
	}

	p := &perfApi.PerfDataSourceJenkins{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fmt.Sprintf("%v-%v", fakeName, "jenkins"),
			Namespace: fakeNamespace,
		},
	}

	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			OwnerReferences: []metaV1.OwnerReference{{
				Kind: "Codebase",
				Name: fakeName,
			}},
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: fakeName,
			BranchName:   fakeName,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c, p, cb)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, p, cb).Build()
	assert.NoError(t, UpdatePerfDataSources{Client: fakeCl}.ServeRequest(cb))
}

func TestUpdatePerfDataSources_CodebaseShouldNotBeFound(t *testing.T) {
	p := &perfApi.PerfDataSourceJenkins{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fmt.Sprintf("%v-%v", fakeName, "jenkins"),
			Namespace: fakeNamespace,
		},
	}

	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			OwnerReferences: []metaV1.OwnerReference{{
				Kind: "Codebase",
				Name: fakeName,
			}},
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: fakeName,
			BranchName:   fakeName,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, p, cb)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(p, cb).Build()
	assert.Error(t, UpdatePerfDataSources{Client: fakeCl}.ServeRequest(cb))
}

func TestUpdatePerfDataSources_PerfDataSourceShouldNotBeFound(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Perf: &codebaseApi.Perf{
				Name:        fakeName,
				DataSources: []string{"Jenkins", "Sonar"},
			},
		},
	}

	p := &perfApi.PerfDataSourceSonar{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fmt.Sprintf("%v-%v", fakeName, "sonar"),
			Namespace: fakeNamespace,
		},
	}

	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			OwnerReferences: []metaV1.OwnerReference{{
				Kind: "Codebase",
				Name: fakeName,
			}},
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: fakeName,
			BranchName:   fakeName,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c, p, cb)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, p, cb).Build()
	assert.Error(t, UpdatePerfDataSources{Client: fakeCl}.ServeRequest(cb))
}

func TestUpdatePerfDataSources_JenkinsDsShouldBeUpdated(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Perf: &codebaseApi.Perf{
				Name:        fakeName,
				DataSources: []string{"Jenkins", "Sonar"},
			},
		},
	}

	p := &perfApi.PerfDataSourceJenkins{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fmt.Sprintf("%v-%v", fakeName, "jenkins"),
			Namespace: fakeNamespace,
		},
		Spec: perfApi.PerfDataSourceJenkinsSpec{
			Config: perfApi.DataSourceJenkinsConfig{
				JobNames: []string{"/fake-name/FAKE-NAME-Build-fake-name"},
			},
		},
	}

	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			OwnerReferences: []metaV1.OwnerReference{{
				Kind: "Codebase",
				Name: fakeName,
			}},
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: fakeName,
			BranchName:   fakeName,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c, p, cb)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, p, cb).Build()
	assert.NoError(t, UpdatePerfDataSources{Client: fakeCl}.ServeRequest(cb))
}

func TestUpdatePerfDataSources_GitLabDsShouldBeUpdated(t *testing.T) {
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			Perf: &codebaseApi.Perf{
				Name:        fakeName,
				DataSources: []string{"GitLab"},
			},
		},
	}

	p := &perfApi.PerfDataSourceGitLab{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fmt.Sprintf("%v-%v", fakeName, "gitlab"),
			Namespace: fakeNamespace,
		},
		Spec: perfApi.PerfDataSourceGitLabSpec{
			Config: perfApi.DataSourceGitLabConfig{
				Repositories: []string{"repo1"},
				Branches:     []string{"master"},
				Url:          fakeName,
			},
		},
	}

	cb := &codebaseApi.CodebaseBranch{
		ObjectMeta: metaV1.ObjectMeta{
			OwnerReferences: []metaV1.OwnerReference{{
				Kind: "Codebase",
				Name: fakeName,
			}},
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: fakeName,
			BranchName:   fakeName,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, c, p, cb)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, p, cb).Build()
	assert.NoError(t, UpdatePerfDataSources{Client: fakeCl}.ServeRequest(cb))
}
