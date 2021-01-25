package update_perf_data_sources

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	perfApi "github.com/epmd-edp/perf-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	fakeName      = "fake-name"
	fakeNamespace = "fake-namespace"
)

func TestUpdatePerfDataSources_ShouldSkipUpdating(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
	}

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			OwnerReferences: []v1.OwnerReference{{
				Kind: "Codebase",
				Name: fakeName,
			}},
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: fakeName,
		},
	}

	objs := []runtime.Object{
		c, cb,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, c, cb)

	assert.NoError(t, UpdatePerfDataSources{Client: fake.NewFakeClient(objs...)}.ServeRequest(cb))
}

func TestUpdatePerfDataSources_DsShouldBeUpdated(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			Perf: &v1alpha1.Perf{
				Name:        fakeName,
				DataSources: []string{"Jenkins", "Sonar"},
			},
		},
	}

	p := &perfApi.PerfDataSourceJenkins{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v-%v", fakeName, "jenkins"),
			Namespace: fakeNamespace,
		},
	}

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			OwnerReferences: []v1.OwnerReference{{
				Kind: "Codebase",
				Name: fakeName,
			}},
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: fakeName,
			BranchName:   fakeName,
		},
	}

	objs := []runtime.Object{
		c, p, cb,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, c, p, cb)
	assert.NoError(t, UpdatePerfDataSources{Client: fake.NewFakeClient(objs...)}.ServeRequest(cb))
}

func TestUpdatePerfDataSources_CodebaseShouldNotBeFound(t *testing.T) {
	p := &perfApi.PerfDataSourceJenkins{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v-%v", fakeName, "jenkins"),
			Namespace: fakeNamespace,
		},
	}

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			OwnerReferences: []v1.OwnerReference{{
				Kind: "Codebase",
				Name: fakeName,
			}},
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: fakeName,
			BranchName:   fakeName,
		},
	}

	objs := []runtime.Object{
		p, cb,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, p, cb)
	assert.Error(t, UpdatePerfDataSources{Client: fake.NewFakeClient(objs...)}.ServeRequest(cb))
}

func TestUpdatePerfDataSources_PerfDataSourceShouldNotBeFound(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			Perf: &v1alpha1.Perf{
				Name:        fakeName,
				DataSources: []string{"Jenkins", "Sonar"},
			},
		},
	}

	p := &perfApi.PerfDataSourceSonar{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v-%v", fakeName, "sonar"),
			Namespace: fakeNamespace,
		},
	}

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			OwnerReferences: []v1.OwnerReference{{
				Kind: "Codebase",
				Name: fakeName,
			}},
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: fakeName,
			BranchName:   fakeName,
		},
	}

	objs := []runtime.Object{
		c, p, cb,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, c, p, cb)
	assert.Error(t, UpdatePerfDataSources{Client: fake.NewFakeClient(objs...)}.ServeRequest(cb))
}

func TestUpdatePerfDataSources_JenkinsDsShouldBeUpdated(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			Perf: &v1alpha1.Perf{
				Name:        fakeName,
				DataSources: []string{"Jenkins", "Sonar"},
			},
		},
	}

	p := &perfApi.PerfDataSourceJenkins{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v-%v", fakeName, "jenkins"),
			Namespace: fakeNamespace,
		},
		Spec: perfApi.PerfDataSourceJenkinsSpec{
			Config: perfApi.DataSourceJenkinsConfig{
				JobNames: []string{"/fake-name/FAKE-NAME-Build-fake-name"},
			},
		},
	}

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			OwnerReferences: []v1.OwnerReference{{
				Kind: "Codebase",
				Name: fakeName,
			}},
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: fakeName,
			BranchName:   fakeName,
		},
	}

	objs := []runtime.Object{
		c, p, cb,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, c, p, cb)
	assert.NoError(t, UpdatePerfDataSources{Client: fake.NewFakeClient(objs...)}.ServeRequest(cb))
}

func TestUpdatePerfDataSources_GitLabDsShouldBeUpdated(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseSpec{
			Perf: &v1alpha1.Perf{
				Name:        fakeName,
				DataSources: []string{"GitLab"},
			},
		},
	}

	p := &perfApi.PerfDataSourceGitLab{
		ObjectMeta: metav1.ObjectMeta{
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

	cb := &v1alpha1.CodebaseBranch{
		ObjectMeta: v1.ObjectMeta{
			OwnerReferences: []v1.OwnerReference{{
				Kind: "Codebase",
				Name: fakeName,
			}},
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: v1alpha1.CodebaseBranchSpec{
			CodebaseName: fakeName,
			BranchName:   fakeName,
		},
	}

	objs := []runtime.Object{
		c, p, cb,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, c, p, cb)
	assert.NoError(t, UpdatePerfDataSources{Client: fake.NewFakeClient(objs...)}.ServeRequest(cb))
}
