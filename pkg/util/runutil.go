package util

import (
	"io"
	"os"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

//nolint:gocritic // This linter fails for ptrToRefParam check, we need to use a pointer for an interface, because we change error
func CloseWithErrorCapture(err *error, closer io.Closer, format string, a ...interface{}) {
	*err = multierr.Append(*err, errors.Wrapf(closer.Close(), format, a...))
}

func CloseWithLogOnErr(logger logr.Logger, closer io.Closer, format string, a ...interface{}) {
	err := closer.Close()
	if err == nil {
		return
	}

	// Not a problem if it has been closed already.
	if errors.Is(err, os.ErrClosed) {
		return
	}

	if logger == nil {
		logger = zap.New()
	}

	logger.Info("detected close error", "error", errors.Wrapf(err, format, a...))
}
