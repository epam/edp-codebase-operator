package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1"
	perfApi "github.com/epam/edp-perf-operator/v2/pkg/apis/edp/v1"
)

const (
	fakeName = "fake-name"
)

func TestPutPerfDataSourcesChain_SkipCreatingPerfDataSource(t *testing.T) {
	ctx := context.Background()
	handler := NewPutPerfDataSources(nil)
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "fake-name",
		},
		Spec: codebaseApi.CodebaseSpec{},
	}
	assert.NoError(t, handler.ServeRequest(ctx, c))
}

func TestPutPerfDataSourcesChain_JenkinsAndSonarDataSourcesShouldBeCreated(t *testing.T) {
	ctx := context.Background()
	pdss := &perfApi.PerfDataSourceSonar{}
	pdsj := &perfApi.PerfDataSourceJenkins{}
	pdsg := &perfApi.PerfDataSourceGitLab{}
	ecJenkins := &edpCompApi.EDPComponent{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "jenkins",
			Namespace: fakeNamespace,
		},
		Spec: edpCompApi.EDPComponentSpec{},
	}

	ecSonar := &edpCompApi.EDPComponent{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "sonar",
			Namespace: fakeNamespace,
		},
		Spec: edpCompApi.EDPComponentSpec{},
	}

	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.GitServerSpec{
			GitHost: fakeName,
		},
	}

	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			DefaultBranch: fakeName,
			GitUrlPath:    util.GetStringP("/fake"),
			GitServer:     fakeName,
			Perf: &codebaseApi.Perf{
				Name:        fakeName,
				DataSources: []string{"Jenkins", "Sonar", "GitLab"},
			},
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, gs)
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, pdsj, pdss, pdsg, ecJenkins, ecSonar)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(pdsj, pdss, pdsg, ecJenkins, ecSonar, gs).Build()

	handler := NewPutPerfDataSources(fakeCl)

	err := handler.ServeRequest(ctx, c)

	assert.NoError(t, err)
}

func TestPutPerfDataSourcesChain_ShouldNotFoundEdpComponent(t *testing.T) {
	ctx := context.Background()
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
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, c)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	handler := NewPutPerfDataSources(fakeCl)

	err := handler.ServeRequest(ctx, c)

	assert.Error(t, err)
}
