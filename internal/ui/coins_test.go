package ui

import (
	"errors"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/dmitrykvasnikov/trademan/internal/binance"
)

// A tab opened without a connection still has to offer coins to pick from.
func TestCatalogFallsBackWhenTheExchangeIsUnreachable(t *testing.T) {
	catalog := &coinCatalog{
		fetch: func() ([]binance.Symbol, error) { return nil, errors.New("no route to host") },
	}

	got := catalog.symbols()

	if len(got) != len(binance.PopularSymbols()) {
		t.Errorf("an unreachable exchange yielded %d coins, want the built-in list", len(got))
	}
}

// An exchange that answers with nothing is as useless as one that does not
// answer at all, and must not leave the dropdown empty.
func TestCatalogFallsBackWhenTheRankingIsEmpty(t *testing.T) {
	catalog := &coinCatalog{
		fetch: func() ([]binance.Symbol, error) { return nil, nil },
	}

	if got := catalog.symbols(); len(got) == 0 {
		t.Error("an empty ranking left the coin list empty")
	}
}

// Ranking every pair on the exchange is a couple of megabytes, so a second tab
// must reuse the answer rather than ask again.
func TestCatalogAsksTheExchangeOnce(t *testing.T) {
	asked := 0
	catalog := &coinCatalog{
		fetch: func() ([]binance.Symbol, error) {
			asked++
			return testSymbols, nil
		},
	}

	catalog.symbols()
	catalog.symbols()

	if asked != 1 {
		t.Errorf("the exchange was asked %d times, want 1", asked)
	}
}

// A failure is not worth remembering: the next tab should get the live ranking
// once the connection is back.
func TestCatalogRetriesAfterAFailure(t *testing.T) {
	asked := 0
	catalog := &coinCatalog{
		fetch: func() ([]binance.Symbol, error) {
			asked++
			if asked == 1 {
				return nil, errors.New("offline")
			}
			return testSymbols, nil
		},
	}

	catalog.symbols()
	got := catalog.symbols()

	if asked != 2 {
		t.Errorf("the exchange was asked %d times, want a retry after the failure", asked)
	}
	if len(got) != len(testSymbols) {
		t.Errorf("the retry yielded %d coins, want the live ranking of %d", len(got), len(testSymbols))
	}
}

// Once the ranking is known, a tab must be able to fill its dropdown on the
// spot instead of waiting on a goroutine.
func TestCatalogDeliversAKnownRankingImmediately(t *testing.T) {
	test.NewTempApp(t)
	catalog := testCatalog(testSymbols)

	var delivered []binance.Symbol
	catalog.load(func(symbols []binance.Symbol) { delivered = symbols })

	if len(delivered) != len(testSymbols) {
		t.Errorf("load handed over %d coins on the spot, want %d", len(delivered), len(testSymbols))
	}
}
