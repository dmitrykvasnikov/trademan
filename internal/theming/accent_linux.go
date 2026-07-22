//go:build linux

package theming

import (
	"image/color"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// DetectSystemAccent reports the accent colour configured for the desktop
// session, which the palettes use as their primary colour. The probes run from
// the most portable source to the most GNOME specific, and the first answer
// wins.
//
// The second result is false when no service answers, in which case the caller
// keeps the palette's own primary.
func DetectSystemAccent() (color.NRGBA, bool) {
	probes := []func() (color.NRGBA, bool){
		portalAccentGDBus,    // XDG desktop portal, works on any DE that ships one
		portalAccentDBusSend, // same setting, for sessions without gdbus
		gsettingsAccent,      // GNOME 47+ named accent
	}
	for _, probe := range probes {
		if accent, ok := probe(); ok {
			return accent, true
		}
	}
	return color.NRGBA{}, false
}

// portalAccentGDBus asks the XDG desktop portal through gdbus. ReadOne is the
// current method; Read is its deprecated predecessor, still the only one on
// older portal builds.
func portalAccentGDBus() (color.NRGBA, bool) {
	const (
		dest   = "org.freedesktop.portal.Desktop"
		path   = "/org/freedesktop/portal/desktop"
		scheme = "org.freedesktop.appearance"
		key    = "accent-color"
	)
	for _, method := range []string{"ReadOne", "Read"} {
		out, ok := run("gdbus", "call", "--session", "--dest", dest,
			"--object-path", path, "--method", "org.freedesktop.portal.Settings."+method,
			scheme, key)
		if !ok {
			continue
		}
		if accent, ok := parseAccent(out); ok {
			return accent, true
		}
	}
	return color.NRGBA{}, false
}

// portalAccentDBusSend reads the same portal setting with dbus-send.
func portalAccentDBusSend() (color.NRGBA, bool) {
	out, ok := run("dbus-send", "--session", "--print-reply",
		"--dest=org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop",
		"org.freedesktop.portal.Settings.Read",
		"string:org.freedesktop.appearance", "string:accent-color")
	if !ok {
		return color.NRGBA{}, false
	}
	return parseAccent(out)
}

// accentTuple matches the three doubles of a gdbus reply, `(<(0.2, 0.5, 0.9)>,)`.
var accentTuple = regexp.MustCompile(`\(\s*(-?[\d.eE+-]+)\s*,\s*(-?[\d.eE+-]+)\s*,\s*(-?[\d.eE+-]+)\s*\)`)

// accentDouble matches one double of a dbus-send reply, which prints the struct
// members one per line and labels each with its type.
var accentDouble = regexp.MustCompile(`double\s+(-?[\d.eE+-]+)`)

// parseAccent reads the portal's (ddd) reply of red, green and blue in the 0..1
// range. The portal answers with all channels negative when the user set no
// accent, and those are rejected along with any other out-of-range reply.
func parseAccent(out string) (color.NRGBA, bool) {
	channels := accentDouble.FindAllStringSubmatch(out, -1)
	if len(channels) != 3 {
		match := accentTuple.FindStringSubmatch(out)
		if match == nil {
			return color.NRGBA{}, false
		}
		channels = [][]string{{"", match[1]}, {"", match[2]}, {"", match[3]}}
	}

	var rgb [3]uint8
	for i, channel := range channels {
		value, err := strconv.ParseFloat(channel[1], 64)
		if err != nil || value < 0 || value > 1 {
			return color.NRGBA{}, false
		}
		rgb[i] = uint8(math.Round(value * 255))
	}
	return color.NRGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: 0xFF}, true
}

// gnomeAccents are the colours GNOME 47 ships behind its named accents. The
// setting only carries the name, so the values have to live here.
var gnomeAccents = map[string]color.NRGBA{
	"blue":   {R: 0x35, G: 0x84, B: 0xE4, A: 0xFF},
	"teal":   {R: 0x21, G: 0x90, B: 0xA4, A: 0xFF},
	"green":  {R: 0x3A, G: 0x94, B: 0x4A, A: 0xFF},
	"yellow": {R: 0xC8, G: 0x88, B: 0x00, A: 0xFF},
	"orange": {R: 0xED, G: 0x5B, B: 0x00, A: 0xFF},
	"red":    {R: 0xE6, G: 0x2D, B: 0x42, A: 0xFF},
	"pink":   {R: 0xD5, G: 0x61, B: 0x99, A: 0xFF},
	"purple": {R: 0x91, G: 0x41, B: 0xAC, A: 0xFF},
	"slate":  {R: 0x6F, G: 0x83, B: 0x96, A: 0xFF},
}

// gsettingsAccent reads the GNOME preference, which reports a bare name such as
// `'teal'`.
func gsettingsAccent() (color.NRGBA, bool) {
	out, ok := run("gsettings", "get", "org.gnome.desktop.interface", "accent-color")
	if !ok {
		return color.NRGBA{}, false
	}

	name := strings.Trim(strings.TrimSpace(out), "'\"")
	accent, known := gnomeAccents[strings.ToLower(name)]
	return accent, known
}
