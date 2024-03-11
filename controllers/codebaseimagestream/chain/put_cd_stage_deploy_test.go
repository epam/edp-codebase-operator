package chain

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestPutCDStageDeploy_ShouldNotFailOnEmptyLables(t *testing.T) {
	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cbis-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: "cb-name",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1"}, cis)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cis).Build()

	chain := PutCDStageDeploy{
		client: fakeCl,
		log:    logr.Discard(),
	}

	err := chain.ServeRequest(cis)
	assert.NoError(t, err)
}

func TestPutCDStageDeploy_ShouldFailWithEmptyCodebase(t *testing.T) {
	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cbis-name",
			Namespace: "stub-namespace",
			Labels: map[string]string{
				"stage-name": "cb-name",
			},
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1"}, cis)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cis).Build()

	chain := PutCDStageDeploy{
		client: fakeCl,
		log:    logr.Discard(),
	}

	err := chain.ServeRequest(cis)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "codebase is not defined in spec") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestPutCDStageDeploy_ShouldFailWithEmptyTags(t *testing.T) {
	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cbis-name",
			Namespace: "stub-namespace",
			Labels: map[string]string{
				"stage-name": "cb-name",
			},
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: "cb-name",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1"}, cis)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cis).Build()

	chain := PutCDStageDeploy{
		client: fakeCl,
		log:    logr.Discard(),
	}

	err := chain.ServeRequest(cis)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "tags are not defined in spec") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestPutCDStageDeploy_ShouldFailWithInvalidLabels(t *testing.T) {
	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cbis-name",
			Namespace: "stub-namespace",
			Labels: map[string]string{
				"stage-name": "cb-name",
			},
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: "cb-name",
			Tags: []codebaseApi.Tag{
				{
					Name:    "master-0.0.1-1",
					Created: "2021-10-20T14:02:31",
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1"}, cis)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cis).Build()

	chain := PutCDStageDeploy{
		client: fakeCl,
		log:    logr.Discard(),
	}

	err := chain.ServeRequest(cis)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "Label must be in format cd-pipeline-name/stage-name") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestPutCDStageDeploy_CdstagedeployShouldFailOnSearch(t *testing.T) {
	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cbis-name",
			Namespace: "stub-namespace",
			Labels: map[string]string{
				"pipeline-name/stage-name": "",
			},
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: "cb-name",
			Tags: []codebaseApi.Tag{
				{
					Name:    "master-0.0.1-1",
					Created: "2021-10-20T14:02:31",
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1"}, cis)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cis).Build()

	chain := PutCDStageDeploy{
		client: fakeCl,
		log:    logr.Discard(),
	}

	err := chain.ServeRequest(cis)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "failed to get pipeline-name-stage-name cd stage deploy") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestPutCDStageDeploy_ShouldPassWhenCdstagedeployExist(t *testing.T) {
	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cbis-name",
			Namespace: "stub-namespace",
			Labels: map[string]string{
				"pipeline-name/stage-name": "",
			},
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: "cb-name",
			Tags: []codebaseApi.Tag{
				{
					Name:    "master-0.0.1-1",
					Created: "2021-10-20T14:02:31",
				},
			},
		},
	}

	cdsd := &codebaseApi.CDStageDeploy{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "pipeline-name-stage-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CDStageDeploySpec{},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1"}, cis, cdsd)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cis, cdsd).Build()

	chain := PutCDStageDeploy{
		client: fakeCl,
		log:    logr.Discard(),
	}

	err := chain.ServeRequest(cis)
	assert.NoError(t, err)

	if err != nil {
		t.Fatalf("unexpected error returned: %s", err.Error())
	}
}

func TestPutCDStageDeploy_ShouldCreateCdstagedeploy(t *testing.T) {
	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cbis-name",
			Namespace: "stub-namespace",
			Labels: map[string]string{
				"pipeline-name/stage-name": "",
			},
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: "cb-name",
			Tags: []codebaseApi.Tag{
				{
					Name:    "master-0.0.1-1",
					Created: "2022-04-12T12:54:04Z",
				},
				{
					Name:    "master-0.0.1-2",
					Created: "2022-04-13T12:54:04Z",
				},
			},
		},
	}

	cdsd := &codebaseApi.CDStageDeploy{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cdsd-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CDStageDeploySpec{},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1"}, cis, cdsd)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cis, cdsd).Build()

	chain := PutCDStageDeploy{
		client: fakeCl,
		log:    logr.Discard(),
	}

	err := chain.ServeRequest(cis)
	assert.NoError(t, err)

	cdsdResp := &codebaseApi.CDStageDeploy{}
	err = fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "pipeline-name-stage-name",
			Namespace: "stub-namespace",
		},
		cdsdResp)
	assert.NoError(t, err)
	assert.Equal(t, cdsdResp.Spec.Tag.Tag, "master-0.0.1-2")
}

func TestPutCDStageDeploy_ShouldFailWithIncorrectTagsTimestamp(t *testing.T) {
	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cbis-name",
			Namespace: "stub-namespace",
			Labels: map[string]string{
				"pipeline-name/stage-name": "",
			},
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: "cb-name",
			Tags: []codebaseApi.Tag{
				{
					Name:    "master-0.0.1-1",
					Created: "2021-10-20",
				},
				{
					Name:    "master-0.0.1-2",
					Created: "14:11:11",
				},
			},
		},
	}

	cdsd := &codebaseApi.CDStageDeploy{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cdsd-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CDStageDeploySpec{},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1"}, cis, cdsd)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cis, cdsd).Build()

	chain := PutCDStageDeploy{
		client: fakeCl,
		log:    logr.Discard(),
	}

	err := chain.ServeRequest(cis)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "failed to construct command to create pipeline-name-stage-name cd stage deploy") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
