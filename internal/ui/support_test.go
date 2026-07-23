package ui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dmitrykvasnikov/trademan/internal/binance"
)

// testSymbols is the coin list the tests build their dropdowns from.
var testSymbols = []binance.Symbol{
	{Base: "BTC", Quote: "USDT"},
	{Base: "ETH", Quote: "USDT"},
	{Base: "SOL", Quote: "USDT"},
}

// testCatalog returns a catalog already holding symbols, so a tab built on it
// fills its Coin dropdown inline — no network call and no background goroutine
// for the test to race against.
func testCatalog(symbols []binance.Symbol) *coinCatalog {
	c := &coinCatalog{fetch: func() ([]binance.Symbol, error) { return symbols, nil }}
	c.symbols()
	return c
}

// offlineClient points at a port nothing listens on, so a test that reaches for
// candles it did not arrange fails at once instead of calling the exchange.
func offlineClient() *binance.Client {
	return &binance.Client{
		BaseURL: "http://127.0.0.1:1",
		HTTP:    &http.Client{Timeout: 50 * time.Millisecond},
	}
}

// klineServer serves n candles walking up from 100 by one per interval, and a
// client pointed at it.
func klineServer(t *testing.T, n int) *binance.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, klinesJSON(n))
	}))
	t.Cleanup(server.Close)

	return &binance.Client{BaseURL: server.URL, HTTP: server.Client()}
}

// failingServer answers every request the way Binance refuses one, and a client
// pointed at it.
func failingServer(t *testing.T, message string) *binance.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"code":-1121,"msg":%q}`, message)
	}))
	t.Cleanup(server.Close)

	return &binance.Client{BaseURL: server.URL, HTTP: server.Client()}
}

// candleServer serves a fixed, hand-built set of candles for every request — a
// gappy series, say — and a client pointed at it, for the tests that need the
// signal to find something specific after a refresh.
func candleServer(t *testing.T, candles []binance.Candle) *binance.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, candlesJSON(candles))
	}))
	t.Cleanup(server.Close)

	return &binance.Client{BaseURL: server.URL, HTTP: server.Client()}
}

// candlesJSON renders a given series in Binance's mixed-array kline encoding, so
// a server can hand back exactly the candles a test built rather than the plain
// rising run klinesJSON makes.
func candlesJSON(candles []binance.Candle) string {
	var b strings.Builder
	b.WriteByte('[')

	for i, c := range candles {
		if i > 0 {
			b.WriteByte(',')
		}
		open := int64(i) * 60_000
		fmt.Fprintf(&b, `[%d,"%g","%g","%g","%g","1.5",%d,"150.0",3,"0.5","50.0","0"]`,
			open, c.Open, c.High, c.Low, c.Close, open+59_999)
	}

	b.WriteByte(']')
	return b.String()
}

// klinesJSON renders n candles in Binance's mixed-array kline encoding.
func klinesJSON(n int) string {
	var b strings.Builder
	b.WriteByte('[')

	for i := range n {
		if i > 0 {
			b.WriteByte(',')
		}
		open := 100 + i
		fmt.Fprintf(&b, `[%d,"%d.00","%d.00","%d.00","%d.00","1.5",%d,"150.0",3,"0.5","50.0","0"]`,
			int64(i)*60_000, open, open+2, open-2, open+1, int64(i)*60_000+59_999)
	}

	b.WriteByte(']')
	return b.String()
}

// gappyCandles is a candle series holding two separated fair-value gaps: one
// jump clear of the run at candle 3 and another at candle 7. FVG marks each gap
// once — indices 3 and 7 — skipping the body it just used, so the marks do not
// overlap.
func gappyCandles() []binance.Candle {
	return []binance.Candle{
		{Open: 15, High: 20, Low: 10, Close: 15},
		{Open: 16, High: 21, Low: 11, Close: 16},
		{Open: 17, High: 22, Low: 12, Close: 17},
		{Open: 35, High: 40, Low: 30, Close: 35}, // first gap completes here (index 3)
		{Open: 36, High: 41, Low: 31, Close: 36},
		{Open: 37, High: 42, Low: 32, Close: 37},
		{Open: 38, High: 43, Low: 33, Close: 38},
		{Open: 65, High: 70, Low: 60, Close: 65}, // second gap completes here (index 7)
		{Open: 66, High: 71, Low: 61, Close: 66},
	}
}

// candles builds n candles directly, for the drawing tests that do not need a
// server behind them.
func candles(n int) []binance.Candle {
	built := make([]binance.Candle, 0, n)
	for i := range n {
		price := float64(100 + i)
		built = append(built, binance.Candle{
			OpenTime:  time.Unix(int64(i)*60, 0),
			Open:      price,
			High:      price + 2,
			Low:       price - 2,
			Close:     price + 1,
			Volume:    1.5,
			CloseTime: time.Unix(int64(i)*60+59, 0),
		})
	}
	return built
}
