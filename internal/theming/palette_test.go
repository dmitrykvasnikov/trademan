package theming

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

// allColorNames is every colour Fyne v2.8 asks a theme for.
var allColorNames = []fyne.ThemeColorName{
	theme.ColorNameBackground,
	theme.ColorNameButton,
	theme.ColorNameDisabled,
	theme.ColorNameDisabledButton,
	theme.ColorNameError,
	theme.ColorNameFocus,
	theme.ColorNameForeground,
	theme.ColorNameForegroundOnError,
	theme.ColorNameForegroundOnPrimary,
	theme.ColorNameForegroundOnSuccess,
	theme.ColorNameForegroundOnWarning,
	theme.ColorNameHeaderBackground,
	theme.ColorNameHover,
	theme.ColorNameHyperlink,
	theme.ColorNameInnerWindowBorder,
	theme.ColorNameInnerWindowBorderInactive,
	theme.ColorNameInputBackground,
	theme.ColorNameInputBorder,
	theme.ColorNameMenuBackground,
	theme.ColorNameOverlayBackground,
	theme.ColorNamePlaceHolder,
	theme.ColorNamePressed,
	theme.ColorNamePrimary,
	theme.ColorNameScrollBar,
	theme.ColorNameScrollBarBackground,
	theme.ColorNameSelection,
	theme.ColorNameSeparator,
	theme.ColorNameShadow,
	theme.ColorNameSuccess,
	theme.ColorNameWarning,
}

// The palettes must answer for every colour Fyne knows about, so nothing falls
// back to the default theme and renders off-scheme.
func TestPalettesCoverEveryColorName(t *testing.T) {
	for _, mode := range []Mode{Light, Dark} {
		for _, name := range allColorNames {
			if _, ok := paletteFor(mode).color(name); !ok {
				t.Errorf("%v palette has no colour for %q", mode, name)
			}
		}
	}
}

// An unknown name has to fall through to Fyne rather than return nil, which
// would panic the renderer. Fyne's own theme reads the running app, so this one
// needs an app in place.
func TestThemeAnswersUnknownColorName(t *testing.T) {
	test.NewTempApp(t)

	if c := Theme(Dark).Color("trademan-no-such-colour", theme.VariantDark); c == nil {
		t.Error("unknown colour name resolved to nil, want the Fyne default")
	}
}

// The theme pins its mode: the variant Fyne passes in is the desktop's opinion,
// and the switcher in the header outranks it.
func TestThemeIgnoresRequestedVariant(t *testing.T) {
	for _, mode := range []Mode{Light, Dark} {
		app := Theme(mode)
		light := app.Color(theme.ColorNameBackground, theme.VariantLight)
		dark := app.Color(theme.ColorNameBackground, theme.VariantDark)
		if light != dark {
			t.Errorf("%v theme background changed with the variant: %v vs %v", mode, light, dark)
		}
	}
}

// Fonts, icons and sizes stay Fyne's; only colours are ours.
func TestThemeDelegatesNonColorLookups(t *testing.T) {
	app := Theme(Light)
	if app.Font(fyne.TextStyle{}) == nil {
		t.Error("theme returned no font")
	}
	if app.Icon(theme.IconNameHome) == nil {
		t.Error("theme returned no icon")
	}
	if app.Size(theme.SizeNameText) <= 0 {
		t.Error("theme returned a non-positive text size")
	}
}

// The two modes have to actually look different, and each has to be on the side
// of the brightness scale its name promises.
func TestModesAreLightAndDark(t *testing.T) {
	l, d := lightPalette, darkPalette
	if l.Background == d.Background {
		t.Fatal("both palettes share a background colour")
	}
	if luminance(l.Background) < 0.5 {
		t.Errorf("light background %v is not light", l.Background)
	}
	if luminance(d.Background) > 0.5 {
		t.Errorf("dark background %v is not dark", d.Background)
	}
}

