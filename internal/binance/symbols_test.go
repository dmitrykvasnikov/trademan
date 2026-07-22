package binance

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// tickerServer answers the 24-hour ticker with the given symbol/turnover pairs.
func tickerServer(t *testing.T, turnover map[string]string) *Client {
	t.Helper()

	type ticker struct {
		Symbol      string `json:"symbol"`
		QuoteVolume string `json:"quoteVolume"`
	}

	tickers := make([]ticker, 0, len(turnover))
	for symbol, volume := range turnover {
		tickers = append(tickers, ticker{symbol, volume})
	}

	return serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/ticker/24hr" {
			t.Errorf("the coin ranking was requested from %q", r.URL.Path)
		}
		if err := json.NewEncoder(w).Encode(tickers); err != nil {
			t.Errorf("serving the ranking: %v", err)
		}
	})
}

// labels renders symbols the way the dropdown shows them, for readable
// comparisons in the tests below.
func labels(symbols []Symbol) []string {
	rendered := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		rendered = append(rendered, symbol.Label())
	}
	return rendered
}

func TestTopSymbolsRanksByTurnover(t *testing.T) {
	client := tickerServer(t, map[string]string{
		"BTCUSDT":  "1000",
		"ETHUSDT":  "3000",
		"SOLUSDT":  "2000",
		"DOGEUSDT": "500",
	})

	top, err := client.TopSymbols(context.Background(), 10)
	if err != nil {
		t.Fatalf("ranking coins: %v", err)
	}

	const want = "ETH/USDT SOL/USDT BTC/USDT DOGE/USDT"
	if got := strings.Join(labels(top), " "); got != want {
		t.Errorf("ranking is %q, want the busiest first: %q", got, want)
	}
}

func TestTopSymbolsStopsAtTheCountAskedFor(t *testing.T) {
	client := tickerServer(t, map[string]string{
		"BTCUSDT": "4000", "ETHUSDT": "3000", "SOLUSDT": "2000", "XRPUSDT": "1000",
	})

	top, err := client.TopSymbols(context.Background(), 2)
	if err != nil {
		t.Fatalf("ranking coins: %v", err)
	}

	if got := strings.Join(labels(top), " "); got != "BTC/USDT ETH/USDT" {
		t.Errorf("ranking is %q, want the two busiest", got)
	}
}

// The dropdown holds coins, so pairs quoted in something else, dollar
// stablecoins and leveraged tokens all have to be filtered out — the last two
// out-trade most real coins and would otherwise fill the list.
func TestTopSymbolsKeepsOnlyChartableCoins(t *testing.T) {
	client := tickerServer(t, map[string]string{
		"USDCUSDT":  "9000", // a stablecoin
		"FDUSDUSDT": "8000", // another one
		"EURUSDT":   "7000", // tokenised fiat
		"PAXGUSDT":  "6000", // tokenised gold
		"BTCUPUSDT": "5000", // a leveraged token
		"ETHBTC":    "4000", // quoted in BTC, not USDT
		"USDT":      "3000", // the quote asset alone
		"BTCUSDT":   "2000",
		"ETHUSDT":   "1000",
	})

	top, err := client.TopSymbols(context.Background(), 20)
	if err != nil {
		t.Fatalf("ranking coins: %v", err)
	}

	if got := strings.Join(labels(top), " "); got != "BTC/USDT ETH/USDT" {
		t.Errorf("ranking is %q, want only the coins BTC/USDT and ETH/USDT", got)
	}
}

// A pair the exchange cannot put a number on must not take the ranking down
// with it; it just sorts last.
func TestTopSymbolsToleratesAnUnreadableTurnover(t *testing.T) {
	client := tickerServer(t, map[string]string{
		"BTCUSDT": "1000",
		"ETHUSDT": "",
	})

	top, err := client.TopSymbols(context.Background(), 10)
	if err != nil {
		t.Fatalf("ranking coins: %v", err)
	}

	if got := strings.Join(labels(top), " "); got != "BTC/USDT ETH/USDT" {
		t.Errorf("ranking is %q, want the unpriced pair last", got)
	}
}

func TestSymbolNaming(t *testing.T) {
	symbol := Symbol{Base: "BTC", Quote: "USDT"}

	if got := symbol.Pair(); got != "BTCUSDT" {
		t.Errorf("symbol is sent to the API as %q, want %q", got, "BTCUSDT")
	}
	if got := symbol.Label(); got != "BTC/USDT" {
		t.Errorf("symbol is shown as %q, want %q", got, "BTC/USDT")
	}
}

// The fallback list stands in for the ranking, so it has to be the same length
// and made of the same kind of entries.
func TestPopularSymbolsCanStandInForTheRanking(t *testing.T) {
	symbols := PopularSymbols()

	if len(symbols) != TopSymbolCount {
		t.Errorf("the fallback list holds %d coins, want %d", len(symbols), TopSymbolCount)
	}

	seen := map[string]bool{}
	for _, symbol := range symbols {
		if symbol.Quote != quoteAsset {
			t.Errorf("%s is quoted in %s, want %s", symbol.Label(), symbol.Quote, quoteAsset)
		}
		if _, ok := chartable(symbol.Pair()); !ok {
			t.Errorf("%s would be filtered out of the live ranking", symbol.Label())
		}
		if seen[symbol.Base] {
			t.Errorf("%s appears in the fallback list twice", symbol.Label())
		}
		seen[symbol.Base] = true
	}
}
