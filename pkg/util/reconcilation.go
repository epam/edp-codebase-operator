package util

import (
	"math"
	"time"
)

func GetTimeout(factor int64, baseDuration time.Duration) time.Duration {
	if factor < 10 {
		return time.Duration(factor+1) * baseDuration
	}

	t := float64(baseDuration) * math.Pow(math.E, float64(factor+1))
	return time.Duration(t)
}
