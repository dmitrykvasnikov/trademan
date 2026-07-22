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
