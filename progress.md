# TradeMan — Progress

Project version: **0.0.1**
Language: Go · Target platform: Linux (Arch, distro-agnostic tooling)
Data source: Binance API (candlesticks)
GUI toolkit: Fyne v2.8

Legend: `[ ]` not started · `[~]` in progress · `[x]` done

---

## 01 — Base UI implementation

| Status | Feature | Description |
|---|---|---|
| [x] | main screen | Header with app name `TradeMan` + version on the left, light/dark theme switcher on the right; tab section fills the rest of the window and starts empty. Keymap: `q` quit (with confirmation), `Ctrl-t` new tab named `No chart` and focus it, `Ctrl-w` close tab (with confirmation) then focus next tab, or previous if none. |
| [x] | tab interface | Each tab has a header row of three dropdowns — `Coin`, `Interval`, `No of candles` — with the chart area filling the remaining tab space below. |

## 02 — Themes implementation

| Status | Feature | Description |
|---|---|---|
| [x] | light / dark themes | Define color sets for light and dark themes, preferring system colors with sensible fallbacks; theme switcher on the main screen applies them, and its initial state follows the system theme (light if the system value is unavailable). |

Both color sets live in `internal/theming/palette.go` and cover every color Fyne
asks for, so nothing falls back to the default palette. The desktop's accent
color is read from the XDG portal, with the GNOME named accent as a fallback,
and becomes the primary color; it is nudged towards the background until it
clears 3:1 contrast, so an accent chosen for a light desktop still reads in dark
mode. The switcher pins the mode against the variant Fyne reports, so the
desktop changing its own scheme mid-session cannot override the user's choice.

## 03 — Tab functionality

| Status | Feature | Description |
|---|---|---|
| [x] | Dropdown lists | `Coin` holds the 20 most popular Binance coins; `Interval` holds 1s, 1m, 3m, 5m, 15m, 30m, 1h, 2h, 4h, 6h, 8h, 12h, 1d, 3d, 1w, 1M; `No of candles` holds 100, 200, 300, 500. All three default to empty. |
| [x] | Live chart | Once all three dropdowns are non-empty, draw a live chart for the selected coin / interval / candle count; redraw on any dropdown change and refresh on the selected interval. |

The coin list is the 20 busiest USDT pairs by 24-hour turnover, read from the
exchange once and shared by every tab. Stablecoins, tokenised fiat and metals
and leveraged tokens are filtered out — they out-trade most real coins but hold
a flat line — and a fixed list of majors stands in when the exchange cannot be
reached. Market data comes from `data-api.binance.vision`, Binance's public
read-only host, which needs no API key and is not regionally blocked.

A complete selection starts a feed that redraws at the pace of its own interval,
clamped to between a second and a minute so a 1s chart cannot flood the API and
a daily one still shows its forming candle moving. Changing a dropdown retires
the running feed, and a reply that arrives after its feed was retired is
discarded rather than drawn over the new chart. Closing a tab stops its feed. A
failed refresh keeps the candles already on screen and marks them stale instead
of blanking the panel. Each tab renames itself after what it is charting.

The chart is a custom Fyne widget: a candle body and wick per interval, a price
scale, a time scale whose format follows how much time is on screen, and a
marker on the latest close. It draws in theme colours, so it follows the
light/dark switch, and reuses its shapes between redraws rather than
reallocating them every tick.

---

## Layout

| Path | Holds |
|---|---|
| `main.go` | Entry point and Fyne app metadata |
| `internal/version` | App name and version, kept in sync with `context.md` |
| `internal/theming` | Light/dark color sets, system scheme and accent detection, theme application |
| `internal/binance` | Market-data client: candlesticks, the coin ranking, interval spans |
| `internal/ui` | Main window, header, tab manager, tab contents, coin catalog, chart area and candlestick widget |

Build and run: `go build ./...` then `go run .` · Tests: `go test ./...`

---

## Change log

| Date | Change |
|---|---|
| 2026-07-22 | Created `progress.md` from the `features/` directory. No code written yet. |
| 2026-07-22 | Section 01 implemented on Fyne v2.8: main screen (header, empty tab section, `q`/`Ctrl-t`/`Ctrl-w` keymap with confirmations) and the tab interface (three dropdowns above a chart area). System theme detection added via XDG portal, gsettings and GTK theme name, falling back to light. 21 tests added. |
| 2026-07-22 | Section 02 implemented: the project's own light and dark color sets covering every Fyne color name, desktop accent detection (XDG portal, GNOME named accents) feeding the primary color, and contrast helpers that keep text readable on any accent. `Apply` now installs this theme instead of wrapping Fyne's default. 13 tests added (34 in total), including WCAG contrast checks on both sets. |
| 2026-07-22 | Section 03 implemented: new `internal/binance` market-data client (candlesticks, turnover-ranked coin list, interval spans) and, in `internal/ui`, the shared coin catalog, the per-tab live feed and a custom candlestick chart widget. Dropdowns are populated and still start empty; a complete selection draws a live chart that redraws on the selected interval and on any dropdown change. 62 tests added (96 in total), all packages passing under `-race`. |

---

## Token & model usage

Cumulative across the project. Figures are approximate session estimates and
count the full context of every request, so they grow with conversation length
rather than with the amount of code produced.

| Date | Model | Input tokens | Output tokens | Total |
|---|---|---|---|---|
| 2026-07-22 | claude-opus-4-8 | ~14,000 | ~1,200 | ~15,200 |
| 2026-07-22 | claude-opus-4-8 | ~1,100,000 | ~19,000 | ~1,119,000 |
| 2026-07-22 | claude-opus-4-8 | ~700,000 | ~15,000 | ~715,000 |
| 2026-07-22 | claude-opus-4-8 | ~900,000 | ~34,000 | ~934,000 |
| **Cumulative** | | **~2,714,000** | **~69,200** | **~2,783,200** |
