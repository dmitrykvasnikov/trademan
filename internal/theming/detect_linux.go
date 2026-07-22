//go:build linux

package theming

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// probeTimeout bounds every external lookup so a stalled desktop service can
// never hold up application start-up.
const probeTimeout = 700 * time.Millisecond

// DetectSystemMode reports the colour scheme configured for the desktop
// session. The probes are ordered from the most portable and authoritative
// source to the weakest hint, so the first one that answers wins.
//
// The second result is false when nothing could answer, in which case Light is
// returned as the documented fallback.
func DetectSystemMode() (Mode, bool) {
	probes := []func() (Mode, bool){
		portalGDBus,          // XDG desktop portal, works on any DE that ships one
		portalDBusSend,       // same setting, for sessions without gdbus
		gsettingsColorScheme, // GNOME/GTK explicit preference
		gtkThemeName,         // theme name carrying a "dark" suffix
	}
	for _, probe := range probes {
		if mode, ok := probe(); ok {
			return mode, true
		}
	}
	return Light, false
}

// colorSchemeValue matches the org.freedesktop.appearance color-scheme reply of
// both gdbus (`(<uint32 1>,)`) and dbus-send (`variant uint32 1`).
var colorSchemeValue = regexp.MustCompile(`uint32\s+(\d+)`)

// portalGDBus asks the XDG desktop portal through gdbus. ReadOne is the current
// method; Read is its deprecated predecessor, still the only one on older
// portal builds.
func portalGDBus() (Mode, bool) {
	const (
		dest   = "org.freedesktop.portal.Desktop"
		path   = "/org/freedesktop/portal/desktop"
		scheme = "org.freedesktop.appearance"
		key    = "color-scheme"
	)
	for _, method := range []string{"ReadOne", "Read"} {
		out, ok := run("gdbus", "call", "--session", "--dest", dest,
			"--object-path", path, "--method", "org.freedesktop.portal.Settings."+method,
			scheme, key)
		if !ok {
			continue
		}
		if mode, ok := parseColorScheme(out); ok {
			return mode, true
		}
	}
	return Light, false
}

// portalDBusSend reads the same portal setting with dbus-send.
func portalDBusSend() (Mode, bool) {
	out, ok := run("dbus-send", "--session", "--print-reply",
		"--dest=org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop",
		"org.freedesktop.portal.Settings.Read",
		"string:org.freedesktop.appearance", "string:color-scheme")
	if !ok {
		return Light, false
	}
	return parseColorScheme(out)
}

// parseColorScheme maps the portal's colour scheme enum: 1 prefers dark and 2
// prefers light. 0 means the user expressed no preference, which is not an
// answer, so the caller keeps probing.
func parseColorScheme(out string) (Mode, bool) {
	match := colorSchemeValue.FindStringSubmatch(out)
	if match == nil {
		return Light, false
	}
	switch match[1] {
	case "1":
		return Dark, true
	case "2":
		return Light, true
	}
	return Light, false
}

// gsettingsColorScheme reads the GNOME preference, which reports "default" when
// the user never made a choice.
func gsettingsColorScheme() (Mode, bool) {
	out, ok := run("gsettings", "get", "org.gnome.desktop.interface", "color-scheme")
	if !ok {
		return Light, false
	}
	switch {
	case strings.Contains(out, "prefer-dark"):
		return Dark, true
	case strings.Contains(out, "prefer-light"):
		return Light, true
	}
	return Light, false
}

// gtkThemeName treats a theme named like "Adwaita-dark" as a dark session. A
// name without that marker is not evidence of a deliberate light choice — it is
// also what an untouched desktop looks like — so only dark is reported here.
func gtkThemeName() (Mode, bool) {
	names := []string{os.Getenv("GTK_THEME"), os.Getenv("QT_STYLE_OVERRIDE")}
	if out, ok := run("gsettings", "get", "org.gnome.desktop.interface", "gtk-theme"); ok {
		names = append(names, out)
	}
	for _, name := range names {
		if strings.Contains(strings.ToLower(name), "dark") {
			return Dark, true
		}
	}
	return Light, false
}

// run executes a desktop lookup and returns its output, reporting false when
// the tool is missing, fails or times out.
func run(name string, args ...string) (string, bool) {
	if _, err := exec.LookPath(name); err != nil {
		return "", false
	}
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}
