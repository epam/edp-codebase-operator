package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFieldsMap(t *testing.T) {
	t.Parallel()

	type args struct {
		payload      string
		keysToDelete []string
	}

	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "should successfully delete keys",
			args: args{
				payload:      `{"one": 1, "two": 2, "three": 3}`,
				keysToDelete: []string{"one", "three"},
			},
			want:    map[string]interface{}{"two": 2.0},
			wantErr: require.NoError,
		},
		{
			name: "should succeed with no keys to delete",
			args: args{
				payload:      `{"one": 1, "two": 2, "three": 3}`,
				keysToDelete: []string{"five"},
			},
			want:    map[string]interface{}{"one": 1.0, "two": 2.0, "three": 3.0},
			wantErr: require.NoError,
		},
		{
			name: "should fail because of the invalid json",
			args: args{
				payload:      "invalid json",
				keysToDelete: []string{},
			},
			want:    map[string]interface{}(nil),
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := GetFieldsMap(tt.args.payload, tt.args.keysToDelete)

			tt.wantErr(t, err)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCheckElementInArray(t *testing.T) {
	t.Parallel()

	type args struct {
		array   []string
		element string
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "should contain the element",
			args: args{
				array:   []string{"one", "two", "three"},
				element: "two",
			},
			want: true,
		},
		{
			name: "should not contain the element",
			args: args{
				array:   []string{"one", "two"},
				element: "three",
			},
			want: false,
		},
		{
			name: "should return false because the array is empty",
			args: args{
				array:   []string{},
				element: "element",
			},
			want: false,
		},
		{
			name: "should return false because the array is nil",
			args: args{
				array:   nil,
				element: "element",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := CheckElementInArray(tt.args.array, tt.args.element)

			assert.Equal(t, tt.want, got)
		})
	}
}
