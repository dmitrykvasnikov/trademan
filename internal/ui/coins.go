package ui

import (
	"context"
	"sync"
	"time"

	"fyne.io/fyne/v2"

	"github.com/dmitrykvasnikov/trademan/internal/binance"
)

// catalogTimeout caps the coin ranking request. Ranking every pair on the
// exchange means reading a couple of megabytes, so it gets longer than a chart
// refresh would.
const catalogTimeout = 30 * time.Second

// coinCatalog hands out the contents of the Coin dropdown. Every tab shares one
// catalog, because the ranking is the same for all of them and downloading it
// once per tab would be wasteful.
type coinCatalog struct {
	fetch func() ([]binance.Symbol, error)

	mu     sync.Mutex
	cached []binance.Symbol
}

func newCoinCatalog(client *binance.Client) *coinCatalog {
	return &coinCatalog{
		fetch: func() ([]binance.Symbol, error) {
			ctx, cancel := context.WithTimeout(context.Background(), catalogTimeout)
			defer cancel()
			return client.TopSymbols(ctx, binance.TopSymbolCount)
		},
	}
}

// load hands the coin list to deliver. A list the exchange has already answered
// with goes straight there, so the second tab is populated the moment it opens;
// otherwise the request runs in the background and deliver is called on the main
// goroutine, where it is safe to drop the result into a widget.
func (c *coinCatalog) load(deliver func([]binance.Symbol)) {
	if symbols, ok := c.known(); ok {
		deliver(symbols)
		return
	}

	go func() {
		symbols := c.symbols()
		fyne.Do(func() { deliver(symbols) })
	}()
}

// known returns the coin list if the exchange has already been asked for it.
func (c *coinCatalog) known() ([]binance.Symbol, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.cached, c.cached != nil
}

// symbols returns the ranked coin list, falling back to the built-in one when
// the exchange cannot be reached. Only a successful answer is remembered, so a
// tab opened after the connection comes back still gets the live ranking.
//
// The lock is deliberately held across the request: tabs opened while one is in
// flight wait for it and share the answer instead of starting their own.
func (c *coinCatalog) symbols() []binance.Symbol {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cached != nil {
		return c.cached
	}

	symbols, err := c.fetch()
	if err != nil || len(symbols) == 0 {
		return binance.PopularSymbols()
	}

	c.cached = symbols
	return c.cached
}
