package gitserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_publicKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "success",
			key:     testKey,
			wantErr: assert.NoError,
		},
		{
			name:    "success",
			key:     "invalid-key",
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := publicKey(tt.key)
			tt.wantErr(t, err)
		})
	}
}
