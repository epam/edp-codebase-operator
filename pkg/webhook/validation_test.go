package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestIsCodebaseValid(t *testing.T) {
	type args struct {
		cr *codebaseApi.Codebase
	}

	tests := []struct {
		name string
		args args
		want require.ErrorAssertionFunc
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
						Versioning: codebaseApi.Versioning{
							Type: codebaseApi.VersioningTypDefault,
						},
					},
					Status: codebaseApi.CodebaseStatus{},
				},
			},
			want: require.NoError,
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
			want: func(t require.TestingT, err error, i ...any) {
				require.ErrorContains(t, err, "provided unsupported repository strategy: test-strategy")
			},
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
			want: func(t require.TestingT, err error, i ...any) {
				require.ErrorContains(t, err, "provided unsupported language: test-lang")
			},
		},
		{
			name: "should fail on versioning",
			args: args{
				cr: &codebaseApi.Codebase{
					TypeMeta:   metaV1.TypeMeta{},
					ObjectMeta: metaV1.ObjectMeta{},
					Spec: codebaseApi.CodebaseSpec{
						Lang:     "go",
						Strategy: "create",
						Versioning: codebaseApi.Versioning{
							Type: codebaseApi.VersioningTypeSemver,
						},
					},
					Status: codebaseApi.CodebaseStatus{},
				},
			},
			want: func(t require.TestingT, err error, i ...any) {
				require.ErrorContains(t, err, "versioning start from is required when versioning type is not default")
			},
		},
		{
			name: "should fail on gitUrlPath ending with space",
			args: args{
				cr: &codebaseApi.Codebase{
					TypeMeta:   metaV1.TypeMeta{},
					ObjectMeta: metaV1.ObjectMeta{},
					Spec: codebaseApi.CodebaseSpec{
						Lang:       "go",
						Strategy:   "create",
						GitUrlPath: "/path/to/repo ",
						Versioning: codebaseApi.Versioning{
							Type: codebaseApi.VersioningTypDefault,
						},
					},
					Status: codebaseApi.CodebaseStatus{},
				},
			},
			want: func(t require.TestingT, err error, i ...any) {
				require.ErrorContains(t, err, "gitUrlPath should not end with space")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsCodebaseValid(tt.args.cr)

			tt.want(t, err)
		})
	}
}

func Test_validateCodBaseName(t *testing.T) {
	tests := []struct {
		name        string
		codBaseName string
		wantErr     assert.ErrorAssertionFunc
	}{
		{
			name:        "valid codebase name",
			codBaseName: "test-codebase",
			wantErr:     assert.NoError,
		},
		{
			name:        "invalid codebase name",
			codBaseName: "test--codebase",
			wantErr:     assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.wantErr(t, validateCodBaseName(tt.codBaseName))
		})
	}
}
