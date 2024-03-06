package codebaseimagestream

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestGetLastTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tags    []codebaseApi.Tag
		want    codebaseApi.Tag
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should return latest tag",
			tags: []codebaseApi.Tag{
				{
					Name:    "master-0.0.1-4",
					Created: "2022-04-11T12:00:00Z",
				},
				{
					Name:    "master-0.0.1-6",
					Created: "2022-04-12T12:54:04Z",
				},
			},
			want: codebaseApi.Tag{
				Name:    "master-0.0.1-6",
				Created: "2022-04-12T12:54:04Z",
			},
			wantErr: assert.NoError,
		},
		{
			name: "should skip tag with invalid created time",
			tags: []codebaseApi.Tag{
				{
					Name:    "master-0.0.1-6",
					Created: "2022-04-12T12:54:04Z",
				},
				{
					Name:    "master-0.0.1-7",
					Created: "2022-04-12",
				},
			},
			want: codebaseApi.Tag{
				Name:    "master-0.0.1-6",
				Created: "2022-04-12T12:54:04Z",
			},
			wantErr: assert.NoError,
		},
		{
			name:    "should return error if latest tag is not found",
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := GetLastTag(tt.tags, logr.Discard())
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
