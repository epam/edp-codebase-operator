package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodebaseSpec_GetProjectID(t *testing.T) {
	t.Parallel()

	type fields struct {
		GitUrlPath string
	}

	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "should remove prefix from the git url",
			fields: fields{
				GitUrlPath: "/group/proj1",
			},
			want: "group/proj1",
		},
		{
			name: "should skip prefix removal if git url doesn't contain prefix",
			fields: fields{
				GitUrlPath: "group/proj1",
			},
			want: "group/proj1",
		},
		{
			name: "should return empty string if GitUrlPath empty",
			fields: fields{
				GitUrlPath: "",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			in := &CodebaseSpec{
				GitUrlPath: tt.fields.GitUrlPath,
			}

			got := in.GetProjectID()

			assert.Equal(t, tt.want, got)
		})
	}
}
