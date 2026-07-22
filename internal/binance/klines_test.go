package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"
)

// oneKline is a candle as the exchange sends it, with every trailing field the
// documentation lists.
const oneKline = `[1784742240000,"66175.99000000","66179.17000000","66175.99000000","66176.00000000",` +
	`"1.11744000",1784742299999,"73950.65455970",600,"0.25894000","17136.18697600","0"]`

func TestCandleReadsTheExchangeEncoding(t *testing.T) {
	var candle Candle
	if err := json.Unmarshal([]byte(oneKline), &candle); err != nil {
		t.Fatalf("reading a candle: %v", err)
	}

	want := Candle{
		OpenTime:  time.UnixMilli(1784742240000),
		Open:      66175.99,
		High:      66179.17,
		Low:       66175.99,
		Close:     66176.00,
		Volume:    1.11744,
		CloseTime: time.UnixMilli(1784742299999),
	}
	if !candle.OpenTime.Equal(want.OpenTime) || !candle.CloseTime.Equal(want.CloseTime) {
		t.Errorf("candle spans %v–%v, want %v–%v",
			candle.OpenTime, candle.CloseTime, want.OpenTime, want.CloseTime)
	}
	if candle.Open != want.Open || candle.High != want.High ||
		candle.Low != want.Low || candle.Close != want.Close || candle.Volume != want.Volume {
		t.Errorf("candle prices are %+v, want %+v", candle, want)
	}
}

func TestCandleRejectsMalformedEncodings(t *testing.T) {
	cases := map[string]string{
		"an object":           `{"open":1}`,
		"a truncated array":   `[1784742240000,"1.0","2.0","0.5"]`,
		"an unquoted price":   `[1784742240000,1.0,"2.0","0.5","1.5","1.0",1784742299999]`,
		"a bad timestamp":     `["soon","1.0","2.0","0.5","1.5","1.0",1784742299999]`,
		"an unparsable price": `[1784742240000,"cheap","2.0","0.5","1.5","1.0",1784742299999]`,
	}

	for name, encoded := range cases {
		t.Run(name, func(t *testing.T) {
			var candle Candle
			if err := json.Unmarshal([]byte(encoded), &candle); err == nil {
				t.Errorf("%s was accepted as a candle: %+v", name, candle)
			}
		})
	}
}

func TestCandleRisingFollowsTheClose(t *testing.T) {
	cases := []struct {
		name        string
		open, close float64
		want        bool
	}{
		{"a gain", 100, 110, true},
		{"a loss", 110, 100, false},
		{"no change", 100, 100, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			candle := Candle{Open: c.open, Close: c.close}
			if got := candle.Rising(); got != c.want {
				t.Errorf("%v–%v rising is %v, want %v", c.open, c.close, got, c.want)
			}
		})
	}
}

func TestKlinesAsksForTheSelectedChart(t *testing.T) {
	var asked url.Values
	client := serve(t, func(w http.ResponseWriter, r *http.Request) {
		asked = r.URL.Query()
		if r.URL.Path != "/api/v3/klines" {
			t.Errorf("candles were requested from %q", r.URL.Path)
		}
		fmt.Fprintf(w, "[%s]", oneKline)
	})

	candles, err := client.Klines(context.Background(), "BTCUSDT", "15m", 300)
	if err != nil {
		t.Fatalf("reading candles: %v", err)
	}

	if len(candles) != 1 {
		t.Errorf("got %d candles, want 1", len(candles))
	}
	for key, want := range map[string]string{"symbol": "BTCUSDT", "interval": "15m", "limit": "300"} {
		if got := asked.Get(key); got != want {
			t.Errorf("request asked for %s=%q, want %q", key, got, want)
		}
	}
}

// The dropdown never offers a count outside the exchange's range, but a limit
// out of range must still produce a chart rather than an error.
func TestKlinesKeepsTheLimitInRange(t *testing.T) {
	cases := map[int]string{0: "1", -5: "1", 100: "100", 5000: "1000"}

	for asked, want := range cases {
		t.Run(want, func(t *testing.T) {
			var sent url.Values
			client := serve(t, func(w http.ResponseWriter, r *http.Request) {
				sent = r.URL.Query()
				fmt.Fprintf(w, "[%s]", oneKline)
			})

			if _, err := client.Klines(context.Background(), "BTCUSDT", "1h", asked); err != nil {
				t.Fatalf("reading candles: %v", err)
			}
			if got := sent.Get("limit"); got != want {
				t.Errorf("asking for %d candles sent limit=%q, want %q", asked, got, want)
			}
		})
	}
}

// A symbol that exists but has never traded answers with an empty array, which
// would otherwise reach the chart as a plot of nothing.
func TestKlinesRejectsAnEmptyChart(t *testing.T) {
	client := serve(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "[]")
	})

	if _, err := client.Klines(context.Background(), "BTCUSDT", "1h", 100); err == nil {
		t.Error("an empty reply was accepted as a chart")
	}
}

func TestCandleCountsAreOffered(t *testing.T) {
	want := []int{100, 200, 300, 500}

	if len(CandleCounts) != len(want) {
		t.Fatalf("candle counts are %v, want %v", CandleCounts, want)
	}
	for i, count := range want {
		if CandleCounts[i] != count {
			t.Fatalf("candle counts are %v, want %v", CandleCounts, want)
		}
		if count > maxCandles {
			t.Errorf("%d candles is more than the endpoint serves (%d)", count, maxCandles)
		}
	}
}
