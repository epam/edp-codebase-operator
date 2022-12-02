package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
)

func TestBuildRepoUrl(t *testing.T) {
	t.Parallel()

	type fields struct {
		lang      string
		buildTool string
		specType  string
		framework string
	}

	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "should complete without a database",
			fields: fields{
				lang:      "Java",
				buildTool: "Maven",
				specType:  "application",
				framework: "java11",
			},
			want: "https://github.com/epmd-edp/java-maven-java11.git",
		},
		{
			name: "should complete without a framework",
			fields: fields{
				lang:      "javascript",
				buildTool: "npm",
				specType:  "library",
			},
			want: "https://github.com/epmd-edp/javascript-npm-react.git",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spec := &codebaseApi.CodebaseSpec{
				Lang:      tt.fields.lang,
				BuildTool: tt.fields.buildTool,
				Type:      tt.fields.specType,
				Framework: &tt.fields.framework,
			}

			got := BuildRepoUrl(spec)

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_tryGetRepoUrl(t *testing.T) {
	t.Parallel()

	type args struct {
		spec *codebaseApi.CodebaseSpec
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "should pass",
			args: args{
				spec: &codebaseApi.CodebaseSpec{
					Repository: &codebaseApi.Repository{
						Url: "test",
					},
				},
			},
			want:    "test",
			wantErr: require.NoError,
		},
		{
			name: "should fail",
			args: args{
				spec: &codebaseApi.CodebaseSpec{},
			},
			want:    "",
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tryGetRepoUrl(tt.args.spec)

			tt.wantErr(t, err)

			if err != nil {
				require.Nil(t, got)

				return
			}

			assert.Equal(t, &tt.want, got)
		})
	}
}

func TestGetRepoUrl(t *testing.T) {
	t.Parallel()

	type fields struct {
		strategy   codebaseApi.Strategy
		lang       string
		buildTool  string
		framework  string
		repository *codebaseApi.Repository
	}

	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "should create",
			fields: fields{
				strategy:   codebaseApi.Create,
				lang:       "java11",
				buildTool:  "maven",
				framework:  "java11",
				repository: nil,
			},
			want:    "https://github.com/epmd-edp/java11-maven-java11.git",
			wantErr: require.NoError,
		},
		{
			name: "should clone",
			fields: fields{
				strategy: codebaseApi.Clone,
				repository: &codebaseApi.Repository{
					Url: "link",
				},
			},
			want:    "link",
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			codebase := &codebaseApi.Codebase{
				Spec: codebaseApi.CodebaseSpec{
					Strategy:   tt.fields.strategy,
					Lang:       tt.fields.lang,
					BuildTool:  tt.fields.buildTool,
					Framework:  &tt.fields.framework,
					Repository: tt.fields.repository,
				},
			}

			got, err := GetRepoUrl(codebase)

			tt.wantErr(t, err)

			assert.Equal(t, &tt.want, got)
		})
	}
}
