package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestIsCodebaseValid(t *testing.T) {
	t.Parallel()

	type args struct {
		cr *codebaseApi.Codebase
	}

	tests := []struct {
		name string
		args args
		want assert.ErrorAssertionFunc
	}{
		{
			name: "should be valid",
			args: args{
				cr: &codebaseApi.Codebase{
					TypeMeta:   metaV1.TypeMeta{},
					ObjectMeta: metaV1.ObjectMeta{},
					Spec: codebaseApi.CodebaseSpec{
						Lang:     "go",
						Strategy: "create",
					},
					Status: codebaseApi.CodebaseStatus{},
				},
			},
			want: assert.NoError,
		},
		{
			name: "should fail on strategy",
			args: args{
				cr: &codebaseApi.Codebase{
					TypeMeta:   metaV1.TypeMeta{},
					ObjectMeta: metaV1.ObjectMeta{},
					Spec: codebaseApi.CodebaseSpec{
						Lang:     "go",
						Strategy: "test-strategy",
					},
					Status: codebaseApi.CodebaseStatus{},
				},
			},
			want: assert.Error,
		},
		{
			name: "should fail on language",
			args: args{
				cr: &codebaseApi.Codebase{
					TypeMeta:   metaV1.TypeMeta{},
					ObjectMeta: metaV1.ObjectMeta{},
					Spec: codebaseApi.CodebaseSpec{
						Lang:     "test-lang",
						Strategy: "create",
					},
					Status: codebaseApi.CodebaseStatus{},
				},
			},
			want: assert.Error,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := IsCodebaseValid(tt.args.cr)

			tt.want(t, err)
		})
	}
}
