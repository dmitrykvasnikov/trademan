package theming

import (
	"image/color"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Theme returns the TradeMan theme rendered in the given mode. It answers with
// the project's own colour set and leaves fonts, icons and sizes to Fyne.
func Theme(m Mode) fyne.Theme {
	return appTheme{base: theme.DefaultTheme(), mode: m, palette: paletteFor(m)}
}

// Apply makes every widget render in the given mode, regardless of the variant
// Fyne would otherwise pick up from the desktop.
func Apply(app fyne.App, m Mode) {
	app.Settings().SetTheme(Theme(m))
}

// appTheme paints the application from a single palette. The variant Fyne
// passes in is ignored on purpose: the mode comes from the switcher in the
// header, so the desktop changing its own scheme mid-session must not override
// what the user chose here.
type appTheme struct {
	base    fyne.Theme
	mode    Mode
	palette palette
}

func (t appTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	if c, ok := t.palette.color(name); ok {
		return c
	}
	return t.base.Color(name, t.variant())
}

func (t appTheme) Font(style fyne.TextStyle) fyne.Resource { return t.base.Font(style) }

func (t appTheme) Icon(name fyne.ThemeIconName) fyne.Resource { return t.base.Icon(name) }

func (t appTheme) Size(name fyne.ThemeSizeName) float32 { return t.base.Size(name) }

func (t appTheme) variant() fyne.ThemeVariant {
	if t.mode == Dark {
		return theme.VariantDark
	}
	return theme.VariantLight
}

// accent caches the desktop accent colour. The probes shell out to desktop
// services, so they run once per process rather than on every theme switch.
var accent struct {
	once  sync.Once
	color color.NRGBA
	found bool
}

func systemAccent() (color.NRGBA, bool) {
	accent.once.Do(func() {
		accent.color, accent.found = DetectSystemAccent()
	})
	return accent.color, accent.found
}
