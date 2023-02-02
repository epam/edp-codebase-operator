package util

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/go-logr/logr"
	"go.uber.org/multierr"
)

//nolint:gocritic // This linter fails for ptrToRefParam check, we need to use a pointer for an interface, because we change error
func CloseWithErrorCapture(err *error, closer io.Closer, format string, a ...any) {
	if closeErr := closer.Close(); closeErr != nil {
		*err = multierr.Append(*err, fmt.Errorf(format+": %w", append(a, closeErr)...))
	}
}

func CloseWithLogOnErr(logger logr.Logger, closer io.Closer, format string, a ...any) {
	err := closer.Close()
	if err == nil {
		return
	}

	// Not a problem if it has been closed already.
	if errors.Is(err, os.ErrClosed) {
		return
	}

	logger.Info("detected close error", "error", fmt.Errorf(format+": %w", append(a, err)...))
}
