package binance

import (
	"testing"
	"time"
)

// The offered intervals are exactly the ones the feature lists, in order.
func TestIntervalsMatchTheFeature(t *testing.T) {
	want := []string{
		"1s", "1m", "3m", "5m", "15m", "30m",
		"1h", "2h", "4h", "6h", "8h", "12h",
		"1d", "3d", "1w", "1M",
	}

	if len(Intervals) != len(want) {
		t.Fatalf("intervals are %v, want %v", Intervals, want)
	}
	for i, interval := range want {
		if Intervals[i] != interval {
			t.Fatalf("intervals are %v, want %v", Intervals, want)
		}
	}
}

// Every offered interval needs a span, since the refresh rate is derived from
// it; and the list has to climb, because that is the order it is shown in.
func TestEveryIntervalHasAnAscendingSpan(t *testing.T) {
	var previous time.Duration

	for _, interval := range Intervals {
		span, ok := IntervalDuration(interval)
		if !ok {
			t.Fatalf("%q is offered but has no span", interval)
		}
		if span <= previous {
			t.Errorf("%q spans %v, which does not follow %v", interval, span, previous)
		}
		previous = span
	}
}

func TestIntervalDurationRejectsUnknownIntervals(t *testing.T) {
	for _, interval := range []string{"", "7m", "1y", "1H"} {
		if span, ok := IntervalDuration(interval); ok {
			t.Errorf("%q reported a span of %v, want it rejected", interval, span)
		}
	}
}
