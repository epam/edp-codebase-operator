package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
)

func TestIsCodebaseValid(t *testing.T) {
	t.Parallel()

	type args struct {
		cr *codebaseApi.Codebase
	}

	tests := []struct {
		name string
		args args
		want bool
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
			want: true,
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
			want: false,
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
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IsCodebaseValid(tt.args.cr)

			assert.Equal(t, tt.want, got)
		})
	}
}
