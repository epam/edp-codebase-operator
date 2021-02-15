package util

import (
	"math"
	"time"
)

func GetTimeout(factor int64, baseDuration time.Duration) time.Duration {
	t := float64(baseDuration) * math.Pow(math.E, float64(factor+1))
	return time.Duration(t)
}
