package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// CandleCounts are the chart depths TradeMan offers in the "No of candles"
// dropdown.
var CandleCounts = []int{100, 200, 300, 500}

// maxCandles is the ceiling the kline endpoint puts on limit.
const maxCandles = 1000

// Candle is one candlestick: what a symbol opened, reached and closed at over a
// single interval, and how much of it changed hands.
type Candle struct {
	OpenTime  time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	CloseTime time.Time
}

// Rising reports whether the candle closed at or above its open, which is what
// decides the colour it is drawn in.
func (c Candle) Rising() bool { return c.Close >= c.Open }

// UnmarshalJSON reads Binance's kline encoding, which is a mixed array rather
// than an object: [openTime, open, high, low, close, volume, closeTime, ...].
// Everything past the close time is ignored, so entries added by a later API
// revision pass through harmlessly.
func (c *Candle) UnmarshalJSON(data []byte) error {
	var fields []json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("candle: %w", err)
	}
	if len(fields) < 7 {
		return fmt.Errorf("candle: has %d fields, want at least 7", len(fields))
	}

	openTime, err := epoch(fields[0])
	if err != nil {
		return fmt.Errorf("candle: open time: %w", err)
	}
	closeTime, err := epoch(fields[6])
	if err != nil {
		return fmt.Errorf("candle: close time: %w", err)
	}

	prices := make([]float64, 5)
	for i := range prices {
		if prices[i], err = decimal(fields[i+1]); err != nil {
			return fmt.Errorf("candle: field %d: %w", i+1, err)
		}
	}

	*c = Candle{
		OpenTime:  openTime,
		Open:      prices[0],
		High:      prices[1],
		Low:       prices[2],
		Close:     prices[3],
		Volume:    prices[4],
		CloseTime: closeTime,
	}
	return nil
}

// Klines returns the last limit candles for a symbol, oldest first. The limit is
// pulled into the range the endpoint accepts rather than rejected, so a caller
// asking for more than the exchange will serve still gets a chart.
func (c *Client) Klines(ctx context.Context, symbol, interval string, limit int) ([]Candle, error) {
	limit = min(max(limit, 1), maxCandles)

	query := url.Values{
		"symbol":   {symbol},
		"interval": {interval},
		"limit":    {strconv.Itoa(limit)},
	}

	var candles []Candle
	if err := c.get(ctx, "/api/v3/klines", query, &candles); err != nil {
		return nil, err
	}
	if len(candles) == 0 {
		return nil, fmt.Errorf("binance: no %s candles for %s", interval, symbol)
	}
	return candles, nil
}

// decimal reads a Binance price or size, which is quoted as a string so that no
// precision is lost on the way — "66175.99000000" rather than 66175.99.
func decimal(raw json.RawMessage) (float64, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return 0, err
	}
	return strconv.ParseFloat(text, 64)
}

// epoch reads a Binance timestamp, given in milliseconds since the Unix epoch.
func epoch(raw json.RawMessage) (time.Time, error) {
	var ms int64
	if err := json.Unmarshal(raw, &ms); err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(ms), nil
}
