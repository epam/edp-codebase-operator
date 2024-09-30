package chain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestCreateChain(t *testing.T) {
	t.Parallel()

	cl := fake.NewClientBuilder().Build()

	tests := []struct {
		name        string
		stageDeploy *codebaseApi.CDStageDeploy
		want        CDStageDeployHandler
	}{
		{
			name: "create Auto-stable chain",
			stageDeploy: &codebaseApi.CDStageDeploy{
				Spec: codebaseApi.CDStageDeploySpec{
					TriggerType: pipelineApi.TriggerTypeAutoStable,
				},
			},
		},
		{
			name: "create Auto chain",
			stageDeploy: &codebaseApi.CDStageDeploy{
				Spec: codebaseApi.CDStageDeploySpec{
					TriggerType: pipelineApi.TriggerTypeAutoDeploy,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, CreateChain(cl, tt.stageDeploy))
		})
	}
}