// Text has to be readable on the surface it sits on. The thresholds are the
// WCAG ones: 4.5 for body text, 3.0 for the larger or dimmed elements.
func TestPaletteContrast(t *testing.T) {
	tests := []struct {
		what  string
		of    func(palette) color.NRGBA
		on    func(palette) color.NRGBA
		least float64
	}{
		{"foreground on background", func(p palette) color.NRGBA { return p.Foreground }, func(p palette) color.NRGBA { return p.Background }, 4.5},
		{"foreground on button", func(p palette) color.NRGBA { return p.Foreground }, func(p palette) color.NRGBA { return p.Button }, 4.5},
		{"foreground on input", func(p palette) color.NRGBA { return p.Foreground }, func(p palette) color.NRGBA { return p.InputBackground }, 4.5},
		{"foreground on overlay", func(p palette) color.NRGBA { return p.Foreground }, func(p palette) color.NRGBA { return p.OverlayBackground }, 4.5},
		{"placeholder on input", func(p palette) color.NRGBA { return p.Placeholder }, func(p palette) color.NRGBA { return p.InputBackground }, 3.0},
		{"disabled on background", func(p palette) color.NRGBA { return p.Disabled }, func(p palette) color.NRGBA { return p.Background }, 3.0},
		{"primary on background", func(p palette) color.NRGBA { return p.Primary }, func(p palette) color.NRGBA { return p.Background }, 3.0},
		{"error on background", func(p palette) color.NRGBA { return p.Error }, func(p palette) color.NRGBA { return p.Background }, 3.0},
		{"success on background", func(p palette) color.NRGBA { return p.Success }, func(p palette) color.NRGBA { return p.Background }, 3.0},
		{"warning on background", func(p palette) color.NRGBA { return p.Warning }, func(p palette) color.NRGBA { return p.Background }, 3.0},
	}

	palettes := map[string]palette{"light": lightPalette, "dark": darkPalette}
	for name, p := range palettes {
		for _, test := range tests {
			t.Run(name+" "+test.what, func(t *testing.T) {
				if got := contrastRatio(test.of(p), test.on(p)); got < test.least {
					t.Errorf("contrast is %.2f:1, want at least %.1f:1", got, test.least)
				}
			})
		}
	}
}

// Labels printed on a filled swatch must be readable whatever the fill is,
// including a system accent colour the palette never saw.
func TestOnColorIsReadable(t *testing.T) {
	fills := []color.NRGBA{
		rgb(0x00, 0x00, 0x00), rgb(0xFF, 0xFF, 0xFF),
		rgb(0x1F, 0x6F, 0xEB), rgb(0xC8, 0x88, 0x00), rgb(0x80, 0x80, 0x80),
	}
	for _, fill := range fills {
		if got := contrastRatio(onColor(fill), fill); got < 4.5 {
			t.Errorf("text on %v has contrast %.2f:1, want at least 4.5:1", fill, got)
		}
	}
}

// A system accent picked for the opposite mode still has to be visible, and it
// has to keep looking like the colour the user chose.
func TestLegibleLiftsWeakAccents(t *testing.T) {
	tests := []struct {
		name   string
		accent color.NRGBA
		bg     color.NRGBA
	}{
		{"dark accent on dark background", rgb(0x1A, 0x33, 0x5C), darkPalette.Background},
		{"light accent on light background", rgb(0xE8, 0xE0, 0xB0), lightPalette.Background},
		{"black accent on dark background", rgb(0x00, 0x00, 0x00), darkPalette.Background},
		{"white accent on light background", rgb(0xFF, 0xFF, 0xFF), lightPalette.Background},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := legible(test.accent, test.bg)
			if ratio := contrastRatio(got, test.bg); ratio < 3.0 {
				t.Errorf("adjusted accent %v has contrast %.2f:1 on %v, want at least 3.0:1", got, ratio, test.bg)
			}
		})
	}
}

// An accent that already stands out must be passed through untouched.
func TestLegibleLeavesStrongAccentsAlone(t *testing.T) {
	accent := rgb(0x35, 0x84, 0xE4)
	if got := legible(accent, darkPalette.Background); got != accent {
		t.Errorf("legible(%v) = %v, want it unchanged", accent, got)
	}
}
