package util

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"
)

func Timer(name string, log logr.Logger) func() {
	start := time.Now()
	return func() {
		log.Info(fmt.Sprintf("%s took %v", name, time.Since(start)))
	}
}
