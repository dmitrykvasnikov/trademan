package binance

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// serve returns a client talking to a server running handler.
func serve(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &Client{BaseURL: server.URL, HTTP: server.Client()}
}

// A refusal from Binance carries an explanation the chart area can show, so it
// has to survive the trip out of the client.
func TestRequestReportsExchangeExplanation(t *testing.T) {
	client := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"code":-1121,"msg":"Invalid symbol."}`)
	})

	_, err := client.Klines(context.Background(), "NOPE", "1h", 100)

	if err == nil {
		t.Fatal("a refused request returned no error")
	}
	if !strings.Contains(err.Error(), "Invalid symbol.") {
		t.Errorf("error is %q, want it to carry the exchange's explanation", err)
	}
}

// A gateway in front of the exchange answers in HTML, not JSON; the status has
// to stand in for the missing explanation.
func TestRequestReportsStatusWhenReplyIsNotJSON(t *testing.T) {
	client := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, "<html>down for maintenance</html>")
	})

	_, err := client.Klines(context.Background(), "BTCUSDT", "1h", 100)

	if err == nil {
		t.Fatal("a failed request returned no error")
	}
	if !strings.Contains(err.Error(), "Bad Gateway") {
		t.Errorf("error is %q, want it to name the HTTP status", err)
	}
}

func TestRequestReportsUnreadableReply(t *testing.T) {
	client := serve(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "not json at all")
	})

	_, err := client.Klines(context.Background(), "BTCUSDT", "1h", 100)

	if err == nil {
		t.Fatal("an unreadable reply returned no error")
	}
	if !strings.Contains(err.Error(), "unreadable") {
		t.Errorf("error is %q, want it to say the reply could not be read", err)
	}
}

// A cancelled context is how a chart's feed is retired when the user changes a
// dropdown, so a request must stop rather than run to completion.
func TestRequestStopsWhenCancelled(t *testing.T) {
	client := serve(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "[]")
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := client.Klines(ctx, "BTCUSDT", "1h", 100); err == nil {
		t.Error("a cancelled request returned no error")
	}
}

// The zero client is expected to be usable, so the defaults have to fill in.
func TestZeroClientUsesDefaults(t *testing.T) {
	var client Client

	if got := client.baseURL(); got != DefaultBaseURL {
		t.Errorf("zero client reads from %q, want %q", got, DefaultBaseURL)
	}
	if client.client() == nil {
		t.Error("zero client has no HTTP client to use")
	}
}

func TestNewClientHasATimeout(t *testing.T) {
	if got := New().HTTP.Timeout; got != requestTimeout {
		t.Errorf("client timeout is %v, want %v", got, requestTimeout)
	}
}
