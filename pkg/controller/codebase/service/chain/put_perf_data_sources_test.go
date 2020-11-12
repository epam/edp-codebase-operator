package chain

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpCompApi "github.com/epmd-edp/edp-component-operator/pkg/apis/v1/v1alpha1"
	perfApi "github.com/epmd-edp/perf-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	fakeName = "fake-name"
)

func TestPutPerfDataSourcesChain_SkipCreatingPerfDataSource(t *testing.T) {
	sources := PutPerfDataSources{
		client: nil,
	}
	c := &v1alpha1.Codebase{
		ObjectMeta: v12.ObjectMeta{
			Name: "fake-name",
		},
		Spec: v1alpha1.CodebaseSpec{},
	}
	assert.NoError(t, sources.ServeRequest(c))
}

func TestPutPerfDataSourcesChain_JenkinsAndSonarDataSourcesShouldBeCreated(t *testing.T) {
	ecJenkins := &edpCompApi.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jenkins",
			Namespace: fakeNamespace,
		},
		Spec: edpCompApi.EDPComponentSpec{},
	}

	ecSonar := &edpCompApi.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sonar",
			Namespace: fakeNamespace,
		},
		Spec: edpCompApi.EDPComponentSpec{},
	}

	objs := []runtime.Object{
		ecJenkins, ecSonar,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, ecJenkins, ecSonar,
		&perfApi.PerfDataSourceJenkins{}, &perfApi.PerfDataSourceJenkinsList{},
		&perfApi.PerfDataSourceSonar{}, &perfApi.PerfDataSourceSonarList{})

	c := &v1alpha1.Codebase{
		ObjectMeta: v12.ObjectMeta{
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
	assert.NoError(t, PutPerfDataSources{client: fake.NewFakeClient(objs...)}.ServeRequest(c))
}

func TestPutPerfDataSourcesChain_ShouldNotFoundEdpComponent(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion,
		&perfApi.PerfDataSourceJenkins{}, &perfApi.PerfDataSourceJenkinsList{},
		&perfApi.PerfDataSourceSonar{}, &perfApi.PerfDataSourceSonarList{})

	c := &v1alpha1.Codebase{
		ObjectMeta: v12.ObjectMeta{
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
	assert.Error(t, PutPerfDataSources{client: fake.NewFakeClient([]runtime.Object{}...)}.ServeRequest(c))
}
