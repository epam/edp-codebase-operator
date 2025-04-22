package util

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/go-logr/logr"
)

func CloseWithLogOnErr(logger logr.Logger, closer io.Closer, format string, a ...any) {
	err := closer.Close()
	if err == nil {
		return
	}

	// Not a problem if it has been closed already.
	if errors.Is(err, os.ErrClosed) {
		return
	}

	logger.Error(err, fmt.Sprintf(format, a...))
}
