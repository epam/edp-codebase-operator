package util

import (
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"go.uber.org/multierr"

	commonmock "github.com/epam/edp-common/pkg/mock"

	ioMock "github.com/epam/edp-codebase-operator/v2/mocks/io"
)

func TestCloseWithErrorCapture(t *testing.T) {
	t.Parallel()

	type args struct {
		err    error
		closer io.Closer
		format string
		a      []any
	}

	tests := []struct {
		name string
		args args
		want error
	}{
		{
			name: "should not modify given error",
			args: args{
				err:    errors.New("new given error"),
				closer: ioMock.NewMockCloser(nil),
				format: "test-format %d %s %v",
				a:      []any{1, "two", []rune("three")},
			},
			want: errors.New("new given error"),
		},
		{
			name: "should wrap given error with returned io.Closer error and provided arguments",
			args: args{
				err:    errors.New("new given error"),
				closer: ioMock.NewMockCloser(errors.New("new io.Closer error")),
				format: "test-format %d %s %v",
				a:      []any{1, "two", []rune("three")},
			},
			want: multierr.Append(
				errors.New("new given error"),
				fmt.Errorf(
					"test-format %d %s %v: %w",
					1,
					"two",
					[]rune("three"),
					errors.New("new io.Closer error"),
				),
			),
		},
		{
			name: "should wrap given error with returned io.Closer",
			args: args{
				err:    errors.New("new given error"),
				closer: ioMock.NewMockCloser(errors.New("new io.Closer error")),
				format: "test-format",
				a:      nil,
			},
			want: multierr.Append(
				errors.New("new given error"),
				fmt.Errorf("test-format: %w", errors.New("new io.Closer error")),
			),
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			CloseWithErrorCapture(&tt.args.err, tt.args.closer, tt.args.format, tt.args.a...)

			assert.Equal(t, tt.want, tt.args.err)
		})
	}
}

func TestCloseWithLogOnErr(t *testing.T) {
	t.Parallel()

	type args struct {
		logger logr.Logger
		closer io.Closer
		format string
		a      []any
	}

	tests := []struct {
		name    string
		args    args
		wantLog []any
	}{
		{
			name: "should return on nil closer error",
			args: args{
				logger: commonmock.NewLogr(),
				closer: ioMock.NewMockCloser(nil),
				format: "",
				a:      nil,
			},
			wantLog: nil,
		},
		{
			name: "should return on os.ErrClosed closer error",
			args: args{
				logger: commonmock.NewLogr(),
				closer: ioMock.NewMockCloser(os.ErrClosed),
				format: "",
				a:      nil,
			},
			wantLog: nil,
		},
		{
			name: "should return wrapped error",
			args: args{
				logger: commonmock.NewLogr(),
				closer: ioMock.NewMockCloser(errors.New("closer error")),
				format: "error format %d %s %v",
				a:      []any{1, "two", []rune("three")},
			},
			wantLog: []any{
				"error",
				fmt.Errorf(
					"error format %d %s %v: %w",
					1,
					"two",
					[]rune("three"),
					errors.New("closer error"),
				),
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			CloseWithLogOnErr(tt.args.logger, tt.args.closer, tt.args.format, tt.args.a...)

			loggerSink, ok := tt.args.logger.GetSink().(*commonmock.Logger)
			assert.True(t, ok)

			assert.Equal(t, loggerSink.InfoMessages()["detected close error"], tt.wantLog)
		})
	}
}
