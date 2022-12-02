package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsString(t *testing.T) {
	t.Parallel()

	type args struct {
		slice []string
		s     string
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Contains_string",
			args: args{
				slice: []string{"foo", "bar", "buz"},
				s:     "bar",
			},
			want: true,
		},
		{
			name: "No_string",
			args: args{
				slice: []string{"foo", "bar", "buz"},
				s:     "asd",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ContainsString(tt.args.slice, tt.args.s)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRemoveString(t *testing.T) {
	t.Parallel()

	type args struct {
		slice []string
		s     string
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Remove_existing_string",
			args: args{
				slice: []string{"foo", "bar", "buz"},
				s:     "bar",
			},
			want: []string{"foo", "buz"},
		},
		{
			name: "Nothing_to_remove",
			args: args{
				slice: []string{"foo", "bar", "buz"},
				s:     "asd",
			},
			want: []string{"foo", "bar", "buz"},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := RemoveString(tt.args.slice, tt.args.s)

			assert.Equal(t, tt.want, got)
		})
	}
}
