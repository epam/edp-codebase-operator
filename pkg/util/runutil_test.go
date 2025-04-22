package util

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"

	commonmock "github.com/epam/edp-common/pkg/mock"

	ioMock "github.com/epam/edp-codebase-operator/v2/mocks/io"
)

func TestCloseWithLogOnErr(t *testing.T) {
	t.Parallel()

	type args struct {
		logger logr.Logger
		closer io.Closer
		format string
		a      []any
	}

	tests := []struct {
		name       string
		args       args
		wantLogErr bool
	}{
		{
			name: "should return on nil closer error",
			args: args{
				logger: commonmock.NewLogr(),
				closer: ioMock.NewMockCloser(nil),
				format: "",
				a:      nil,
			},
		},
		{
			name: "should return on os.ErrClosed closer error",
			args: args{
				logger: commonmock.NewLogr(),
				closer: ioMock.NewMockCloser(os.ErrClosed),
				format: "",
				a:      nil,
			},
		},
		{
			name: "should return wrapped error",
			args: args{
				logger: commonmock.NewLogr(),
				closer: ioMock.NewMockCloser(errors.New("closer error")),
				format: "error format %d %s",
				a:      []any{1, "two"},
			},
			wantLogErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			CloseWithLogOnErr(tt.args.logger, tt.args.closer, tt.args.format, tt.args.a...)

			loggerSink, ok := tt.args.logger.GetSink().(*commonmock.Logger)
			assert.True(t, ok)

			assert.Equal(t, tt.wantLogErr, loggerSink.LastError() != nil, "expected error to be logged")
		})
	}
}
