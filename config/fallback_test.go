package config

import (
	"math"
	"testing"
	"time"
)

func TestDwellSaturatesInsteadOfWrapping(t *testing.T) {
	if got := Dwell(0); got != DefaultFallbackDwell {
		t.Fatalf("default dwell = %v, want %v", got, DefaultFallbackDwell)
	}
	if got := Dwell(5); got != 5*time.Second {
		t.Fatalf("five-second dwell = %v", got)
	}
	if got := Dwell(math.MaxInt); got <= 0 {
		t.Fatalf("large dwell wrapped to %v", got)
	}
}
