package binance

import "time"

// Intervals are the candlestick intervals TradeMan offers, written the way the
// kline endpoint expects them and ordered from shortest to longest.
var Intervals = []string{
	"1s", "1m", "3m", "5m", "15m", "30m",
	"1h", "2h", "4h", "6h", "8h", "12h",
	"1d", "3d", "1w", "1M",
}

// intervalSpans is how much time one candle of each interval covers. Weeks and
// months are nominal — a month is counted as 30 days — because the span is only
// ever used to decide how often to ask for fresh candles, never to place one on
// the time axis, where the exchange's own timestamps are used instead.
var intervalSpans = map[string]time.Duration{
	"1s":  time.Second,
	"1m":  time.Minute,
	"3m":  3 * time.Minute,
	"5m":  5 * time.Minute,
	"15m": 15 * time.Minute,
	"30m": 30 * time.Minute,
	"1h":  time.Hour,
	"2h":  2 * time.Hour,
	"4h":  4 * time.Hour,
	"6h":  6 * time.Hour,
	"8h":  8 * time.Hour,
	"12h": 12 * time.Hour,
	"1d":  24 * time.Hour,
	"3d":  3 * 24 * time.Hour,
	"1w":  7 * 24 * time.Hour,
	"1M":  30 * 24 * time.Hour,
}

// IntervalDuration reports how long one candle of the given interval covers.
// The second result is false for anything that is not one of Intervals.
func IntervalDuration(interval string) (time.Duration, bool) {
	span, ok := intervalSpans[interval]
	return span, ok
}
