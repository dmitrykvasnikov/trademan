//go:build !linux

package theming

// DetectSystemMode has no implementation outside Linux, the platform TradeMan
// targets, so callers get the documented Light fallback.
func DetectSystemMode() (Mode, bool) {
	return Light, false
}
