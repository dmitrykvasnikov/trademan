package theming

import (
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// palette is one complete colour set. Every colour Fyne asks for is either a
// field here or derived from one, so both modes are described in a single place
// and stay comparable side by side.
//
// The surfaces climb away from Background as widgets stack on top of it —
// Background, then Button and the input fields, then menus and dialogs — which
// is what gives depth to a dark scheme without borders everywhere.
type palette struct {
	Background          color.NRGBA
	Button              color.NRGBA
	DisabledButton      color.NRGBA
	Disabled            color.NRGBA
	Error               color.NRGBA
	Foreground          color.NRGBA
	HeaderBackground    color.NRGBA
	Hover               color.NRGBA
	InputBackground     color.NRGBA
	InputBorder         color.NRGBA
	MenuBackground      color.NRGBA
	OverlayBackground   color.NRGBA
	Placeholder         color.NRGBA
	Pressed             color.NRGBA
	Primary             color.NRGBA
	ScrollBar           color.NRGBA
	ScrollBarBackground color.NRGBA
	Separator           color.NRGBA
	Shadow              color.NRGBA
	Success             color.NRGBA
	Warning             color.NRGBA
}

// lightPalette is the default light colour set: a cool neutral grey page with white
// raised surfaces, the tones desktop toolkits settled on and the ones a chart
// drawn in ink reads best against.
var lightPalette = palette{
	Background:          rgb(0xF6, 0xF7, 0xF9),
	Button:              rgb(0xEB, 0xEE, 0xF2),
	DisabledButton:      rgb(0xE3, 0xE6, 0xEB),
	Disabled:            rgb(0x86, 0x8E, 0x9C),
	Error:               rgb(0xC4, 0x28, 0x2E),
	Foreground:          rgb(0x1B, 0x20, 0x27),
	HeaderBackground:    rgb(0xFF, 0xFF, 0xFF),
	Hover:               rgba(0x00, 0x00, 0x00, 0x14),
	InputBackground:     rgb(0xFF, 0xFF, 0xFF),
	InputBorder:         rgb(0xD3, 0xD9, 0xE0),
	MenuBackground:      rgb(0xFF, 0xFF, 0xFF),
	OverlayBackground:   rgb(0xFF, 0xFF, 0xFF),
	Placeholder:         rgb(0x70, 0x79, 0x87),
	Pressed:             rgba(0x00, 0x00, 0x00, 0x26),
	Primary:             rgb(0x1F, 0x6F, 0xEB),
	ScrollBar:           rgba(0x00, 0x00, 0x00, 0x4D),
	ScrollBarBackground: rgba(0x00, 0x00, 0x00, 0x0F),
	Separator:           rgb(0xDE, 0xE3, 0xE9),
	Shadow:              rgba(0x00, 0x00, 0x00, 0x40),
	Success:             rgb(0x0B, 0x7A, 0x55),
	Warning:             rgb(0x9A, 0x5B, 0x06),
}

// darkPalette is the default dark colour set: the near-black blue-grey trading desks
// use, kept off pure black so shadows and raised surfaces stay visible.
var darkPalette = palette{
	Background:          rgb(0x12, 0x16, 0x1C),
	Button:              rgb(0x1E, 0x24, 0x2E),
	DisabledButton:      rgb(0x19, 0x1E, 0x26),
	Disabled:            rgb(0x69, 0x73, 0x82),
	Error:               rgb(0xF0, 0x63, 0x67),
	Foreground:          rgb(0xE6, 0xEA, 0xF0),
	HeaderBackground:    rgb(0x17, 0x1C, 0x24),
	Hover:               rgba(0xFF, 0xFF, 0xFF, 0x1A),
	InputBackground:     rgb(0x1A, 0x20, 0x2A),
	InputBorder:         rgb(0x2E, 0x37, 0x44),
	MenuBackground:      rgb(0x17, 0x1C, 0x24),
	OverlayBackground:   rgb(0x17, 0x1C, 0x24),
	Placeholder:         rgb(0x8A, 0x94, 0xA3),
	Pressed:             rgba(0xFF, 0xFF, 0xFF, 0x33),
	Primary:             rgb(0x4D, 0x96, 0xFF),
	ScrollBar:           rgba(0xFF, 0xFF, 0xFF, 0x40),
	ScrollBarBackground: rgba(0xFF, 0xFF, 0xFF, 0x0F),
	Separator:           rgb(0x26, 0x2E, 0x39),
	Shadow:              rgba(0x00, 0x00, 0x00, 0x8C),
	Success:             rgb(0x2E, 0xBD, 0x85),
	Warning:             rgb(0xF0, 0xA6, 0x2E),
}

// paletteFor returns the colour set for mode, with the desktop's accent colour
// substituted for the built-in primary when the session exposes one. The accent
// is pulled towards the background until it is legible on it, so a light accent
// picked for a light desktop still reads in dark mode and vice versa.
func paletteFor(m Mode) palette {
	p := lightPalette
	if m == Dark {
		p = darkPalette
	}

	if accent, ok := systemAccent(); ok {
		p.Primary = legible(accent, p.Background)
	}
	return p
}

// color maps a Fyne colour name onto the palette. The second result is false
// for names the palette does not describe, which the theme then answers from
// Fyne's own defaults — so a name added by a later Fyne release still renders.
func (p palette) color(name fyne.ThemeColorName) (color.Color, bool) {
	switch name {
	case theme.ColorNameBackground:
		return p.Background, true
	case theme.ColorNameButton:
		return p.Button, true
	case theme.ColorNameDisabled:
		return p.Disabled, true
	case theme.ColorNameDisabledButton:
		return p.DisabledButton, true
	case theme.ColorNameError:
		return p.Error, true
	case theme.ColorNameFocus:
		// The focus ring is the accent showing through the widget below it.
		return alpha(p.Primary, 0x7F), true
	case theme.ColorNameForeground:
		return p.Foreground, true
	case theme.ColorNameForegroundOnError:
		return onColor(p.Error), true
	case theme.ColorNameForegroundOnPrimary:
		return onColor(p.Primary), true
	case theme.ColorNameForegroundOnSuccess:
		return onColor(p.Success), true
	case theme.ColorNameForegroundOnWarning:
		return onColor(p.Warning), true
	case theme.ColorNameHeaderBackground:
		return p.HeaderBackground, true
	case theme.ColorNameHover:
		return p.Hover, true
	case theme.ColorNameHyperlink:
		return p.Primary, true
	case theme.ColorNameInnerWindowBorder:
		return p.HeaderBackground, true
	case theme.ColorNameInnerWindowBorderInactive:
		return p.Button, true
	case theme.ColorNameInputBackground:
		return p.InputBackground, true
	case theme.ColorNameInputBorder:
		return p.InputBorder, true
	case theme.ColorNameMenuBackground:
		return p.MenuBackground, true
	case theme.ColorNameOverlayBackground:
		return p.OverlayBackground, true
	case theme.ColorNamePlaceHolder:
		return p.Placeholder, true
	case theme.ColorNamePressed:
		return p.Pressed, true
	case theme.ColorNamePrimary:
		return p.Primary, true
	case theme.ColorNameScrollBar:
		return p.ScrollBar, true
	case theme.ColorNameScrollBarBackground:
		return p.ScrollBarBackground, true
	case theme.ColorNameSelection:
		// Selected text stays readable, so the accent is thinner here than focus.
		return alpha(p.Primary, 0x4D), true
	case theme.ColorNameSeparator:
		return p.Separator, true
	case theme.ColorNameShadow:
		return p.Shadow, true
	case theme.ColorNameSuccess:
		return p.Success, true
	case theme.ColorNameWarning:
		return p.Warning, true
	}
	return nil, false
}

func rgb(r, g, b uint8) color.NRGBA { return color.NRGBA{R: r, G: g, B: b, A: 0xFF} }

func rgba(r, g, b, a uint8) color.NRGBA { return color.NRGBA{R: r, G: g, B: b, A: a} }

// alpha returns c at the given opacity.
func alpha(c color.NRGBA, a uint8) color.NRGBA {
	c.A = a
	return c
}

// contrastRatio is the WCAG ratio between two opaque colours, from 1 (identical)
// to 21 (black on white).
func contrastRatio(a, b color.NRGBA) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// luminance is the WCAG relative luminance of an opaque colour.
func luminance(c color.NRGBA) float64 {
	return 0.2126*channel(c.R) + 0.7152*channel(c.G) + 0.0722*channel(c.B)
}

// channel linearises one sRGB channel.
func channel(v uint8) float64 {
	f := float64(v) / 255
	if f <= 0.04045 {
		return f / 12.92
	}
	return math.Pow((f+0.055)/1.055, 2.4)
}

// onColor picks the text colour to print on top of a filled swatch: near-black
// on light fills, near-white on dark ones. Deriving it rather than storing it
// keeps labels readable on a system accent colour of any brightness.
func onColor(fill color.NRGBA) color.NRGBA {
	ink, paper := rgb(0x10, 0x13, 0x18), rgb(0xFF, 0xFF, 0xFF)
	if contrastRatio(fill, ink) >= contrastRatio(fill, paper) {
		return ink
	}
	return paper
}

// legible lightens or darkens c away from bg until the two are far enough apart
// to be told apart at a glance. It is only ever a nudge: the hue survives, since
// mixing happens with white or black.
func legible(c, bg color.NRGBA) color.NRGBA {
	// 3:1 is the WCAG floor for large text and UI components, which is what an
	// accent colour is used for here.
	const wanted = 3.0

	// Mix towards white on a dark background and towards black on a light one.
	target := rgb(0xFF, 0xFF, 0xFF)
	if luminance(bg) > 0.5 {
		target = rgb(0x00, 0x00, 0x00)
	}

	adjusted := c
	for step := 0; step < 20 && contrastRatio(adjusted, bg) < wanted; step++ {
		adjusted = mix(adjusted, target, 0.1)
	}
	return adjusted
}

// mix blends amount of b into a, keeping a's opacity.
func mix(a, b color.NRGBA, amount float64) color.NRGBA {
	blend := func(x, y uint8) uint8 {
		return uint8(math.Round(float64(x)*(1-amount) + float64(y)*amount))
	}
	return color.NRGBA{R: blend(a.R, b.R), G: blend(a.G, b.G), B: blend(a.B, b.B), A: a.A}
}
