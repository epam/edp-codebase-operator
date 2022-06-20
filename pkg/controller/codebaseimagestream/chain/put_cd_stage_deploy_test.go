package chain

import (
	"context"
	"strings"
	"testing"

	k8sMockClient "github.com/epam/edp-common/pkg/mock/controller-runtime/client"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
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
		log:    logr.DiscardLogger{},
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
		log:    logr.DiscardLogger{},
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
		log:    logr.DiscardLogger{},
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
		log:    logr.DiscardLogger{},
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
				"pipeline-name/stage-name": "cb-name",
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
		log:    logr.DiscardLogger{},
	}

	err := chain.ServeRequest(cis)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "couldn't get pipeline-name-stage-name-cb-name cd stage deploy") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

}

func TestPutCDStageDeploy_ShouldFailWhenCdstagedeployExist(t *testing.T) {
	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cbis-name",
			Namespace: "stub-namespace",
			Labels: map[string]string{
				"pipeline-name/stage-name": "cb-name",
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
			Name:      "pipeline-name-stage-name-cb-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CDStageDeploySpec{},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1"}, cis, cdsd)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cis, cdsd).Build()

	chain := PutCDStageDeploy{
		client: fakeCl,
		log:    logr.DiscardLogger{},
	}

	err := chain.ServeRequest(cis)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "stage-name-cb-name has not been processed for previous version of application yet") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestPutCDStageDeploy_ShouldCreateCdstagedeploy(t *testing.T) {
	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cbis-name",
			Namespace: "stub-namespace",
			Labels: map[string]string{
				"pipeline-name/stage-name": "cb-name",
			},
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: "cb-name",
			Tags: []codebaseApi.Tag{
				{
					Name:    "master-0.0.1-1",
					Created: "2021-10-20T14:00:00",
				},
				{
					Name:    "master-0.0.1-2",
					Created: "2021-10-20T14:11:11",
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
		log:    logr.DiscardLogger{},
	}

	err := chain.ServeRequest(cis)
	assert.NoError(t, err)

	cdsdResp := &codebaseApi.CDStageDeploy{}
	err = fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "pipeline-name-stage-name-cb-name",
			Namespace: "stub-namespace",
		},
		cdsdResp)
	assert.NoError(t, err)
	assert.Equal(t, cdsdResp.Spec.Tag.Tag, "master-0.0.1-2")
}

func TestPutCDStageDeploy_ShouldFailWithIncorrectTagstimestamp(t *testing.T) {
	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cbis-name",
			Namespace: "stub-namespace",
			Labels: map[string]string{
				"pipeline-name/stage-name": "cb-name",
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
		log:    logr.DiscardLogger{},
	}

	err := chain.ServeRequest(cis)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "couldn't construct command to create pipeline-name-stage-name-cb-name cd stage deploy") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestPutCDStageDeploy_ShouldGetLastTag(t *testing.T) {
	cis := []codebaseApi.Tag{
		{
			Name:    "master-0.0.1-1",
			Created: "2021-10-20T11:11:11",
		},
		{
			Name:    "master-0.0.1-2",
			Created: "2021-10-20T10:10:10",
		},
	}

	cb, err := getLastTag(cis)
	assert.Nil(t, err)
	assert.Equal(t, cb.Name, "master-0.0.1-1")
}

func TestPutCDStageDeploy_ShouldSkipTagsWithWrongDatetimeFormat(t *testing.T) {
	cis := []codebaseApi.Tag{
		{
			Name:    "master-0.0.1-1",
			Created: "2021-10-19T15:15:15",
		},
		{
			Name:    "master-0.0.1-2",
			Created: "2022-10-21",
		},
		{
			Name:    "master-0.0.1-3",
			Created: "2021-10-20T15:15:15",
		},
		{
			Name:    "master-0.0.1-4",
			Created: "2021-10-20T11:11:11",
		},
		{
			Name:    "master-0.0.1-5",
			Created: "14:11:11",
		},
		{
			Name:    "master-0.0.1-6",
			Created: "2021-10-20T12:12:12",
		},
	}

	cb, err := getLastTag(cis)
	assert.Nil(t, err)
	assert.Equal(t, cb.Name, "master-0.0.1-3")
}

func TestPutCDStageDeploy_ShouldFailWithInvalidTags(t *testing.T) {
	cis := []codebaseApi.Tag{
		{
			Name:    "master-0.0.1-1",
			Created: "2021-10-20",
		},
		{
			Name:    "master-0.0.1-2",
			Created: "10:10:10",
		},
	}

	_, err := getLastTag(cis)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "There are no valid tags") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func Test_generateCdStageDeployName(t *testing.T) {
	type args struct {
		env      string
		codebase string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"1", args{"env", "codebase"}, "env-codebase"},
		{"2", args{"env/2", "codebase"}, "env-2-codebase"},
		{"3", args{"env/1/1-2", "codebase"}, "env-1-1-2-codebase"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateCdStageDeployName(tt.args.env, tt.args.codebase); got != tt.want {
				t.Errorf("generateCdStageDeployName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPutCDStageDeploy_ShouldFailToCreateCdstagedeploy(t *testing.T) {
	mc := k8sMockClient.Client{}

	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cbis-name",
			Namespace: "stub-namespace",
			Labels: map[string]string{
				"pipeline-name/stage-name": "cb-name",
			},
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: "cb-name",
			Tags: []codebaseApi.Tag{
				{
					Name:    "master-0.0.1-1",
					Created: "2021-10-20T14:00:00",
				},
				{
					Name:    "master-0.0.1-2",
					Created: "2021-10-20T14:11:11",
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

	existingCdsd := &codebaseApi.CDStageDeploy{
		TypeMeta: metaV1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1",
			Kind:       "CDStageDeploy",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "pipeline-name-stage-name-cb-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CDStageDeploySpec{
			Pipeline: "pipeline-name",
			Stage:    "stage-name",
			Tag: jenkinsApi.Tag{
				Codebase: "cb-name",
				Tag:      "master-0.0.1-2",
			},
			Tags: []jenkinsApi.Tag{
				{
					Codebase: "cb-name",
					Tag:      "master-0.0.1-2",
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1"}, cis, cdsd)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cis, cdsd).Build()

	mockErr := errors.New("FATAL")

	mc.On("Get", types.NamespacedName{
		Namespace: "stub-namespace",
		Name:      "pipeline-name-stage-name-cb-name",
	}, &codebaseApi.CDStageDeploy{}).Return(fakeCl)

	var createOpts []client.CreateOption
	mc.On("Create", existingCdsd, createOpts).Return(mockErr)

	chain := PutCDStageDeploy{
		client: &mc,
		log:    logr.DiscardLogger{},
	}

	err := chain.ServeRequest(cis)
	assert.Error(t, err)

	if errors.Cause(err) != mockErr {
		t.Fatal("wrong error returned")
	}
}
