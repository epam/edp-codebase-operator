package util

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDirectoryEmpty(t *testing.T) {
	tests := []struct {
		name    string
		dirPath func(t *testing.T) string
		want    bool
	}{
		{
			name: "Directory is not empty",
			dirPath: func(t *testing.T) string {
				tmp := t.TempDir()
				_, err := os.Create(path.Join(tmp, "test.txt"))

				require.NoError(t, err)

				return tmp
			},
			want: false,
		},
		{
			name: "Directory is empty",
			dirPath: func(t *testing.T) string {
				return t.TempDir()
			},
			want: true,
		},
		{
			name: "Directory does not exist",
			dirPath: func(t *testing.T) string {
				return "/tmp/does-not-exist"
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsDirectoryEmpty(tt.dirPath(t)))
		})
	}
}
