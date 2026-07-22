// Package theming resolves the colour scheme the application runs with and
// applies it to the Fyne app.
package theming

// Mode is the colour scheme the application renders with.
type Mode int

const (
	Light Mode = iota
	Dark
)

// String returns the human readable name shown on the theme switcher.
func (m Mode) String() string {
	if m == Dark {
		return "Dark"
	}
	return "Light"
}

// Toggle returns the opposite mode.
func (m Mode) Toggle() Mode {
	if m == Dark {
		return Light
	}
	return Dark
}
