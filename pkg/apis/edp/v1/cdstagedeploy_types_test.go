package v1

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCDStageDeploy_SetFailedStatus(t *testing.T) {
	t.Parallel()

	type fields struct {
		in CDStageDeploy
	}

	type args struct {
		err error
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   CDStageDeployStatus
	}{
		{
			name: "should change status to failed",
			fields: fields{
				in: CDStageDeploy{
					TypeMeta:   v1.TypeMeta{},
					ObjectMeta: v1.ObjectMeta{},
					Spec:       CDStageDeploySpec{},
					Status:     CDStageDeployStatus{},
				},
			},
			args: args{
				err: errors.New("error message"),
			},
			want: CDStageDeployStatus{
				Status:  "failed",
				Message: errors.New("error message").Error(),
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.fields.in.SetFailedStatus(tt.args.err)

			assert.Equal(t, tt.want, tt.fields.in.Status)
		})
	}
}
