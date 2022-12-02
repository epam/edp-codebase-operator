package chain

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPostponeError_Error(t *testing.T) {
	t.Parallel()

	type fields struct {
		Timeout time.Duration
		Message string
	}

	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "should return timeout",
			fields: fields{
				Timeout: time.Second,
				Message: "",
			},
			want: fmt.Sprintf("postpone for: %s", time.Second.String()),
		},
		{
			name: "should return error message",
			fields: fields{
				Timeout: 0,
				Message: "error message",
			},
			want: "error message",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := PostponeError{
				Timeout: tt.fields.Timeout,
				Message: tt.fields.Message,
			}

			got := p.Error()

			assert.Equal(t, tt.want, got)
		})
	}
}
