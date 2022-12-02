package chain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateDefChain(t *testing.T) {
	t.Parallel()

	type args struct {
		c client.Client
	}

	tests := []struct {
		name string
		args args
		want PutCDStageJenkinsDeployment
	}{
		{
			name: "should succeed",
			args: args{
				c: fake.NewClientBuilder().Build(),
			},
			want: PutCDStageJenkinsDeployment{
				log: ctrl.Log.WithName("put-cd-stage-jenkins-deployment-controller"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.want.client = tt.args.c

			got := CreateDefChain(tt.args.c)

			assert.Equal(t, tt.want, got)
		})
	}
}
