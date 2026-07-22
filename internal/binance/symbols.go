package binance

import (
	"context"
	"slices"
	"strconv"
	"strings"
)

// TopSymbolCount is how many coins the Coin dropdown offers.
const TopSymbolCount = 20

// quoteAsset is what TradeMan charts everything against. Restricting the list
// to one quote keeps the coins comparable — a ranking mixing BTC and USDT pairs
// would sort the same coin twice — and USDT is where the depth is.
const quoteAsset = "USDT"

// Symbol is one tradable pair, split into what is being bought and what it is
// priced in.
type Symbol struct {
	Base  string // BTC
	Quote string // USDT
}

// Pair is the symbol as the API spells it.
func (s Symbol) Pair() string { return s.Base + s.Quote }

// Label is the symbol as the dropdown shows it.
func (s Symbol) Label() string { return s.Base + "/" + s.Quote }

// pegged lists the bases that are not coins to chart: stablecoins, tokenised
// fiat and tokenised metals. They dominate the turnover ranking — a dollar
// stablecoin usually out-trades Bitcoin — while holding a flat line no trader
// looks for signals in, so they are kept out of the list.
var pegged = map[string]bool{
	"AEUR": true, "BUSD": true, "DAI": true, "EUR": true, "EURI": true,
	"FDUSD": true, "GBP": true, "JPY": true, "PAXG": true, "PYUSD": true,
	"RLUSD": true, "TRY": true, "TUSD": true, "USD1": true, "USDC": true,
	"USDE": true, "USDP": true, "USDS": true, "XAUT": true, "XUSD": true,
}

// leveraged suffixes mark Binance's leveraged tokens (BTCUP, ETHDOWN and the
// like). They track a coin rather than being one, so they are dropped too.
var leveraged = []string{"UP", "DOWN", "BULL", "BEAR"}

// TopSymbols returns the n busiest coins on the exchange, ranked by the value
// traded over the last 24 hours and paired against USDT.
func (c *Client) TopSymbols(ctx context.Context, n int) ([]Symbol, error) {
	var tickers []struct {
		Symbol      string `json:"symbol"`
		QuoteVolume string `json:"quoteVolume"`
	}
	if err := c.get(ctx, "/api/v3/ticker/24hr", nil, &tickers); err != nil {
		return nil, err
	}

	type ranked struct {
		symbol Symbol
		volume float64
	}

	busiest := make([]ranked, 0, len(tickers))
	for _, ticker := range tickers {
		base, ok := chartable(ticker.Symbol)
		if !ok {
			continue
		}
		// A pair the exchange cannot price is one nobody is trading; ignoring
		// the parse error drops it to the bottom, which is where it belongs.
		volume, _ := strconv.ParseFloat(ticker.QuoteVolume, 64)
		busiest = append(busiest, ranked{Symbol{Base: base, Quote: quoteAsset}, volume})
	}

	slices.SortFunc(busiest, func(a, b ranked) int {
		if a.volume != b.volume {
			// Busiest first.
			if a.volume > b.volume {
				return -1
			}
			return 1
		}
		// Ties sort by name so the list does not shuffle between refreshes.
		return strings.Compare(a.symbol.Base, b.symbol.Base)
	})

	top := make([]Symbol, 0, n)
	for _, r := range busiest[:min(n, len(busiest))] {
		top = append(top, r.symbol)
	}
	return top, nil
}

// chartable reports whether a ticker symbol belongs in the coin list, and what
// its base asset is if so.
func chartable(symbol string) (string, bool) {
	base, ok := strings.CutSuffix(symbol, quoteAsset)
	if !ok || base == "" || pegged[base] {
		return "", false
	}
	for _, suffix := range leveraged {
		if strings.HasSuffix(base, suffix) {
			return "", false
		}
	}
	return base, true
}

// PopularSymbols is the coin list used when the exchange cannot be reached, so
// that a tab opened offline still offers something to select. It is a fixed set
// of long-standing majors rather than a snapshot of a ranking, because the
// coins topping the turnover charts on any given day are often short-lived.
func PopularSymbols() []Symbol {
	bases := []string{
		"BTC", "ETH", "BNB", "SOL", "XRP",
		"ADA", "DOGE", "TRX", "LINK", "AVAX",
		"DOT", "LTC", "BCH", "XLM", "UNI",
		"NEAR", "ATOM", "SUI", "PEPE", "AAVE",
	}

	symbols := make([]Symbol, 0, len(bases))
	for _, base := range bases {
		symbols = append(symbols, Symbol{Base: base, Quote: quoteAsset})
	}
	return symbols
}
