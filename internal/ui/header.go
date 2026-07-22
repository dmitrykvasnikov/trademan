package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/dmitrykvasnikov/trademan/internal/theming"
	"github.com/dmitrykvasnikov/trademan/internal/version"
)

// header is the top bar of the main screen: the application name and version on
// the left, the light/dark switcher on the right.
type header struct {
	mode     theming.Mode
	onChange func(theming.Mode)
	switcher *widget.Button
}

// newHeader builds the bar starting in mode, reporting every switch to onChange.
func newHeader(mode theming.Mode, onChange func(theming.Mode)) *header {
	h := &header{mode: mode, onChange: onChange}
	h.switcher = widget.NewButtonWithIcon(mode.String(), theme.ColorPaletteIcon(), h.toggle)
	h.switcher.Importance = widget.LowImportance
	return h
}

// toggle flips the colour scheme and reports the new one.
func (h *header) toggle() {
	h.mode = h.mode.Toggle()
	h.switcher.SetText(h.mode.String())
	h.onChange(h.mode)
}

func (h *header) view() fyne.CanvasObject {
	name := widget.NewLabelWithStyle(version.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	release := widget.NewLabel("v" + version.Version)

	// A border keeps the identity pinned left and the switcher pinned right at
	// any window width.
	bar := container.NewBorder(nil, nil, container.NewHBox(name, release), h.switcher)
	return container.NewVBox(bar, widget.NewSeparator())
}
