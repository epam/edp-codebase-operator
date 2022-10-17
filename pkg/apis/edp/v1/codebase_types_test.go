package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodebaseSpec_GetProjectID(t *testing.T) {
	tests := []struct {
		name       string
		gitUrlPath func() *string
		want       string
	}{
		{
			name: "should remove prefix from git url",
			gitUrlPath: func() *string {
				p := "/group/proj1"

				return &p
			},
			want: "group/proj1",
		},
		{
			name: "should skip prefix removal if git url doesn't contain prefix",
			gitUrlPath: func() *string {
				p := "group/proj1"

				return &p
			},
			want: "group/proj1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CodebaseSpec{GitUrlPath: tt.gitUrlPath()}
			assert.Equal(t, tt.want, c.GetProjectID())
		})
	}
}
