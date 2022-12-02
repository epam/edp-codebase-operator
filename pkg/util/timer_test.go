package util

import (
	"testing"

	"github.com/go-logr/logr"
)

func TestTimer(t *testing.T) {
	t.Parallel()

	type args struct {
		name string
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "should return func",
			args: args{
				name: "test timer",
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			log := logr.Discard()

			got := Timer(tt.args.name, log)

			got()
		})
	}
}
