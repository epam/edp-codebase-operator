package util

import (
	"math"
	"time"
)

func GetTimeout(factor int64) time.Duration {
	t := float64(500*time.Millisecond) * math.Pow(math.E, float64(factor+1))
	return time.Duration(t)
}
