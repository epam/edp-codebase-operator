package util

import (
	"testing"
	"time"
)

func TestGetTimeout(t *testing.T) {
	tm := GetTimeout(0, 10*time.Second)
	if tm != 10*time.Second {
		t.Fatalf("wring timeout returned: %d", tm)
	}

	tm = GetTimeout(15, 10*time.Second)
	if tm != 88861105205078672 {
		t.Fatalf("wring timeout returned: %d", tm)
	}
}
