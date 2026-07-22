//go:build linux

package theming

import (
	"image/color"
	"testing"
)

func TestParseAccent(t *testing.T) {
	tests := []struct {
		name   string
		out    string
		want   color.NRGBA
		wantOK bool
	}{
		{
			name:   "gdbus ReadOne",
			out:    "(<(0.2078431372549020, 0.5176470588235295, 0.8941176470588236)>,)\n",
			want:   color.NRGBA{R: 0x35, G: 0x84, B: 0xE4, A: 0xFF},
			wantOK: true,
		},
		{
			name:   "gdbus Read is double wrapped",
			out:    "(<<(1.0, 1.0, 1.0)>>,)\n",
			want:   color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
			wantOK: true,
		},
		{
			name: "dbus-send prints one double per line",
			out: "method return time=1755000000.5 sender=:1.5 -> destination=:1.9 serial=3\n" +
				"   variant       variant       struct {\n" +
				"         double 0.207843\n         double 0.517647\n         double 0.894118\n      }\n",
			want:   color.NRGBA{R: 0x35, G: 0x84, B: 0xE4, A: 0xFF},
			wantOK: true,
		},
		{name: "no accent set is reported as all negative", out: "(<(-1.0, -1.0, -1.0)>,)\n", wantOK: false},
		{name: "out of range", out: "(<(1.5, 0.5, 0.5)>,)\n", wantOK: false},
		{name: "portal error", out: "Requested setting not found\n", wantOK: false},
		{name: "empty", out: "", wantOK: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			accent, ok := parseAccent(test.out)
			if ok != test.wantOK {
				t.Fatalf("parseAccent(%q) answered %v, want %v", test.out, ok, test.wantOK)
			}
			if ok && accent != test.want {
				t.Errorf("parseAccent(%q) = %v, want %v", test.out, accent, test.want)
			}
		})
	}
}

// Detection must never hand back a colour it did not find.
func TestDetectSystemAccentReportsWhenUnknown(t *testing.T) {
	accent, found := DetectSystemAccent()
	if !found && accent != (color.NRGBA{}) {
		t.Errorf("undetected accent resolved to %v, want the zero colour", accent)
	}
	if found && accent.A != 0xFF {
		t.Errorf("detected accent %v is not opaque", accent)
	}
}

// Every named GNOME accent has to be usable as a primary colour in both modes.
func TestGNOMEAccentsAreUsable(t *testing.T) {
	for name, accent := range gnomeAccents {
		if accent.A != 0xFF {
			t.Errorf("accent %q is not opaque", name)
		}
		for _, p := range []palette{lightPalette, darkPalette} {
			fitted := legible(accent, p.Background)
			if ratio := contrastRatio(fitted, p.Background); ratio < 3.0 {
				t.Errorf("accent %q on %v has contrast %.2f:1, want at least 3.0:1", name, p.Background, ratio)
			}
		}
	}
}
