package chain

import (
	"fmt"
	"time"
)

type PostponeError struct {
	Timeout time.Duration
	Message string
}

func (p PostponeError) Error() string {
	if p.Message != "" {
		return p.Message
	}

	return fmt.Sprintf("postpone for: %s", p.Timeout.String())
}
