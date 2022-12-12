package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/gittag/chain/handler"
)

func TestDeleteGitTagCr_ServeRequest(t *testing.T) {
	t.Parallel()

	type fields struct {
		next   handler.GitTagHandler
		client client.Client
	}

	type args struct {
		gt *codebaseApi.GitTag
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should serve request",
			fields: fields{
				next:   nil,
				client: fake.NewClientBuilder().Build(),
			},
			args: args{
				gt: &codebaseApi.GitTag{
					TypeMeta: metaV1.TypeMeta{
						Kind:       "test-kind",
						APIVersion: "v.test",
					},
					ObjectMeta: metaV1.ObjectMeta{
						Name: "git object name",
					},
					Spec:   codebaseApi.GitTagSpec{},
					Status: codebaseApi.GitTagStatus{},
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

			err = tt.fields.client.Create(ctx, tt.args.gt)
			assert.NoError(t, err)

			h := DeleteGitTagCr{
				next:   tt.fields.next,
				client: tt.fields.client,
			}

			gotErr := h.ServeRequest(tt.args.gt)

			tt.wantErr(t, gotErr)
		})
	}
}
