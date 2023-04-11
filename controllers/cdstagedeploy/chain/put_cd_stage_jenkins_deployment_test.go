package chain

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestPutCDStageJenkinsDeployment_ServeRequest_ShouldPass(t *testing.T) {
	j := &jenkinsApi.Jenkins{}
	jl := &jenkinsApi.JenkinsList{
		Items: []jenkinsApi.Jenkins{
			{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "jenkins",
					Namespace: "namespace",
				},
			},
		},
	}
	jcdsd := &jenkinsApi.CDStageJenkinsDeployment{}
	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "pipeline-stage",
			Namespace: "namespace",
		},
	}
	cdsd := &codebaseApi.CDStageDeploy{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CDStageDeploySpec{
			Pipeline: "pipeline",
			Stage:    "stage",
			Tag: codebaseApi.CodebaseTag{
				Codebase: "codebase",
				Tag:      "tag",
			},
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(metaV1.SchemeGroupVersion, jl, j)
	scheme.AddKnownTypes(codebaseApi.GroupVersion, cdsd)
	scheme.AddKnownTypes(cdPipeApi.GroupVersion, s)
	scheme.AddKnownTypes(jenkinsApi.SchemeGroupVersion, jcdsd)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdsd, s, jcdsd, j, jl).Build()

	jd := PutCDStageJenkinsDeployment{
		client: fakeCl,
		log:    logr.Discard(),
	}

	err := jd.ServeRequest(context.Background(), cdsd)
	assert.NoError(t, err)
}

func TestPutCDStageJenkinsDeployment_ServeRequest_ShouldFailWithExistingCR(t *testing.T) {
	cdsd := &codebaseApi.CDStageDeploy{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CDStageDeploySpec{
			Pipeline: "pipeline",
			Stage:    "stage",
			Tag: codebaseApi.CodebaseTag{
				Codebase: "codebase",
				Tag:      "tag",
			},
		},
	}
	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "pipeline-stage",
			Namespace: "namespace",
		},
	}
	jcdsd := &jenkinsApi.CDStageJenkinsDeployment{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(cdPipeApi.GroupVersion, s)
	scheme.AddKnownTypes(codebaseApi.GroupVersion, cdsd)
	scheme.AddKnownTypes(jenkinsApi.SchemeGroupVersion, jcdsd)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdsd, jcdsd, s).Build()

	jd := PutCDStageJenkinsDeployment{
		client: fakeCl,
		log:    logr.Discard(),
	}

	err := jd.ServeRequest(context.Background(), cdsd)
	assert.ErrorIs(t, err, ErrCDStageJenkinsDeploymentHasNotBeenProcessed)
}

func TestPutCDStageJenkinsDeployment_ServeRequest_ShouldFailGenerateLabels(t *testing.T) {
	j := &jenkinsApi.Jenkins{}
	jl := &jenkinsApi.JenkinsList{}
	jcdsd := &jenkinsApi.CDStageJenkinsDeployment{}
	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "pipeline-stage",
			Namespace: "namespace",
		},
	}
	cdsd := &codebaseApi.CDStageDeploy{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
		Spec: codebaseApi.CDStageDeploySpec{
			Pipeline: "pipeline",
			Stage:    "stage",
			Tag: codebaseApi.CodebaseTag{
				Codebase: "codebase",
				Tag:      "tag",
			},
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(metaV1.SchemeGroupVersion, jl, j)
	scheme.AddKnownTypes(codebaseApi.GroupVersion, cdsd)
	scheme.AddKnownTypes(cdPipeApi.GroupVersion, s)
	scheme.AddKnownTypes(jenkinsApi.SchemeGroupVersion, jcdsd)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdsd, s, jcdsd, j, jl).Build()

	jd := PutCDStageJenkinsDeployment{
		client: fakeCl,
		log:    logr.Discard(),
	}

	err := jd.ServeRequest(context.Background(), cdsd)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "failed to generate labels") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestPutCDStageJenkinsDeployment_ServeRequest_ShouldFailGetJenkinsDeployment(t *testing.T) {
	cdsd := &codebaseApi.CDStageDeploy{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewCDStageDeploy",
			Namespace: "namespace",
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, cdsd)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdsd).Build()

	jd := PutCDStageJenkinsDeployment{
		client: fakeCl,
		log:    logr.Discard(),
	}

	err := jd.ServeRequest(context.Background(), cdsd)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "failed to get NewCDStageDeploy cd stage jenkins deployment") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
