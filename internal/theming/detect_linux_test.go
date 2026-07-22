//go:build linux

package theming

import "testing"

func TestParseColorScheme(t *testing.T) {
	tests := []struct {
		name   string
		out    string
		want   Mode
		wantOK bool
	}{
		{name: "gdbus ReadOne dark", out: "(<uint32 1>,)\n", want: Dark, wantOK: true},
		{name: "gdbus Read light", out: "(<<uint32 2>>,)\n", want: Light, wantOK: true},
		{
			name: "dbus-send dark",
			out:  "method return time=1 sender=:1.5 -> destination=:1.9\n   variant       variant       uint32 1\n",
			want: Dark, wantOK: true,
		},
		{name: "no preference is not an answer", out: "(<uint32 0>,)\n", wantOK: false},
		{name: "portal error", out: "Requested setting not found\n", wantOK: false},
		{name: "empty", out: "", wantOK: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mode, ok := parseColorScheme(test.out)
			if ok != test.wantOK {
				t.Fatalf("parseColorScheme(%q) answered %v, want %v", test.out, ok, test.wantOK)
			}
			if ok && mode != test.want {
				t.Errorf("parseColorScheme(%q) = %v, want %v", test.out, mode, test.want)
			}
		})
	}
}

// A missing tool must be reported as "no answer" rather than stall or panic.
func TestRunMissingTool(t *testing.T) {
	if out, ok := run("trademan-no-such-desktop-tool"); ok {
		t.Errorf("run of a missing tool answered %q, want no answer", out)
	}
}

// Detection must always yield a usable mode, and Light whenever nothing knows.
func TestDetectSystemModeFallsBackToLight(t *testing.T) {
	t.Setenv("GTK_THEME", "")
	t.Setenv("QT_STYLE_OVERRIDE", "")

	mode, detected := DetectSystemMode()
	if !detected && mode != Light {
		t.Errorf("undetected system theme resolved to %v, want %v", mode, Light)
	}
	if mode != Light && mode != Dark {
		t.Errorf("DetectSystemMode returned unknown mode %v", mode)
	}
}

func TestGTKThemeNameDetectsDarkFromEnv(t *testing.T) {
	t.Setenv("GTK_THEME", "Adwaita:dark")

	mode, ok := gtkThemeName()
	if !ok || mode != Dark {
		t.Errorf("gtkThemeName() = %v, %v; want %v, true", mode, ok, Dark)
	}
}

func TestModeToggle(t *testing.T) {
	if got := Light.Toggle(); got != Dark {
		t.Errorf("Light.Toggle() = %v, want %v", got, Dark)
	}
	if got := Dark.Toggle(); got != Light {
		t.Errorf("Dark.Toggle() = %v, want %v", got, Light)
	}
}
