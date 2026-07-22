// Package binance reads public market data from the Binance REST API: the
// candlesticks a chart is drawn from and the ranking the coin list comes from.
package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// DefaultBaseURL is Binance's public market-data host. It serves the same spot
// endpoints as api.binance.com, needs no API key, and is not subject to the
// regional blocks the trading hosts apply — which is all TradeMan needs, since
// it only ever reads.
const DefaultBaseURL = "https://data-api.binance.vision"

const (
	// requestTimeout caps a single call, so a stalled connection cannot hold a
	// chart's refresh loop open forever.
	requestTimeout = 20 * time.Second

	// maxResponse caps how much of a reply is read. The 24-hour ticker covers
	// every symbol on the exchange and runs to a couple of megabytes; anything
	// past this is a malformed or hostile answer rather than market data.
	maxResponse = 16 << 20
)

// Client talks to one Binance host. The zero value works and reads from the
// default host with the default HTTP client; New sets up the timeout as well.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// New returns a client pointed at the public market-data host.
func New() *Client {
	return &Client{
		BaseURL: DefaultBaseURL,
		HTTP:    &http.Client{Timeout: requestTimeout},
	}
}

// get performs a GET against path and decodes the JSON reply into out.
func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	endpoint := c.baseURL() + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("binance: %s: %w", path, err)
	}

	resp, err := c.client().Do(req)
	if err != nil {
		return fmt.Errorf("binance: %s: %w", path, err)
	}
	defer resp.Body.Close()

	body := io.LimitReader(resp.Body, maxResponse)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("binance: %s: %s", path, reason(resp.StatusCode, body))
	}
	if err := json.NewDecoder(body).Decode(out); err != nil {
		return fmt.Errorf("binance: %s: unreadable reply: %w", path, err)
	}
	return nil
}

func (c *Client) baseURL() string {
	if c.BaseURL == "" {
		return DefaultBaseURL
	}
	return c.BaseURL
}

func (c *Client) client() *http.Client {
	if c.HTTP == nil {
		return http.DefaultClient
	}
	return c.HTTP
}

// reason describes a failed request in the words the chart area will show.
// Binance explains its refusals in a JSON {code, msg} document — "Invalid
// symbol." says far more than "Bad Request" — and the status line is the
// fallback for everything else, including the HTML a proxy might return.
func reason(status int, body io.Reader) string {
	var apiErr struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.NewDecoder(body).Decode(&apiErr); err == nil && apiErr.Msg != "" {
		return apiErr.Msg
	}
	if text := http.StatusText(status); text != "" {
		return text
	}
	return fmt.Sprintf("HTTP %d", status)
}
