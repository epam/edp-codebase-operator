package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodebaseSpec_GetProjectID(t *testing.T) {
	t.Parallel()

	type fields struct {
		GitUrlPath func() *string
	}

	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "should remove prefix from the git url",
			fields: fields{
				GitUrlPath: func() *string {
					p := "/group/proj1"

					return &p
				},
			},
			want: "group/proj1",
		},
		{
			name: "should skip prefix removal if git url doesn't contain prefix",
			fields: fields{
				GitUrlPath: func() *string {
					p := "group/proj1"

					return &p
				},
			},
			want: "group/proj1",
		},
		{
			name: "should return empty string if GitUrlPath returns nil",
			fields: fields{
				GitUrlPath: func() *string {
					return nil
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			in := &CodebaseSpec{
				GitUrlPath: tt.fields.GitUrlPath(),
			}

			got := in.GetProjectID()

			assert.Equal(t, tt.want, got)
		})
	}
}
