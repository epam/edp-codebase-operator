package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/imagestreamtag/chain/handler"
)

func TestDeleteTagCodebaseImageStreamCr_ServeRequest(t *testing.T) {
	t.Parallel()

	type fields struct {
		next   handler.ImageStreamTagHandler
		client client.Client
	}

	type args struct {
		ist *codebaseApi.ImageStreamTag
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should serve request successfully",
			fields: fields{
				next:   nil,
				client: fake.NewClientBuilder().Build(),
			},
			args: args{
				ist: &codebaseApi.ImageStreamTag{
					TypeMeta: metaV1.TypeMeta{
						Kind:       "test-kind",
						APIVersion: "v.test",
					},
					ObjectMeta: metaV1.ObjectMeta{
						Name: "Test Image Stream Tag",
					},
					Spec:   codebaseApi.ImageStreamTagSpec{},
					Status: codebaseApi.ImageStreamTagStatus{},
				},
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := codebaseApi.AddToScheme(tt.fields.client.Scheme())
			assert.NoError(t, err)

			ctx := context.Background()

			err = tt.fields.client.Create(ctx, tt.args.ist)
			assert.NoError(t, err)

			h := DeleteTagCodebaseImageStreamCr{
				next:   tt.fields.next,
				client: tt.fields.client,
			}

			tt.wantErr(t, h.ServeRequest(tt.args.ist))
		})
	}
}
