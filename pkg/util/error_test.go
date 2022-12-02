package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCodebaseBranchReconcileError(t *testing.T) {
	t.Parallel()

	type args struct {
		msg string
	}

	tests := []struct {
		name string
		args args
		want *CodebaseBranchReconcileError
	}{
		{
			name: "should create error with a given message",
			args: args{
				msg: "given error message",
			},
			want: &CodebaseBranchReconcileError{
				Message: "given error message",
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewCodebaseBranchReconcileError(tt.args.msg)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCodebaseBranchReconcileError_Error(t *testing.T) {
	t.Parallel()

	type fields struct {
		Message string
	}

	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "should return given error message",
			fields: fields{
				Message: "given error message",
			},
			want: "given error message",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := &CodebaseBranchReconcileError{
				Message: tt.fields.Message,
			}

			got := e.Error()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCDStageDeployHasNotBeenProcessedError_Error(t *testing.T) {
	t.Parallel()

	type fields struct {
		Message string
	}

	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "should return given error message",
			fields: fields{
				Message: "given error message",
			},
			want: "given error message",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := &CDStageDeployHasNotBeenProcessedError{
				Message: tt.fields.Message,
			}

			assert.Equalf(t, tt.want, e.Error(), "Error()")
		})
	}
}

func TestCDStageJenkinsDeploymentHasNotBeenProcessedError_Error(t *testing.T) {
	t.Parallel()

	type fields struct {
		Message string
	}

	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "should return given error message",
			fields: fields{
				Message: "given error message",
			},
			want: "given error message",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := &CDStageJenkinsDeploymentHasNotBeenProcessedError{
				Message: tt.fields.Message,
			}

			got := e.Error()

			assert.Equal(t, tt.want, got)
		})
	}
}
