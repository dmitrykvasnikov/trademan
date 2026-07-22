//go:build !linux

package theming

import "image/color"

// DetectSystemAccent has no implementation outside Linux, the platform TradeMan
// targets, so callers keep the palette's own primary colour.
func DetectSystemAccent() (color.NRGBA, bool) {
	return color.NRGBA{}, false
}
